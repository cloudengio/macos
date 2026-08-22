// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils_test

import (
	"io/fs"
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
		{"trailing slash", valid + "/", true},
		{"trailing dot", valid + "/.", true},
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

	// An executable nested deeper than Contents/MacOS: the whole bundle is
	// searched, not just Contents/MacOS.
	nested := writeFile(t, filepath.Join(bundle, "Contents", "Library", "nested.app", "Contents", "MacOS", "nested-binary"), "#!/bin/sh\n", 0700)

	// A readable, non-executable resource, found only when the permission
	// predicate allows it.
	data := writeFile(t, filepath.Join(bundle, "Contents", "Resources", "data-file"), "data\n", 0600)

	for _, tc := range []struct {
		name    string
		bundle  string
		file    string
		matches func(fs.FileMode) bool
		want    string
	}{
		{"found in Contents/MacOS", bundle, "host-binary", macosutils.IsExecutable, wantPath},
		{"found nested", bundle, "nested-binary", macosutils.IsExecutable, nested},
		{"nil matchPerms defaults to executable", bundle, "host-binary", nil, wantPath},
		{"nil matchPerms rejects non-executable", bundle, "data-file", nil, ""},
		{"not executable", bundle, "data-file", macosutils.IsExecutable, ""},
		{"readable resource", bundle, "data-file", macosutils.IsReadable, data},
		{"executable is also readable", bundle, "host-binary", macosutils.IsReadable, wantPath},
		{"no such file", bundle, "no-such-binary", macosutils.IsExecutable, ""},
		{"not a bundle", tmpDir, "host-binary", macosutils.IsExecutable, ""},
		{"bundle does not exist", filepath.Join(tmpDir, "Missing.app"), "host-binary", macosutils.IsExecutable, ""},
	} {
		got, ok := macosutils.LocateInBundle(tc.bundle, tc.file, tc.matches)
		if got != tc.want {
			t.Errorf("%v: LocateInBundle(%q, %q) = %q, want %q", tc.name, tc.bundle, tc.file, got, tc.want)
		}
		if want := tc.want != ""; ok != want {
			t.Errorf("%v: LocateInBundle(%q, %q) ok = %v, want %v", tc.name, tc.bundle, tc.file, ok, want)
		}
	}
}

