// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"fmt"
	"path/filepath"
)

// AppBundle represents a macOS application bundle.
type AppBundle struct {
	Path string
	Info InfoPlist
}

// Create returns the steps required to create the app bundle directory structure
// and Info.plist.
func (b AppBundle) Create() []Step {
	steps := []Step{
		MkdirAll(b.Path),
		MkdirAll(filepath.Join(b.Path, "Contents", "MacOS")),
		MkdirAll(filepath.Join(b.Path, "Contents", "Resources")),
	}
	infoPlist := filepath.Join(b.Path, "Contents", "Info.plist")
	steps = append(steps, writeInfoPlist(infoPlist, b.Info))
	return steps
}

// CopyContents returns the step required to copy a file into the app bundle
// dst is relative to the bundle Contents root.
func (b AppBundle) CopyContents(src string, dst ...string) Step {
	p := filepath.Join(dst...)
	if src == "" || len(p) == 0 {
		return ErrorStep(fmt.Errorf("source (%q) or destination (%q) not specified", src, dst), "cp", src, p)
	}
	return Copy(src, filepath.Join(b.Path, "Contents", p))
}

// SignContents returns the step required to sign a file within the app bundle,
// dst is relative to the bundle Contents root.
func (b AppBundle) SignContents(signer Signer, dst ...string) Step {
	p := filepath.Join(dst...)
	if len(p) == 0 {
		return ErrorStep(fmt.Errorf("destination (%q) not specified", dst), "codesign", "", p)
	}
	return signer.SignPath(filepath.Join(b.Path, "Contents", p))
}

// Contents returns the path to the specified element within the app bundle's
// Contents directory.
func (b AppBundle) Contents(elem ...string) string {
	return filepath.Join(b.Path, "Contents", filepath.Join(elem...))
}

func (b AppBundle) CopyIcons(src string) Step {
	if len(b.Info.IconSet) == 0 {
		return NoopStep()
	}
	dst := filepath.Join(b.Path, "Contents", "Resources", b.Info.IconSet)
	return Copy(src, dst)
}

func (b AppBundle) Sign(signer Signer) Step {
	if signer.Identity == "" {
		return NoopStep()
	}
	return signer.SignPath(b.Path)
}

func (b AppBundle) VerifySignatures(signer Signer) []Step {
	if signer.Identity == "" {
		return []Step{NoopStep()}
	}
	steps := []Step{
		signer.VerifyPath(b.Path),
	}
	return steps
}
