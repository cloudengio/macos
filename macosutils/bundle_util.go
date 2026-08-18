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

// IsAppBundle returns true if path is a directory ending with .app and contains
// a Contents/Info.plist file.
func IsAppBundle(path string) bool {
	path = filepath.Clean(path)
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
// and returns its path within that bundle.
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
			if info, err := d.Info(); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				location = filepath.Join(bundlePath, path)
				return fs.SkipAll
			}
		}
		return nil
	})
	return location
}

// InAppBundle determines if the running process is within an app bundle and
// returns the path of the requested binary inside that bundle. It uses the
// heuristic of checking if the executable is located under
// .../<app-bundle>/Contents/MacOS and that <app-bundle> satisfies IsAppBundle.
func InAppBundle(binary string) (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	// Let's see if this binary is in an app bundle.
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
	if len(plugin) > 0 {
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

// LookPathBundleAll is like LookPathBundle but returns all instances
// of bundle on pathList without duplicates.
func LookPathBundleAll(bundle, pathList string) []string {
	if filepath.IsAbs(bundle) {
		if IsAppBundle(bundle) {
			return []string{bundle}
		}
		return nil
	}

	if bundle != filepath.Base(bundle) || bundle == "." || bundle == ".." {
		return nil
	}

	var found []string
	dirs := filepath.SplitList(pathList)
	seen := make(map[string]struct{}, len(dirs))
	for _, dir := range dirs {
		dir = filepath.Clean(dir)
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
