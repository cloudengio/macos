// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/macos/macosutils"
)

// writeFile creates path, and any missing parents, with the given contents
// and permissions.
func writeFile(t *testing.T, path, contents string, perm os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll(%v): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), perm); err != nil {
		t.Fatalf("WriteFile(%v): %v", path, err)
	}
	return path
}

// makeBundle creates a minimal, valid app bundle at dir/name: a directory
// with a .app suffix containing Contents/MacOS and Contents/Info.plist.
func makeBundle(t *testing.T, dir, name string) string {
	t.Helper()
	bundle := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(bundle, "Contents", "MacOS"), 0700); err != nil {
		t.Fatalf("MkdirAll(%v): %v", bundle, err)
	}
	writeFile(t, filepath.Join(bundle, "Contents", "Info.plist"), "<plist/>\n", 0600)
	return bundle
}

func TestIsAppBundle(t *testing.T) {
	tmpDir := t.TempDir()

	valid := makeBundle(t, tmpDir, "Valid.app")

	// A bundle whose suffix differs only in case: IsAppBundle lowercases
	// before testing the suffix.
	upper := makeBundle(t, tmpDir, "Upper.APP")

	// A directory with the right contents but no .app suffix.
	notSuffixed := makeBundle(t, tmpDir, "NoSuffix")

	// A .app directory with no Info.plist.
	noPlist := filepath.Join(tmpDir, "NoPlist.app")
	if err := os.MkdirAll(filepath.Join(noPlist, "Contents", "MacOS"), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// A .app directory whose Info.plist is itself a directory.
	plistIsDir := filepath.Join(tmpDir, "PlistIsDir.app")
	if err := os.MkdirAll(filepath.Join(plistIsDir, "Contents", "Info.plist"), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// A regular file named .app rather than a directory.
	fileBundle := writeFile(t, filepath.Join(tmpDir, "File.app"), "not a bundle", 0600)

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"valid", valid, true},
		{"uppercase suffix", upper, true},
		{"no .app suffix", notSuffixed, false},
		{"no Info.plist", noPlist, false},
		{"Info.plist is a directory", plistIsDir, false},
		{"regular file", fileBundle, false},
		{"does not exist", filepath.Join(tmpDir, "Missing.app"), false},
		{"empty path", "", false},
	} {
		if got, want := macosutils.IsAppBundle(tc.path), tc.want; got != want {
			t.Errorf("%v: IsAppBundle(%q) = %v, want %v", tc.name, tc.path, got, want)
		}
	}
}

func TestLocateInBundle(t *testing.T) {
	tmpDir := t.TempDir()
	bundle := makeBundle(t, tmpDir, "Host.app")

	// The binary being searched for, in the conventional location.
	wantPath := writeFile(t, filepath.Join(bundle, "Contents", "MacOS", "host-binary"), "#!/bin/sh\n", 0700)

	// A non-executable file, which must not be reported.
	writeFile(t, filepath.Join(bundle, "Contents", "Resources", "data-file"), "data\n", 0600)

	// An executable nested deeper than Contents/MacOS: the whole bundle is
	// searched, not just Contents/MacOS.
	nested := writeFile(t, filepath.Join(bundle, "Contents", "Library", "nested.app", "Contents", "MacOS", "nested-binary"), "#!/bin/sh\n", 0700)

	for _, tc := range []struct {
		name   string
		bundle string
		binary string
		want   string
	}{
		{"found in Contents/MacOS", bundle, "host-binary", wantPath},
		{"found nested", bundle, "nested-binary", nested},
		{"not executable", bundle, "data-file", ""},
		{"no such binary", bundle, "no-such-binary", ""},
		{"not a bundle", tmpDir, "host-binary", ""},
		{"bundle does not exist", filepath.Join(tmpDir, "Missing.app"), "host-binary", ""},
	} {
		if got, want := macosutils.LocateInBundle(tc.bundle, tc.binary), tc.want; got != want {
			t.Errorf("%v: LocateInBundle(%q, %q) = %q, want %q", tc.name, tc.bundle, tc.binary, got, want)
		}
	}
}

// A symlink inside a bundle pointing at an executable outside it must not be
// reported: the located binary is subsequently executed, so escaping the
// bundle would defeat the point of confining the walk.
func TestLocateInBundleSymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()
	outside := writeFile(t, filepath.Join(tmpDir, "outside", "escapee"), "#!/bin/sh\n", 0700)

	bundle := makeBundle(t, tmpDir, "Host.app")
	link := filepath.Join(bundle, "Contents", "MacOS", "escapee")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	if got := macosutils.LocateInBundle(bundle, "escapee"); got != "" {
		t.Errorf("LocateInBundle followed a symlink out of the bundle: got %q", got)
	}
}

