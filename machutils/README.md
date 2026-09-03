# Package [cloudeng.io/macos/machutils](https://pkg.go.dev/cloudeng.io/macos/machutils?tab=doc)

```go
import cloudeng.io/macos/machutils
```

Package machutils provides low level utilities for interacting with the
macOS kernel.

## Variables
### ErrFailedToRetrieveParentUID
```go
ErrFailedToRetrieveParentUID = errors.New("failed to retrieve parent process UID from the kernel")

```
ErrFailedToRetrieveParentUID is returned when the parent process UID cannot
be retrieved from the kernel.



## Functions
### Func EnsureParentProcessSafe
```go
func EnsureParentProcessSafe() error
```
EnsureParentProcessSafe checks that:
 1. the parent process UID matches the executable's UID
 2. the current process UID matches that of the parent
 3. the executable is not group- or world-writable or executable
 4. the current process is not running with elevated privileges (SUID/SGID)
 5. the process has not been orphaned (reparented to launchd)

1 ensures that only the executable's owner can launch it, 2 ensures that the
current process UID matches that of the executable owner, 3 ensures that
the executable cannot be modified or run by other users, 4 ensures that the
process has not escalated privileges, and 5 ensures that parent identity
cannot be spoofed via orphaning.

### Func GetExecutableInfo
```go
func GetExecutableInfo() (uint32, os.FileInfo, error)
```
GetExecutableInfo retrieves the executable path, its owner UID, and the file
info of the executable.

### Func GetParentUID
```go
func GetParentUID() (uint32, error)
```
GetParentUID retrieves the real user ID (RUID) of the parent process.




