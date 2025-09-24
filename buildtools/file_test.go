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
)

func TestFileOperations(t *testing.T) {
	// Create a temporary directory for our test
	tempDir := t.TempDir()

	// Create a command runner for executing steps
	runner := buildtools.NewCommandRunner()
	ctx := context.Background()

	// Test directory creation
	testDirPath := filepath.Join(tempDir, "test_dir", "nested")
	mkdirStep := buildtools.MkdirAll(testDirPath)

	result, err := mkdirStep.Run(ctx, runner)
	if err != nil {
		t.Fatalf("mkdir step failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(testDirPath); err != nil {
		t.Fatalf("expected directory %q to exist, but it doesn't: %v", testDirPath, err)
	}

	// Test file creation
	testContent := []byte("test content")
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	if err := os.WriteFile(testFilePath, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test file copy
	destPath := filepath.Join(testDirPath, "copied_file.txt")
	copyStep := buildtools.Copy(testFilePath, destPath)

	result, err = copyStep.Run(ctx, runner)
	if err != nil {
		t.Fatalf("copy step failed: %v", err)
	}

	// Verify file was copied
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("expected file %q to exist, but it doesn't: %v", destPath, err)
	}

	// Verify content is correct
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	if string(copiedContent) != string(testContent) {
		t.Fatalf("copied content does not match original, got %q, want %q", string(copiedContent), string(testContent))
	}

	// Test command execution with StepFunc
	cmdStep := buildtools.StepFunc(func(ctx context.Context, cmdRunner *buildtools.CommandRunner) (buildtools.StepResult, error) {
		return buildtools.NewStepResult("test command", []string{"arg1", "arg2"}, []byte("test output"), nil), nil
	})

	result, err = cmdStep.Run(ctx, runner)
	if err != nil {
		t.Fatalf("command step failed: %v", err)
	}

	if result.Executable() != "test command" {
		t.Fatalf("command mismatch, got %q, want %q", result.Executable(), "test command")
	}
	if len(result.Args()) != 2 || result.Args()[0] != "arg1" || result.Args()[1] != "arg2" {
		t.Fatalf("Args mismatch, got %v, want %v", result.Args(), []string{"arg1", "arg2"})
	}
	if string(result.Output()) != "test output" {
		t.Fatalf("output mismatch, got %q, want %q", string(result.Output()), "test output")
	}
}
