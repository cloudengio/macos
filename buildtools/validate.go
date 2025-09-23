// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"path/filepath"
)

type HasSuffix string

func (s HasSuffix) Check(path string) Step {
	p := string(path)
	return StepFunc(func(_ context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		cmd := fmt.Sprintf("filepath.Ext(%q) == %q", p, s)
		if cmdRunner.DryRun() {
			return NewStepResult(cmd, nil, nil, nil), nil
		}
		if filepath.Ext(p) != ".iconset" {
			return NewStepResult(cmd, nil, nil, fmt.Errorf("icon set dir %q must have .iconset extension", p)), nil
		}
		return NewStepResult(cmd, nil, nil, nil), nil
	})
}
