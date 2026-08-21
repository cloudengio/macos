// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

// Launcher provides an interface for launching macOS applications.
type Launcher struct {
	opts *launchOptions

	quitting atomic.Bool
	mu       sync.Mutex
	launched bool
	childCmd *exec.Cmd
}

var (
	ErrFailedToLaunch    = errors.New("failed to launch application")
	ErrLaunchedAppFailed = errors.New("launched application failed")
	ErrAlreadyLaunched   = errors.New("application already launched")
)

// LaunchOption defines a function type for configuring the Launcher.
type LaunchOption func(o *launchOptions)

func WithCmdEnv(env func() []string) LaunchOption {
	return func(o *launchOptions) {
		o.cmdenv = env
	}
}

func WithWorkingDir(dir string) LaunchOption {
	return func(o *launchOptions) {
		o.dir = dir
	}
}

func WithStdoutStderr(stdout, stderr io.Writer) LaunchOption {
	return func(o *launchOptions) {
		o.stdout = stdout
		o.stderr = stderr
	}
}

type launchOptions struct {
	cmdenv         func() []string
	dir            string
	stdout, stderr io.Writer
}

// NewLauncher creates a new Launcher with the provided options.
func NewLauncher(opts ...LaunchOption) *Launcher {
	lo := &launchOptions{}
	for _, opt := range opts {
		opt(lo)
	}
	return &Launcher{
		opts: lo,
	}
}

func (l *Launcher) cmdWithEnvDir(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = l.opts.dir
	if l.opts.cmdenv != nil {
		c.Env = l.opts.cmdenv()
	}

	return c
}

// RunApp executes the specified command with the provided arguments and returns
// its combined output. Use it for short running commands, eg. to configure or
// setup the application.
func (l *Launcher) RunApp(ctx context.Context, cmd string, args ...string) (string, error) {
	c := l.cmdWithEnvDir(ctx, cmd, args...)
	out, err := c.CombinedOutput()
	return string(out), err
}

// LaunchApp launches a long-running application and waits for it to exit.
// Use TerminateLaunchedApp to signal the application to exit. Interrupt and
// terminate signals are forwarded to the launched application.
func (l *Launcher) LaunchApp(ctx context.Context, cmd string, args ...string) error {
	l.mu.Lock()
	if l.launched {
		l.mu.Unlock()
		return fmt.Errorf("%v: %w", cmd, ErrAlreadyLaunched)
	}
	l.launched = true
	l.mu.Unlock()

	c := l.cmdWithEnvDir(ctx, cmd, args...)
	if l.opts.stdout != nil {
		c.Stdout = l.opts.stdout
	}
	if l.opts.stderr != nil {
		c.Stderr = l.opts.stderr
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	defer func() {
		signal.Stop(sigCh)
		close(done)
	}()

	if err := c.Start(); err != nil {
		return fmt.Errorf("%v: %w: %w", cmd, ErrFailedToLaunch, err)
	}
	l.setChild(c)
	defer l.setChild(nil)
	go func() {
		for {
			select {
			case s := <-sigCh:
				if c.Process != nil {
					_ = c.Process.Signal(s)
				}
			case <-done:
				return
			}
		}
	}()

	// Don't show the failure dialog if the process was stopped because the user
	// is quitting the app.
	if err := c.Wait(); err != nil && !l.quitting.Load() {
		return fmt.Errorf("%v: %w: %w", cmd, ErrLaunchedAppFailed, err)
	}
	return nil
}

func (l *Launcher) setChild(cmd *exec.Cmd) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.childCmd = cmd
}

func (l *Launcher) TerminateLaunchedApp() bool {
	l.mu.Lock()
	c := l.childCmd
	if c != nil && c.Process != nil {
		l.quitting.Store(true)
		l.mu.Unlock()
		_ = c.Process.Signal(syscall.SIGTERM)
		return true
	}
	l.mu.Unlock()
	return false
}
