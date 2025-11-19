// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !darwin

package keychain

import "errors"

// Option represents an option for configuring a keychain.T
type Option func(o *options)

type options struct{}

type T struct{}

// NewKeychain creates a new Keychain.
func NewKeychain(typ KeychainType, account string, opts ...Option) *T {
	return &T{}
}

// NewKeychainReadonly creates a new readonly Keychain.
func NewKeychainReadonly(typ KeychainType, account string, opts ...Option) SecureNoteReader {
	return T{}
}

func (u T) ReadSecureNote(service string) ([]byte, error) {
	return nil, errors.New("not implemented on this platform")
}

func (u T) UpdateSecureNote(service string, data []byte) error {
	return errors.New("not implemented on this platform")
}

func (u T) WriteSecureNote(service, account string, data []byte) error {
	return errors.New("not implemented on this platform")
}
