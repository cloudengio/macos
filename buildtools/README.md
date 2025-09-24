# Package [cloudeng.io/macos/buildtools](https://pkg.go.dev/cloudeng.io/macos/buildtools?tab=doc)

```go
import cloudeng.io/macos/buildtools
```


## Functions
### Func MarshalInfoPlist
```go
func MarshalInfoPlist(info InfoPlist) ([]byte, error)
```

### Func SwiftBinDir
```go
func SwiftBinDir(ctx context.Context, release bool) (string, error)
```
SwiftBinDir returns the directory containing the swift build products.



## Types
### Type AppBundle
```go
type AppBundle struct {
	Path string
	Info InfoPlist
}
```
AppBundle represents a macOS application bundle.

### Methods

```go
func (b AppBundle) Contents(elem ...string) string
```
Contents returns the path to the specified element within the app bundle's
Contents directory.


```go
func (b AppBundle) CopyContents(src string, dst ...string) Step
```
CopyContents returns the step required to copy a file into the app bundle
dst is relative to the bundle Contents root.


```go
func (b AppBundle) CopyIcons(src string) Step
```


```go
func (b AppBundle) Create() []Step
```
Create returns the steps required to create the app bundle directory
structure and Info.plist.


```go
func (b AppBundle) Sign(signer Signer) Step
```


```go
func (b AppBundle) SignContents(signer Signer, dst ...string) Step
```
SignContents returns the step required to sign a file within the app bundle,
dst is relative to the bundle Contents root.


```go
func (b AppBundle) VerifySignatures(signer Signer) []Step
```




### Type AsPNG
```go
type AsPNG struct {
	InputPath  string
	OutputPath string
}
```

### Methods

```go
func (j AsPNG) Convert() Step
```




### Type CommandRunner
```go
type CommandRunner struct {
	// contains filtered or unexported fields
}
```
CommandRunner executes system commands.

### Functions

```go
func NewCommandRunner(opts ...CommandRunnerOption) *CommandRunner
```
NewCommandRunner creates a new CommandRunner with the provided options.



### Methods

```go
func (r *CommandRunner) DryRun() bool
```


```go
func (r *CommandRunner) Run(ctx context.Context, name string, args ...string) (StepResult, error)
```
Run executes the specified command with arguments and returns the combined
output and any error encountered.


```go
func (r *CommandRunner) WriteFile(ctx context.Context, path string, data []byte, perm uint32) (string, error)
```




### Type CommandRunnerOption
```go
type CommandRunnerOption func(o *commandRunnerOptions)
```
CommandRunnerOption configures a CommandRunner.

### Functions

```go
func WithDryRun(dryRun bool) CommandRunnerOption
```
WithDryRun configures the CommandRunner to simulate command execution
without actually running commands.


```go
func WithStderr(w io.Writer) CommandRunnerOption
```
WithStderr configures the CommandRunner to write standard error to the
provided io.Writer.


```go
func WithStdout(w io.Writer) CommandRunnerOption
```
WithStdout configures the CommandRunner to write standard output to the
provided io.Writer.




### Type IconSet
```go
type IconSet struct {
	Icon       string
	IconSetDir string
	IconSet    string
}
```

### Methods

```go
func (i IconSet) CreateIcns() Step
```


```go
func (i IconSet) CreateIcons(twoX bool, sizes ...int) []Step
```




### Type IconSetDir
```go
type IconSetDir string
```
IconSetDir represents a directory for an icon set.


### Type InfoPlist
```go
type InfoPlist struct {
	Identifier   string       `plist:"CFBundleIdentifier,omitempty" yaml:"identifier"`
	Name         string       `plist:"CFBundleName,omitempty" yaml:"name"`
	Version      string       `plist:"CFBundleVersion,omitempty" yaml:"version"`
	ShortVersion string       `plist:"CFBundleShortVersionString,omitempty" yaml:"short_version"`
	Executable   string       `plist:"CFBundleExecutable,omitempty" yaml:"executable"`
	IconSet      string       `plist:"CFBundleIconFile,omitempty" yaml:"icon_set"`
	Type         string       `plist:"CFBundlePackageType"`
	XPCService   XPCInfoPlist `plist:"XPCService,omitempty" yaml:"xpc_service"`
}
```
InfoPlist captures the common fields for an app bundle's Info.plist.


