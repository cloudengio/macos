// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"strings"
)

type Git struct {
	dir string
}

// NewGit creates a new Git instance rooted at the specified directory which
// must be within a git repository.
func NewGit(dir string) Git {
	return Git{dir: dir}
}

/*()
func (g Git) HashStep(ctx context.Context, branch string, n int) Step {
	if len(branch) == 0 {
		branch = "HEAD"
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		ctx = ContextWithCWD(ctx, g.dir)
		if n == 0 {
			n = 8
		}
		short := fmt.Sprintf("--short=%d", n)
		return cmdRunner.Run(ctx, "git", "rev-parse", short, branch)
	})
}*/

func (g Git) Hash(ctx context.Context, cmdRunner *CommandRunner, branch string, n int) (StepResult, error) {
	if len(branch) == 0 {
		branch = "HEAD"
	}
	ctx = ContextWithCWD(ctx, g.dir)
	if n == 0 {
		n = 8
	}
	short := fmt.Sprintf("--short=%d", n)
	return cmdRunner.Run(ctx, "git", "rev-parse", short, branch)
}

func (g Git) key() string {
	return "+git:"
}

func (g Git) GetBranch(version string) string {
	index := strings.Index(version, g.key())
	if index == -1 {
		return ""
	}
	return version[index+len(g.key()):]
}

func (g Git) ReplaceBranch(version, buildID string) string {
	index := strings.Index(version, g.key())
	if index == -1 {
		return version
	}
	return version[:index] + "+" + buildID
}

/*()
func (g Git) AppendGitHash(ctx context.Context, version string, n int) (string, error) {
	key := "+git:"
	index := strings.Index(version, key)
	if index == -1 {
		return version, nil
	}
	branch := version[index+len(key):]
	fmt.Printf("branch: %q\n", branch)
	hash, err := g.Hash(ctx, branch, 8)
	if err != nil {
		return "", err
	}
	return version[:index] + "+" + hash, nil
}
*/
