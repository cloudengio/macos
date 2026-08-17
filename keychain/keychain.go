// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychain

import (
	"encoding/json"
	"fmt"

	"cloudeng.io/cmdutil/flags"
	"gopkg.in/yaml.v3"
)

// SecureNoteReader defines the interface for reading secure notes from the keychain.
type SecureNoteReader interface {
	ReadSecureNote(service string) (data []byte, err error)
}

// Type represents the type of keychain to use, it maps to the underlying
// system keychain types but also includes the pseudotype 'all' which
// can be used, only when reading, to specify all keychains.
type Type int

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
	// KeychainAll represents any keychain type, it can only be used for
	// reading and indicates that all keychains will be searched for
	// the requested item.
	KeychainAll
)

func (t Type) MarshalYAML() (any, error) {
	return t.String(), nil
}

func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t Type) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *Type) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("failed to decode keychain type: %w", err)
	}
	kt, err := ParseType(s)
	if err != nil {
		return err
	}
	*t = kt
	return nil
}

func (t *Type) UnmarshalText(text []byte) error {
	kt, err := ParseType(string(text))
	if err != nil {
		return err
	}
	*t = kt
	return nil
}

func (t *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("failed to unmarshal keychain type from JSON: %w", err)
	}
	kt, err := ParseType(s)
	if err != nil {
		return err
	}
	*t = kt
	return nil
}

func (t Type) String() string {
	return flags.Enum[Type]{Value: t}.String()
}

// EnumValues satisfies flags.EnumType[Type].
func (Type) EnumValues() map[string]Type {
	return map[string]Type{
		"file":                  KeychainFileBased,
		"data-protection-local": KeychainDataProtectionLocal,
		"icloud":                KeychainICloud,
		"all":                   KeychainAll,
	}
}

// ParseType parses a string into a KeychainType.
func ParseType(s string) (Type, error) {
	var e flags.Enum[Type]
	if err := e.Set(s); err != nil {
		return 0, err
	}
	return e.Value, nil
}

/*

// WriteType is like Type, except that it does not allow the value 'all'.
type WriteType int

// EnumValues satisfies flags.EnumType[ReadType].
func (WriteType) EnumValues() map[string]WriteType {
	return map[string]WriteType{
		"file":                  WriteType(KeychainFileBased),
		"data-protection-local": WriteType(KeychainDataProtectionLocal),
		"icloud":                WriteType(KeychainICloud),
	}
}

// ParseWriteType parses a string into a KeychainType.
func ParseWriteType(s string) (WriteType, error) {
	var e flags.Enum[WriteType]
	if err := e.Set(s); err != nil {
		return 0, err
	}
	if e.Value == WriteType(KeychainAll) {
		return 0, fmt.Errorf("keychain type %s cannot be used for writing", KeychainAll)
	}
	return e.Value, nil
}

func (wt *WriteType) UnmarshalYAML(node *yaml.Node) error {
	var t Type
	if err := node.Decode(&t); err != nil {
		return fmt.Errorf("failed to decode keychain type: %w", err)
	}
	if t == KeychainAll {
		return fmt.Errorf("keychain type %s is not supported for writing", t)
	}
	*wt = WriteType(t)
	return nil
}

func (wt *WriteType) UnmarshalText(text []byte) error {
	kt, err := ParseWriteType(string(text))
	if err != nil {
		return err
	}
	*wt = kt
	return nil
}

func (wt *WriteType) UnmarshalJSON(data []byte) error {
	var t Type
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("failed to unmarshal keychain type from JSON: %w", err)
	}
	if t == KeychainAll {
		return fmt.Errorf("keychain type %s is not supported for writing", t)
	}
	*wt = WriteType(t)
	return nil
}
*/
