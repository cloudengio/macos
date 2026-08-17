// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

// Command builder builds the keychain app bundle: the keychain client as the
// bundle's main executable, wrapping the macos-keychain-plugin as a nested,
// entitled helper bundle.
//
//	keychain.app/
//	  Contents/MacOS/keychain                          <- client, no entitlements
//	  Contents/Library/macos-keychain-plugin.app/      <- nested, entitled + profile
//	    Contents/MacOS/macos-keychain-plugin
//	    Contents/embedded.provisionprofile
//
// The client needs no entitlements or provisioning profile. The plugin holds the
// keychain-access-groups + app-sandbox entitlements, which — being
// provisioning-profile restricted — are only AMFI-authorized for a bundle's own
// main executable; hence the plugin is a nested bundle with its own profile.
//
// The nested plugin's signing identity, entitlements, provisioning profile and
// notary credentials are read from keychain-plugin/gobundle-app.yml, the same
// config used to build the plugin standalone with gobundle, so there is a single
// source of truth.
//
// Run via `go generate` (see keychain_cmd.go) or directly:
//
//	go run builder.go [-o keychain.app] [-notarize]
package main

import (
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"

	"cloudeng.io/macos/buildtools"
	"cloudeng.io/macos/keychain/plugin"
	"gopkg.in/yaml.v3"
)

const (
	clientExecutable = "cloudeng-keychain"
	pluginPackage    = "./cloudeng-keychain-plugin"
	pluginConfigYML  = "cloudeng-keychain-plugin/gobundle-app.yml"
	// pluginExecutable must match keychain/plugin.DefaultPluginBinary so the
	// client can locate the plugin inside the bundle.
	pluginExecutable = plugin.DefaultPluginBinary
	// nestedDir is where the plugin bundle lives within the outer app's Contents.
	nestedDir = "Library"
)

// bundleConfig mirrors the parts of the plugin's gobundle-app.yml that the
// nested plugin bundle needs.
type bundleConfig struct {
	buildtools.SigningConfig `yaml:",inline"`
	Info                     map[string]any          `yaml:"info.plist"`
	ProvisioningProfile      string                  `yaml:"profile"`
	Notarize                 bool                    `yaml:"notarize"`
	Notary                   buildtools.NotaryConfig `yaml:"notary"`
}

