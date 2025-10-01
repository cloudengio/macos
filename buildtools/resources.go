// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resources represents the resources needed to build an app bundle.
type Resources struct {
	Executable string    `yaml:"executable"`
	Icons      []IconSet `yaml:"icons"` // multiple icon sets can be specified
}

// IconSetSteps returns the steps needed to create the icon sets specified in the Resources.
func (r Resources) IconSetSteps() []Step {
	steps := []Step{}
	for _, is := range r.Icons {
		// Already exists.
		if _, err := os.Stat(is.IconSetFile()); err == nil {
			steps = append(steps, NoopStep(fmt.Sprintf("Icon set %q already exists", is.IconSetFile())))
			continue
		}
		steps = append(steps, MkdirAll(is.Dir))
		format := is.IconFormat()
		baseImage := filepath.Join(is.IconSetDir(), "icon."+format)
		steps = append(steps, ReformatIcon{
			InputPath:  is.Icon,
			OutputPath: baseImage,
		}.Convert(format))
		steps = append(steps, is.CreateIconVariants(baseImage, is.IconSetDir())...)
	}
	return steps
}
