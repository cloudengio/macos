// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"path/filepath"
	"strconv"
)

type AsPNG struct {
	InputPath  string
	OutputPath string
}

func (j AsPNG) Convert() Step {
	if filepath.Ext(string(j.InputPath)) == ".png" {
		return Copy(j.InputPath, j.OutputPath)
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "sips", "-s", "format", "png", j.InputPath, "--out", j.OutputPath)
	})
}

type IconSet struct {
	Icon       string
	IconSetDir string
	IconSet    string
}

type iconStep struct {
	IconSet
	size int
	twoX bool
}

func (i iconStep) Run(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
	name := "icon_" + strconv.Itoa(i.size)
	if i.twoX {
		name += "@2x.png"
	} else {
		name += ".png"
	}
	outPath := filepath.Join(i.IconSetDir, name)
	return cmdRunner.Run(ctx, "sips", "-z", strconv.Itoa(i.size), strconv.Itoa(i.size), i.Icon, "--out", outPath)
}

func (i IconSet) CreateIcons(twoX bool, sizes ...int) []Step {
	if len(sizes) == 0 {
		sizes = []int{16, 32, 64, 128, 256, 512, 1024}
	}
	var steps []Step
	for _, size := range sizes {
		steps = append(steps, iconStep{IconSet: i, size: size, twoX: false})
		if twoX {
			steps = append(steps, iconStep{IconSet: i, size: size * 2, twoX: true})
		}
	}
	return append(steps, i.CreateIcns())
}

func (i IconSet) CreateIcns() Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "iconutil", "-c", "icns", "-o", filepath.Join(i.IconSetDir, i.IconSet), i.IconSetDir)
	})
}
