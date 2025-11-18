// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

	sharedConfg := `
identity: shared-identity
entitlements:
  com.apple.security.app-sandbox: true
`
	appConfig := `
CFBundleIdentifier: com.shared.bundle
CFBundleDisplayName: My App
`

	mergedConfig := `
identity: shared-identity
entitlements:
  com.apple.security.app-sandbox: true
CFBundleIdentifier: com.shared.bundle
CFBundleDisplayName: My App
`

	newConfigFile(t, tmpDir, "gobundle-shared.yaml", sharedConfg)
	newConfigFile(t, tmpDir, "gobundle-app.yaml", appConfig)

	// load from files in current directory.
	mergedYAML, err := readAndMergeConfigs()
	if err != nil {
		t.Fatalf("loadAndMergeConfigs failed: %v", err)
	}

	if got, want := parseConfig(t, mergedYAML), parseConfig(t, []byte(mergedConfig)); !reflect.DeepEqual(got, want) {
		t.Fatalf("merged config does not match expected:\nGot:\n%+v\nExpected:\n%+v", got, want)
	}
}

func parseConfig(t *testing.T, merged []byte) config {
	t.Helper()
	cfg, err := configForGoBuild("binary", merged)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	return cfg
}
