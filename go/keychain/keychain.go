// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychain

// KeychainType represents the type of keychain to use.
type KeychainType int

// SecureNoteReader defines the interface for reading secure notes from the keychain.
type SecureNoteReader interface {
	ReadSecureNote(service string) (data []byte, err error)
}
