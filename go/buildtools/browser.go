// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
)

// Browser represents a web browser.
type Browser struct{}

// The mapping used by Chrome to convert hexadecimal digits (0-9, a-f)
// to the first 16 letters of the alphabet (a-p).
const chromeAlphabet = "abcdefghijklmnop"

// generateExtensionID takes a DER-encoded public key and returns the Chrome extension ID.
func generateExtensionID(publicKey []byte) (string, error) {
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
			return "", fmt.Errorf("invalid hex character in string: %c", char)
		}
		// Map the value (0-15) to the corresponding letter in the chromeAlphabet.
		extensionID[i] = chromeAlphabet[val]
	}

	return string(extensionID), nil
}

// CreateChromeExtensionID generates a stable Chrome Extension ID suitable for development use.
// Note that this ID is derived from a newly generated RSA key pair each time
// the function is called, so it will be different on each invocation.
// For a stable ID, you would need to persist the generated key pair.
func (b Browser) CreateChromeExtensionID() ([]byte, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate RSA key pair: %v", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	// MarshalPKCS1PrivateKey returns the DER encoding of the private key.
	// To obtain PEM encoding, wrap the DER bytes using pem.Encode or pem.EncodeToMemory.
	// Example:
	pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// 2. Extract the public key and encode it into the DER PKCS#1 format
	// required for the SHA-256 hash calculation by Chrome.
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	if publicKeyBytes == nil {
		return nil, "", fmt.Errorf("failed to marshal public key to PKCS#1 format")
	}

	// 3. Generate the final extension ID.
	id, err := generateExtensionID(publicKeyBytes)
	return pemBytes, id, err
}

// ReadChromeExtensionID reads the RSA private key from the specified PEM-encoded file
// to obtain the public key and corresponding Chrome Extension ID.
func (b Browser) ReadChromeExtensionID(keyFile string) ([]byte, string, error) {
	// Read the private key from the specified file.
	privateKeyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read private key file: %v", err)
	}

	// Decode the PEM block containing the private key.
	block, _ := pem.Decode(privateKeyData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, "", fmt.Errorf("failed to decode PEM block containing RSA private key")
	}

	// Parse the RSA private key.
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse RSA private key: %v", err)
	}

	// Extract the public key and encode it into the DER PKCS#1 format
	// required for the SHA-256 hash calculation by Chrome.
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	id, err := generateExtensionID(publicKeyBytes)
	return publicKeyBytes, id, err
}

type BrowserType int

const (
	Chrome BrowserType = iota
	Firefox
	Safari
	Edge
)

func (b BrowserType) String() string {
	switch b {
	case Chrome:
		return "chrome"
	case Firefox:
		return "firefox"
	case Safari:
		return "safari"
	case Edge:
		return "edge"
	default:
		return "unknown"
	}
}

// NativeMessagingConfig represents the configuration for a native messaging host.
type NativeMessagingConfig struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`                         // "stdio" or one of the other allowed communication types
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`    // chrome-extension://<extension-id>/
	AllowedExtensions []string `json:"allowed_extensions,omitempty"` // firefox extension ids
}

// Validate validates the native messaging configuration for the specified browser.
func (nm *NativeMessagingConfig) Validate(browser BrowserType) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		switch browser {
		default:
			err := fmt.Errorf("unsupported browser: %q", browser)
			return NewStepResult("validate native messaging config", nil, nil, err), err
		case Chrome:
			err := nm.ValidateChrome()
			return NewStepResult("validate chrome native messaging config", nil, nil, err), err
		}
	})
}

var (
	chromeAllowedCharacters = regexp.MustCompile(`^[a-z0-9_.]+$`)
	chromeNoConsecutiveDots = regexp.MustCompile(`\.\.`)
)

// ValidateChrome validates the native messaging configuration for Chrome.
func (nm *NativeMessagingConfig) ValidateChrome() error {
	// Name validation: lowercase alphanumeric, underscores, dots; cannot start/end with dot; no consecutive dots.
	name := nm.Name
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	// Only allowed characters
	if !chromeAllowedCharacters.MatchString(name) {
		return fmt.Errorf("name %q must only contain lowercase alphanumeric characters, underscores, and dots", name)
	}
	// Cannot start or end with a dot
	if name[0] == '.' || name[len(name)-1] == '.' {
		return fmt.Errorf("name %q cannot start or end with a dot", name)
	}
	// No consecutive dots
	if chromeNoConsecutiveDots.FindStringIndex(name) != nil {
		return fmt.Errorf("name %q cannot contain consecutive dots", name)
	}
	return nil
}

// AppendChromeOrigin appends the specified Chrome extension ID to the list of allowed origins.
func (nm *NativeMessagingConfig) AppendChromeOrigin(extensionID string) {
	nm.AllowedOrigins = append(nm.AllowedOrigins, fmt.Sprintf("chrome-extension://%s/", extensionID))
}
