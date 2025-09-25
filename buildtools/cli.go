// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"flag"
	"fmt"
	"os"

	"cloudeng.io/cmdutil/flags"
)

// CommonFlags represents flags commonly used by buildtools command line tools.
type CommonFlags struct {
	DryRun     bool   `subcmd:"dry-run,false,'if set, execute the commands in dry-run mode'"`
	Release    bool   `subcmd:"swift-release,false,'if set, use swift release build, otherwise debug'"`
	BundlePath string `subcmd:"bundle-path,'','path for the output bundle, overrides any specified in a configf ile'"`
	Signer     string `subcmd:"signer,'','signing identity to use, overrides any specified in a config file'"`
	ConfigFile string `subcmd:"config-file,'spec.yaml','path to the build specification yaml file'"`
}

// RegisterCommonFlagsOrDie registers an instance of CommonFlags with the provided
// FlagSet, panicing on error.
func RegisterCommonFlagsOrDie(f *CommonFlags, fs *flag.FlagSet) {
	if err := flags.RegisterFlagsInStruct(fs, "subcmd", f, nil, nil); err != nil {
		panic(err)
	}
}

// SwiftBinDir returns the path to the swift binary directory for the specified
// build type.
func (f CommonFlags) SwiftBinDirOrDie(ctx context.Context) string {
	bin, err := SwiftBinDir(ctx, f.Release)
	if err != nil {
		panic(err)
	}
	return bin
}

// CommandRunnerOptions returns options for the CommandRunner based on the flags.
func (f CommonFlags) CommandRunnerOptions() []CommandRunnerOption {
	if f.DryRun {
		return []CommandRunnerOption{WithDryRun(f.DryRun)}
	}
	return []CommandRunnerOption{}
}

// Config represents common configuration options
// that can be read from a yaml config file.
type Config struct {
	Bundle  string        `yaml:"bundle"`
	Signing SigningConfig `yaml:"signing"`
}

// SigningConfig represents signing related configuration
// that can be read from a yaml config file.
type SigningConfig struct {
	Identity            string               `yaml:"identity"`
	CodeSignArguments   []string             `yaml:"codesign-args"`
	Entitlements        *Entitlements        `yaml:"entitlements"`
	PerFileEntitlements *PerFileEntitlements `yaml:"perfile_entitlements"`
}

// Signer returns a Signer based on the configuration.
func (s SigningConfig) Signer() Signer {
	return NewSigner(s.Identity, s.Entitlements, s.PerFileEntitlements, s.CodeSignArguments)
}

// PrintResultAndExitOnErrorf prints the results of running steps and exits with a non-zero
// status if any of the steps failed.
func PrintResultAndExitOnErrorf(result RunResult) {
	if err := result.Error(); err != nil {
		for _, r := range result {
			fmt.Println(r.String())
		}
		os.Exit(1)
	}
	for _, r := range result {
		fmt.Println(r.CommandLine())
	}
}
