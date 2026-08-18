# Package [cloudeng.io/macos/macosutils](https://pkg.go.dev/cloudeng.io/macos/macosutils?tab=doc)

```go
import cloudeng.io/macos/macosutils
```


## Functions
### Func InAppBundle
```go
func InAppBundle(binary string) (string, bool)
```
InAppBundle determines if binary is an app bundle and returns the path of
the bundle. It uses the simple heurestic of checking to see if the binary
has parents .../<app-bundle>/Contents/MacOS and that <app-bundle> satisfies
InAppBundle.

### Func IsAppBundle
```go
func IsAppBundle(path string) bool
```
IsAppBundle returns true if path is a directory ending .app and contains a
Contents/Info.plist.

### Func LocateInBundle
```go
func LocateInBundle(bundlePath, binary string) string
```
LocateInBundle finds the requested binary in the specified app bundle
returns its absolute path.

### Func LookPathBundle
```go
func LookPathBundle(bundle, pathList string) string
```
LookPathBundle is like exec.LookPath but for app bundles.

### Func LookPathBundleAll
```go
func LookPathBundleAll(bundle, pathList string) []string
```
LookPathBundle is like LookPathBundle but returns all instances of bundle on
pathList without duplicates.




