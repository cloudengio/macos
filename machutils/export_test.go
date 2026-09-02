// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package machutils

// SetGetppidForTesting overrides getppid for testing and returns a restore function.
func SetGetppidForTesting(fn func() int) func() {
	orig := getppid
	getppid = fn
	return func() {
		getppid = orig
	}
}
