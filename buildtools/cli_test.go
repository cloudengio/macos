// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"testing"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

const cliConfig = `bundle: ./testing.app
signing:
    identity: "Apple Development: some id"
    entitlements:
      com.apple.security.app-sandbox: true
    perfile_entitlements:
      xpcHelper:
        com.apple.security.app-sandbox: false
      Contents/MacOS/xpcHelper:
        com.apple.security.app-sandbox: false
`

func TestCLIConfig(t *testing.T) {

	var cfg buildtools.Config
	if err := yaml.Unmarshal([]byte(cliConfig), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got, want := cfg.Bundle, "./testing.app"; got != want {
		t.Fatalf("unexpected bundle, got %q, want %q", got, want)
	}
	if got, want := cfg.Signing.Identity, "Apple Development: some id"; got != want {
		t.Fatalf("unexpected signing identity, got %q, want %q", got, want)
	}
	if cfg.Signing.Entitlements == nil {
		t.Fatal("expected entitlements but got nil")
	}

	pl, err := cfg.Signing.Entitlements.MarshalIndent(" ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	expectedEntitlements := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
 </dict>
</plist>`

	if got, want := string(pl), expectedEntitlements; got != want {
		t.Errorf("unexpected entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}

	efor := func(p string) []byte {
		data, err := getEntitlements(cfg.Signing.PerFileEntitlements, p)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	pl = efor("xpcHelper")

	expectedPerFileEntitlements := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <false/>
 </dict>
</plist>`

	if got, want := string(pl), expectedPerFileEntitlements; got != want {
		t.Errorf("unexpected per file entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}

	pl = efor("Contents/MacOS/xpcHelper")

	if got, want := string(pl), expectedPerFileEntitlements; got != want {
		t.Errorf("unexpected per file entitlements, got:\n%s\nwant to contain:\n%s", got, want)
	}
}
