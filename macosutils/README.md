# Package [cloudeng.io/macos/macosutils](https://pkg.go.dev/cloudeng.io/macos/macosutils?tab=doc)

```go
import cloudeng.io/macos/macosutils
```

Package macosutils contains macos specific utilities.

## Functions
### Func InAppBundle
```go
func InAppBundle(binary string) (string, bool)
```
InAppBundle determines if the running process is within an app bundle
and returns the path of the requested binary inside that bundle.
It uses the heuristic of checking if the executable is located under
.../<app-bundle>/Contents/MacOS and that <app-bundle> satisfies IsAppBundle.

### Func IsAppBundle
```go
func IsAppBundle(path string) bool
```
IsAppBundle returns true if path is a directory ending with .app and
contains a Contents/Info.plist file.

### Func LocateInBundle
```go
func LocateInBundle(bundlePath, binary string) string
```
LocateInBundle finds the requested binary in the specified app bundle and
returns its path within that bundle.

### Func LookPathBundle
```go
func LookPathBundle(bundle, pathList string) string
```
LookPathBundle is like exec.LookPath but for app bundles.

### Func LookPathBundleAll
```go
func LookPathBundleAll(bundle, pathList string) []string
```
LookPathBundleAll is like LookPathBundle but returns all instances of bundle
on pathList without duplicates.




