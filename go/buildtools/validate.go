// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"path/filepath"
)

type Suffix string

// Assert returns a Step that checks if the provided path has the specified suffix.
func (s Suffix) Assert(path string) Step {
	p := string(path)
	return StepFunc(func(_ context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		cmd := fmt.Sprintf("filepath.Ext(%q) == %q", p, s)
		if cmdRunner.DryRun() {
			return NewStepResult(cmd, nil, nil, nil), nil
		}
		if filepath.Ext(p) != string(s) {
			err := fmt.Errorf("filepath.Ext(%q) != %q", p, s)
			return NewStepResult(cmd, nil, nil, err), err
		}
		return NewStepResult(cmd, nil, nil, nil), nil
	})
}
