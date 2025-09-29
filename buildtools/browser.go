// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
)

// Browser represents a web browser.
type Browser struct{}

// The mapping used by Chrome to convert hexadecimal digits (0-9, a-f)
// to the first 16 letters of the alphabet (a-p).
const chromeAlphabet = "abcdefghijklmnop"

// generateExtensionID takes a DER-encoded public key and returns the Chrome extension ID.
func generateExtensionID(publicKey []byte) string {
	// 1. Calculate the SHA-256 hash of the public key.
	hash := sha256.Sum256(publicKey)

	// 2. Take the first 16 bytes (128 bits) of the hash.
	// This is the source for the 32-character ID.
	hash16Bytes := hash[:16]

	// 3. Convert the 16 bytes to a 32-character hexadecimal string.
	hexString := hex.EncodeToString(hash16Bytes)

	// 4. Map the hex string characters (0-9, a-f) to the Chrome alphabet (a-p).
	extensionID := make([]byte, 32)
	for i, char := range hexString {
		// Convert the hex character to its integer value (0-15).
		var val int
		if char >= '0' && char <= '9' {
			val = int(char - '0')
		} else if char >= 'a' && char <= 'f' {
			val = int(char - 'a' + 10)
		} else {
			// This shouldn't happen with hex.EncodeToString, but for safety:
			log.Fatalf("Invalid hex character in string: %c", char)
		}
		// Map the value (0-15) to the corresponding letter in the chromeAlphabet.
		extensionID[i] = chromeAlphabet[val]
	}

	return string(extensionID)
}

// ChromeExtensionID generates a stable Chrome Extension ID suitable for development use.
// Note that this ID is derived from a newly generated RSA key pair each time
// the function is called, so it will be different on each invocation.
// For a stable ID, you would need to persist the generated key pair.
func (b Browser) ChromeExtensionID() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("Failed to generate RSA key pair: %v", err)
	}

	// 2. Extract the public key and encode it into the DER PKCS#1 format
	// required for the SHA-256 hash calculation by Chrome.
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	if publicKeyBytes == nil {
		return "", fmt.Errorf("Failed to marshal public key to PKCS#1 format.")
	}

	// 3. Generate the final extension ID.
	extensionID := generateExtensionID(publicKeyBytes)

	return extensionID, nil
}
