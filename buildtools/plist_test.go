// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"os"
	"testing"

	"cloudeng.io/macos/buildtools"
)

func TestInfoPlist(t *testing.T) {
	info := buildtools.InfoPlist{
		Identifier:   "io.cloudeng.TestApp",
		Name:         "TestApp",
		Version:      "1.0.0",
		ShortVersion: "1.0",
		Executable:   "TestExecutable",
		IconSet:      "AppIcon",
		Type:         "APPL",
		XPCService: buildtools.XPCInfoPlist{
			ServiceName: "io.cloudeng.TestService",
			ServiceType: "Application",
			ProcessType: "Interactive",
			ProgramArguments: []string{
				"TestExecutable", "--arg1", "--arg2",
			},
		},
	}

	data, err := buildtools.MarshalInfoPlist(info)
	if err != nil {
		t.Fatalf("failed to marshal info plist: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("marshaled data is empty")
	}

	// Simple validation that it contains expected values
	str := string(data)
	expected := []string{
		"<key>CFBundleIdentifier</key>",
		"<string>io.cloudeng.TestApp</string>",
		"<key>CFBundleName</key>",
		"<string>TestApp</string>",
		"<key>XPCService</key>",
	}

	for _, e := range expected {
		if !contains(str, e) {
			t.Errorf("expected marshaled data to contain %q but it doesn't", e)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	// Standard library's strings.Contains is better for this purpose in real code
	for i := 0; i <= len(s)-len(substr); i++ {
		found := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

func TestStepExecution(t *testing.T) {
	// Create a test context
	ctx := context.Background()
	tmpDir := t.TempDir()
	runner := buildtools.NewCommandRunner()

	// Create a simple step that just checks if a file exists
	step := buildtools.StepFunc(func(ctx context.Context, cmdRunner *buildtools.CommandRunner) (buildtools.StepResult, error) {
		_, err := os.Stat(tmpDir)
		return buildtools.NewStepResult("stat", []string{tmpDir}, nil, err), err
	})

	// Execute the step
	result, err := step.Run(ctx, runner)
	if err != nil {
		t.Fatalf("step execution failed: %v", err)
	}

	if result.Executable() != "stat" {
		t.Fatal("step execution returned unexpected executable")
	}
}
