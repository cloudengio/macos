// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// NativeMessagingScope determines whether a native messaging manifest is
// installed for the current user or for all users of the machine.
type NativeMessagingScope int

const (
	// UserScope installs the manifest for the current user only.
	UserScope NativeMessagingScope = iota
	// SystemScope installs the manifest for all users, which requires
	// privileges to write outside of the user's home directory.
	SystemScope
)

func (s NativeMessagingScope) String() string {
	switch s {
	case UserScope:
		return "user"
	case SystemScope:
		return "system"
	default:
		return "unknown"
	}
}

// NativeMessagingHostsDir returns the directory that browser searches for
// native messaging manifests, into which the manifest for a helper must be
// installed for an extension to be able to launch it. home is the user's home
// directory and is ignored for SystemScope.
//
// Safari does not use native messaging manifests: a Safari web extension
// sends native messages to the .appex that contains it, so an error is
// returned for it. See AppBundle.AddSafariWebExtension.
//
// See https://developer.chrome.com/docs/apps/nativeMessaging and
// https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/Native_manifests
func NativeMessagingHostsDir(browser BrowserType, scope NativeMessagingScope, home string) (string, error) {
	const hosts = "NativeMessagingHosts"
	if scope != UserScope && scope != SystemScope {
		return "", fmt.Errorf("unsupported scope: %v", scope)
	}
	user := scope == UserScope
	if user && home == "" {
		return "", fmt.Errorf("home directory not specified for %v scope", scope)
	}
	switch browser {
	case Chrome:
		if user {
			return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", hosts), nil
		}
		return filepath.Join("/", "Library", "Google", "Chrome", hosts), nil
	case Edge:
		if user {
			return filepath.Join(home, "Library", "Application Support", "Microsoft Edge", hosts), nil
		}
		return filepath.Join("/", "Library", "Microsoft", "Edge", hosts), nil
	case Firefox:
		if user {
			return filepath.Join(home, "Library", "Application Support", "Mozilla", hosts), nil
		}
		return filepath.Join("/", "Library", "Application Support", "Mozilla", hosts), nil
	case Safari:
		return "", fmt.Errorf("safari does not use native messaging manifests: a web extension sends native messages to its containing .appex, see AddSafariWebExtension")
	default:
		return "", fmt.Errorf("unsupported browser: %v", browser)
	}
}

// firefoxAllowedName is the form required of a firefox native messaging host
// name: word characters separated by single dots.
var firefoxAllowedName = regexp.MustCompile(`^\w+(\.\w+)*$`)

// ValidateFirefox validates the native messaging configuration for Firefox.
func (nm *NativeMessagingConfig) ValidateFirefox() error {
	if nm.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if !firefoxAllowedName.MatchString(nm.Name) {
		return fmt.Errorf("name %q must be word characters separated by single dots", nm.Name)
	}
	if len(nm.AllowedExtensions) == 0 {
		return fmt.Errorf("allowed_extensions must list at least one extension id for firefox")
	}
	if nm.Path != "" && !filepath.IsAbs(nm.Path) {
		return fmt.Errorf("path %q must be an absolute path", nm.Path)
	}
	return nil
}

// NativeMessagingHelper describes a native messaging host that is shipped
// inside an app bundle, so that a browser extension can launch it.
//
// The helper is an executable placed in the bundle's Contents/Helpers
// directory, alongside a manifest for each browser it serves. The manifest
// names the absolute path of the helper, which is where the bundle will be
// once installed rather than where it is built, hence InstalledBundlePath.
type NativeMessagingHelper struct {
	// Executable is the path to the built helper binary that is copied into
	// the bundle.
	Executable string

	// Name is the name of the helper within the bundle. It defaults to the
	// base name of Executable.
	Name string

	// Config is the manifest for the helper. Its Path is filled in by
	// Manifest and need not be set; Type defaults to "stdio".
	Config NativeMessagingConfig

	// InstalledBundlePath is the location the bundle will occupy once
	// installed, such as /Applications/Example.app. The manifest must name
	// the helper's path as it will be at runtime, which is not where the
	// bundle is built. It defaults to the bundle's own path, which is
	// appropriate when the bundle is used from where it is built.
	InstalledBundlePath string
}

// name returns the name of the helper within the bundle.
func (h NativeMessagingHelper) name() string {
	if h.Name != "" {
		return h.Name
	}
	return filepath.Base(h.Executable)
}

// Helpers returns the path to elem within the bundle's Contents/Helpers
// directory, which is where a native messaging helper is placed.
func (b AppBundle) Helpers(elem ...string) string {
	return filepath.Join(b.Path, "Contents", "Helpers", filepath.Join(elem...))
}

// NativeMessagingHelperPath returns the absolute path of the helper as it
// will be once the bundle is installed, which is the path named by the
// manifest.
func (b AppBundle) NativeMessagingHelperPath(h NativeMessagingHelper) string {
	root := h.InstalledBundlePath
	if root == "" {
		root = b.Path
	}
	return filepath.Join(root, "Contents", "Helpers", h.name())
}