// TestIsExecutableIsReadable covers the permission predicates passed to
// LocateInBundle and ProcessInBundle.
func TestIsExecutableIsReadable(t *testing.T) {
	for _, tc := range []struct {
		mode                 fs.FileMode
		executable, readable bool
	}{
		{0700, true, true},
		{0600, false, true},
		{0500, true, true},
		{0400, false, true},
		{0100, true, false},
		{0000, false, false},
		{0111, true, false},
		{0444, false, true},
	} {
		if got, want := macosutils.IsExecutable(tc.mode), tc.executable; got != want {
			t.Errorf("IsExecutable(%o) = %v, want %v", tc.mode, got, want)
		}
		if got, want := macosutils.IsReadable(tc.mode), tc.readable; got != want {
			t.Errorf("IsReadable(%o) = %v, want %v", tc.mode, got, want)
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

	if got, ok := macosutils.LocateInBundle(bundle, "escapee", macosutils.IsExecutable); ok || got != "" {
		t.Errorf("LocateInBundle followed a symlink out of the bundle: got %q, %v", got, ok)
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
		{"trailing slashes in pathList dirs", "Tool.app", pathList(dirA+"/", dirB+"/"), []string{bundleA, bundleB}},
		{"path traversal rejected", "../../Tool.app", pathList(dirA, dirB), nil},
		{"nested relative bundle path rejected", "nested/Tool.app", pathList(dirA, dirB), nil},
		{"dot relative bundle rejected", ".", pathList(dirA, dirB), nil},
		{"dot dot relative bundle rejected", "..", pathList(dirA, dirB), nil},
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

// joinDirs builds a $PATH style list from dirs.
func joinDirs(dirs ...string) string {
	return strings.Join(dirs, string(filepath.ListSeparator))
}

func TestLookupBundleBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// A Tool.app containing no binary at all.
	dirNone := filepath.Join(tmpDir, "none")
	makeBundle(t, dirNone, "Tool.app")

	// A Tool.app whose tool-binary is not executable.
	dirNonExec := filepath.Join(tmpDir, "nonexec")
	nonExecBundle := makeBundle(t, dirNonExec, "Tool.app")
	writeFile(t, filepath.Join(nonExecBundle, "Contents", "MacOS", "tool-binary"), "data\n", 0600)

	// A Tool.app with the binary in the conventional location.
	dirB := filepath.Join(tmpDir, "b")
	bundleB := makeBundle(t, dirB, "Tool.app")
	binaryB := writeFile(t, filepath.Join(bundleB, "Contents", "MacOS", "tool-binary"), "#!/bin/sh\n", 0700)

	// A Tool.app with the binary nested deeper in the bundle.
	dirC := filepath.Join(tmpDir, "c")
	bundleC := makeBundle(t, dirC, "Tool.app")
	binaryC := writeFile(t, filepath.Join(bundleC, "Contents", "Library", "helpers", "tool-binary"), "#!/bin/sh\n", 0700)

	// A directory that is not a bundle, for the absolute path cases.
	dirOther := filepath.Join(tmpDir, "other")
	makeBundle(t, dirOther, "Other")

	for _, tc := range []struct {
		name       string
		bundle     string
		binary     string
		pathList   string
		wantBundle string
		wantBinary string
	}{
		{"found in the only bundle", "Tool.app", "tool-binary", joinDirs(dirB), bundleB, binaryB},
		{"found nested in the bundle", "Tool.app", "tool-binary", joinDirs(dirC), bundleC, binaryC},

		// The search continues past bundles that exist but do not contain a
		// usable binary; it is the first bundle *containing* it that wins, and
		// the bundle returned is that one, not the first on the path.
		{"skips bundle without the binary", "Tool.app", "tool-binary", joinDirs(dirNone, dirB), bundleB, binaryB},
		{"skips bundle with a non-executable match", "Tool.app", "tool-binary", joinDirs(dirNonExec, dirB), bundleB, binaryB},
		{"skips two unusable bundles", "Tool.app", "tool-binary", joinDirs(dirNone, dirNonExec, dirC), bundleC, binaryC},

		// When several bundles contain the binary, path order decides.
		{"first match wins", "Tool.app", "tool-binary", joinDirs(dirB, dirC), bundleB, binaryB},
		{"first match wins, reversed", "Tool.app", "tool-binary", joinDirs(dirC, dirB), bundleC, binaryC},
		{"duplicate dirs deduped", "Tool.app", "tool-binary", joinDirs(dirB, dirB), bundleB, binaryB},

		{"binary in no bundle", "Tool.app", "no-such-binary", joinDirs(dirNone, dirB, dirC), "", ""},
		{"bundle not on the path", "Missing.app", "tool-binary", joinDirs(dirB), "", ""},
		{"empty path list", "Tool.app", "tool-binary", "", "", ""},
		{"empty binary name", "Tool.app", "", joinDirs(dirB), "", ""},

		// Absolute bundles are used directly and the path list is ignored.
		{"absolute bundle", bundleB, "tool-binary", joinDirs(dirNone), bundleB, binaryB},
		{"absolute bundle without the binary", filepath.Join(dirNone, "Tool.app"), "tool-binary", joinDirs(dirB), "", ""},
		{"absolute non-bundle", filepath.Join(dirOther, "Other"), "tool-binary", joinDirs(dirB), "", ""},

		// Relative bundles must be bare names; anything else is rejected by
		// LookPathBundleAll before the filesystem is touched.
		{"path traversal rejected", "../b/Tool.app", "tool-binary", joinDirs(dirB), "", ""},
		{"nested relative bundle rejected", "b/Tool.app", "tool-binary", joinDirs(tmpDir), "", ""},
		{"dot rejected", ".", "tool-binary", joinDirs(dirB), "", ""},
		{"dot dot rejected", "..", "tool-binary", joinDirs(dirB), "", ""},
	} {
		gotBundle, gotBinary, ok := macosutils.LookupBundleBinary(tc.bundle, tc.binary, tc.pathList)
		if gotBundle != tc.wantBundle || gotBinary != tc.wantBinary {
			t.Errorf("%v: LookupBundleBinary(%q, %q, %q) = %q, %q, want %q, %q",
				tc.name, tc.bundle, tc.binary, tc.pathList,
				gotBundle, gotBinary, tc.wantBundle, tc.wantBinary)
		}
		if want := tc.wantBundle != ""; ok != want {
			t.Errorf("%v: LookupBundleBinary(%q, %q, %q) ok = %v, want %v",
				tc.name, tc.bundle, tc.binary, tc.pathList, ok, want)
		}
	}
}

// TestInBundle covers the parent-directory heuristic, including the nested
// bundle layout: a bundle nested inside another must live in the outer bundle's
// Contents/MacOS, so that InBundle(nested, "Contents", "MacOS") resolves to the
// outer bundle. A bundle placed in Contents/Library is not found this way.
func TestInBundle(t *testing.T) {
	tmpDir := t.TempDir()

	outer := makeBundle(t, tmpDir, "Outer.app")
	// The supported nesting: Outer.app/Contents/MacOS/Inner.app.
	inner := makeBundle(t, filepath.Join(outer, "Contents", "MacOS"), "Inner.app")
	innerExe := writeFile(t, filepath.Join(inner, "Contents", "MacOS", "inner-binary"), "#!/bin/sh\n", 0700)
	// The unsupported nesting: Outer.app/Contents/Library/Stray.app.
	stray := makeBundle(t, filepath.Join(outer, "Contents", "Library"), "Stray.app")

	outerExe := writeFile(t, filepath.Join(outer, "Contents", "MacOS", "outer-binary"), "#!/bin/sh\n", 0700)
	upper := makeBundle(t, tmpDir, "Upper.APP")
	upperExe := writeFile(t, filepath.Join(upper, "Contents", "MacOS", "upper-binary"), "#!/bin/sh\n", 0700)
	loose := writeFile(t, filepath.Join(tmpDir, "loose-binary"), "#!/bin/sh\n", 0700)

	for _, tc := range []struct {
		name    string
		path    string
		parents []string
		want    string
	}{
		{"executable in a bundle", outerExe, []string{"Contents", "MacOS"}, outer},
		{"executable in an uppercase bundle", upperExe, []string{"Contents", "MacOS"}, upper},
		{"executable in a nested bundle", innerExe, []string{"Contents", "MacOS"}, inner},
		{"nested bundle in Contents/MacOS", inner, []string{"Contents", "MacOS"}, outer},
		{"nested bundle in Contents/Library", stray, []string{"Contents", "MacOS"}, ""},
		{"nested bundle, matching parents", stray, []string{"Contents", "Library"}, outer},
		{"wrong parents", outerExe, []string{"Contents", "Resources"}, ""},
		{"no parents", outerExe, nil, ""},
		{"not in a bundle", loose, []string{"Contents", "MacOS"}, ""},
	} {
		got, ok := macosutils.InBundle(tc.path, tc.parents...)
		if got != tc.want {
			t.Errorf("%v: InBundle(%q, %v) = %q, want %q", tc.name, tc.path, tc.parents, got, tc.want)
		}
		if want := tc.want != ""; ok != want {
			t.Errorf("%v: InBundle(%q, %v) ok = %v, want %v", tc.name, tc.path, tc.parents, ok, want)
		}
	}
}

func TestExecutablePath(t *testing.T) {
	got, err := macosutils.ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ExecutablePath = %q, want an absolute path", got)
	}
	// Symlinks are resolved, so the result is its own resolution.
	resolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(%v): %v", got, err)
	}
	if got != resolved {
		t.Errorf("ExecutablePath = %q, want the symlink-resolved %q", got, resolved)
	}
}

// The test binary is not itself inside an app bundle, so ProcessInBundle must
// report that it is not. The positive case is covered by
// TestProcessInBundleFromBundle below.
func TestProcessInBundleNotInBundle(t *testing.T) {
	if os.Getenv(inBundleEnv) != "" {
		t.Skip("running inside the re-executed child")
	}
	if got, ok := macosutils.ProcessInBundle(); ok || got != "" {
		t.Errorf("ProcessInBundle = %q, %v, want \"\", false", got, ok)
	}
}

const (
	// inBundleEnv marks the re-executed child process.
	inBundleEnv = "CLOUDENG_MACOSUTILS_TEST_IN_BUNDLE"
	// childPlugin is the name of the executable the child searches for.
	childPlugin = "nested-plugin"
)

// TestProcessInBundleFromBundle copies the test binary into a synthetic app
// bundle and re-executes it there, so that os.Executable inside the child
// resolves to a path within a bundle. That is the only way to exercise the
// positive path of ProcessInBundle, which inspects the running executable
// rather than an argument.
func TestProcessInBundleFromBundle(t *testing.T) {
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

	cmd := exec.Command(hosted, "-test.run=TestProcessInBundleChild", "-test.v")
	cmd.Env = append(os.Environ(), inBundleEnv+"=1", childBundleEnv+"="+bundle)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("re-executed test failed: %v\n%s", err, out)
	}
	// Require the child's own result line: a skipped child would otherwise
	// still report an overall PASS and this test would prove nothing.
	if !strings.Contains(string(out), "--- PASS: TestProcessInBundleChild") {
		t.Errorf("re-executed test did not run TestProcessInBundleChild:\n%s", out)
	}
}

