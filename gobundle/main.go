// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var verbose bool

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		printHelpAndExit()
	}

	verbose = len(os.Getenv(verboseEnvVar)) != 0

	if len(os.Args) < 2 {
		rungoExit(ctx)
		return
	}

	verb := os.Args[1]
	var mergedConfig []byte
	if verb != "run" {
		merged, err := readAndMergeConfigs()
		if err != nil {
			exit(1, "error loading config: %v\n", err)
		}
		mergedConfig = merged
	}

	switch verb {
	case "__runsign__":
		if len(os.Args) < 3 {
			exit(1, "no path provided to sign\n")
		}
		binary := os.Args[2]
		runAndExit(func() error {
			return handleGoRunExec(ctx, mergedConfig, binary)
		})
	case "build":
		runAndExit(func() error {
			return handleGoBuild(ctx, mergedConfig, os.Args[2:])
		})
	case "install":
		runAndExit(func() error {
			return handleGoInstall(ctx, mergedConfig, os.Args[2:])
		})
	case "run":
		handleGoRun(ctx, os.Args[2:])
	default:
		rungoExit(ctx, os.Args[1:]...)
	}
}

func rungo(ctx context.Context, args []string) error {
	if verbose {
		fmt.Printf("go %v\n", strings.Join(args, " "))
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func rungoExit(ctx context.Context, args ...string) {
	runAndExit(func() error {
		return rungo(ctx, args)
	})
}

func exit(code int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(code)
}

func runAndExit(fn func() error) {
	err := fn()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		exit(1, "error: %v\n", err)
	}
	os.Exit(0)
}
