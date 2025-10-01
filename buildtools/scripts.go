// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"fmt"
	"strings"
)

type MacOSBashScript struct {
	out *strings.Builder
}

const installPreamble = `#!/usr/bin/env bash
set -euo pipefail
mount="$1"
installer_pid="$2"
target="$3"

`

func NewInstallScript() *MacOSBashScript {
	scr := &MacOSBashScript{
		out: &strings.Builder{},
	}
	scr.out.WriteString(installPreamble)
	return scr
}

func (b *MacOSBashScript) AddLine(line string) {
	b.out.WriteString(line + "\n")
}

func (b *MacOSBashScript) InstallChromeNativeMessagingManifest(systemWide bool, manifestPath string) {
	location := "$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
	if systemWide {
		location = "/Library/Google/Chrome/NativeMessagingHosts"
	}
	fmt.Fprintf(b.out, `
location="%s"
echo "Installing Chrome native messaging manifest in $location"
mkdir -p "$location"
cp "$target/%s" "$location"
`, location, manifestPath)
}

func (b *MacOSBashScript) Bytes() []byte {
	return []byte(b.out.String())
}
