// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"cloudeng.io/macos/buildtools"
)

const privKeyFile = "chrome_extension_private_key.pem"

// A simple tool to generate a stable Chrome Extension ID for development.
func main() {
	if len(os.Args) == 1 {
		if _, err := os.Stat(privKeyFile); err == nil {
			log.Fatalf("Key file %q already exists, refusing to overwrite", privKeyFile)
		}
		br := buildtools.Browser{}
		privKey, _, err := br.CreateChromeExtensionID()
		if err != nil {
			log.Fatalf("Failed to generate Chrome Extension ID: %v", err)
		}
		if err := os.WriteFile(privKeyFile, privKey, 0400); err != nil {
			log.Fatalf("Failed to write key file: %v", err)
		}
		return
	}
	for _, arg := range os.Args[1:] {
		if err := printValues(arg); err != nil {
			log.Fatalf("Failed to print values for %q: %v", arg, err)
		}
	}
}

func printValues(file string) error {
	br := buildtools.Browser{}
	publicKey, extensionID, err := br.ReadChromeExtensionID(file)
	if err != nil {
		log.Fatalf("Failed to generate Chrome Extension ID from %q: %v", file, err)
	}
	fmt.Printf("Key file: %s\n", file)
	fmt.Printf("Public key (add as \"key\" to the browser manifest):\n%s\n", base64.StdEncoding.EncodeToString(publicKey))
	fmt.Printf("Extension ID (use as allowed_origin in native host manifest): %s\n", extensionID)
	return nil
}
