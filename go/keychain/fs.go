// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package keychain

import "context"

func (kc T) ReadFileCtx(ctx context.Context, service string) ([]byte, error) {
	return kc.ReadSecureNote(service)
}

func (kc T) WriteFileCtx(ctx context.Context, service string, data []byte) error {
	return kc.WriteSecureNote(service, data)
}
