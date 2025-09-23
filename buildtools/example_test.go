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
)

// This example demonstrates how to create a basic macOS application bundle structure
// with Info.plist and copy resources into it.
func Example_createAppBundle() {
	// Create a temporary directory for the example
	tempDir, err := os.MkdirTemp("", "example_app_bundle_*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test executable (just a placeholder file for the example)
	exeContent := []byte("#!/bin/bash\necho 'Hello from Example App'")
	exePath := filepath.Join(tempDir, "ExampleExecutable")
	if err := os.WriteFile(exePath, exeContent, 0755); err != nil {
		log.Fatalf("Failed to create example executable: %v", err)
	}

	// Define the app bundle with required Info.plist contents
	bundle := buildtools.AppBundle{
		Path: filepath.Join(tempDir, "ExampleApp.app"),
		Info: buildtools.InfoPlist{
			Identifier:   "io.cloudeng.ExampleApp",
			Name:         "Example App",
			Version:      "1.0.0",
			ShortVersion: "1.0",
			Executable:   "ExampleExecutable",
			IconSet:      "AppIcon",
			Type:         "APPL",
		},
	}
	ctx := context.Background()

	runner := buildtools.NewRunner()

	// Get the steps to create the basic bundle structure
	runner.AddSteps(bundle.Create()...)
	runner.AddSteps(bundle.CopyContents(exePath, "MacOS", "ExampleExecutable"))
	results := runner.Run(ctx, buildtools.NewCommandRunner())
	if err := results.Error(); err != nil {
		log.Fatalf("Failed to create app bundle: %v", err)
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