// NativeMessagingManifestPath returns the path within the bundle of the
// manifest written for browser, relative to the bundle's Resources directory.
// An installer copies this file into the directory returned by
// NativeMessagingHostsDir.
func (b AppBundle) NativeMessagingManifestPath(h NativeMessagingHelper, browser BrowserType) string {
	return b.Resources("NativeMessagingHosts", browser.String(), h.Config.Name+".json")
}

// NativeMessagingManifest returns the manifest for h as it is written into
// the bundle, with Path set to the helper's installed location and Type
// defaulted to "stdio". If a browser is specified, the configuration is
// tailored to that browser (e.g. AllowedOrigins for Chrome/Edge,
// AllowedExtensions for Firefox).
func (b AppBundle) NativeMessagingManifest(h NativeMessagingHelper, browser ...BrowserType) NativeMessagingConfig {
	cfg := h.Config
	cfg.Path = b.NativeMessagingHelperPath(h)
	if cfg.Type == "" {
		cfg.Type = "stdio"
	}
	if len(browser) > 0 {
		switch browser[0] {
		case Chrome, Edge:
			cfg.AllowedExtensions = nil
		case Firefox:
			cfg.AllowedOrigins = nil
		}
	}
	return cfg
}

// AddNativeMessagingHelper returns the steps required to add a native
// messaging helper to the bundle: the Contents/Helpers directory is created,
// the helper is copied into it, and a manifest is written into
// Contents/Resources/NativeMessagingHosts/<browser> for each browser. Each
// manifest is validated for the browser it is written for. Safari does not
// use native messaging manifests and is rejected.
func (b AppBundle) AddNativeMessagingHelper(h NativeMessagingHelper, browsers ...BrowserType) []Step {
	if h.Executable == "" {
		return []Step{ErrorStep(fmt.Errorf("helper executable path not specified"), "cp")}
	}
	if len(browsers) == 0 {
		return []Step{ErrorStep(fmt.Errorf("no browsers specified for helper %q", h.name()), "manifest")}
	}
	for _, browser := range browsers {
		if browser == Safari {
			return []Step{ErrorStep(fmt.Errorf("safari does not use native messaging manifests: a web extension sends native messages to its containing .appex, see AddSafariWebExtension"), "manifest", "safari")}
		}
	}
	steps := []Step{
		MkdirAll(b.Helpers()),
		Copy(h.Executable, b.Helpers(h.name())),
	}
	for _, browser := range browsers {
		cfg := b.NativeMessagingManifest(h, browser)
		steps = append(steps,
			cfg.Validate(browser),
			MkdirAll(b.Resources("NativeMessagingHosts", browser.String())),
			WriteJSONFile(cfg, b.NativeMessagingManifestPath(h, browser)),
		)
	}
	return steps
}

// SignNativeMessagingHelper returns the step required to sign the helper
// within the app bundle. A helper is a separate executable and so must be
// signed before the bundle that contains it.
func (b AppBundle) SignNativeMessagingHelper(signer Signer, h NativeMessagingHelper) Step {
	return b.SignHelper(signer, h.name())
}

// InstallNativeMessagingManifest returns the step required to install the
// manifest for h into the directory that browser searches, so that an
// extension can launch the helper. It is intended for development use; a
// shipped application would normally install its own manifest when it runs.
// Safari does not use native messaging manifests and is rejected.
func (b AppBundle) InstallNativeMessagingManifest(h NativeMessagingHelper, browser BrowserType, scope NativeMessagingScope) []Step {
	if browser == Safari {
		return []Step{ErrorStep(fmt.Errorf("safari does not use native messaging manifests: a web extension sends native messages to its containing .appex, see AddSafariWebExtension"), "install", "safari")}
	}
	home, err := os.UserHomeDir()
	if err != nil && scope == UserScope {
		return []Step{ErrorStep(fmt.Errorf("failed to determine home directory: %w", err), "install")}
	}
	dir, err := NativeMessagingHostsDir(browser, scope, home)
	if err != nil {
		return []Step{ErrorStep(err, "install")}
	}
	cfg := b.NativeMessagingManifest(h, browser)
	return []Step{
		cfg.Validate(browser),
		MkdirAll(dir),
		WriteJSONFile(cfg, filepath.Join(dir, cfg.Name+".json")),
	}
}

// AddHelperExecutable returns the steps required to add an additional
// executable to the bundle's Contents/Helpers directory, such as a tool that a
// native messaging helper invokes, or a helper shared by more than one
// browser. name defaults to the base name of src.
//
// Each executable added this way is a separate Mach-O and so must be signed in
// its own right, before the bundle that contains it, see SignHelper.
func (b AppBundle) AddHelperExecutable(src string, name ...string) []Step {
	if src == "" {
		return []Step{ErrorStep(fmt.Errorf("helper executable path not specified"), "cp", "", "")}
	}
	dst := filepath.Base(src)
	if len(name) > 0 && name[0] != "" {
		dst = name[0]
	}
	return []Step{
		MkdirAll(b.Helpers()),
		Copy(src, b.Helpers(dst)),
	}
}

// SignHelper returns the step required to sign the named executable within the
// bundle's Contents/Helpers directory.
func (b AppBundle) SignHelper(signer Signer, name string) Step {
	if name == "" {
		return ErrorStep(fmt.Errorf("helper name not specified"), "codesign", "")
	}
	return signer.SignPath(b.Path, filepath.Join("Contents", "Helpers", name))
}
