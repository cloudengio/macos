// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/keys/keyscmd"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/file"
	"cloudeng.io/macos/keychain/plugin"
	"cloudeng.io/security/keys/keychain/plugins"
	"github.com/aws/smithy-go"
)

const cmdSpec = `name: cloudeng-aws-secrets
summary: provide access to aws secretsmanager on macos with aws credentials stored in the keychain
commands:
  - name: read
    summary: read a secret from aws secretsmanager writing it to filename, if filename is - the secret will be written to stdout
    arguments:
      - <filename>
  - name: write
    summary: write a secret read from <filename>, to aws secretsmanager, if filename is - the secret will be read from stdin
    arguments:
      - <filename>
`

func cli() *subcmd.CommandSetYAML {
	cmd := subcmd.MustFromYAML(cmdSpec)
	var secretsCmd secretsCmd
	cmd.Set("read").MustRunner(secretsCmd.Read, &ReadFlags{})
	cmd.Set("write").MustRunner(secretsCmd.Write, &WriteFlags{})
	return cmd
}

func main() {
	ctx := context.Background()
	subcmd.Dispatch(ctx, cli())
}

type secretsCmd struct{}

type ARNFlags struct {
	ARN string `subcmd:"arn,,arn of the secret to use instead of the filename"`
}

type Flags struct {
	awsconfig.AWSFlags
	plugin.ReadFlags
	KeychainItem string `subcmd:"keychain-item,,keychain item to use instead of the filename"`
	ARNFlags
}

type ReadFlags struct {
	Flags
}

type WriteFlags struct {
	Flags
}

func (sc secretsCmd) config(ctx context.Context, writeable bool, fv Flags) (context.Context, *awssecretsfs.T, error) {
	if fv.ARN == "" {
		return ctx, nil, fmt.Errorf("missing secret ARN or name; use --arn to specify it")
	}
	if fv.KeychainItem == "" {
		return ctx, nil, fmt.Errorf("no keychain item provided")
	}
	if fv.AWSKeyInfoID == "" {
		return ctx, nil, fmt.Errorf("no key info ID provided")
	}
	var fs file.ReadFileFS
	if rwfs, ok := file.ReadWriteFSFromContext(ctx); ok {
		fs = rwfs
	} else if rfs, ok := file.FSFromContext(ctx); ok && len(rfs) > 0 {
		fs = rfs[0]
	} else {
		kcCfg, err := fv.ReadFlags.Config()
		if err != nil {
			return ctx, nil, fmt.Errorf("failed to get keychain config from flags: %w", err)
		}
		kcFS, err := kcCfg.FS(false)
		if err != nil {
			return ctx, nil, fmt.Errorf("failed to create keychain fs: %w", err)
		}
		fs = kcFS
	}
	ims := keys.NewInMemoryKeyStore()
	if err := ims.ReadYAML(ctx, fs, fv.KeychainItem); err != nil {
		return ctx, nil, fmt.Errorf("failed to read: %v: %w", fv.KeychainItem, err)
	}
	ctx = keys.ContextWithKeyStore(ctx, ims)
	fv.AWS = true
	cfg := fv.AWSFlags.Config()
	awscfg, err := cfg.Load(ctx)
	if err != nil {
		return ctx, nil, err
	}
	if writeable {
		return ctx, awssecretsfs.New(awscfg,
			awssecretsfs.WithAllowCreation(true),
			awssecretsfs.WithAllowUpdates(true)), nil
	}
	return ctx, awssecretsfs.New(awscfg), nil
}

func (sc secretsCmd) Read(ctx context.Context, f any, args []string) error {
	fl := f.(*ReadFlags)
	ctx, fs, err := sc.config(ctx, false, fl.Flags)
	if err != nil {
		return handleError(err)
	}
	return handleError(keyscmd.SafeWriteToLocal(ctx, fs, fl.ARN, args[0], 0600))
}

func (sc secretsCmd) Write(ctx context.Context, f any, args []string) error {
	fl := f.(*WriteFlags)
	ctx, fs, err := sc.config(ctx, true, fl.Flags)
	if err != nil {
		return handleError(err)
	}
	return handleError(keyscmd.ReadFromLocal(ctx, args[0], fs, fl.ARN, 0600))
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	name := filepath.Base(os.Args[0])
	if pluginErr := plugins.AsError(err); pluginErr != nil {
		fmt.Fprintf(os.Stderr, "%s: keychain plugin error: %s: %s\n", name, pluginErr.Message, pluginErr.Detail)
		if pluginErr.Stderr != "" {
			fmt.Fprintf(os.Stderr, "%s: keychain plugin stderr: %s\n", name, pluginErr.Stderr)
		}
	}
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		if opErr, ok := errors.AsType[*smithy.OperationError](err); ok {
			fmt.Fprintf(os.Stderr, "%s: AWS error (%s/%s): %s: %s\n", name, opErr.Service(), opErr.Operation(), apiErr.ErrorCode(), apiErr.ErrorMessage())
		} else {
			fmt.Fprintf(os.Stderr, "%s: AWS API error: %s: %s\n", name, apiErr.ErrorCode(), apiErr.ErrorMessage())
		}
	}
	return err
}