func main() {
	out := flag.String("o", plugin.DefaultKeyChainAppBundle, "output app bundle path")
	notarize := flag.Bool("notarize", false, "notarize and staple the bundle (slow; requires notary credentials)")
	flag.Parse()

	if err := run(*out, *notarize); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(out string, notarize bool) error {
	ctx := context.Background()

	cfg, err := loadPluginConfig(pluginConfigYML)
	if err != nil {
		return err
	}

	client, cleanupClient, err := build(ctx, ".", clientExecutable)
	if err != nil {
		return err
	}
	defer cleanupClient()
	plugin, cleanupPlugin, err := build(ctx, pluginPackage, pluginExecutable)
	if err != nil {
		return err
	}
	defer cleanupPlugin()

	// The nested plugin bundle keeps its own identifier from gobundle-app.yml; the
	// outer app takes a distinct one (macOS requires nested identifiers to differ)
	// and needs no provisioning profile.
	pluginID, _ := cfg.Info["CFBundleIdentifier"].(string)
	// A nested helper has no UI, so drop any icon reference.
	delete(cfg.Info, "CFBundleIconFile")

	innerInfo, err := buildInfoPlist(cfg.Info, pluginExecutable, pluginID)
	if err != nil {
		return err
	}
	outerInfo, err := buildInfoPlist(nil, clientExecutable, pluginID+".app")
	if err != nil {
		return err
	}

	return buildBundle(ctx, out, cfg, outerInfo, innerInfo, client, plugin, notarize)
}

func buildBundle(ctx context.Context, out string, cfg bundleConfig, outerInfo, innerInfo buildtools.InfoPlist, client, plugin string, notarize bool) error {
	outer := buildtools.AppBundle{Path: out, Info: outerInfo}
	inner := buildtools.AppBundle{
		Path: outer.Contents(nestedDir, pluginExecutable+".app"),
		Info: innerInfo,
	}

	runner := buildtools.NewRunner()
	runner.AddSteps(outer.Clean())
	runner.AddSteps(outer.Create()...)

	// Nested plugin bundle: the entitled executable, authorized by its own profile.
	runner.AddSteps(inner.Create()...)
	runner.AddSteps(inner.WriteInfoPlist(), inner.CopyExecutable(plugin))
	if cfg.ProvisioningProfile != "" {
		runner.AddSteps(inner.InstallProvisioningProfile(cfg.ProvisioningProfile))
	}

	// Outer client app.
	runner.AddSteps(outer.WriteInfoPlist(), outer.CopyExecutable(client))

	if cfg.Identity != "" {
		// Sign the nested plugin (with entitlements) fully first, then the client
		// (no entitlements), then seal the outer bundle.
		entitled := cfg.Signer()
		plain := buildtools.NewSigner(cfg.Identity, nil, nil, cfg.CodesignArguments)
		runner.AddSteps(
			inner.SignExecutable(entitled),
			inner.Sign(entitled),
			outer.SignExecutable(plain),
			outer.Sign(plain),
		)
	}

	if notarize {
		if cfg.Identity == "" {
			return fmt.Errorf("-notarize requires a signing identity in %s", pluginConfigYML)
		}
		if !cfg.Notary.Configured() {
			return fmt.Errorf("-notarize requires a notary section in %s", pluginConfigYML)
		}
		runner.AddSteps(outer.Notarize(cfg.Notary)...)
	}

	results := runner.Run(ctx, buildtools.NewCommandRunner())
	for _, r := range results {
		if r.Error() != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n%s\n", r.CommandLine(), r.Error(), r.Output())
		}
	}
	if err := results.Error(); err != nil {
		return err
	}
	fmt.Printf("created app bundle %s\n", out)
	return nil
}

// loadPluginConfig reads the plugin's gobundle-app.yml, expands ${ENV}
// references, and unmarshals the fields the nested bundle needs.
func loadPluginConfig(path string) (bundleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return bundleConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return bundleConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	expandEnv(raw)
	expanded, err := yaml.Marshal(raw)
	if err != nil {
		return bundleConfig{}, err
	}
	var cfg bundleConfig
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		return bundleConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Info == nil {
		cfg.Info = map[string]any{}
	}
	return cfg, nil
}

// expandEnv walks a decoded YAML value and expands ${ENV} references in every
// string, in place.
func expandEnv(v any) any {
	switch t := v.(type) {
	case string:
		return os.ExpandEnv(t)
	case map[string]any:
		for k, val := range t {
			t[k] = expandEnv(val)
		}
	case []any:
		for i, val := range t {
			t[i] = expandEnv(val)
		}
	}
	return v
}

// buildInfoPlist merges the caller-supplied Info.plist keys over the standard
// defaults for a launchable bundle with the given executable and identifier.
func buildInfoPlist(user map[string]any, executable, bundleID string) (buildtools.InfoPlist, error) {
	raw := map[string]any{
		"CFBundleExecutable":     executable,
		"CFBundleName":           executable,
		"CFBundleDisplayName":    executable,
		"CFBundleIdentifier":     bundleID,
		"CFBundlePackageType":    "APPL",
		"CFBundleVersion":        "0.0.0",
		"LSMinimumSystemVersion": "15.0", // macOS Sequoia
	}
	maps.Copy(raw, user)
	merged, err := yaml.Marshal(raw)
	if err != nil {
		return buildtools.InfoPlist{}, err
	}
	var info buildtools.InfoPlist
	if err := yaml.Unmarshal(merged, &info); err != nil {
		return buildtools.InfoPlist{}, fmt.Errorf("building Info.plist: %w", err)
	}
	return info, nil
}

// build compiles the Go package pkg to a temporary file named executable,
// returning its path and a cleanup function.
func build(ctx context.Context, pkg, executable string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "keychain-build-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	bin := dir + "/" + executable
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, pkg)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("building %s: %w", pkg, err)
	}
	return bin, cleanup, nil
}
