// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package machutils_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/macos/machutils"
	"cloudeng.io/os/executil"
)

var testBinary string

func buildTestBinary() (string, string, error) {
	tmpDir, err := os.MkdirTemp("", "machutils-test")
	if err != nil {
		return "", "", err
	}
	bin, err := executil.GoBuild(context.Background(), filepath.Join(tmpDir, "machutils-test-bin"), "./testdata")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("failed to build test binary: %w", err)
	}
	return tmpDir, bin, nil
}

func TestMain(m *testing.M) {
	tmpDir, bin, err := buildTestBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	testBinary = bin
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func runSubprocess(t *testing.T, env []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestSubprocessGetParentUID(t *testing.T) {
	stdout, stderr, err := runSubprocess(t, nil)
	if err != nil {
		t.Fatalf("subprocess failed: %v (stderr: %s)", err, stderr)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output: %s", stderr)
	}
	gotStr := strings.TrimSpace(stdout)
	gotUID, err := strconv.ParseUint(gotStr, 10, 32)
	if err != nil {
		t.Fatalf("failed to parse UID %q: %v", gotStr, err)
	}
	wantUID := uint64(os.Getuid())
	if gotUID != wantUID {
		t.Errorf("got parent UID %d, want %d", gotUID, wantUID)
	}
}

func TestSubprocessCheckFlag(t *testing.T) {
	selfUID := os.Getuid()

	t.Run("matching UID", func(t *testing.T) {
		stdout, stderr, err := runSubprocess(t, nil, "-check-uid", strconv.Itoa(selfUID))
		if err != nil {
			t.Fatalf("subprocess failed: %v (stderr: %s)", err, stderr)
		}
		if stderr != "" {
			t.Errorf("unexpected stderr output: %s", stderr)
		}
		gotUID, err := strconv.ParseUint(strings.TrimSpace(stdout), 10, 32)
		if err != nil {
			t.Fatalf("failed to parse UID: %v", err)
		}
		if gotUID != uint64(selfUID) {
			t.Errorf("got UID %d, want %d", gotUID, selfUID)
		}
	})

	t.Run("mismatched UID", func(t *testing.T) {
		mismatchedUID := selfUID + 999
		_, stderr, err := runSubprocess(t, nil, "-check-uid", strconv.Itoa(mismatchedUID))
		if err == nil {
			t.Fatal("expected subprocess to fail with mismatched UID, but it succeeded")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *exec.ExitError, got: %T: %v", err, err)
		}
		if exitErr.ExitCode() != 2 {
			t.Errorf("got exit code %d, want 2", exitErr.ExitCode())
		}
		if !strings.Contains(stderr, "UID mismatch") {
			t.Errorf("expected stderr to contain 'UID mismatch', got %q", stderr)
		}
	})
}

func TestDirectGetParentUID(t *testing.T) {
	uid, err := machutils.GetParentUID()
	if err != nil {
		t.Fatalf("GetParentUID failed: %v", err)
	}
	// The parent of the test process is go test, which is owned by the
	// current user.
	if got, want := uid, uint32(os.Getuid()); got != want {
		t.Errorf("got parent UID %d, want %d", got, want)
	}
}

// TestSubprocessGetParentUIDOrphaned documents what GetParentUID reports when
// the process that launched the caller has exited. The kernel reparents an
// orphan to launchd, which runs as root, so the UID reported is root's rather
// than that of whoever actually started the process. Anything relying on
// GetParentUID as an identity has to account for this.
func TestSubprocessGetParentUIDOrphaned(t *testing.T) {
	out := filepath.Join(t.TempDir(), "parent-uid")

	// The subprocess starts a copy of itself and exits immediately, orphaning
	// the copy, which writes its report once it has been reparented.
	if _, stderr, err := runSubprocess(t, nil, "-orphan", out); err != nil {
		t.Fatalf("subprocess failed: %v (stderr: %s)", err, stderr)
	}

	var data []byte
	deadline := time.Now().Add(30 * time.Second)
	for {
		var err error
		if data, err = os.ReadFile(out); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the orphaned process did not report within the deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}

	fields := strings.Fields(strings.TrimSpace(string(data)))
	if len(fields) != 2 {
		t.Fatalf("got %q, want a parent PID and UID", strings.TrimSpace(string(data)))
	}
	if got, want := fields[0], "1"; got != want {
		t.Errorf("got parent PID %v, want %v, ie. reparented to launchd", got, want)
	}
	if fields[1] == "error" {
		t.Fatalf("GetParentUID failed for the reparented process: %q", strings.TrimSpace(string(data)))
	}
	gotUID, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		t.Fatalf("failed to parse UID %q: %v", fields[1], err)
	}
	// launchd runs as root, so the reported UID is 0 and not that of the user
	// who ran the tests.
	if gotUID != 0 {
		t.Errorf("got parent UID %d, want 0, the UID of launchd", gotUID)
	}
	if gotUID == uint64(os.Getuid()) && os.Getuid() != 0 {
		t.Errorf("got the launching user's UID %d, want launchd's", gotUID)
	}
}

// executableOwner returns the UID and permissions of path, read independently
// of the package under test.
func executableOwner(t *testing.T, path string) (uint32, fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %v: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %v: got %T, want *syscall.Stat_t", path, info.Sys())
	}
	return stat.Uid, info.Mode().Perm()
}

// cloneTestBinary copies the test binary so that it can be modified or removed
// without disturbing the tests that share it.
func cloneTestBinary(t *testing.T, perm fs.FileMode) string {
	t.Helper()
	data, err := os.ReadFile(testBinary)
	if err != nil {
		t.Fatalf("read %v: %v", testBinary, err)
	}
	clone := filepath.Join(t.TempDir(), "machutils-test-clone")
	if err := os.WriteFile(clone, data, perm); err != nil {
		t.Fatalf("write %v: %v", clone, err)
	}
	// WriteFile is subject to the umask, so set the mode explicitly.
	if err := os.Chmod(clone, perm); err != nil {
		t.Fatalf("chmod %v: %v", clone, err)
	}
	return clone
}

