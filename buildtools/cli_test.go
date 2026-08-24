// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"errors"
	"testing"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

const cliConfig = `bundle: ./testing.app
signing:
    identity: "Apple Development: some id"
    entitlements:
      com.apple.security.app-sandbox: true
    perfile_entitlements:
      xpcHelper:
        com.apple.security.app-sandbox: false
      Contents/MacOS/xpcHelper:
        com.apple.security.app-sandbox: false
`

func TestCLIConfig(t *testing.T) {

	var cfg buildtools.Config
	if err := yaml.Unmarshal([]byte(cliConfig), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got, want := cfg.AppBundle, "./testing.app"; got != want {
		t.Fatalf("unexpected bundle, got %q, want %q", got, want)
	}
	if got, want := cfg.Signing.Identity, "Apple Development: some id"; got != want {
		t.Fatalf("unexpected signing identity, got %q, want %q", got, want)
	}
	if cfg.Signing.Entitlements == nil {
		t.Fatal("expected entitlements but got nil")
	}

	pl, err := cfg.Signing.Entitlements.MarshalIndent(" ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	expectedEntitlements := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
 </dict>
</plist>`

	if got, want := string(pl), expectedEntitlements; got != want {
		t.Errorf("unexpected entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}

	efor := func(p string) []byte {
		data, err := getEntitlements(cfg.Signing.PerFileEntitlements, p)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	pl = efor("xpcHelper")

	expectedPerFileEntitlements := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <false/>
 </dict>
</plist>`

	if got, want := string(pl), expectedPerFileEntitlements; got != want {
		t.Errorf("unexpected per file entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}

	pl = efor("Contents/MacOS/xpcHelper")

	if got, want := string(pl), expectedPerFileEntitlements; got != want {
		t.Errorf("unexpected per file entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}
}

type mockStep struct {
	exec string
	args []string
	err  error
}

func (m *mockStep) Run(_ context.Context, _ *buildtools.CommandRunner) (buildtools.StepResult, error) {
	return buildtools.NewStepResult(m.exec, m.args, []byte("mock output"), m.err), m.err
}

func TestStepRunner(t *testing.T) {
	ctx := t.Context()
	runner := buildtools.NewRunner(
		buildtools.WithStepTiming(true),
		buildtools.WithStepVerbose(true),
	)

	step1 := &mockStep{exec: "echo", args: []string{"hello"}}
	step2 := &mockStep{exec: "echo", args: []string{"world"}}
	runner.AddSteps(step1, step2)

	results := runner.Run(ctx, nil)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if err := results.Error(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got, want := results[0].CommandLine(), "echo hello "; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStepRunnerFailure(t *testing.T) {
	ctx := t.Context()
	runner := buildtools.NewRunner(buildtools.WithStepVerbose(true))

	stepErr := errors.New("command failed")
	step1 := &mockStep{exec: "failing-cmd", args: []string{"arg1"}, err: stepErr}
	step2 := &mockStep{exec: "unreachable", args: []string{"arg2"}}
	runner.AddSteps(step1, step2)

	results := runner.Run(ctx, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result due to early termination, got %d", len(results))
	}
	if !errors.Is(results.Error(), stepErr) {
		t.Errorf("got error %v, want %v", results.Error(), stepErr)
	}
}

func TestCommonFlagsStepRunnerOptions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flags     buildtools.CommonFlags
		wantCount int
	}{
		{"default", buildtools.CommonFlags{}, 0},
		{"timing only", buildtools.CommonFlags{Timing: true}, 1},
		{"verbose only", buildtools.CommonFlags{Verbose: true}, 1},
		{"both timing and verbose", buildtools.CommonFlags{Timing: true, Verbose: true}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.flags.StepRunnerOptions()
			if got := len(opts); got != tc.wantCount {
				t.Errorf("StepRunnerOptions() returned %d options, want %d", got, tc.wantCount)
			}
		})
	}
}

