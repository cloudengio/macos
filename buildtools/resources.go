// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"os"
	"path/filepath"
)

// Resources represents the resources needed to build an app bundle.
type Resources struct {
	Executable    string `yaml:"executable"`
	XPCExecutable string `yaml:"xpc_executable"` // optional
	Icon          string `yaml:"icon"`           // optional
	IconSetDir    string `yaml:"icon_dir"`       // optional
	IconSetName   string `yaml:"icon_set_name"`  // optional - defaults to AppIcon.icns
}

func (r Resources) IconSteps(twoX bool, sizes ...int) []Step {
	if r.Icon == "" && r.IconSetDir == "" {
		return []Step{NoopStep()}
	}
	if r.IconSetDir != "" {
		icns := r.IconSetPath()
		if _, err := os.Stat(string(icns)); err == nil {
			return []Step{NoopStep()}
		}
	}
	steps := []Step{MkdirAll(r.IconSetDir)}
	pngFile := filepath.Join(r.IconSetDir, "icon.png")
	steps = append(steps, AsPNG{
		InputPath:  r.Icon,
		OutputPath: pngFile,
	}.Convert())

	set := IconSet{
		Icon:       pngFile,
		IconSetDir: r.IconSetDir,
		IconSet:    r.IconSetFile(),
	}
	steps = append(steps, set.CreateIcons(twoX, sizes...)...)
	return steps
}

func (r Resources) IconSetFile() string {
	if r.IconSetName == "" {
		return "AppIcon.icns"
	}
	return r.IconSetName
}

func (r Resources) IconSetPath() string {
	return filepath.Join(r.IconSetDir, r.IconSetFile())
}