// childBundleEnv carries the bundle the child is expected to report.
const childBundleEnv = "CLOUDENG_MACOSUTILS_TEST_BUNDLE"

// TestProcessInBundleChild runs only inside the process re-executed by
// TestProcessInBundleFromBundle.
func TestProcessInBundleChild(t *testing.T) {
	if os.Getenv(inBundleEnv) == "" {
		t.Skip("only runs as the child of TestProcessInBundleFromBundle")
	}

	bundle, ok := macosutils.ProcessInBundle()
	if !ok {
		t.Fatalf("ProcessInBundle = %q, false; want this process's bundle", bundle)
	}
	// ProcessInBundle resolves symlinks, so resolve the expectation too:
	// macOS temp directories live under a symlinked /var.
	want := os.Getenv(childBundleEnv)
	if resolved, err := filepath.EvalSymlinks(want); err == nil {
		want = resolved
	}
	if bundle != want {
		t.Errorf("ProcessInBundle = %q, want %q", bundle, want)
	}

	// The bundle is then searched for the wanted file.
	got, ok := macosutils.LocateInBundle(bundle, childPlugin, macosutils.IsExecutable)
	if !ok {
		t.Fatalf("LocateInBundle(%q) = %q, false; want the nested plugin", childPlugin, got)
	}
	if want := filepath.Join("Contents", "Library", childPlugin); !strings.HasSuffix(got, want) {
		t.Errorf("LocateInBundle = %q, want a path ending in %q", got, want)
	}
	fi, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat(%v): %v", got, err)
	}
	if !fi.Mode().IsRegular() || !macosutils.IsExecutable(fi.Mode().Perm()) {
		t.Errorf("LocateInBundle returned %v, which is not an executable file", fi.Mode())
	}

	// A binary that is not in the bundle must not be found.
	if got, ok := macosutils.LocateInBundle(bundle, "no-such-plugin", macosutils.IsExecutable); ok || got != "" {
		t.Errorf("LocateInBundle(no-such-plugin) = %q, %v, want \"\", false", got, ok)
	}
}
