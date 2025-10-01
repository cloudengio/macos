// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"path/filepath"
)

type PkgBuild struct {
	BuildDir        string `yaml:"build_dir"`        // Directory to use for building the package
	Identifier      string `yaml:"identifier"`       // Package identifier, e.g. com.cloudeng.myapp
	Version         string `yaml:"version"`          // Package version, e.g. 1.0.0
	InstallLocation string `yaml:"install_location"` // Installation location, e.g. /Applications/MyApp.app

}

func (p PkgBuild) Clean() Step {
	if len(p.BuildDir) == 0 {
		return ErrorStep(fmt.Errorf("no build dir specified"), "rm", "-rf")
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "rm", "-rf", p.BuildDir)
	})
}

func (p PkgBuild) Create() []Step {
	steps := []Step{
		MkdirAll(p.BuildDir),
		MkdirAll(filepath.Join(p.BuildDir, "root", "Applications")),
		MkdirAll(filepath.Join(p.BuildDir, "scripts")),
	}
	return steps

}

func (p PkgBuild) CopyApplication(src string) Step {
	if len(src) == 0 {
		return NoopStep("no application specified")
	}
	return RSync(src, filepath.Join(p.BuildDir, "root", "Applications"))
}

func (p PkgBuild) CopyScripts(src string) Step {
	if len(src) == 0 {
		return NoopStep("no scripts specified")
	}
	return RSync(src+"/", filepath.Join(p.BuildDir, "scripts"))
}

func (p PkgBuild) WriteScript(data []byte, name string) Step {
	if len(data) == 0 {
		return NoopStep("no script data specified")
	}
	if len(name) == 0 {
		return ErrorStep(fmt.Errorf("no script name specified"), "write script")
	}
	return WriteFile(data, 0500, filepath.Join(p.BuildDir, "scripts", name))
}

func (p PkgBuild) CopyLibrary(src, library string) Step {
	if len(src) == 0 {
		return NoopStep("no library specified")
	}
	return RSync(src+"/", filepath.Join(p.BuildDir, "root", "Library", library))
}

func (p PkgBuild) WritePlist(cfg []PkgComponentPlist) Step {
	return writeInfoPlist(filepath.Join(p.BuildDir, "component.plist"), "component.plist", cfg)
}

func (p PkgBuild) ScriptsPath() string {
	return filepath.Join(p.BuildDir, "scripts")
}

func (p PkgBuild) CreateLibrary(library string) Step {
	return MkdirAll(filepath.Join(p.BuildDir, "root", "Library", library))
}

func (p PkgBuild) LibraryPath(library string) string {
	return filepath.Join(p.BuildDir, "root", "Library", library)
}

func (p PkgBuild) Build(outputPath string) Step {
	if len(outputPath) == 0 || len(p.InstallLocation) == 0 || len(p.Identifier) == 0 || len(p.Version) == 0 {
		return ErrorStep(fmt.Errorf("one of outputPath, InstallLocation, Identifier or Version is not set: %+v", p), "pkgbuild")
	}
	args := []string{
		"--root", filepath.Join(p.BuildDir, "root"),
		"--component-plist", filepath.Join(p.BuildDir, "component.plist"),
		"--identifier", p.Identifier,
		"--version", p.Version,
		"--install-location", p.InstallLocation,
		"--scripts", p.ScriptsPath(),
		outputPath,
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "pkgbuild", args...)
	})
}

func (p PkgBuild) Install(outputPath string) Step {
	if len(p.InstallLocation) == 0 {
		return ErrorStep(fmt.Errorf("no install location specified"), "open")
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "sudo", "installer", "-pkg", outputPath, "-target", p.InstallLocation)
	})
}

// PkgConfPkgComponentPlist represents the pkgbuild component plist structure.
type PkgComponentPlist struct {
	RootRelativeBundlePath      string `yaml:"RootRelativeBundlePath" plist:"RootRelativeBundlePath"`
	BundleIsRelocatable         bool   `yaml:"BundleIsRelocatable" plist:"BundleIsRelocatable,omitempty"`
	BundleIsVersionChecked      bool   `yaml:"BundleIsVersionChecked" plist:"BundleIsVersionChecked,omitempty"`
	BundleHasStrictIdentifier   bool   `yaml:"BundleHasStrictIdentifier" plist:"BundleHasStrictIdentifier,omitempty"`
	BundleOverwriteAction       string `yaml:"BundleOverwriteAction" plist:"BundleOverwriteAction,omitempty"`
	BundlePreInstallScriptPath  string `yaml:"BundlePreInstallScriptPath" plist:"BundlePreInstallScriptPath,omitempty"`
	BundlePostInstallScriptPath string `yaml:"BundlePostInstallScriptPath" plist:"BundlePostInstallScriptPath,omitempty"`
	BundleInstallScriptTimeout  int    `yaml:"BundleInstallScriptTimeout" plist:"BundleInstallScriptTimeout,omitempty"`
}
