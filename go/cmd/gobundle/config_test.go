// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"

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
