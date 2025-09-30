// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

const plistYAML = `
CFBundleIdentifier: io.cloudeng.TestApp
CFBundleName: TestApp
CFBundleVersion: 1.0.0
CFBundleShortVersionString: 1.0
CFBundleExecutable: TestExecutable
CFBundlePackageType: APPL
LSMinimumSystemVersion: "15.0"
CFBundleDisplayName: Swift UI Example
`

func TestAppBundle(t *testing.T) {
	// Create a temporary directory for our test
	tempDir := t.TempDir()
	var info buildtools.InfoPlist
	if err := yaml.Unmarshal([]byte(plistYAML), &info); err != nil {
		t.Fatalf("failed to unmarshal info plist: %v", err)
	}

	// Define a simple app bundle
	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "TestApp.app"),
		Info: info,
	}

	// Create a command runner for executing steps
	runner := buildtools.NewCommandRunner()
	ctx := context.Background()

	// Execute the steps to create the bundle
	steps := bundle.Create()
	if len(steps) == 0 {
		t.Fatal("expected steps to create bundle, but got none")
	}
	steps = append(steps, bundle.WriteInfoPlist())

	// Execute each step
	for i, step := range steps {
		_, err := step.Run(ctx, runner)
		if err != nil {
			t.Fatalf("step %d failed with error: %v", i, err)
		}
	}

	// Verify the bundle structure
	requiredPaths := []string{
		bundle.Path,
		filepath.Join(bundle.Path, "Contents"),
		filepath.Join(bundle.Path, "Contents", "MacOS"),
		filepath.Join(bundle.Path, "Contents", "Resources"),
		filepath.Join(bundle.Path, "Contents", "Info.plist"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected path %q to exist, but it doesn't: %v", path, err)
		}
	}

	// Test copying content
	// Create a test file to copy into the bundle
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Copy the file into the bundle's Resources directory
	copyStep := bundle.CopyContents(testFile, "Resources", "test.txt")
	if _, err := copyStep.Run(ctx, runner); err != nil {
		t.Fatalf("copy step failed: %v", err)
	}

	// Verify the file was copied
	copiedPath := filepath.Join(bundle.Path, "Contents", "Resources", "test.txt")
	if _, err := os.Stat(copiedPath); err != nil {
		t.Fatalf("expected file %q to exist, but it doesn't: %v", copiedPath, err)
	}
}
