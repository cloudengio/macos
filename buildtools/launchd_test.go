// Copyright 2026 cloudeng llc. All rights reserved.
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

const testLabel = "io.cloudeng.buildtools-test"

// testAgent returns a LaunchAgent installed into a temporary directory and a
// fixed domain, so that tests never touch the real LaunchAgents directory.
func testAgent(t *testing.T) buildtools.LaunchAgent {
	t.Helper()
	return buildtools.LaunchAgent{
		Plist: buildtools.LaunchAgentPlist{
			Label:            testLabel,
			ProgramArguments: []string{"/bin/echo", "hello"},
			RunAtLoad:        true,
		},
		Dir:    t.TempDir(),
		Domain: "gui/501",
	}
}

// runSteps executes steps with a dry-run CommandRunner, which reports the
// commands that would be run without invoking launchctl, and returns them.
func runSteps(t *testing.T, steps []buildtools.Step) []string {
	t.Helper()
	runner := buildtools.NewCommandRunner(buildtools.WithDryRun(true))
	var cmds []string
	for _, s := range steps {
		res, err := s.Run(context.Background(), runner)
		if err != nil {
			t.Fatalf("step %v: %v", res.Executable(), err)
		}
		cmds = append(cmds, strings.TrimSpace(res.Executable()+" "+strings.Join(res.Args(), " ")))
	}
	return cmds
}

func TestLaunchAgentPaths(t *testing.T) {
	a := testAgent(t)

	path, err := a.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := path, filepath.Join(a.Dir, testLabel+".plist"); got != want {
		t.Errorf("PlistPath: got %v, want %v", got, want)
	}
	target, err := a.ServiceTarget()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := target, "gui/501/"+testLabel; got != want {
		t.Errorf("ServiceTarget: got %v, want %v", got, want)
	}
}

// TestLaunchAgentNoLabel verifies that a job with no Label, which has neither a
// file name nor a service target, is rejected rather than acted on.
func TestLaunchAgentNoLabel(t *testing.T) {
	a := buildtools.LaunchAgent{Dir: t.TempDir()}

	if _, err := a.PlistPath(); err == nil {
		t.Error("PlistPath: got nil error for a job with no Label")
	}
	if _, err := a.ServiceTarget(); err == nil {
		t.Error("ServiceTarget: got nil error for a job with no Label")
	}
	if a.IsInstalled() {
		t.Error("IsInstalled: got true for a job with no Label")
	}
	runner := buildtools.NewCommandRunner(buildtools.WithDryRun(true))
	for _, steps := range [][]buildtools.Step{a.Install(), a.Uninstall()} {
		if _, err := steps[0].Run(context.Background(), runner); err == nil {
			t.Error("got nil error for a job with no Label")
		}
	}
}

// TestLaunchAgentInstallValidates verifies that an invalid job is rejected
// before anything is written or loaded.
func TestLaunchAgentInstallValidates(t *testing.T) {
	a := buildtools.LaunchAgent{
		Plist: buildtools.LaunchAgentPlist{Label: testLabel}, // nothing to run
		Dir:   t.TempDir(),
	}
	steps := a.Install()
	if _, err := steps[0].Run(context.Background(), buildtools.NewCommandRunner()); err == nil {
		t.Fatal("Install: got nil error for a job with no ProgramArguments")
	}
	if a.IsInstalled() {
		t.Error("a rejected job was installed anyway")
	}
}

// TestLaunchAgentInstallSteps verifies the launchctl invocations Install and
// Uninstall issue, using a dry run so that launchd is never involved.
func TestLaunchAgentInstallSteps(t *testing.T) {
	a := testAgent(t)
	path, err := a.PlistPath()
	if err != nil {
		t.Fatal(err)
	}

	got := runSteps(t, a.Install())
	want := []string{
		"mkdir -p " + a.Dir,
		"write " + testLabel + ".plist " + path,
		"launchctl bootout gui/501/" + testLabel,
		"launchctl bootstrap gui/501 " + path,
	}
	for i, w := range want {
		if i >= len(got) || got[i] != w {
			t.Errorf("Install step %d: got %q, want %q (all: %v)", i, got[min(i, len(got)-1)], w, got)
		}
	}

	got = runSteps(t, a.Uninstall())
	want = []string{
		"launchctl bootout gui/501/" + testLabel,
		"remove " + testLabel + ".plist " + path,
	}
	for i, w := range want {
		if i >= len(got) || got[i] != w {
			t.Errorf("Uninstall step %d: got %q, want %q (all: %v)", i, got[min(i, len(got)-1)], w, got)
		}
	}
}

