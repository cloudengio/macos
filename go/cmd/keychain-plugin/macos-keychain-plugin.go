// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:generate gobundle install ./macos-keychain-plugin.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/macos/keychain/plugin"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--help" {
		fmt.Printf("macos-keychain-plugin is a plugin for the macOS keychain.\n")
		fmt.Printf("To install it run go generate in the go/cmd/keychain-plugin directory\n")
		fmt.Printf("taking care to set up the appropriate Apple signing identity and provisioning profile.\n")
		return
	}
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