### Type Resources
```go
type Resources struct {
	Identity      string `yaml:"identity"` // Apple developer identity
	Entitlements  string `yaml:"entitlements"`
	Executable    string `yaml:"executable"`
	XPCExecutable string `yaml:"xpc_executable"` // optional
	Icon          string `yaml:"icon"`           // optional
	IconSetDir    string `yaml:"icon_dir"`       // optional
	IconSetName   string `yaml:"icon_set_name"`  // optional - defaults to AppIcon.icns
}
```
Resources represents the resources needed to build an app bundle.

### Methods

```go
func (r Resources) IconSetFile() string
```


```go
func (r Resources) IconSetPath() string
```


```go
func (r Resources) IconSteps(twoX bool, sizes ...int) []Step
```


```go
func (r Resources) Signer() Signer
```




### Type RunResult
```go
type RunResult []StepResult
```
RunResult captures the outcome of running the steps.

### Methods

```go
func (r RunResult) Error() error
```
Error returns the last error encountered, if any.




### Type Signer
```go
type Signer struct {
	Identity         string
	EntitlementsFile string
	Arguments        []string
}
```

### Methods

```go
func (s Signer) SignPath(path string) Step
```


```go
func (s Signer) VerifyPath(path string) Step
```




### Type Step
```go
type Step interface {
	// Run executes the step.
	Run(context.Context, *CommandRunner) (StepResult, error)
}
```
Step represents a single operation that can be executed by the StepRunner.

### Functions

```go
func Copy(oldname, newname string) Step
```
Copy returns a Step that copies a file using cp.


```go
func CopyDir(srcDir, dstDir string) Step
```
CopyDir returns a Step that copies a directory recursively using cp -r.


```go
func DirExists(d string) Step
```
DirExists returns a Step that checks for the existence of the directory.


```go
func ErrorStep(err error, cmd string, args ...string) Step
```


```go
func FileExists(f string) Step
```
FileExists returns a Step that checks for the existence of the file.


```go
func IsValidIconSetDir(id IconSetDir) Step
```
IsValidIsValidIconSetDir returns a Step that checks if the directory has a
.iconset extension.


```go
func MkdirAll(d string) Step
```
MkdirAll returns a Step that creates a directory and all necessary parents
using mkdir -p.


```go
func NoopStep() Step
```


```go
func Rename(oldname, newname string) Step
```
Rename retrurns a Step that renames a file using mv.


```go
func StepFunc(f func(context.Context, *CommandRunner) (StepResult, error)) Step
```
StepFunc is a helper to create Steps from functions.




### Type StepResult
```go
type StepResult struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewStepResult(executable string, args []string, output []byte, err error) StepResult
```



### Methods

```go
func (le *StepResult) Args() []string
```


```go
func (le *StepResult) CommandLine() string
```


```go
func (le *StepResult) Error() error
```


```go
func (le *StepResult) Executable() string
```


```go
func (le *StepResult) Output() string
```


```go
func (le *StepResult) String() string
```




### Type StepRunner
```go
type StepRunner struct {
	// contains filtered or unexported fields
}
```
StepRunner manages and executes a series of Steps.

### Functions

```go
func NewRunner(opts ...StepRunnerOption) *StepRunner
```
NewRunner creates a new StepRunner with the provided options.



### Methods

```go
func (r *StepRunner) AddSteps(steps ...Step)
```
AddSteps adds one or more steps to the StepRunner.


```go
func (r *StepRunner) Run(ctx context.Context, cmdRunner *CommandRunner) RunResult
```
Run executes all added steps in sequence and returns a RunResult.




### Type StepRunnerOption
```go
type StepRunnerOption func(o *stepRunnerOptions)
```
StepRunnerOption configures a StepRunner.


### Type Suffix
```go
type Suffix string
```

### Methods

```go
func (s Suffix) Assert(path string) Step
```
Assert returns a Step that checks if the provided path has the specified
suffix.




### Type XPCInfoPlist
```go
type XPCInfoPlist struct {
	ServiceName      string   `plist:"ServiceName,omitempty" yaml:"service_name"`
	ServiceType      string   `plist:"ServiceType,omitempty" yaml:"service_type"`
	ProcessType      string   `plist:"ProcessType,omitempty" yaml:"process_type"`
	ProgramArguments []string `plist:"ProgramArguments,omitempty" yaml:"args"`
}
```
XPCInfoPlist represents the XPC service specific portion of the XPC service
info.plist.




## Examples
### [Example_createAppBundle](https://pkg.go.dev/cloudeng.io/macos/buildtools?tab=doc#example-_createAppBundle)
This example demonstrates how to create a basic macOS application bundle
structure with Info.plist and copy resources into it.




