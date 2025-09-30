// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"path/filepath"
	"strconv"
)

// ReformatIcon represents a step that reformats an icon to the specified format.
type ReformatIcon struct {
	InputPath  string
	OutputPath string
}

// Convert converts the input image/icon to the specified format.
func (j ReformatIcon) Convert(format string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "sips", "-s", "format", format, j.InputPath, "--out", j.OutputPath)
	})
}

// IconSizeMultiple represents the scale factor for icon sizes, e.g., 1x, 2x, 3x.
type IconSizeMultiple int

const (
	IconSize1x IconSizeMultiple = 1
	IconSize2x IconSizeMultiple = 2
	IconSize3x IconSizeMultiple = 3
)

// Suffix returns the filename suffix appropriate for the icon size multiple.
func (m IconSizeMultiple) Suffix() string {
	if m != IconSize1x {
		return strconv.Itoa(int(m)) + "x"
	}
	return ""
}

// IconSet represents a directory that contains the variously sized icons
// needed to create an .icns file from a single source icon.
type IconSet struct {
	Icon     string           `yaml:"icon"`
	Dir      string           `yaml:"dir"`
	Name     string           `yaml:"name"`       // defaults to AppIcon.icns
	Sizes    []int            `yaml:"sizes,flow"` // optional - defaults to standard sizes if not provided
	Multiple IconSizeMultiple `yaml:"multiple"`   // optional - defaults to 3 (1x, 2x and 3x) if not provided
	Format   string           `yaml:"format"`     // optional - defaults to png if not provided
	// if true, the icon is copied to the bundle Resources directory
	// as the file specified by CFBundleIconFile in the Info.plist
	BundleIcon bool `yaml:"bundle_icon"`
}

func (i IconSet) IconSetDir() string {
	if i.Dir == "" {
		return "Icons.iconset"
	}
	return i.Dir
}

func (i IconSet) IconSetName() string {
	if i.Name == "" {
		return "AppIcon.icns"
	}
	return i.Name
}

func (i IconSet) IconSetFile() string {
	return filepath.Join(i.IconSetDir(), i.IconSetName())
}

func (i IconSet) IconFormat() string {
	if i.Format == "" {
		return "png"
	}
	return i.Format
}

type iconStep struct {
	dir      string
	src      string
	format   string
	size     int
	multiple IconSizeMultiple
}

func (i iconStep) Run(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
	name := "icon_" + strconv.Itoa(i.size) + i.multiple.Suffix() + "." + i.format
	outPath := filepath.Join(i.dir, name)
	return cmdRunner.Run(ctx, "sips", "-z", strconv.Itoa(i.size), strconv.Itoa(i.size), i.src, "--out", outPath)
}

// CreateIconVariants creates the variously sized icons needed for the icon set.
// If no sizes are provided, a default set is used.
// The highest_multiple parameter indicates the highest scale factor to use,
// e.g., 2 for 1x and 2x, 3 for 1x, 2x and 3x.
func (i IconSet) CreateIconVariants(src, dir string) []Step {
	if len(i.Sizes) == 0 {
		i.Sizes = []int{16, 32, 64, 128, 256, 512, 1024}
	}
	if i.Multiple == 0 {
		i.Multiple = IconSize3x
	}
	var steps []Step
	for _, size := range i.Sizes {
		for m := IconSize1x; m <= i.Multiple; m++ {
			steps = append(steps, iconStep{src: src, dir: dir, size: size * int(m), multiple: m, format: i.IconFormat()})
		}
	}
	return append(steps, i.CreateIcns())
}

func (i IconSet) CreateIcns() Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "iconutil", "--convert", "icns", "--output",
			i.IconSetFile(), i.IconSetDir())
	})
}
