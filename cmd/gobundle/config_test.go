// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

func newConfigFile(t *testing.T, dir, name, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

func TestLoadAndMergeConfigs(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	sharedConfig := `
identity: shared-identity
entitlements:
  com.apple.security.app-sandbox: true
`
	appConfig := `
info.plist:
  CFBundleIdentifier: com.shared.bundle
  CFBundleDisplayName: My App
`

	mergedConfig := `
identity: shared-identity
entitlements:
  com.apple.security.app-sandbox: true
info.plist:
  CFBundleIdentifier: com.shared.bundle
  CFBundleDisplayName: My App
`

	newConfigFile(t, tmpDir, "gobundle-shared.yaml", sharedConfig)
	newConfigFile(t, tmpDir, "gobundle-app.yaml", appConfig)

	// load from files in current directory.
	mergedYAML, err := readAndMergeConfigs()
	if err != nil {
		t.Fatalf("loadAndMergeConfigs failed: %v", err)
	}

	gotYAML, err := yaml.Marshal(parseConfig(t, mergedYAML))
	if err != nil {
		t.Fatalf("failed to marshal got config: %v", err)
	}
	wantYAML, err := yaml.Marshal(parseConfig(t, []byte(mergedConfig)))
	if err != nil {
		t.Fatalf("failed to marshal want config: %v", err)
	}

	if got, want := string(gotYAML), string(wantYAML); got != want {
		t.Fatalf("merged config does not match expected:\nGot:\n%v\nExpected:\n%v", got, want)
	}
}

func TestExpandEnv(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	t.Setenv("TEST_IDENTITY", "test-identity")
	t.Setenv("TEST_ENTITLEMENT", "true")
	t.Setenv("TEST_BUNDLE_ID", "com.test.bundle")

	sharedConfig := `
identity: ${TEST_IDENTITY}
entitlements:
  "com.apple.security.app-sandbox": "${TEST_ENTITLEMENT}"
`
	appConfig := `
info.plist:
  CFBundleIdentifier: ${TEST_BUNDLE_ID}
  CFBundleDisplayName: My App
`

	mergedConfig := `
identity: test-identity
entitlements:
  com.apple.security.app-sandbox: true
info.plist:
  CFBundleIdentifier: com.test.bundle
  CFBundleDisplayName: My App
`
	newConfigFile(t, tmpDir, "gobundle-shared.yaml", sharedConfig)
	newConfigFile(t, tmpDir, "gobundle-app.yaml", appConfig)

	// load from files in current directory.
	mergedYAML, err := readAndMergeConfigs()
	if err != nil {
		t.Fatalf("loadAndMergeConfigs failed: %v", err)
	}

	gotYAML, err := yaml.Marshal(parseConfig(t, mergedYAML))
	if err != nil {
		t.Fatalf("failed to marshal got config: %v", err)
	}
	wantYAML, err := yaml.Marshal(parseConfig(t, []byte(mergedConfig)))
	if err != nil {
		t.Fatalf("failed to marshal want config: %v", err)
	}

	if got, want := string(gotYAML), string(wantYAML); got != want {
		t.Fatalf("merged config does not match expected:\nGot:\n%v\nExpected:\n%v", got, want)
	}
}

func parseConfig(t *testing.T, merged []byte) config {
	t.Helper()
	cfg, err := configFromMerged(merged, "binary")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	return cfg
}

func TestLoadAndMergeConfigsNotarize(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	sharedConfig := `
identity: "Developer ID Application: Example Inc (TEAM123)"
notary:
  keychain_profile: my-notary-profile
`
	appConfig := `
notarize: true
info.plist:
  CFBundleIdentifier: com.shared.bundle
  CFBundleDisplayName: My App
`

	newConfigFile(t, tmpDir, "gobundle-shared.yaml", sharedConfig)
	newConfigFile(t, tmpDir, "gobundle-app.yaml", appConfig)

	mergedYAML, err := readAndMergeConfigs()
	if err != nil {
		t.Fatalf("readAndMergeConfigs failed: %v", err)
	}

	cfg := parseConfig(t, mergedYAML)
	if !cfg.Notarize {
		t.Errorf("expected Notarize to be true")
	}
	if got, want := cfg.Identity, "Developer ID Application: Example Inc (TEAM123)"; got != want {
		t.Errorf("Identity = %q, want %q", got, want)
	}
	if got, want := cfg.Notary.KeychainProfile, "my-notary-profile"; got != want {
		t.Errorf("Notary.KeychainProfile = %q, want %q", got, want)
	}
	if !cfg.Notary.Configured() {
		t.Errorf("expected Notary to be Configured()")
	}
}

func TestExpandEnvNotary(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	t.Setenv("NOTARY_PROFILE", "env-notary-profile")
	t.Setenv("NOTARY_ENABLE", "true")

	sharedConfig := `
notary:
  keychain_profile: ${NOTARY_PROFILE}
`
	appConfig := `
notarize: ${NOTARY_ENABLE}
info.plist:
  CFBundleIdentifier: com.env.bundle
`
	newConfigFile(t, tmpDir, "gobundle-shared.yaml", sharedConfig)
	newConfigFile(t, tmpDir, "gobundle-app.yaml", appConfig)

	mergedYAML, err := readAndMergeConfigs()
	if err != nil {
		t.Fatalf("readAndMergeConfigs failed: %v", err)
	}

	cfg := parseConfig(t, mergedYAML)
	if !cfg.Notarize {
		t.Errorf("expected Notarize to be true after env expansion")
	}
	if got, want := cfg.Notary.KeychainProfile, "env-notary-profile"; got != want {
		t.Errorf("Notary.KeychainProfile = %q, want %q", got, want)
	}
}

func TestCreateAndSignNotarizeValidation(t *testing.T) {
	ctx := t.Context()

	// 1. Notarize requested but no identity
	b := newBundle(config{
		Notarize: true,
		Notary: buildtools.NotaryConfig{
			KeychainProfile: "my-profile",
		},
	})
	err := b.createAndSign(ctx, "testbin", true)
	if err == nil || err.Error() != "notarize is set but the bundle is not signed: set an 'identity' in the config" {
		t.Errorf("expected missing identity error, got: %v", err)
	}

	// 2. Notarize requested with Apple Development identity
	b = newBundle(config{
		SigningConfig: buildtools.SigningConfig{
			Identity: "Apple Development: Developer (TEAM123)",
		},
		Notarize: true,
		Notary: buildtools.NotaryConfig{
			KeychainProfile: "my-profile",
		},
	})
	err = b.createAndSign(ctx, "testbin", true)
	if err == nil || !strings.Contains(err.Error(), "requires a 'Developer ID Application' identity") {
		t.Errorf("expected Developer ID identity requirement error, got: %v", err)
	}

	// 3. Notarize requested with no notary credentials
	b = newBundle(config{
		SigningConfig: buildtools.SigningConfig{
			Identity: "Developer ID Application: Developer (TEAM123)",
		},
		Notarize: true,
	})
	err = b.createAndSign(ctx, "testbin", true)
	if err == nil || !strings.Contains(err.Error(), "no notarization credentials are configured") {
		t.Errorf("expected unconfigured notary error, got: %v", err)
	}
}
