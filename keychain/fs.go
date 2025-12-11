// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package keychain

import (
	"context"
	"io/fs"
)

func (kc T) ReadFileCtx(_ context.Context, service string) ([]byte, error) {
	return kc.ReadSecureNote(service)
}

func (kc T) WriteFileCtx(_ context.Context, service string, data []byte, _ fs.FileMode) error {
	return kc.WriteSecureNote(service, data)
}

func (kc T) ReadFile(service string) ([]byte, error) {
	return kc.ReadFileCtx(context.Background(), service)
}

func (kc T) WriteFile(service string, data []byte, _ fs.FileMode) error {
	return kc.WriteFileCtx(context.Background(), service, data, 0600)
}
