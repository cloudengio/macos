// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"cloudeng.io/macos/buildtools"
)

const privKeyFile = "chrome_extension_key.pem"

// A simple tool to generate a stable Chrome Extension ID for development.
func main() {
	br := buildtools.Browser{}
	if len(os.Args) == 1 {
		if _, err := os.Stat(privKeyFile); err == nil {
			log.Fatalf("Key file %q already exists, refusing to overwrite", privKeyFile)
		}
		key, extensionID, err := br.CreateChromeExtensionID()
		if err != nil {
			log.Fatalf("Failed to generate Chrome Extension ID: %v", err)
		}
		fmt.Printf("Generated Chrome Extension ID: %s\n", extensionID)
		if err := os.WriteFile(privKeyFile, key, 0400); err != nil {
			log.Fatalf("Failed to write key file: %v", err)
		}
		return
	}
	for _, arg := range os.Args[1:] {
		extensionID, err := br.ChromeExtensionID(arg)
		if err != nil {
			log.Fatalf("Failed to generate Chrome Extension ID from %q: %v", arg, err)
		}
		fmt.Printf("Generated Chrome Extension ID from %q: %s\n", arg, extensionID)
	}
}
