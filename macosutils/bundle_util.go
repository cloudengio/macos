// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package macosutils contains macos specific utilities.
package macosutils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// IsAppBundle returns true if path is a directory ending .app and contains
// a Contents/Info.plist.
func IsAppBundle(path string) bool {
	if !strings.HasSuffix(strings.ToLower(path), ".app") {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	plistPath := filepath.Join(path, "Contents", "Info.plist")
	plistInfo, err := os.Stat(plistPath)
	if err != nil || plistInfo.IsDir() {
		return false
	}

	return true
}

// LocateInBundle finds the requested binary in the specified app bundle
// returns its absolute path.
func LocateInBundle(bundlePath, binary string) string {
	if !IsAppBundle(bundlePath) {
		return ""
	}

	// Confine the walk to the bundle directory: os.Root rejects any path that
	// leaves it, including via symlinks inside the bundle. This matters because
	// the binary located here is subsequently executed.
	root, err := os.OpenRoot(bundlePath)
	if err != nil {
		return ""
	}
	defer root.Close()

	location := ""
	_ = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == binary {
			if info, err := d.Info(); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0100 != 0 {
				location = filepath.Join(bundlePath, path)
				return fs.SkipAll
			}
		}
		return nil
	})
	return location
}

// InAppBundle determines if binary is an app bundle and returns the path of
// the bundle. It uses the simple heurestic of checking to see if the
// binary has parents .../<app-bundle>/Contents/MacOS and that <app-bundle>
// satisfies InAppBundle.
func InAppBundle(binary string) (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	// Let's see if this binary is an app bundle.
	macos := filepath.Dir(exe)
	if !strings.EqualFold(filepath.Base(macos), "macos") {
		return "", false
	}
	contents := filepath.Dir(macos)
	if !strings.EqualFold(filepath.Base(contents), "contents") {
		return "", false
	}
	appBundle := filepath.Dir(contents)

	if !IsAppBundle(appBundle) {
		return "", false
	}

	// Look for the binary.
	plugin := LocateInBundle(appBundle, binary)
	if len(plugin) == 0 {
		return "", false
	}
	if fi, err := os.Stat(plugin); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0100 != 0 {
		return plugin, true
	}
	return "", false
}

// LookPathBundle is like exec.LookPath but for app bundles.
func LookPathBundle(bundle, pathList string) string {
	found := LookPathBundleAll(bundle, pathList)
	if len(found) > 0 {
		return found[0]
	}
	return ""
}

// LookPathBundle is like LookPathBundle but returns all instances
// of bundle on pathList without duplicates.
func LookPathBundleAll(bundle, pathList string) []string {
	if filepath.IsAbs(bundle) {
		if IsAppBundle(bundle) {
			return []string{bundle}
		}
		return nil
	}

	var found []string
	seen := make(map[string]struct{}, len(pathList))
	for _, dir := range filepath.SplitList(pathList) {
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		candidate := filepath.Join(dir, bundle)
		if IsAppBundle(candidate) {
			found = append(found, candidate)
		}
	}
	return found
}
