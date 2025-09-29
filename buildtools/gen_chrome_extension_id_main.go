// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"fmt"
	"log"

	"cloudeng.io/macos/buildtools"
)

// A simple tool to generate a stable Chrome Extension ID for development.
func main() {
	br := buildtools.Browser{}
	extensionID, err := br.ChromeExtensionID()
	if err != nil {
		log.Fatalf("Failed to generate Chrome Extension ID: %v", err)
	}
	fmt.Printf("Generated Chrome Extension ID: %s\n", extensionID)
}
