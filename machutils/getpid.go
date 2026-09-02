// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package machutils

/*
#include <sys/types.h>
#include <sys/sysctl.h>
#include <unistd.h>
#include <errno.h>

// Helper function to safely fetch the real UID of a specific PID via sysctl
int64_t get_process_ruid(pid_t pid) {
    struct kinfo_proc info;
    size_t length = sizeof(info);

    // Define the Management Information Base (MIB) query path
    int mib[4] = { CTL_KERN, KERN_PROC, KERN_PROC_PID, pid };

    // Call the macOS kernel sysctl interface
    if (sysctl(mib, 4, &info, &length, NULL, 0) < 0) {
        return -1; // System call failed (e.g., process exited or permission denied)
    }

    if (length == 0) {
        return -1; // No process data returned
    }

    // kp_eproc.e_pcred.p_ruid holds the immutable REAL user identity assigned at login,
    // protecting it against tricks involving temporary privilege escalations (SUID).
    return info.kp_eproc.e_pcred.p_ruid;
}
*/
import "C"

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// ErrFailedToRetrieveParentUID is returned when the parent process UID
// cannot be retrieved from the kernel.
var ErrFailedToRetrieveParentUID = errors.New("failed to retrieve parent process UID from the kernel")

// GetParentUID handles the safe extraction of the caller's User ID context.
func GetParentUID() (uint32, error) {
	// 1. Get the parent's Process ID directly from the running Go environment
	ppid := os.Getppid()

	// 2. Pass the PID to our C layer to query the kernel
	cUID := int64(C.get_process_ruid(C.pid_t(ppid)))

	// 3. Evaluate results (-1 indicates a failure to read the process credentials)
	if cUID == int64(-1) {
		return 0, ErrFailedToRetrieveParentUID
	}

	return uint32(cUID), nil
}

// GetExecutableOwnerInfo retrieves the UID and permissions
// of the currently running executable.
func GetExecutableOwnerInfo() (uint32, fs.FileMode, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, 0, err
	}

	fileInfo, err := os.Stat(execPath)
	if err != nil {
		return 0, 0, err
	}

	systat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("failed to retrieve file stat information")
	}

	return systat.Uid, fileInfo.Mode().Perm(), nil
}

// EnsureParentProcessSafe checks that:
//  1. the parent process UID matches the executable's UID
//  2. the current process UID matches that of the parent
//  3. the executable is not world-executable
//
// 1 ensures that the only the executable's owner can launch it,
// 2 ensures that the current process is not running with elevated
// privileges, and 3 ensures that the executable can only be run by
// its owner and not by any other user or group on the system.
func EnsureParentProcessSafe() error {
	parentUID, err := GetParentUID()
	if err != nil {
		return err
	}
	execUID, perms, err := GetExecutableOwnerInfo()
	if err != nil {
		return err
	}
	if perms&011 != 0 {
		return errors.New("executable is group or world executable; potential privilege escalation detected")
	}
	if parentUID != execUID {
		return errors.New("parent process UID does not match executable UID; potential privilege escalation detected")
	}
	currentUID := uint32(os.Getuid())
	if currentUID != execUID {
		return errors.New("current process UID does not match executable UID; potential privilege escalation detected")
	}
	return nil
}