func TestLookPathBundleAll(t *testing.T) {
	tmpDir := t.TempDir()

	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")
	dirEmpty := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(dirEmpty, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	bundleA := makeBundle(t, dirA, "Tool.app")
	bundleB := makeBundle(t, dirB, "Tool.app")
	// A directory that looks like a bundle by name only.
	makeBundle(t, dirB, "Other")

	pathList := func(dirs ...string) string {
		return strings.Join(dirs, string(filepath.ListSeparator))
	}

	for _, tc := range []struct {
		name     string
		bundle   string
		pathList string
		want     []string
	}{
		{"found in first dir", "Tool.app", pathList(dirA, dirEmpty), []string{bundleA}},
		{"found in second dir", "Tool.app", pathList(dirEmpty, dirB), []string{bundleB}},
		{"found in both, in order", "Tool.app", pathList(dirA, dirB), []string{bundleA, bundleB}},
		{"duplicate dirs deduped", "Tool.app", pathList(dirA, dirA, dirA), []string{bundleA}},
		{"not found", "Missing.app", pathList(dirA, dirB), nil},
		{"not a bundle", "Other", pathList(dirB), nil},
		{"empty path list", "Tool.app", "", nil},
		{"absolute bundle", bundleA, pathList(dirB), []string{bundleA}},
		{"absolute non-bundle", filepath.Join(dirB, "Other"), pathList(dirB), nil},
		{"absolute missing", filepath.Join(tmpDir, "Nope.app"), pathList(dirA), nil},
	} {
		got := macosutils.LookPathBundleAll(tc.bundle, tc.pathList)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v: LookPathBundleAll(%q) = %v, want %v", tc.name, tc.bundle, got, tc.want)
		}

		// LookPathBundle returns the first result, or "" when there are none.
		want := ""
		if len(tc.want) > 0 {
			want = tc.want[0]
		}
		if got, want := macosutils.LookPathBundle(tc.bundle, tc.pathList), want; got != want {
			t.Errorf("%v: LookPathBundle(%q) = %q, want %q", tc.name, tc.bundle, got, want)
		}
	}
}

// The test binary is not itself inside an app bundle, so InAppBundle must
// report that it is not. The positive case is covered by
// TestInAppBundleFromBundle below.
func TestInAppBundleNotInBundle(t *testing.T) {
	if os.Getenv(inBundleEnv) != "" {
		t.Skip("running inside the re-executed child")
	}
	if got, ok := macosutils.InAppBundle("anything"); ok || got != "" {
		t.Errorf("InAppBundle = %q, %v, want \"\", false", got, ok)
	}
}

const (
	// inBundleEnv marks the re-executed child process.
	inBundleEnv = "CLOUDENG_MACOSUTILS_TEST_IN_BUNDLE"
	// childPlugin is the name of the executable the child searches for.
	childPlugin = "nested-plugin"
)

// TestInAppBundleFromBundle copies the test binary into a synthetic app bundle
// and re-executes it there, so that os.Executable inside the child resolves to
// a path within a bundle. That is the only way to exercise the positive path of
// InAppBundle, which inspects the running executable rather than an argument.
func TestInAppBundleFromBundle(t *testing.T) {
	if os.Getenv(inBundleEnv) != "" {
		t.Skip("running inside the re-executed child")
	}

	exe, err := os.Executable()
	if err != nil {
		t.Skipf("cannot determine test executable: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Skipf("cannot read test executable %v: %v", exe, err)
	}

	tmpDir := t.TempDir()
	bundle := makeBundle(t, tmpDir, "Host.app")
	// makeBundle already created Contents/MacOS. The copy must be executable
	// so that it can be re-executed; it lives in this test's temp directory.
	hosted := filepath.Join(bundle, "Contents", "MacOS", "host")
	if err := os.WriteFile(hosted, data, 0700); err != nil { //nolint:gosec // G306: the copy must be executable.
		t.Fatalf("copying test executable: %v", err)
	}
	// The executable the child will look for inside its own bundle.
	writeFile(t, filepath.Join(bundle, "Contents", "Library", childPlugin), "#!/bin/sh\n", 0700)

	cmd := exec.Command(hosted, "-test.run=TestInAppBundleChild", "-test.v")
	cmd.Env = append(os.Environ(), inBundleEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("re-executed test failed: %v\n%s", err, out)
	}
	// Require the child's own result line: a skipped child would otherwise
	// still report an overall PASS and this test would prove nothing.
	if !strings.Contains(string(out), "--- PASS: TestInAppBundleChild") {
		t.Errorf("re-executed test did not run TestInAppBundleChild:\n%s", out)
	}
}

// TestInAppBundleChild runs only inside the process re-executed by
// TestInAppBundleFromBundle.
func TestInAppBundleChild(t *testing.T) {
	if os.Getenv(inBundleEnv) == "" {
		t.Skip("only runs as the child of TestInAppBundleFromBundle")
	}

	got, ok := macosutils.InAppBundle(childPlugin)
	if !ok {
		t.Fatalf("InAppBundle(%q) = %q, false; want the nested plugin", childPlugin, got)
	}
	if want := filepath.Join("Contents", "Library", childPlugin); !strings.HasSuffix(got, want) {
		t.Errorf("InAppBundle = %q, want a path ending in %q", got, want)
	}
	fi, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat(%v): %v", got, err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm()&0100 == 0 {
		t.Errorf("InAppBundle returned %v, which is not an executable file", fi.Mode())
	}

	// A binary that is not in the bundle must not be found.
	if got, ok := macosutils.InAppBundle("no-such-plugin"); ok || got != "" {
		t.Errorf("InAppBundle(no-such-plugin) = %q, %v, want \"\", false", got, ok)
	}
}
