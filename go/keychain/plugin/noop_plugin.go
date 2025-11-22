// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

// This file contains an example implementation of a keychain plugin.
package main

import (
	"flag"
	"log/slog"
	"os"

	"cloudeng.io/macos/keychain/plugin"
	"cloudeng.io/security/keys/keychain/plugins"
)

var storeFlag string
var logFile string

func main() {
	flag.StringVar(&storeFlag, "store", "", "Path to the keychain store")
	flag.StringVar(&logFile, "logfile", "", "Path to the log file")
	flag.Parse()

	lf, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer lf.Close()

	logger := slog.New(slog.NewJSONHandler(lf, nil))
	p := plugin.NewPluginServer(logger)

	cfg, req, resp := p.ReadRequest(os.Stdin)
	if resp != nil {
		p.SendResponse(os.Stdout, resp)
		return
	}
	logger.Info("Received request", "request", req, "config", cfg)
	if req.Write {
		os.WriteFile(storeFlag, []byte(req.Contents), 0600)
	} else {
		contents, err := os.ReadFile(storeFlag)
		if err != nil {
			resp = &plugins.Response{
				ID: req.ID,
				Error: &plugins.Error{
					Message: "failed to read from store",
					Detail:  err.Error(),
				},
			}
			p.SendResponse(os.Stdout, resp)
			return
		}
		resp = &plugins.Response{
			ID:       req.ID,
			Contents: string(contents),
		}
	}
	p.SendResponse(os.Stdout, resp)
}
