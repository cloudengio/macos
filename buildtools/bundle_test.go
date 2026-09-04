// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"io/fs"
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
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
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

func TestWriteInfoPlistGitBuild(t *testing.T) {
	tempDir := t.TempDir()
	var info buildtools.InfoPlist
	if err := yaml.Unmarshal([]byte(plistYAML), &info); err != nil {
		t.Fatalf("failed to unmarshal info plist: %v", err)
	}

	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "TestApp.app"),
		Info: info,
	}

	runner := buildtools.NewCommandRunner()
	ctx := context.Background()

	for _, step := range bundle.Create() {
		if _, err := step.Run(ctx, runner); err != nil {
			t.Fatalf("bundle create failed: %v", err)
		}
	}

	// Case 1: CFBundleVersion has no branch pattern ("1.0.0").
	// Previously this deadlocked because versionCh was never closed.
	git := buildtools.NewGit(".")
	steps := bundle.WriteInfoPlistGitBuild(ctx, git)
	for i, step := range steps {
		if _, err := step.Run(ctx, runner); err != nil {
			t.Fatalf("step %d failed: %v", i, err)
		}
	}

	// Case 2: CFBundleVersion specifies branch ("1.0.0+git:HEAD")
	bundleWithGit := bundle
	bundleWithGit.Info.CFBundleVersion = "1.0.0+git:HEAD"
	gitSteps := bundleWithGit.WriteInfoPlistGitBuild(ctx, git)
	for i, step := range gitSteps {
		if _, err := step.Run(ctx, runner); err != nil {
			t.Fatalf("git step %d failed: %v", i, err)
		}
	}
}

func TestAppBundlePermissions(t *testing.T) {
	tempDir := t.TempDir()
	var info buildtools.InfoPlist
	if err := yaml.Unmarshal([]byte(plistYAML), &info); err != nil {
		t.Fatalf("failed to unmarshal info plist: %v", err)
	}

	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "TestApp.app"),
		Info: info,
	}

	runner := buildtools.NewCommandRunner()
	ctx := context.Background()

	for _, step := range bundle.Create() {
		if _, err := step.Run(ctx, runner); err != nil {
			t.Fatalf("bundle create failed: %v", err)
		}
	}

	// Create a dummy executable file
	exePath := filepath.Join(bundle.Path, "Contents", "MacOS", info.CFBundleExecutable)
	if err := os.WriteFile(exePath, []byte("#!/bin/sh\necho ok\n"), 0600); err != nil {
		t.Fatalf("failed to create dummy executable: %v", err)
	}

	// Set executable permissions to 0700
	stepExe := bundle.SetExecutablePermissions("", 0700)
	if _, err := stepExe.Run(ctx, runner); err != nil {
		t.Fatalf("SetExecutablePermissions failed: %v", err)
	}
	fi, err := os.Stat(exePath)
	if err != nil {
		t.Fatalf("stat executable failed: %v", err)
	}
	if got, want := fi.Mode().Perm(), fs.FileMode(0700); got != want {
		t.Errorf("executable permissions = %04o, want %04o", got, want)
	}

	// Set MacOS dir permissions to 0750
	dirPath := filepath.Join(bundle.Path, "Contents", "MacOS")
	stepDir := bundle.SetMacOSDirPermissions(0750)
	if _, err := stepDir.Run(ctx, runner); err != nil {
		t.Fatalf("SetMacOSDirPermissions failed: %v", err)
	}
	dirFi, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("stat MacOS dir failed: %v", err)
	}
	if got, want := dirFi.Mode().Perm(), fs.FileMode(0750); got != want {
		t.Errorf("MacOS dir permissions = %04o, want %04o", got, want)
	}

	// Passing dirFi.Mode() directly (which includes os.ModeDir) should succeed
	stepDirMode := bundle.SetMacOSDirPermissions(dirFi.Mode())
	if _, err := stepDirMode.Run(ctx, runner); err != nil {
		t.Fatalf("SetMacOSDirPermissions with ModeDir failed: %v", err)
	}
}

func TestAppBundleSetExecutablePermissionsErrors(t *testing.T) {
	tempDir := t.TempDir()
	runner := buildtools.NewCommandRunner()
	ctx := context.Background()

	// Error case: both CFBundleExecutable and src are empty
	emptyBundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "Empty.app"),
	}
	stepEmpty := emptyBundle.SetExecutablePermissions("", 0700)
	if _, err := stepEmpty.Run(ctx, runner); err == nil {
		t.Fatal("expected SetExecutablePermissions to fail when both CFBundleExecutable and src are empty")
	}

	// Fallback case: CFBundleExecutable is empty, but src is provided
	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "Fallback.app"),
	}
	for _, step := range bundle.Create() {
		if _, err := step.Run(ctx, runner); err != nil {
			t.Fatalf("bundle create failed: %v", err)
		}
	}
	customExe := filepath.Join(bundle.Path, "Contents", "MacOS", "custom_bin")
	if err := os.WriteFile(customExe, []byte("#!/bin/sh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stepFallback := bundle.SetExecutablePermissions("/path/to/custom_bin", 0755)
	if _, err := stepFallback.Run(ctx, runner); err != nil {
		t.Fatalf("SetExecutablePermissions with src fallback failed: %v", err)
	}
	customFi, err := os.Stat(customExe)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := customFi.Mode().Perm(), fs.FileMode(0755); got != want {
		t.Errorf("custom_bin permissions = %04o, want %04o", got, want)
	}
}