// TestLaunchAgentWriteAndRemove verifies the file side of install and
// uninstall for real, skipping the launchctl steps.
func TestLaunchAgentWriteAndRemove(t *testing.T) {
	a := testAgent(t)
	path, err := a.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	runner := buildtools.NewCommandRunner()

	if a.IsInstalled() {
		t.Fatal("IsInstalled: got true before installing")
	}
	// Steps 0 and 1 of Install create the directory and write the plist.
	for _, s := range a.Install()[:2] {
		if _, err := s.Run(context.Background(), runner); err != nil {
			t.Fatalf("install step: %v", err)
		}
	}
	if !a.IsInstalled() {
		t.Error("IsInstalled: got false after writing the plist")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<key>Label</key>", "<string>" + testLabel + "</string>",
		"<key>ProgramArguments</key>", "<string>/bin/echo</string>",
		"<key>RunAtLoad</key>", "<true/>",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("plist missing %q:\n%s", want, data)
		}
	}

	// The removal step of Uninstall deletes it, and is idempotent.
	remove := a.Uninstall()[1]
	for range 2 {
		if _, err := remove.Run(context.Background(), runner); err != nil {
			t.Fatalf("uninstall step: %v", err)
		}
	}
	if a.IsInstalled() {
		t.Error("IsInstalled: got true after removing the plist")
	}
}

func TestGUIDomain(t *testing.T) {
	if got, want := buildtools.GUIDomain(), "gui/"; !strings.HasPrefix(got, want) {
		t.Errorf("GUIDomain: got %v, want it to start with %v", got, want)
	}
}

// TestUserLaunchAgentsDir verifies the default install location.
func TestUserLaunchAgentsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := buildtools.UserLaunchAgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dir, filepath.Join(home, "Library", "LaunchAgents"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// An agent with no Dir installs there.
	a := buildtools.LaunchAgent{Plist: buildtools.LaunchAgentPlist{Label: testLabel}}
	path, err := a.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := path, filepath.Join(dir, testLabel+".plist"); got != want {
		t.Errorf("PlistPath: got %v, want %v", got, want)
	}
}

func TestLaunchAgentInvalidLabel(t *testing.T) {
	invalidLabels := []string{
		"../foo",
		"foo/bar",
		"foo\\bar",
		"a/b/c",
		"../../evil",
	}

	for _, label := range invalidLabels {
		a := buildtools.LaunchAgent{
			Plist: buildtools.LaunchAgentPlist{
				Label:            label,
				ProgramArguments: []string{"/bin/echo"},
			},
			Dir: t.TempDir(),
		}
		if err := a.Plist.Validate(); err == nil {
			t.Errorf("Validate() succeeded for invalid label %q", label)
		}
		if _, err := a.PlistPath(); err == nil {
			t.Errorf("PlistPath() succeeded for invalid label %q", label)
		}
		if _, err := a.ServiceTarget(); err == nil {
			t.Errorf("ServiceTarget() succeeded for invalid label %q", label)
		}
	}
}

func TestLaunchAgentProgramInExtra(t *testing.T) {
	// Valid Program in Extra
	aValid := buildtools.LaunchAgentPlist{
		Label: testLabel,
		Extra: map[string]any{"Program": "/bin/ls"},
	}
	if err := aValid.Validate(); err != nil {
		t.Errorf("expected valid plist with Program in Extra, got: %v", err)
	}

	// Invalid empty Program in Extra
	aEmpty := buildtools.LaunchAgentPlist{
		Label: testLabel,
		Extra: map[string]any{"Program": "   "},
	}
	if err := aEmpty.Validate(); err == nil {
		t.Error("expected error for empty Program in Extra, got nil")
	}

	// Invalid non-string Program in Extra
	aNonString := buildtools.LaunchAgentPlist{
		Label: testLabel,
		Extra: map[string]any{"Program": 12345},
	}
	if err := aNonString.Validate(); err == nil {
		t.Error("expected error for non-string Program in Extra, got nil")
	}
}

func TestLaunchAgentContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := testAgent(t)
	runner := buildtools.NewCommandRunner()

	// Status (which uses launchctlIgnoringError) should fail when context is cancelled
	statusStep := a.Status()
	if _, err := statusStep.Run(ctx, runner); err == nil {
		t.Error("Status: expected error for cancelled context, got nil")
	}

	// Uninstall step with context cancelled
	uninstSteps := a.Uninstall()
	for _, s := range uninstSteps {
		if _, err := s.Run(ctx, runner); err == nil {
			t.Error("Uninstall step: expected error for cancelled context, got nil")
		}
	}
}
