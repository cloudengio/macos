// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"howett.net/plist"
)

// PerFileEntitlements represents a set of macOS app entitlements that
// are specific to individual files within an app bundle. These are specified
// as YAML. The key should be the file within the bundle and the value is the
// entitlements dictionary for that file. The file name can be either the base
// name (eg. "executable") or the full path within the bundle
// (e.g. "Contents/MacOS/executable").
type PerFileEntitlements struct {
	raw map[string]Entitlements `yaml:",inline"`
}

func (e *PerFileEntitlements) UnmarshalYAML(node *yaml.Node) error {
	return node.Decode(&e.raw)
}

// For returns the entitlements for the specified path or nil if none exist.
// It will first look for the base name of the path and then the full path.
func (e PerFileEntitlements) For(path string) (Entitlements, bool) {
	base := filepath.Base(path)
	if ent, ok := e.raw[base]; ok {
		return ent, true
	}
	ent, ok := e.raw[path]
	return ent, ok
}

// Entitlements represents a set of macOS app entitlements that
// are specified as YAML.
type Entitlements struct {
	raw map[string]any `yaml:",inline"`
}

func (e *Entitlements) UnmarshalYAML(node *yaml.Node) error {
	return node.Decode(&e.raw)
}

func (e Entitlements) MarshalPlist() (any, error) {
	return e.raw, nil
}

func (e Entitlements) MarshalYAML() (any, error) {
	return e.raw, nil
}

func (e PerFileEntitlements) MarshalPlist() (any, error) {
	return nil, fmt.Errorf("cannot marshal PerFileEntitlements to plist")
}

func (e PerFileEntitlements) MarshalYAML() (any, error) {
	return e.raw, nil
}

// MarshalIndent returns the XML plist representation of the entitlements
// with the specified indent.
func (e Entitlements) MarshalIndent(indent string) ([]byte, error) {
	return plist.MarshalIndent(e.raw, plist.XMLFormat, indent)
}
