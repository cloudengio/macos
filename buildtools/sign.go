// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
)

type Signer struct {
	Identity         string
	EntitlementsFile string
	Arguments        []string
}

func (s Signer) SignPath(path string) Step {
	args := []string{"--sign", s.Identity}
	if len(s.Arguments) == 0 {
		args = append(args, "--options", "runtime", "--force", "--timestamp")
	} else {
		args = append(args, s.Arguments...)
	}
	if s.Identity == "" {
		return ErrorStep(fmt.Errorf("cannot sign path %q: no identity specified", path), "codesign")
	}
	if s.EntitlementsFile != "" {
		args = append(args, "--entitlements", s.EntitlementsFile)
	}
	args = append(args, path)
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "codesign", args...)
	})
}

func (s Signer) VerifyPath(path string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "codesign", "--verify", "--strict", path)
	})
}
