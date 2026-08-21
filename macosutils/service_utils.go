// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package macosutils

import (
	"os"
	"path/filepath"
	"strings"
)

// IsServiceInstalled returns true if the specified service is installed in the
// current user's LaunchAgents directory.
func IsServiceInstalled(serviceLabel string) bool {
	if len(serviceLabel) == 0 || filepath.Base(serviceLabel) != serviceLabel || strings.ContainsAny(serviceLabel, `/\`) {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(home, "Library", "LaunchAgents", serviceLabel+".plist"))
	if err != nil || info.IsDir() {
		return false
	}
	return true
}
