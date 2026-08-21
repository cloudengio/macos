// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/macos/macosutils"
)

const testServiceLabel = "io.cloudeng.macosutils-test"

// launchAgents returns a temporary home directory, set as HOME for the duration
// of the test, and the LaunchAgents directory within it.
func launchAgents(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, "Library", "LaunchAgents")
}

func TestIsServiceInstalled(t *testing.T) {
	agents := launchAgents(t)

	if macosutils.IsServiceInstalled(testServiceLabel) {
		t.Error("reported installed with no LaunchAgents directory")
	}

	writeFile(t, filepath.Join(agents, testServiceLabel+".plist"), "<plist/>\n", 0600)

	if !macosutils.IsServiceInstalled(testServiceLabel) {
		t.Error("reported not installed with the plist present")
	}
	// The lookup is by label: another service's plist must not match.
	if macosutils.IsServiceInstalled(testServiceLabel + ".other") {
		t.Error("a different label matched an unrelated plist")
	}
}

// TestIsServiceInstalledNoHome verifies that an undiscoverable home directory
// reports "not installed" rather than failing.
func TestIsServiceInstalledNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	if macosutils.IsServiceInstalled(testServiceLabel) {
		t.Error("reported installed with no home directory")
	}
}

func TestIsServiceInstalledValidation(t *testing.T) {
	agents := launchAgents(t)

	writeFile(t, filepath.Join(agents, testServiceLabel+".plist"), "<plist/>\n", 0600)

	dirPlist := filepath.Join(agents, "directory-service.plist")
	if err := os.MkdirAll(dirPlist, 0700); err != nil {
		t.Fatal(err)
	}

	if macosutils.IsServiceInstalled("") {
		t.Error("IsServiceInstalled(\"\") reported true, want false")
	}
	if macosutils.IsServiceInstalled("directory-service") {
		t.Error("IsServiceInstalled(\"directory-service\") reported true for directory, want false")
	}
	if macosutils.IsServiceInstalled("../LaunchAgents/" + testServiceLabel) {
		t.Error("IsServiceInstalled with path traversal reported true, want false")
	}
	if macosutils.IsServiceInstalled("sub/dir") {
		t.Error("IsServiceInstalled with path separator reported true, want false")
	}
}
