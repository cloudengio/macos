// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
	"howett.net/plist"
)

const plistWithXPCYAML = `
CFBundleIdentifier: io.cloudeng.TestApp
CFBundleName: TestApp
CFBundleVersion: 1.0.0
CFBundleShortVersionString: 1.0
CFBundleExecutable: TestExecutable
CFBundleIconFile: AppIcon
CFBundlePackageType: APPL
LSMinimumSystemVersion: "15.0"
CFBundleDisplayName: Swift UI Example
SomethingNew: SomeValue
XPCService:
  ServiceName: io.cloudeng.TestService
  ServiceType: Application
  ProcessType: Interactive
  ProgramArguments:
    - TestExecutable
    - --arg1
`

func TestInfoPlist(t *testing.T) {
	var info buildtools.InfoPlist
	if err := yaml.Unmarshal([]byte(plistWithXPCYAML), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got, want := info.CFBundleIdentifier, "io.cloudeng.TestApp"; got != want {
		t.Fatalf("unexpected CFBundleIdentifier, got %q, want %q", got, want)
	}
	if got, want := info.CFBundleExecutable, "TestExecutable"; got != want {
		t.Fatalf("unexpected CFBundleExecutable, got %q, want %q", got, want)
	}
	if got, want := info.CFBundleIconFile, "AppIcon"; got != want {
		t.Fatalf("unexpected CFBundleIconFile, got %q, want %q", got, want)
	}
	if got, want := info.XPCService.ServiceName, "io.cloudeng.TestService"; got != want {
		t.Fatalf("unexpected XPCService.ServiceName, got %q, want %q", got, want)
	}

	data, err := plist.MarshalIndent(info, plist.XMLFormat, "\t")
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
		"<key>ProcessType</key>",
		"<string>Interactive</string>",
		"<key>ProgramArguments</key>",
		"<array>",
		"<string>TestExecutable</string>",
		"<string>--arg1</string>",
		"</array>",
	}
	for _, e := range expected {
		if !strings.Contains(str, e) {
			t.Errorf("expected marshaled data to contain %q but it doesn't", e)
		}
	}
}

func TestStepExecution(t *testing.T) {
	// Create a test context
	ctx := context.Background()
	tmpDir := t.TempDir()
	runner := buildtools.NewCommandRunner()

	// Create a simple step that just checks if a file exists
	step := buildtools.StepFunc(func(_ context.Context, _ *buildtools.CommandRunner) (buildtools.StepResult, error) {
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
