// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MkdirAll returns a Step that creates a directory and all necessary parents using mkdir -p.
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

// IconSetDir represents a directory for an icon set.
type IconSetDir string

// IsValidIsValidIconSetDir returns a Step that checks if the directory has a .iconset extension.
func IsValidIconSetDir(id IconSetDir) Step {
	p := string(id)
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return Suffix(".iconset").Assert(p).Run(ctx, cmdRunner)
	})
}

// Rename returns a Step that renames a file using mv.
func Rename(oldname, newname string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "mv", string(oldname), newname)
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

// RSync returns a Step that synchronizes files and directories using rsync.
func RSync(src, dst string, args ...string) Step {
	allArgs := append([]string{"-a", "--delete"}, args...)
	allArgs = append(allArgs, src, dst)
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		return cmdRunner.Run(ctx, "rsync", allArgs...)
	})
}

// WriteFile returns a Step that writes data to the specified path with the specified permissions.
func WriteFile(data []byte, perm os.FileMode, elems ...string) Step {
	return StepFunc(func(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
		path := filepath.Join(elems...)
		err := os.WriteFile(path, data, os.FileMode(perm))
		return NewStepResult("os.WriteFile", []string{path, fmt.Sprintf("%o", perm)}, nil, err), err
	})
}

func WriteJSON(v any, perm os.FileMode, elems ...string) Step {
	data, err := json.Marshal(v)
	if err != nil {
		return ErrorStep(fmt.Errorf("failed to marshal json: %w", err), "json.Marshal")
	}
	return WriteFile(data, perm, elems...)
}
