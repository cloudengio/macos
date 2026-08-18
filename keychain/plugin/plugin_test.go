// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package plugin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	iofs "io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/macos/keychain"
	"cloudeng.io/macos/keychain/plugin"
	"cloudeng.io/security/keys/keychain/plugins"
)

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic("failed to get current working directory: " + err.Error())
	}
}

func TestPluginFlagsAndConfig(t *testing.T) {
	egp := filepath.Join(cwd, "testdata/example_plugin")
	args := []string{
		"--keychain-plugin=" + egp,
		"--keychain-app-bundle=not-there.app",
		"--keychain-plugin-bundle=not-there-plugin.app",
		"--keychain-type=data-protection-local",
		"--keychain-account=test-account",
		"--keychain-update-in-place=true",
		"--keychain-accessibility=when-unlocked",
	}
	var flagCfg plugin.WriteFlags
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := flags.RegisterFlagsInStruct(fs, "subcmd", &flagCfg, nil, nil); err != nil {
		t.Fatalf("failed to register flags: %v", err)
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	cfg, err := flagCfg.Config()
	if err != nil {
		t.Fatalf("failed to get config from flags: %v", err)
	}
	wantCfg := plugin.Config{
		Binary:               egp,
		KeychainBundle:       "not-there.app",
		KeychainPluginBundle: "not-there-plugin.app",
		OnlyUsePlugin:        false,
		Type:                 keychain.KeychainDataProtectionLocal,
		WriteType:            keychain.KeychainDataProtectionLocal,
		Account:              "test-account",
		UpdateInPlace:        true,
		Accessibility:        keychain.AccessibleWhenUnlocked,
	}

	if got, want := cfg, wantCfg; got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	args = []string{
		"--keychain-plugin=" + egp + "-not-there",
		"--keychain-app-bundle=not-there.app",
		"--keychain-type=data-protection-local",
		"--keychain-account=test-account",
		"--keychain-update-in-place=true",
		"--keychain-accessibility=when-unlocked",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	cfg, err = flagCfg.Config()
	if err != nil {
		t.Fatalf("unexpected error from Config: %v", err)
	}
	// Locating the plugin binary is deferred to FS. With OnlyUsePlugin=false a
	// missing plugin falls back to direct keychain access.
	if _, err := cfg.FS(false); err != nil {
		t.Fatalf("expected fallback to direct keychain fs, got: %v", err)
	}
	// With OnlyUsePlugin, a missing plugin binary is an error.
	cfg.OnlyUsePlugin = true
	if _, err := cfg.FS(false); err == nil {
		t.Fatalf("expected an error for missing plugin binary, got nil")
	}

	args = []string{
		"--keychain-type=all",
	}
	if err := fs.Parse(args); err == nil || !strings.Contains(err.Error(), `invalid value "all"`) {
		t.Errorf("expected an error regarding 'all' not allowed for writing: %v", err)
	}

}

func TestFSPluginFallback(t *testing.T) {
	// Both "not found" cases must fall back to direct access: a bare name not on
	// the PATH (exec.ErrNotFound) and an explicit path that does not exist
	// (fs.ErrNotExist).
	for _, tc := range []struct {
		name   string
		binary string
		want   error
	}{
		{"not-on-path", "definitely-not-a-real-keychain-plugin-xyz", exec.ErrNotFound},
		{"missing-path", filepath.Join(cwd, "testdata", "no-such-plugin"), iofs.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := plugin.Config{Binary: tc.binary, Type: keychain.KeychainAll}

			// OnlyUsePlugin=false falls back to direct, in-process keychain access.
			cfg.OnlyUsePlugin = false
			got, err := cfg.FS(false)
			if err != nil {
				t.Fatalf("expected fallback to direct keychain fs, got error: %v", err)
			}
			if got == nil {
				t.Fatal("expected a non-nil fs")
			}

			// OnlyUsePlugin=true makes a missing plugin an error.
			cfg.OnlyUsePlugin = true
			if _, err := cfg.FS(false); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got: %v", tc.want, err)
			}
		})
	}

	t.Run("non-executable-binary", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "not-executable")
		if err := os.WriteFile(tmpFile, []byte("#!/bin/sh\necho hi"), 0600); err != nil {
			t.Fatalf("writing temp file: %v", err)
		}
		cfg := plugin.Config{
			Binary:        tmpFile,
			Type:          keychain.KeychainAll,
			OnlyUsePlugin: false,
		}
		if _, err := cfg.FS(false); err == nil {
			t.Fatalf("expected error for non-executable binary, got nil")
		}
	})
}

