// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools_test

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"

	"cloudeng.io/macos/buildtools"
)

const plistPreamble = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`

func getEntitlements(pfe *buildtools.PerFileEntitlements, p string) ([]byte, error) {
	ent, ok := pfe.For(p)
	if !ok {
		return nil, fmt.Errorf("no entitlements found for %q", p)
	}
	data, err := ent.MarshalIndent(" ")
	if err != nil {
		return nil, fmt.Errorf("marshal: %v", err)
	}
	return data, nil
}

func TestEntitlements(t *testing.T) {
	yamlData := `file1:
  com.apple.security.app-sandbox: true
  com.apple.security.network.client: false
  keychain-access-groups:
    - a
    - b
file2:
  com.apple.security.app-sandbox: false
  com.apple.security.network.client: true
`
	file1 := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
  <key>com.apple.security.network.client</key>
  <false/>
  <key>keychain-access-groups</key>
  <array>
   <string>a</string>
   <string>b</string>
  </array>
 </dict>
</plist>`

	file2 := plistPreamble + ` <dict>
  <key>com.apple.security.app-sandbox</key>
  <false/>
  <key>com.apple.security.network.client</key>
  <true/>
 </dict>
</plist>`

	var pfe buildtools.PerFileEntitlements
	if err := yaml.Unmarshal([]byte(yamlData), &pfe); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	efor := func(p string) []byte {
		data, err := getEntitlements(&pfe, p)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	pl := efor("file1")

	if got, want := string(pl), file1; got != want {
		t.Fatalf("unexpected plist data\ngot:\n%s\nwant:\n%s", got, want)
	}
	pl = efor("file2")
	if got, want := string(pl), file2; got != want {
		t.Fatalf("unexpected plist data\ngot:\n%s\nwant:\n%s", got, want)
	}
}
