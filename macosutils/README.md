# Package [cloudeng.io/macos/macosutils](https://pkg.go.dev/cloudeng.io/macos/macosutils?tab=doc)

```go
import cloudeng.io/macos/macosutils
```

Package macosutils contains macos specific utilities.

## Variables
### ErrFailedToLaunch, ErrLaunchedAppFailed
```go
ErrFailedToLaunch = errors.New("failed to launch application")
ErrLaunchedAppFailed = errors.New("launched application failed")

```



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

### Func IsServiceInstalled
```go
func IsServiceInstalled(serviceLabel string) bool
```
IsServiceInstalled returns true if the specified service is installed in the
current user's LaunchAgents directory.

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

### Func LookupBundleBinary
```go
func LookupBundleBinary(bundle, binary, pathList string) (string, string)
```
LookupBundleBinary iterates over all instances of bundle in pathList to
locate the first one that contains binary returning the absolute pathname of
the bundle and binary in that bundle or empty strings if not found.



## Types
### Type LaunchOption
```go
type LaunchOption func(o *launchOptions)
```
LaunchOption defines a function type for configuring the Launcher.

### Functions

```go
func WithCmdEnv(env func() []string) LaunchOption
```


```go
func WithStdoutStderr(stdout, stderr io.Writer) LaunchOption
```


```go
func WithWorkingDir(dir string) LaunchOption
```




### Type Launcher
```go
type Launcher struct {
	// contains filtered or unexported fields
}
```
Launcher provides an interface for launching macOS applications.

### Functions

```go
func NewLauncher(opts ...LaunchOption) *Launcher
```
NewLauncher creates a new Launcher with the provided options.



### Methods

```go
func (l *Launcher) LaunchApp(cmd string, args ...string) error
```
LaunceApp launches a long-running application and waits for it to exit.
Use TerminateLaunchedApp to signal the application to exit. Interrupt and
terminate signals are forwarded to the launched application.


```go
func (l *Launcher) RunApp(cmd string, args ...string) (string, error)
```
RunApp executes the specified command with the provided arguments and
returns its combined output. Use it for short running commands, eg.
to configure or setup the application.


```go
func (l *Launcher) TerminateLaunchedApp()
```







