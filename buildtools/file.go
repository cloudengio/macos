// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"fmt"
	"strings"
)

func MkdirAll(d string) Step {
	if d == "" {
		return ErrorStep(fmt.Errorf("cannot create directory with empty name"), "mkdir", "-p")
	}
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "mkdir", "-p", d)
	})
}

// DirExists returns a Step that checks for the existence of the directory.
func DirExists(d string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "test", "-d", d)
	})
}

// FileExists returns a Step that checks for the existence of the file.
func FileExists(f string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "test", "-f", f)
	})
}

/*
// InputFile represents an input file.
type InputFile string

// Exists returns a Step that checks for the existence of the file.
func (f InputFile) Exists() Step {
	return StepFunc(func(_ context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(context.Background(), "test", "-f", string(f))
	})
}

// Dir returns the directory containing the file.
func (f InputFile) Dir() Dir {
	return Dir(filepath.Dir(string(f)))
}

// OutputFile represents an output file.
type OutputFile string

// Dir returns the directory containing the file.
func (f OutputFile) Dir() Dir {
	return Dir(filepath.Dir(string(f)))
}*/

// IconSetDir represents a directory for an icon set.
type IconSetDir string

// IsValidIsValidIconSetDir returns a Step that checks if the directory has a .iconset extension.
func IsValidIconSetDir(id IconSetDir) Step {
	p := string(id)
	return StepFunc(func(_ context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return HasSuffix(".iconset").Check(p).Run(context.Background(), cmdRunner)
	})
}

/*
// Exists returns a Step that checks for the existence of the directory.
func (id IconSetDir) Exists() Step {
	return Dir(id).Exists()
}*/

// Rename retrurns a Step that renames a file using mv.
func Rename(oldname, newname string) Step {
	return StepFunc(func(_ context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(context.Background(), "mv", string(oldname), newname)
	})
}

// Copy returns a Step that copies a file using cp.
func Copy(oldname, newname string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "cp", oldname, newname)
	})
}

// CopyDir returns a Step that copies a directory recursively using cp -r.
func CopyDir(srcDir, dstDir string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "cp", "-r", srcDir, dstDir)
	})
}

// SwiftBinDir returns the directory containing the swift build products.
func SwiftBinDir(ctx context.Context, release bool) (string, error) {
	runner := NewCommandRunner()
	args := []string{"build", "--show-bin-path"}
	if release {
		args = append(args, "--configuration", "release")
	}
	r, err := runner.Run(ctx, "swift", args...)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(r.Output()), err
}
