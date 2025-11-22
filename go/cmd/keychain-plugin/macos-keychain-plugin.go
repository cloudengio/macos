// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:generate gobundle install ./macos-keychain-plugin.go
package main

import (
	"context"
	"log/slog"
	"os"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/macos/keychain/plugin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)
	srv := plugin.NewServer(logger)
	cfg, req, resp := srv.ReadRequest(ctx, os.Stdin)
	if resp != nil {
		srv.SendResponse(ctx, os.Stdout, resp)
		return
	}
	resp = srv.HandleRequest(ctx, cfg, req)
	srv.SendResponse(ctx, os.Stdout, resp)
}
