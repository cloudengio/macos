// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

// This example demonstrates how to create a basic macOS application bundle structure
// with Info.plist and copy resources into it.
func Example_createAppBundle() {
	// Create a temporary directory for the example
	tempDir, err := os.MkdirTemp("", "example_app_bundle_*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create a test executable (just a placeholder file for the example)
	exeContent := []byte("#!/bin/bash\necho 'Hello from Example App'")
	exePath := filepath.Join(tempDir, "ExampleExecutable")
	if err := os.WriteFile(exePath, exeContent, 0755); err != nil { //nolint:gosec // G306
		fmt.Printf("Failed to create example executable: %v", err)
		return
	}

	plistYAML := `
CFBundleIdentifier: io.cloudeng.TestApp
CFBundleName: TestApp
CFBundleVersion: 1.0.0
CFBundleShortVersionString: 1.0
CFBundleExecutable: TestExecutable
CFBundlePackageType: APPL
LSMinimumSystemVersion: "15.0"
CFBundleDisplayName: Swift UI Example
`

	var info buildtools.InfoPlist
	if err := yaml.Unmarshal([]byte(plistYAML), &info); err != nil {
		fmt.Printf("failed to unmarshal info plist: %v", err)
		return
	}

	// Define the app bundle with required Info.plist contents
	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "ExampleApp.app"),
		Info: info,
	}
	ctx := context.Background()

	runner := buildtools.NewRunner()

	// Get the steps to create the basic bundle structure
	runner.AddSteps(bundle.Create()...)
	runner.AddSteps(bundle.WriteInfoPlist())
	runner.AddSteps(bundle.CopyContents(exePath, "MacOS", "ExampleExecutable"))
	results := runner.Run(ctx, buildtools.NewCommandRunner())
	if err := results.Error(); err != nil {
		fmt.Printf("Failed to create app bundle: %v\n", err)
	}
	for _, result := range results {
		fmt.Printf("Step executed: %v %v\n", result.Executable(), result.Error() == nil)
	}

	// Output:
	// Step executed: mkdir true
	// Step executed: mkdir true
	// Step executed: mkdir true
	// Step executed: write Info.plist true
	// Step executed: cp true
}

// ExampleAppBundle_AddNativeMessagingHelper shows how to build an app bundle
// that contains a native messaging helper, so that a browser extension can
// launch it. The helper is placed in the bundle and a manifest is written for
// each browser, naming the helper's path as it will be once the bundle is
// installed.
func ExampleAppBundle_AddNativeMessagingHelper() {
	bundle := buildtools.AppBundle{
		Path: filepath.Join(os.TempDir(), "Example.app"),
		Info: buildtools.InfoPlist{
			CFBundleIdentifier: "io.cloudeng.Example",
			CFBundleName:       "Example",
			CFBundleExecutable: "Example",
		},
	}

	helper := buildtools.NativeMessagingHelper{
		Executable: "bin/example-helper",
		// The manifest must name the helper where it will be at runtime,
		// which is not where the bundle is built.
		InstalledBundlePath: "/Applications/Example.app",
		Config: buildtools.NativeMessagingConfig{
			Name:        "io.cloudeng.example",
			Description: "Example native helper",
			AllowedOrigins: []string{
				"chrome-extension://abcdefghijklmnopabcdefghijklmnop/",
			},
		},
	}

	// The steps are added to a runner along with those that create the
	// bundle, sign it and so on.
	steps := bundle.Create()
	steps = append(steps, bundle.WriteInfoPlist())
	steps = append(steps, bundle.AddNativeMessagingHelper(helper, buildtools.Chrome)...)
	_ = steps

	fmt.Println(bundle.NativeMessagingHelperPath(helper))
	fmt.Println(filepath.Base(bundle.NativeMessagingManifestPath(helper, buildtools.Chrome)))

	dir, err := buildtools.NativeMessagingHostsDir(buildtools.Chrome, buildtools.UserScope, "/Users/example")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	// Output:
	// /Applications/Example.app/Contents/Helpers/example-helper
	// io.cloudeng.example.json
	// /Users/example/Library/Application Support/Google/Chrome/NativeMessagingHosts
}

// ExampleAppBundle_AddSafariWebExtension shows how to build an app bundle that
// contains a Safari web extension. Safari does not launch a separate native
// messaging host: the extension is packaged as an .appex in the app's PlugIns
// directory, and native messages are delivered to that .appex's principal
// class, which is the native helper.
func ExampleAppBundle_AddSafariWebExtension() {
	bundle := buildtools.AppBundle{
		Path: filepath.Join(os.TempDir(), "Example.app"),
		Info: buildtools.InfoPlist{
			CFBundleIdentifier: "io.cloudeng.Example",
			CFBundleName:       "Example",
			CFBundleExecutable: "Example",
		},
	}

	ext := buildtools.SafariWebExtension{
		Name:       "Example Extension",
		Executable: "bin/ExampleHandler",
		// The directory holding the web extension itself: manifest.json and
		// everything it references.
		Resources:      "extension",
		PrincipalClass: "ExampleExtension.SafariWebExtensionHandler",
		Info: buildtools.InfoPlist{
			// An app extension's identifier must be prefixed by that of the
			// app that contains it.
			CFBundleIdentifier: "io.cloudeng.Example.Extension",
			CFBundleName:       "Example Extension",
			CFBundleExecutable: "ExampleHandler",
		},
	}

	steps := bundle.Create()
	steps = append(steps, bundle.WriteInfoPlist())
	steps = append(steps, bundle.AddSafariWebExtension(ext)...)
	_ = steps

	fmt.Println(filepath.Base(bundle.SafariWebExtensionPath(ext)))
	fmt.Println(buildtools.SafariWebExtensionPointIdentifier)

	// Safari has no native messaging manifest to install.
	if _, err := buildtools.NativeMessagingHostsDir(buildtools.Safari, buildtools.UserScope, "/Users/example"); err != nil {
		fmt.Println("no manifest directory for safari")
	}

	// Output:
	// Example Extension.appex
	// com.apple.Safari.web-extension
	// no manifest directory for safari
}
