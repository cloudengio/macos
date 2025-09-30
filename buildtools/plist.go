// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"howett.net/plist"
)

// InfoPlist represents the contents of a macOS Info.plist file.
// The struct fields represent common keys found in such files
// and are extracted from the Raw map for convenience and use
// within this package.
type InfoPlist struct {
	CFBundleIdentifier     string
	CFBundleName           string
	CFBundleExecutable     string
	CFBundleIconFile       string
	CFBundlePackageType    string
	LSMinimumSystemVersion string
	CFBundleDisplayName    string
	XPCService             *XPCServicePlist
	Raw                    map[string]any
}

// XPCServicePlist represents the contents of an XPCService dictionary
// within an Info.plist file.
// The Raw field contains the full dictionary contents while the ServiceName
// field is extracted for convenience.
type XPCServicePlist struct {
	ServiceName string
}

func asString(dict map[string]any, key string) (string, error) {
	if v, ok := dict[key]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("key %q not found or not a string", key)
}

func (ipl *InfoPlist) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&ipl.Raw); err != nil {
		return err
	}
	var err error
	if ipl.CFBundleIdentifier, err = asString(ipl.Raw, "CFBundleIdentifier"); err != nil {
		return err
	}
	if ipl.CFBundleName, err = asString(ipl.Raw, "CFBundleName"); err != nil {
		return err
	}
	if ipl.CFBundleExecutable, err = asString(ipl.Raw, "CFBundleExecutable"); err != nil {
		return err
	}
	if ipl.CFBundlePackageType, err = asString(ipl.Raw, "CFBundlePackageType"); err != nil {
		return err
	}
	if ipl.LSMinimumSystemVersion, err = asString(ipl.Raw, "LSMinimumSystemVersion"); err != nil {
		return err
	}
	if ipl.CFBundleDisplayName, err = asString(ipl.Raw, "CFBundleDisplayName"); err != nil {
		return err
	}
	// optional
	ipl.CFBundleIconFile, _ = asString(ipl.Raw, "CFBundleIconFile")

	if v, ok := ipl.Raw["XPCService"]; ok {
		vm, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("XPCService not a dictionary")
		}
		xpc := &XPCServicePlist{}
		xpc.ServiceName, err = asString(vm, "ServiceName")
		if err != nil {
			return err
		}
		ipl.XPCService = xpc
	}
	return nil
}

func (ipl InfoPlist) MarshalPlist() (any, error) {
	return ipl.Raw, nil
}

func (ipl InfoPlist) MarshalYAML() (any, error) {
	return ipl.Raw, nil
}

func writeInfoPlist(path string, info InfoPlist) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		if cmdRunner.DryRun() {
			return NewStepResult("write Info.plist", []string{path}, nil, nil), nil
		}
		data, err := plist.MarshalIndent(info, plist.XMLFormat, "\t")
		if err != nil {
			return NewStepResult("write Info.plist", []string{path}, nil, err), err
		}
		err = os.WriteFile(path, data, 0644)
		return NewStepResult("write Info.plist", []string{path}, nil, err), err
	})
}
