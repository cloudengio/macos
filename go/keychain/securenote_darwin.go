// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

// Package keychain provides a simple interface for reading and writing
// secure notes to the macOS keychain.
package keychain

// The following are important references for working with the macOS keychain:
// https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains
// https://developer.apple.com/forums/thread/724013

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"

	"github.com/cloudengio/go-keychain"
)

// Option represents an option for configuring a keychain.T
type Option func(o *options)

type options struct {
	updateInPlace bool
	accessibility Accessibility
}

func WithUpdateInPlace(v bool) Option {
	return func(o *options) {
		o.updateInPlace = v
	}
}

func WithAccessibility(v Accessibility) Option {
	return func(o *options) {
		o.accessibility = v
	}
}

const (
	// KeychainFileBased represents the file-based keychain.
	// This is the legacy, local only, file based keychain.
	KeychainFileBased Type = iota
	// KeychainDataProtectionLocal represents the data protection
	// keychain which is local, but integrated with the system's secure
	// enclave. Applications that use must be signed and have
	// appropriate entitlements.
	KeychainDataProtectionLocal
	// KeychainICloud represents the iCloud keychain that can be synced
	// across devices.
	// Applications that use must be signed and have appropriate
	// entitlements.
	KeychainICloud
)

// Accessibility is the items accessibility
type Accessibility int

const (
	AccessibleDefault                        = keychain.AccessibleDefault
	AccessibleWhenUnlocked                   = keychain.AccessibleWhenUnlocked
	AccessibleAfterFirstUnlock               = keychain.AccessibleAfterFirstUnlock
	AccessibleAlways                         = keychain.AccessibleAlways
	AccessibleWhenPasscodeSetThisDeviceOnly  = keychain.AccessibleWhenPasscodeSetThisDeviceOnly
	AccessibleWhenUnlockedThisDeviceOnly     = keychain.AccessibleWhenUnlockedThisDeviceOnly
	AccessibleAfterFirstUnlockThisDeviceOnly = keychain.AccessibleAfterFirstUnlockThisDeviceOnly
	AccessibleAccessibleAlwaysThisDeviceOnly = keychain.AccessibleAccessibleAlwaysThisDeviceOnly
)

func (t Type) String() string {
	switch t {
	case KeychainFileBased:
		return "file"
	case KeychainDataProtectionLocal:
		return "data-protection-local"
	case KeychainICloud:
		return "icloud"
	default:
		return "unknown"
	}
}

// ParseKeychainType parses a string into a KeychainType.
func ParseKeychainType(s string) (Type, error) {
	switch s {
	case "file":
		return KeychainFileBased, nil
	case "data-protection-local", "data-protection", "local":
		return KeychainDataProtectionLocal, nil
	case "icloud":
		return KeychainICloud, nil
	default:
		return 0, fmt.Errorf("invalid keychain type: %s", s)
	}
}

// T represents a keychain that can be used to read and write secure notes.
type T struct {
	typ     Type
	opts    options
	account string
}

func newKeychain(readonly bool, typ Type, account string, opts ...Option) *T {
	var options options
	options.accessibility = keychain.AccessibleWhenUnlocked
	for _, opt := range opts {
		opt(&options)
	}
	if readonly && options.updateInPlace {
		panic("updateInPlace cannot be true for a readonly keychain")
	}
	return &T{typ: typ, account: account, opts: options}
}

// New creates a new Keychain.
func New(typ Type, account string, opts ...Option) *T {
	return newKeychain(false, typ, account, opts...)
}

// NewReadonly creates a new readonly Keychain.
func NewReadonly(typ Type, account string, opts ...Option) SecureNoteReader {
	return newKeychain(true, typ, account, opts...)
}

func (kc T) configure(item *keychain.Item) {
	item.SetSecClass(keychain.SecClassGenericPassword)
	switch kc.typ {
	case KeychainFileBased:
	case KeychainDataProtectionLocal:
		item.SetDataProtectionKeyChain(true)
	case KeychainICloud:
		item.SetSynchronizable(keychain.SynchronizableYes)
	}
}

// WriteSecureNote writes a secure note to the keychain. It will update
// an existing note if it WithUpdateInPlace was set to true.
func (kc T) WriteSecureNote(service string, data []byte) error {
	item := kc.newItem(service, data)
	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem && kc.opts.updateInPlace {
		return kc.UpdateSecureNote(service, data)
	}
	return err
}

// UpdateSecureNote updates an existing secure note in the keychain.
func (kc T) UpdateSecureNote(service string, data []byte) error {
	item := keychain.NewItem()
	item.SetData(data)
	query := kc.queryItem(kc.account, service)
	return keychain.UpdateItem(query, item)
}

func (kc T) queryItem(account, service string) keychain.Item {
	query := keychain.NewItem()
	kc.configure(&query)
	query.SetService(service)
	query.SetAccount(account)
	query.SetReturnData(true)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnAttributes(true)
	return query
}

func (kc T) newItem(service string, data []byte) keychain.Item {
	item := keychain.NewItem()
	kc.configure(&item)
	item.SetService(service)
	item.SetAccount(kc.account)
	item.SetDescription("secure note")
	item.SetData(data)
	item.SetAccessible(keychain.Accessible(kc.opts.accessibility))
	return item
}

func (kc T) queryNote(service string) (keychain.QueryResult, error) {
	query := kc.queryItem(kc.account, service)
	results, err := keychain.QueryItem(query)
	if err != nil {
		return keychain.QueryResult{}, err
	}
	if len(results) == 0 {
		return keychain.QueryResult{}, fs.ErrNotExist
	}
	return results[0], nil
}

// ReadSecureNote reads a secure note from the keychain.
func (kc T) ReadSecureNote(service string) ([]byte, error) {
	result, err := kc.queryNote(service)
	if err != nil {
		return nil, err
	}
	data, err := extractKeychainNote(result.Data)
	if err == io.EOF {
		// Maybe not an XML plist document.
		if len(result.Data) > 0 {
			return result.Data, nil
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return data, err
}

type plist struct {
	Dict dict `xml:"dict"`
}

type dict struct {
	Entries []entry `xml:",any"`
}

type entry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func extractKeychainNote(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewBuffer(data))
	var pl plist
	if err := dec.Decode(&pl); err != nil {
		return nil, err
	}
	for i, v := range pl.Dict.Entries {
		if v.XMLName.Local == "key" && v.Value == "NOTE" {
			if i+1 < len(pl.Dict.Entries) && pl.Dict.Entries[i+1].XMLName.Local == "string" {
				return []byte(pl.Dict.Entries[i+1].Value), nil
			}
		}
	}
	return nil, fs.ErrNotExist
}
