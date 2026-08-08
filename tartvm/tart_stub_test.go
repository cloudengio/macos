// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package tartvm

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fakeTart puts a stub tart on PATH that records the arguments it was called
// with, writes stdout and stderr, and exits with exitCode. It returns a
// function reporting the invocations made so far, so that tests can assert on
// whether tart was run at all as well as on the result.
func fakeTart(t *testing.T, stdout, stderr string, exitCode int) func() []string {
	t.Helper()
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + strconv.Quote(calls) + "\n" +
		"printf '%s' " + strconv.Quote(stdout) + "\n" +
		"printf '%s' " + strconv.Quote(stderr) + " >&2\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "tart"), []byte(script), 0700); err != nil { //nolint:gosec // test stub must be executable
		t.Fatalf("writing tart stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		buf, err := os.ReadFile(calls)
		if err != nil {
			return nil // never invoked
		}
		return strings.Split(strings.TrimSpace(string(buf)), "\n")
	}
}