func runBinary(t *testing.T, bin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestDirectGetExecutableOwnerInfo(t *testing.T) {
	uid, perm, err := machutils.GetExecutableOwnerInfo()
	if err != nil {
		t.Fatalf("GetExecutableOwnerInfo failed: %v", err)
	}
	// The test binary was built by, and so is owned by, whoever is running
	// the tests.
	if got, want := uid, uint32(os.Getuid()); got != want {
		t.Errorf("got UID %d, want %d", got, want)
	}
	// It is running, so it must at least be executable by its owner, and the
	// permissions must be permission bits only.
	if perm&0100 == 0 {
		t.Errorf("got permissions %v, want the owner execute bit set", perm)
	}
	if perm&^fs.FileMode(0777) != 0 {
		t.Errorf("got %v, want permission bits only", perm)
	}
	// It reports properties of the executable file rather than anything that
	// varies between calls.
	againUID, againPerm, err := machutils.GetExecutableOwnerInfo()
	if err != nil {
		t.Fatalf("GetExecutableOwnerInfo failed on the second call: %v", err)
	}
	if againUID != uid || againPerm != perm {
		t.Errorf("got %d/%v on the second call, want %d/%v", againUID, againPerm, uid, perm)
	}
}

// TestSubprocessGetExecutableOwnerInfo verifies that a process reports the
// owner and permissions of its own executable: the subprocess discovers its
// path for itself, while the test stats the binary it launched, so the two are
// arrived at independently.
func TestSubprocessGetExecutableOwnerInfo(t *testing.T) {
	stdout, stderr, err := runSubprocess(t, nil, "-exec-uid")
	if err != nil {
		t.Fatalf("subprocess failed: %v (stderr: %s)", err, stderr)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output: %s", stderr)
	}
	fields := strings.Fields(strings.TrimSpace(stdout))
	if len(fields) != 2 {
		t.Fatalf("got %q, want a UID and permissions", strings.TrimSpace(stdout))
	}
	gotUID, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		t.Fatalf("failed to parse UID %q: %v", fields[0], err)
	}
	gotPerm, err := strconv.ParseUint(fields[1], 8, 32)
	if err != nil {
		t.Fatalf("failed to parse permissions %q: %v", fields[1], err)
	}

	wantUID, wantPerm := executableOwner(t, testBinary)
	if uint32(gotUID) != wantUID {
		t.Errorf("got UID %d, want %d, the owner of %v", gotUID, wantUID, testBinary)
	}
	if fs.FileMode(gotPerm) != wantPerm {
		t.Errorf("got permissions %v, want %v", fs.FileMode(gotPerm), wantPerm)
	}
}

// TestSubprocessGetExecutableOwnerInfoDeleted verifies that an executable which
// no longer exists is reported as an error rather than as a UID of zero, which
// would read as root.
func TestSubprocessGetExecutableOwnerInfoDeleted(t *testing.T) {
	clone := cloneTestBinary(t, 0700)
	stdout, stderr, err := runBinary(t, clone, "-exec-uid", "-delete-self")
	if err == nil {
		t.Fatalf("expected the subprocess to fail, it printed %q", stdout)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	// Exit code 3 is reserved by the test binary for a failure to read the
	// executable's owner, distinguishing it from the other failure modes.
	if got, want := exitErr.ExitCode(), 3; got != want {
		t.Errorf("got exit code %d, want %d (stderr: %s)", got, want, stderr)
	}
	if !strings.Contains(stderr, "no such file") {
		t.Errorf("got stderr %q, want it to report the missing executable", stderr)
	}
	if got := strings.TrimSpace(stdout); got != "" {
		t.Errorf("got stdout %q, want nothing to be printed on failure", got)
	}
}

// TestEnsureParentProcessSafe verifies the check that an executable can only
// be launched by its owner. The test process is the parent of the subprocess
// and both are owned by the current user, so the outcome turns on the
// executable's permissions.
func TestEnsureParentProcessSafe(t *testing.T) {
	for _, tc := range []struct {
		name string
		perm fs.FileMode
		want string // empty when the process is expected to be safe
	}{
		{"owner only", 0700, ""},
		{"owner read and execute", 0500, ""},
		{"group executable", 0710, "group or world executable"},
		{"world executable", 0701, "group or world executable"},
		{"world readable and executable", 0755, "group or world executable"},
	} {
		clone := cloneTestBinary(t, tc.perm)
		stdout, stderr, err := runBinary(t, clone, "-ensure-safe")
		if tc.want == "" {
			if err != nil {
				t.Errorf("%v: expected the process to be safe: %v (stderr: %s)", tc.name, err, stderr)
				continue
			}
			if got := strings.TrimSpace(stdout); got != "safe" {
				t.Errorf("%v: got %q, want %q", tc.name, got, "safe")
			}
			continue
		}
		if err == nil {
			t.Errorf("%v: expected the process to be rejected, it printed %q", tc.name, stdout)
			continue
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Errorf("%v: expected *exec.ExitError, got %T: %v", tc.name, err, err)
			continue
		}
		// Exit code 4 is reserved by the test binary for an unsafe parent.
		if got, want := exitErr.ExitCode(), 4; got != want {
			t.Errorf("%v: got exit code %d, want %d (stderr: %s)", tc.name, got, want, stderr)
		}
		if !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: got stderr %q, want it to contain %q", tc.name, stderr, tc.want)
		}
	}
}
