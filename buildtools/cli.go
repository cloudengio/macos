// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"flag"
	"fmt"
	"os"

	"cloudeng.io/cmdutil/flags"
	"gopkg.in/yaml.v3"
)

// CommonFlags represents flags commonly used by buildtools command line tools.
type CommonFlags struct {
	DryRun     bool   `subcmd:"dry-run,false,'if set, execute the commands in dry-run mode'"`
	Timing     bool   `subcmd:"timing,false,'if set, print timing information for each step'"`
	Release    bool   `subcmd:"swift-release,false,'if set, use swift release build, otherwise debug'"`
	BundlePath string `subcmd:"bundle-path,'','path for the output bundle, overrides any specified in a config file'"`
	Signer     string `subcmd:"signer,'','signing identity to use, overrides any specified in a config file'"`
	ConfigFile string `subcmd:"config,'spec.yaml','path to the build specification yaml file'"`
	Verbose    bool   `subcmd:"verbose,false,'if set, print verbose output'"`
}

// RegisterFlagsOrDie registers a struct that contains an instance of CommonFlags with the provided
// FlagSet, panicing on error.
func RegisterFlagsOrDie(f any, fs *flag.FlagSet) {
	if err := flags.RegisterFlagsInStruct(fs, "subcmd", f, nil, nil); err != nil {
		panic(err)
	}
}

// CommandRunnerOptions returns options for the CommandRunner based on the flags.
func (f CommonFlags) CommandRunnerOptions() []CommandRunnerOption {
	opts := []CommandRunnerOption{}
	if f.DryRun {
		opts = append(opts, WithDryRun(f.DryRun))
	}
	if f.Timing {
		opts = append(opts, WithCommandTiming(f.Timing))
	}
	return opts
}

// StepRunnerOptions returns options for the StepRunner based on the flags.
func (f CommonFlags) StepRunnerOptions() []StepRunnerOption {
	if f.Timing {
		return []StepRunnerOption{WithStepTiming(f.Timing)}
	}
	return nil
}

// ParseFile parses the specified config file into cfg.
func (f CommonFlags) ParseFile(cfg any) error {
	data, err := os.ReadFile(f.ConfigFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

// Config represents common configuration options
// that can be read from a yaml config file.
type Config struct {
	AppBundle string        `yaml:"bundle"`
	Signing   SigningConfig `yaml:"signing"`
}

// SigningConfig represents signing related configuration
// that can be read from a yaml config file.
type SigningConfig struct {
	Identity            string               `yaml:"identity"`
	CodesignArguments   []string             `yaml:"codesign-args"`
	Entitlements        *Entitlements        `yaml:"entitlements"`
	PerFileEntitlements *PerFileEntitlements `yaml:"perfile_entitlements"`
}

// Signer returns a Signer based on the configuration.
func (s SigningConfig) Signer() Signer {
	return NewSigner(s.Identity, s.Entitlements, s.PerFileEntitlements, s.CodesignArguments)
}

// PrintResultAndExitOnErrorf prints the results of running steps and exits with a non-zero
// status if any of the steps failed.
func (f CommonFlags) PrintResultAndExitOnErrorf(spec any, result RunResult) {
	err := result.Error()
	verbose := f.Verbose || err != nil
	if verbose {
		if out, err := yaml.Marshal(spec); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal spec parsed from %v: %v\n", f.ConfigFile, err)
		} else {
			fmt.Printf("%v: %s\n", f.ConfigFile, out)
		}
		for _, r := range result {
			if r.Error() != nil {
				fmt.Println(r.String())
				continue
			}
			fmt.Println(r.CommandLine())
		}
	}
	if err != nil {
		os.Exit(1)
	}
}
