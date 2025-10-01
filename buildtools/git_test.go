// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/macos/buildtools"
)

func TestGitGetBranch(t *testing.T) {
	git := buildtools.NewGit("/tmp")

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "version with git key and branch",
			version:  "1.0.0+git:main",
			expected: "main",
		},
		{
			name:     "version with git key and HEAD",
			version:  "2.0.0+git:HEAD",
			expected: "HEAD",
		},
		{
			name:     "version without git key",
			version:  "1.0.0",
			expected: "",
		},
		{
			name:     "version with git key and feature branch",
			version:  "1.5.0+git:feature/new-feature",
			expected: "feature/new-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := git.GetBranch(tt.version)
			if result != tt.expected {
				t.Errorf("GetBranch(%q) = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

func TestGitReplaceBranch(t *testing.T) {
	git := buildtools.NewGit("/tmp")

	tests := []struct {
		name     string
		version  string
		buildID  string
		expected string
	}{
		{
			name:     "version with git key",
			version:  "1.0.0+git:main",
			buildID:  "abc12345",
			expected: "1.0.0+abc12345",
		},
		{
			name:     "version without git key",
			version:  "1.0.0",
			buildID:  "abc12345",
			expected: "1.0.0",
		},
		{
			name:     "version with git key and HEAD",
			version:  "2.0.0+git:HEAD",
			buildID:  "def67890",
			expected: "2.0.0+def67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := git.ReplaceBranch(tt.version, tt.buildID)
			if result != tt.expected {
				t.Errorf("ReplaceBranch(%q, %q) = %q, want %q", tt.version, tt.buildID, result, tt.expected)
			}
		})
	}
}

func TestGitHash(t *testing.T) {
	// Create a temporary directory for our test git repository
	tempDir := t.TempDir()

	// Initialize a git repository for testing
	ctx := context.Background()
	runner := buildtools.NewCommandRunner()
	ctx = buildtools.ContextWithCWD(ctx, tempDir)

	// Initialize git repo
	_, err := runner.Run(ctx, "git", "init")
	if err != nil {
		t.Skipf("git not available, skipping test: %v", err)
	}

	// Configure git user for the test
	_, err = runner.Run(ctx, "git", "config", "user.name", "Test User")
	if err != nil {
		t.Skipf("failed to configure git user.name: %v", err)
	}
	_, err = runner.Run(ctx, "git", "config", "user.email", "test@example.com")
	if err != nil {
		t.Skipf("failed to configure git user.email: %v", err)
	}

	// Create a test file and commit it
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runner.Run(ctx, "git", "add", "test.txt")
	if err != nil {
		t.Fatalf("failed to add file to git: %v", err)
	}

	_, err = runner.Run(ctx, "git", "commit", "-m", "Initial commit")
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create Git instance
	git := buildtools.NewGit(tempDir)

	// Test Hash function
	result, err := git.Hash(ctx, runner, "HEAD", 8)
	if err != nil {
		t.Fatalf("failed to get git hash: %v", err)
	}

	if result.Error() != nil {
		t.Fatalf("git hash command failed: %s", result.Error())
	}

	hash := strings.TrimSpace(result.Output())
	if len(hash) != 8 {
		t.Errorf("expected hash length of 8, got %d: %q", len(hash), hash)
	}

	// Verify it's a valid hex string
	for _, c := range hash {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("hash contains invalid character %q: %q", string(c), hash)
		}
	}

	// Test with different hash length
	result12, err := git.Hash(ctx, runner, "HEAD", 12)
	if err != nil {
		t.Fatalf("failed to get git hash with length 12: %v", err)
	}

	if result12.Error() != nil {
		t.Fatalf("git hash command failed: %s", result12.Error())
	}

	hash12 := strings.TrimSpace(result12.Output())
	if len(hash12) != 12 {
		t.Errorf("expected hash length of 12, got %d: %q", len(hash12), hash12)
	}

	// Verify shorter hash is prefix of longer hash
	if !strings.HasPrefix(hash12, hash) {
		t.Errorf("expected shorter hash %q to be prefix of longer hash %q", hash, hash12)
	}

	// Test with default length (0 should use 8)
	resultDefault, err := git.Hash(ctx, runner, "HEAD", 0)
	if err != nil {
		t.Fatalf("failed to get git hash with default length: %v", err)
	}

	if resultDefault.Error() != nil {
		t.Fatalf("git hash command failed: %s", resultDefault.Error())
	}

	hashDefault := strings.TrimSpace(resultDefault.Output())
	if len(hashDefault) != 8 {
		t.Errorf("expected default hash length of 8, got %d: %q", len(hashDefault), hashDefault)
	}

	if hashDefault != hash {
		t.Errorf("expected default hash %q to equal explicit 8-char hash %q", hashDefault, hash)
	}

	// Test with empty branch (should default to HEAD)
	resultEmpty, err := git.Hash(ctx, runner, "", 8)
	if err != nil {
		t.Fatalf("failed to get git hash with empty branch: %v", err)
	}

	if resultEmpty.Error() != nil {
		t.Fatalf("git hash command failed: %s", resultEmpty.Error())
	}

	hashEmpty := strings.TrimSpace(resultEmpty.Output())
	if hashEmpty != hash {
		t.Errorf("expected empty branch hash %q to equal HEAD hash %q", hashEmpty, hash)
	}
}
