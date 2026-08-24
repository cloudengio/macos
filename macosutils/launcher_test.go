// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudeng.io/macos/macosutils"
)

// shell is used rather than a fixture binary so that the tests exercise real
// process start, exit-status and signal handling.
const shell = "/bin/sh"

func TestRunApp(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()

	// Combined output: both streams are captured.
	out, err := l.RunApp(ctx, shell, "-c", "echo to-stdout; echo to-stderr >&2")
	if err != nil {
		t.Fatalf("RunApp: %v", err)
	}
	for _, want := range []string{"to-stdout", "to-stderr"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q does not contain %q", out, want)
		}
	}
}

func TestRunAppFailure(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()

	out, err := l.RunApp(ctx, shell, "-c", "echo failing >&2; exit 7")
	if err == nil {
		t.Fatal("RunApp: got nil error, want a failure")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("RunApp: got %v, want an *exec.ExitError", err)
	}
	if got, want := exitErr.ExitCode(), 7; got != want {
		t.Errorf("exit code: got %d, want %d", got, want)
	}
	// The output produced before the failure is still returned.
	if !strings.Contains(out, "failing") {
		t.Errorf("output %q does not contain the command's output", out)
	}
}

// TestRunAppWithStdoutStderr verifies that RunApp can be used on a Launcher
// configured with WithStdoutStderr to capture combined output without conflict.
func TestRunAppWithStdoutStderr(t *testing.T) {
	ctx := t.Context()
	var stdout, stderr bytes.Buffer
	l := macosutils.NewLauncher(macosutils.WithStdoutStderr(&stdout, &stderr))

	out, err := l.RunApp(ctx, shell, "-c", "echo to-stdout; echo to-stderr >&2")
	if err != nil {
		t.Fatalf("RunApp: %v", err)
	}
	for _, want := range []string{"to-stdout", "to-stderr"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q does not contain %q", out, want)
		}
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("stdout/stderr buffers unexpectedly received RunApp output: %q, %q", stdout.String(), stderr.String())
	}
}

func TestRunAppContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	l := macosutils.NewLauncher()

	_, err := l.RunApp(ctx, shell, "-c", "sleep 10")
	if err == nil {
		t.Fatal("RunApp: got nil error, want context canceled error")
	}
}