func TestPluginReadRequest(t *testing.T) {
	ctx := t.Context()
	cfg := plugin.Config{
		Binary:        "not-needed-since-we-run-the-server-directly",
		Type:          keychain.KeychainDataProtectionLocal,
		Account:       "test-account",
		UpdateInPlace: true,
		Accessibility: keychain.AccessibleWhenUnlocked,
	}

	logBuf := &strings.Builder{}
	logger := slog.New(slog.NewTextHandler(logBuf, nil))
	ps := plugin.NewServer(plugin.WithLogger(logger))

	req, err := plugin.NewRequest("test_key", cfg)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	req.ID = 123
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	rCfg, rReq, resp := ps.ReadRequest(ctx, bytes.NewReader(data))
	if resp != nil {
		t.Fatalf("expected nil response, got %v (error: %v)", resp, resp.Error)
	}

	// Normalize expected request by round-tripping through JSON
	var expectedReq plugins.Request
	if err := json.Unmarshal(data, &expectedReq); err != nil {
		t.Fatalf("failed to unmarshal expected request: %v", err)
	}

	if got, want := rReq.ID, expectedReq.ID; got != want {
		t.Errorf("got request ID %v, want %v", got, want)
	}
	if got, want := rReq.Keyname, expectedReq.Keyname; got != want {
		t.Errorf("got request Keyname %v, want %v", got, want)
	}
	if got, want := rReq.Write, expectedReq.Write; got != want {
		t.Errorf("got request Write %v, want %v", got, want)
	}
	if !bytes.Equal(rReq.Contents, expectedReq.Contents) {
		t.Errorf("got request Contents %v, want %v", rReq.Contents, expectedReq.Contents)
	}

	// Normalize expected config by round-tripping through JSON to account for any
	// zero-values or type aliases that json.Marshal/Unmarshal might coerce.
	var expectedCfg plugin.Config
	cfgData, _ := json.Marshal(cfg)
	_ = json.Unmarshal(cfgData, &expectedCfg)

	if got, want := rCfg, &expectedCfg; !reflect.DeepEqual(got, want) {
		t.Errorf("got config %+v, want %+v", got, want)
	}

	logged := logBuf.String()
	checks := []string{
		"new request",
		"id=123",
		"test-account",
		"test_key",
		"data-protection",
		"when-unlocked",
		"write=false",
		"update_in_place=true",
	}
	for _, check := range checks {
		if !strings.Contains(logged, check) {
			t.Errorf("expected log to contain %q, got %q", check, logged)
		}
	}

}

func TestSendResponse(t *testing.T) {
	ctx := t.Context()
	logBuf := &strings.Builder{}
	logger := slog.New(slog.NewTextHandler(logBuf, nil))
	ps := plugin.NewServer(plugin.WithLogger(logger))

	resp := plugins.Response{
		ID:       123,
		Contents: []byte("test contents"),
		Error: &plugins.Error{
			Message: "test error",
			Detail:  "error details",
		},
	}

	var output strings.Builder
	ps.SendResponse(ctx, &output, &resp)
	logged := logBuf.String()
	if !strings.Contains(logged, "sent response") {
		t.Errorf("expected log to contain 'sent response', got %q", logged)
	}
	if !strings.Contains(logged, "id=123") {
		t.Errorf("expected log to contain 'id=123', got %q", logged)
	}

}

func TestReadFlags(t *testing.T) {
	egp := filepath.Join(cwd, "testdata/example_plugin")
	args := []string{
		"--keychain-plugin=" + egp,
		"--keychain-app-bundle=not-there.app",
		"--keychain-type=all",
		"--keychain-account=test-account",
	}
	var flagCfg plugin.ReadFlags
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := flags.RegisterFlagsInStruct(fs, "subcmd", &flagCfg, nil, nil); err != nil {
		t.Fatalf("failed to register flags: %v", err)
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	cfg, err := flagCfg.Config()
	if err != nil {
		t.Fatalf("failed to get config from flags: %v", err)
	}
	if got, want := cfg.Binary, filepath.Join(cwd, "testdata/example_plugin"); got != want {
		t.Errorf("got Binary %q, want %q", got, want)
	}
	if got, want := cfg.KeychainBundle, "not-there.app"; got != want {
		t.Errorf("got KeychainBundle %q, want %q", got, want)
	}
	if got, want := cfg.Type, keychain.KeychainAll; got != want {
		t.Errorf("got Type %v, want %v", got, want)
	}
	if got, want := cfg.Account, "test-account"; got != want {
		t.Errorf("got Account %q, want %q", got, want)
	}

	args = []string{
		"--keychain-plugin=" + egp + "-not-there",
		"--keychain-app-bundle=not-there.app",
		"--keychain-type=all",
		"--keychain-account=test-account",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	cfg, err = flagCfg.Config()
	if err != nil {
		t.Fatalf("unexpected error from Config: %v", err)
	}
	// Locating the plugin binary is deferred to FS. With OnlyUsePlugin=false a
	// missing plugin falls back to direct keychain access.
	if _, err := cfg.FS(false); err != nil {
		t.Fatalf("expected fallback to direct keychain fs, got: %v", err)
	}
	// With OnlyUsePlugin, a missing plugin binary is an error.
	cfg.OnlyUsePlugin = true
	if _, err := cfg.FS(false); err == nil {
		t.Fatalf("expected an error for missing plugin binary, got nil")
	}
}

func TestLocateKeychainBinaryInAppBundle(t *testing.T) {
	// 1. Non-bare relative path should return error immediately
	if _, _, err := plugin.LocateKeychainBinaryInAppBundle("sub/Tool.app", "tool"); err == nil || !strings.Contains(err.Error(), "must be a bare bundle name") {
		t.Errorf("expected bare bundle name error, got: %v", err)
	}

	// 2. Absolute bundle path that doesn't exist
	missingAbs := filepath.Join(t.TempDir(), "Missing.app")
	if _, _, err := plugin.LocateKeychainBinaryInAppBundle(missingAbs, "tool"); err == nil || !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("expected exec.ErrNotFound for missing absolute bundle, got: %v", err)
	}

	// 3. Bare bundle name not found with empty PATH
	t.Setenv("PATH", "")
	if _, _, err := plugin.LocateKeychainBinaryInAppBundle("NonExistent.app", "tool"); err == nil || !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("expected exec.ErrNotFound with empty PATH, got: %v", err)
	}
}
