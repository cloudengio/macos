// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/macos/keychain/plugin"
	"cloudeng.io/security/keys/keychain/plugins"
)

const cmdSpec = `name: keychain
summary: provide access to local keychains across multiple operating systems
commands:
  - name: read
    summary: read items from the keychain
    arguments:
      - <item-name>
  - name: write
    summary: write items to the keychain
    arguments:
      - <filename>
`

func cli() *subcmd.CommandSetYAML {
	cmd := subcmd.MustFromYAML(cmdSpec)
	var pluginCmd pluginCmd
	cmd.Set("read").MustRunner(pluginCmd.Read, &ReadFlags{})
	cmd.Set("write").MustRunner(pluginCmd.Write, &WriteFlags{})
	return cmd
}

func main() {
	ctx := context.Background()
	subcmd.Dispatch(ctx, cli())
}

type pluginCmd struct{}

type VerboseFlag struct {
	Verbose bool `subcmd:"verbose,false,enable verbose logging"`
}

type ReadFlags struct {
	plugin.ReadFlags
}

type WriteFlags struct {
	plugin.WriteFlags
	Name string `subcmd:"name,,name of the item to use instead of the filename"`
}

func (pluginCmd) Read(ctx context.Context, f any, args []string) error {
	fl := f.(*ReadFlags)
	cfg := fl.Config()
	fs := plugins.NewFS(cfg.Binary, cfg)
	contents, err := fs.ReadFileCtx(ctx, args[0])
	if err != nil {
		return handleError(err)
	}
	fmt.Printf("%s", string(contents))
	return nil
}

func (pluginCmd) Write(ctx context.Context, f any, args []string) error {
	fl := f.(*WriteFlags)
	cfg := fl.Config()
	fs := plugins.NewFS(cfg.Binary, cfg)
	filename := args[0]
	name := filepath.Base(filename)
	if fl.Name != "" {
		name = fl.Name
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Printf("writing item %q to keychain\n", name)
	err = fs.WriteFileCtx(ctx, name, contents)
	return handleError(err)
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	pluginErr := plugins.AsError(err)
	if pluginErr == nil {
		return err
	}
	fmt.Printf("plugin error: %s: %s\n", pluginErr.Message, pluginErr.Detail)
	if pluginErr.Stderr != "" {
		fmt.Printf("plugin stderr: %s\n", pluginErr.Stderr)
	}
	return err
}
