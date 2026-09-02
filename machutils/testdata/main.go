// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"cloudeng.io/macos/machutils"
)

func main() {
	var (
		checkUID = flag.Int64("check-uid", -1, "expected UID to check against")
		mockPPID = flag.Int("mock-ppid", 0, "mock parent PID for error testing")
	)
	flag.Parse()

	if *mockPPID != 0 {
		_ = os.Setenv("TEST_MOCK_PPID", strconv.Itoa(*mockPPID))
	}

	uid, err := machutils.GetParentUID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if *checkUID >= 0 && int64(uid) != *checkUID {
		fmt.Fprintf(os.Stderr, "UID mismatch: expected %d, got %d\n", *checkUID, uid)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stdout, "%d\n", uid)
}
