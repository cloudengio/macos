// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LaunchAgent manages the installation of a launchd job as a per-user
// LaunchAgent, ie. a plist in ~/Library/LaunchAgents that launchd starts when
// the user logs in.
//
// A LaunchAgent rather than a system-wide LaunchDaemon is the right choice for
// a job that needs a logged-in user's GUI session, for example to use
// Virtualization.framework.
type LaunchAgent struct {
	// Plist describes the job. Its Label names both the installed file and the
	// launchd service, so it must be set.
	Plist LaunchAgentPlist

	// Dir is the directory the plist is installed into. If empty,
	// UserLaunchAgentsDir is used. Set it to install elsewhere, or in tests.
	Dir string

	// Domain is the launchd domain the job is bootstrapped into, eg. "gui/501".
	// If empty, GUIDomain for the current user is used.
	Domain string
}

// UserLaunchAgentsDir returns the current user's LaunchAgents directory.
func UserLaunchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// GUIDomain returns the launchd GUI domain target for the current user.
func GUIDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

// dir returns the directory the plist is installed into.
func (l LaunchAgent) dir() (string, error) {
	if l.Dir != "" {
		return l.Dir, nil
	}
	return UserLaunchAgentsDir()
}

// domain returns the launchd domain the job is bootstrapped into.
func (l LaunchAgent) domain() string {
	if l.Domain != "" {
		return l.Domain
	}
	return GUIDomain()
}

// PlistPath returns the path the job's plist is installed at.
func (l LaunchAgent) PlistPath() (string, error) {
	if l.Plist.Label == "" {
		return "", fmt.Errorf("launchd job has no Label")
	}
	if filepath.Base(l.Plist.Label) != l.Plist.Label || strings.ContainsAny(l.Plist.Label, `/\`) {
		return "", fmt.Errorf("invalid Label %q: must not contain path separators", l.Plist.Label)
	}
	dir, err := l.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, l.Plist.Label+".plist"), nil
}

// ServiceTarget returns the launchd service target for the job, ie. the domain
// and label that launchctl subcommands operate on.
func (l LaunchAgent) ServiceTarget() (string, error) {
	if l.Plist.Label == "" {
		return "", fmt.Errorf("launchd job has no Label")
	}
	if filepath.Base(l.Plist.Label) != l.Plist.Label || strings.ContainsAny(l.Plist.Label, `/\`) {
		return "", fmt.Errorf("invalid Label %q: must not contain path separators", l.Plist.Label)
	}
	return l.domain() + "/" + l.Plist.Label, nil
}

// IsInstalled reports whether the job's plist is present. It says nothing about
// whether launchd has the job loaded; use Status for that.
func (l LaunchAgent) IsInstalled() bool {
	path, err := l.PlistPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Install returns the steps that write the job's plist and load it: any
// previously loaded instance is booted out first, so Install doubles as a
// reinstall of a changed job.
func (l LaunchAgent) Install() []Step {
	if err := l.Plist.Validate(); err != nil {
		return []Step{ErrorStep(err, "validate", "LaunchAgent")}
	}
	path, err := l.PlistPath()
	if err != nil {
		return []Step{ErrorStep(err, "validate", "LaunchAgent")}
	}
	dir, _ := l.dir()
	return []Step{
		MkdirAll(dir),
		WritePlistFile(l.Plist, path),
		l.bootout(),
		l.launchctl("bootstrap", l.domain(), path),
	}
}

// Uninstall returns the steps that unload the job and remove its plist. It
// succeeds whether or not the job is currently loaded or installed.
func (l LaunchAgent) Uninstall() []Step {
	path, err := l.PlistPath()
	if err != nil {
		return []Step{ErrorStep(err, "validate", "LaunchAgent")}
	}
	name := filepath.Base(path)
	return []Step{
		l.bootout(),
		StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
			if err := ctx.Err(); err != nil {
				return ErrorStep(err, "remove "+name, path).Run(ctx, cmdRunner)
			}
			if cmdRunner.DryRun() {
				return NewStepResult("remove "+name, []string{path}, nil, nil), nil
			}
			err := os.Remove(path)
			if os.IsNotExist(err) {
				err = nil
			}
			return NewStepResult("os.Remove", []string{path}, nil, err), err
		}),
	}
}

// Restart returns a Step that restarts the job, starting it if it is loaded but
// not running.
func (l LaunchAgent) Restart() Step {
	target, err := l.ServiceTarget()
	if err != nil {
		return ErrorStep(err, "validate", "LaunchAgent")
	}
	return l.launchctl("kickstart", "-k", target)
}

// Status returns a Step that prints launchd's view of the job. launchctl
// reports a useful message for a job that is not loaded, so the step succeeds
// either way.
func (l LaunchAgent) Status() Step {
	target, err := l.ServiceTarget()
	if err != nil {
		return ErrorStep(err, "validate", "LaunchAgent")
	}
	return l.launchctlIgnoringError("print", target)
}

// bootout unloads the job, ignoring the error launchctl returns when it is not
// loaded, so that it can be used to make loading idempotent.
func (l LaunchAgent) bootout() Step {
	target, err := l.ServiceTarget()
	if err != nil {
		return ErrorStep(err, "validate", "LaunchAgent")
	}
	return l.launchctlIgnoringError("bootout", target)
}

func (l LaunchAgent) launchctl(args ...string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "launchctl", args...)
	})
}

// launchctlIgnoringError runs launchctl and reports success regardless of the
// exit status, retaining the output in the StepResult.
func (l LaunchAgent) launchctlIgnoringError(args ...string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		if err := ctx.Err(); err != nil {
			return ErrorStep(err, "launchctl", args...).Run(ctx, cmdRunner)
		}
		res, _ := cmdRunner.Run(ctx, "launchctl", args...)
		if err := ctx.Err(); err != nil {
			return ErrorStep(err, "launchctl", args...).Run(ctx, cmdRunner)
		}
		return NewStepResult(res.Executable(), res.Args(), []byte(res.Output()), nil), nil
	})
}
