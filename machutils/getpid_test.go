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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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

func TestSubprocessError(t *testing.T) {
	testCases := []struct {
		name     string
		env      []string
		args     []string
		wantExit int
	}{
		{
			name:     "flag mock-ppid -1",
			args:     []string{"-mock-ppid", "-1"},
			wantExit: 1,
		},
		{
			name:     "flag mock-ppid non-existent pid",
			args:     []string{"-mock-ppid", "99999999"},
			wantExit: 1,
		},
		{
			name:     "env TEST_MOCK_PPID -1",
			env:      append(os.Environ(), "TEST_MOCK_PPID=-1"),
			wantExit: 1,
		},
		{
			name:     "env TEST_MOCK_PPID non-existent pid",
			env:      append(os.Environ(), "TEST_MOCK_PPID=99999999"),
			wantExit: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runSubprocess(t, tc.env, tc.args...)
			if err == nil {
				t.Fatalf("expected error, but command succeeded (stdout: %s)", stdout)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected *exec.ExitError, got: %T: %v", err, err)
			}
			if got, want := exitErr.ExitCode(), tc.wantExit; got != want {
				t.Errorf("got exit code %d, want %d", got, want)
			}
			if !strings.Contains(stderr, "failed to retrieve parent process UID from the kernel") {
				t.Errorf("expected stderr to contain error message, got %q", stderr)
			}
		})
	}
}

func TestDirectGetParentUID(t *testing.T) {
	uid, err := machutils.GetParentUID()
	if err != nil {
		t.Fatalf("GetParentUID failed: %v", err)
	}
	// The parent of the test process is also owned by the current user.
	if uid != uint32(os.Getuid()) {
		t.Errorf("got parent UID %d, want %d", uid, os.Getuid())
	}
}

func TestDirectGetParentUIDErrors(t *testing.T) {
	t.Run("mock ppid -1 via SetGetppidForTesting", func(t *testing.T) {
		restore := machutils.SetGetppidForTesting(func() int { return -1 })
		defer restore()

		uid, err := machutils.GetParentUID()
		if err == nil {
			t.Fatalf("expected error, got uid %d", uid)
		}
		if !errors.Is(err, machutils.ErrFailedToRetrieveParentUID) {
			t.Errorf("got error %v, want %v", err, machutils.ErrFailedToRetrieveParentUID)
		}
		if uid != 0 {
			t.Errorf("got uid %d on error, want 0", uid)
		}
	})

	t.Run("mock ppid 99999999 via SetGetppidForTesting", func(t *testing.T) {
		restore := machutils.SetGetppidForTesting(func() int { return 99999999 })
		defer restore()

		uid, err := machutils.GetParentUID()
		if err == nil {
			t.Fatalf("expected error, got uid %d", uid)
		}
		if !errors.Is(err, machutils.ErrFailedToRetrieveParentUID) {
			t.Errorf("got error %v, want %v", err, machutils.ErrFailedToRetrieveParentUID)
		}
		if uid != 0 {
			t.Errorf("got uid %d on error, want 0", uid)
		}
	})

	t.Run("env TEST_MOCK_PPID -1", func(t *testing.T) {
		t.Setenv("TEST_MOCK_PPID", "-1")

		uid, err := machutils.GetParentUID()
		if err == nil {
			t.Fatalf("expected error, got uid %d", uid)
		}
		if !errors.Is(err, machutils.ErrFailedToRetrieveParentUID) {
			t.Errorf("got error %v, want %v", err, machutils.ErrFailedToRetrieveParentUID)
		}
		if uid != 0 {
			t.Errorf("got uid %d on error, want 0", uid)
		}
	})

	t.Run("env TEST_MOCK_PPID non-numeric fallback", func(t *testing.T) {
		t.Setenv("TEST_MOCK_PPID", "invalid-int")

		uid, err := machutils.GetParentUID()
		if err != nil {
			t.Fatalf("expected success with invalid env var, got error: %v", err)
		}
		if uid != uint32(os.Getuid()) {
			t.Errorf("got parent UID %d, want %d", uid, os.Getuid())
		}
	})
}