func TestLaunchAppStdoutStderr(t *testing.T) {
	ctx := t.Context()
	var stdout, stderr bytes.Buffer
	l := macosutils.NewLauncher(macosutils.WithStdoutStderr(&stdout, &stderr))

	if err := l.LaunchApp(ctx, shell, "-c", "echo to-stdout; echo to-stderr >&2"); err != nil {
		t.Fatalf("LaunchApp: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "to-stdout"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(stderr.String()), "to-stderr"; got != want {
		t.Errorf("stderr: got %q, want %q", got, want)
	}
}

func TestLaunchAppCmdEnv(t *testing.T) {
	ctx := t.Context()
	var stdout bytes.Buffer
	l := macosutils.NewLauncher(
		macosutils.WithCmdEnv(func() []string { return []string{"LAUNCHER_TEST=set-by-option"} }),
		macosutils.WithStdoutStderr(&stdout, io.Discard))

	if err := l.LaunchApp(ctx, shell, "-c", "echo $LAUNCHER_TEST"); err != nil {
		t.Fatalf("LaunchApp: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "set-by-option"; got != want {
		t.Errorf("child env: got %q, want %q", got, want)
	}
}

func TestLaunchAppWorkingDir(t *testing.T) {
	ctx := t.Context()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	l := macosutils.NewLauncher(
		macosutils.WithWorkingDir(dir),
		macosutils.WithStdoutStderr(&stdout, io.Discard))

	if err := l.LaunchApp(ctx, shell, "-c", "pwd -P"); err != nil {
		t.Fatalf("LaunchApp: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), dir; got != want {
		t.Errorf("working dir: got %q, want %q", got, want)
	}
}

// TestLaunchAppFailedToLaunch verifies that a command that never starts is
// reported with ErrFailedToLaunch, and that the underlying error survives so
// callers can inspect the cause.
func TestLaunchAppFailedToLaunch(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()

	err := l.LaunchApp(ctx, filepath.Join(t.TempDir(), "no-such-binary"))
	if !errors.Is(err, macosutils.ErrFailedToLaunch) {
		t.Errorf("got %v, want it to wrap ErrFailedToLaunch", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, want the underlying cause to survive wrapping", err)
	}
	if errors.Is(err, macosutils.ErrLaunchedAppFailed) {
		t.Errorf("got %v, want it not to match ErrLaunchedAppFailed", err)
	}
}

// TestLaunchAppLaunchedAppFailed verifies that a command that starts and then
// exits non-zero is reported with ErrLaunchedAppFailed, with the exit status
// still reachable.
func TestLaunchAppLaunchedAppFailed(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()

	err := l.LaunchApp(ctx, shell, "-c", "exit 3")
	if !errors.Is(err, macosutils.ErrLaunchedAppFailed) {
		t.Errorf("got %v, want it to wrap ErrLaunchedAppFailed", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("got %v, want an *exec.ExitError in the chain", err)
	}
	if got, want := exitErr.ExitCode(), 3; got != want {
		t.Errorf("exit code: got %d, want %d", got, want)
	}
	if errors.Is(err, macosutils.ErrFailedToLaunch) {
		t.Errorf("got %v, want it not to match ErrFailedToLaunch", err)
	}
}

func TestLaunchAppContextCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	l := macosutils.NewLauncher()

	err := l.LaunchApp(ctx, shell, "-c", "sleep 10")
	if err == nil {
		t.Fatal("LaunchApp: got nil error, want context timeout error")
	}
	if !errors.Is(err, macosutils.ErrLaunchedAppFailed) {
		t.Errorf("LaunchApp: got %v, want ErrLaunchedAppFailed", err)
	}
}

// TestTerminateLaunchedApp verifies that terminating a running app stops it and
// that the resulting failure is suppressed: a caller that asked for the shutdown
// should not be told the app failed.
func TestTerminateLaunchedApp(t *testing.T) {
	ctx := t.Context()
	pr, pw := io.Pipe()
	// The pipe must be drained continuously: os/exec copies the child's output
	// in a goroutine that Wait blocks on, so a stalled reader would stall Wait.
	ready := make(chan struct{})
	go func() {
		buf := bufio.NewReader(pr)
		if line, err := buf.ReadString('\n'); err == nil && strings.TrimSpace(line) == "ready" {
			close(ready)
		}
		_, _ = io.Copy(io.Discard, buf)
	}()

	l := macosutils.NewLauncher(macosutils.WithStdoutStderr(pw, io.Discard))
	errCh := make(chan error, 1)
	go func() {
		// exec so that the shell replaces itself with sleep: the signal must
		// reach the process that is actually running, and no other process may
		// be left holding the output pipe open.
		errCh <- l.LaunchApp(ctx, shell, "-c", "echo ready; exec sleep 30")
	}()

	select {
	case <-ready:
	case <-time.After(20 * time.Second):
		t.Fatal("the child never reported that it had started")
	}

	if !l.TerminateLaunchedApp() {
		t.Error("TerminateLaunchedApp reported false for a running app")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("LaunchApp: got %v, want nil: a requested shutdown is not a failure", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("LaunchApp did not return after TerminateLaunchedApp")
	}
}

// TestTerminateLaunchedAppNotRunning verifies that terminating when nothing has
// been launched reports false and does not suppress errors on subsequent launches.
func TestTerminateLaunchedAppNotRunning(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()
	if l.TerminateLaunchedApp() {
		t.Error("TerminateLaunchedApp reported true when no app was running")
	}
	// Verify a subsequent failing app still reports its failure.
	err := l.LaunchApp(ctx, shell, "-c", "exit 2")
	if !errors.Is(err, macosutils.ErrLaunchedAppFailed) {
		t.Errorf("LaunchApp: got %v, want ErrLaunchedAppFailed", err)
	}
}

// TestLaunchAppSecondAttemptFails verifies that attempting to launch a second
// application using the same Launcher returns ErrAlreadyLaunched.
func TestLaunchAppSecondAttemptFails(t *testing.T) {
	ctx := t.Context()
	l := macosutils.NewLauncher()

	if err := l.LaunchApp(ctx, shell, "-c", "true"); err != nil {
		t.Fatalf("LaunchApp: %v", err)
	}

	err := l.LaunchApp(ctx, shell, "-c", "true")
	if !errors.Is(err, macosutils.ErrAlreadyLaunched) {
		t.Errorf("second LaunchApp: got %v, want %v", err, macosutils.ErrAlreadyLaunched)
	}
}

func TestTailBytes(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("hello world, testing tail bytes functionality!")
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	emptyFilePath := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFilePath, []byte{}, 0600); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}

	for _, tc := range []struct {
		name    string
		file    string
		n       int
		want    []byte
		wantErr bool
	}{
		{
			name:    "file larger than n",
			file:    filePath,
			n:       14,
			want:    []byte("functionality!"),
			wantErr: false,
		},
		{
			name:    "file equal to n",
			file:    filePath,
			n:       len(content),
			want:    content,
			wantErr: false,
		},
		{
			name:    "file smaller than n",
			file:    filePath,
			n:       len(content) + 50,
			want:    content,
			wantErr: false,
		},
		{
			name:    "empty file with positive n",
			file:    emptyFilePath,
			n:       10,
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "zero n",
			file:    filePath,
			n:       0,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "negative n",
			file:    filePath,
			n:       -5,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			file:    filepath.Join(tmpDir, "nonexistent.txt"),
			n:       10,
			want:    nil,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := macosutils.TailBytes(tc.file, tc.n)
			if (err != nil) != tc.wantErr {
				t.Fatalf("TailBytes(%q, %d): got error %v, wantErr %v", tc.file, tc.n, err, tc.wantErr)
			}
			if !bytes.Equal(got, tc.want) {
				t.Errorf("TailBytes(%q, %d): got %q, want %q", tc.file, tc.n, string(got), string(tc.want))
			}
		})
	}
}
