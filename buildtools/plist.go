// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"os"

	"howett.net/plist"
)

// InfoPlist captures the common fields for an app bundle's Info.plist.
type InfoPlist struct {
	Identifier   string       `plist:"CFBundleIdentifier,omitempty" yaml:"identifier"`
	Name         string       `plist:"CFBundleName,omitempty" yaml:"name"`
	Version      string       `plist:"CFBundleVersion,omitempty" yaml:"version"`
	ShortVersion string       `plist:"CFBundleShortVersionString,omitempty" yaml:"short_version"`
	Executable   string       `plist:"CFBundleExecutable,omitempty" yaml:"executable"`
	IconSet      string       `plist:"CFBundleIconFile,omitempty" yaml:"icon_set"`
	Type         string       `plist:"CFBundlePackageType"`
	XPCService   XPCInfoPlist `plist:"XPCService,omitempty" yaml:"xpc_service"`
}

// XPCInfoPlist represents the XPC service specific portion of the
// XPC service info.plist.
type XPCInfoPlist struct {
	ServiceName      string   `plist:"ServiceName,omitempty" yaml:"service_name"`
	ServiceType      string   `plist:"ServiceType,omitempty" yaml:"service_type"`
	ProcessType      string   `plist:"ProcessType,omitempty" yaml:"process_type"`
	ProgramArguments []string `plist:"ProgramArguments,omitempty" yaml:"args"`
}

func MarshalInfoPlist(info InfoPlist) ([]byte, error) {
	return plist.MarshalIndent(info, plist.XMLFormat, "\t")
}

func writeInfoPlist(path string, info InfoPlist) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		if cmdRunner.DryRun() {
			return NewStepResult("write Info.plist", []string{path}, nil, nil), nil
		}
		data, err := MarshalInfoPlist(info)
		if err != nil {
			return NewStepResult("write Info.plist", []string{path}, nil, err), err
		}
		err = os.WriteFile(path, data, 0644)
		return NewStepResult("write Info.plist", []string{path}, nil, err), err
	})
}
