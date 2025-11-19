// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package buildtools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// StepRunnerOption configures a StepRunner.
type StepRunnerOption func(o *stepRunnerOptions)

func WithStepTiming(timing bool) StepRunnerOption {
	return func(o *stepRunnerOptions) {
		o.timing = timing
	}
}

type stepRunnerOptions struct {
	timing bool
}

// StepRunner manages and executes a series of Steps.
type StepRunner struct {
	options stepRunnerOptions
	steps   []Step
}

// NewRunner creates a new StepRunner with the provided options.
func NewRunner(opts ...StepRunnerOption) *StepRunner {
	var options stepRunnerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &StepRunner{options: options}
}

// Step represents a single operation that can be executed by the StepRunner.
type Step interface {
	// Run executes the step.
	Run(context.Context, *CommandRunner) (StepResult, error)
}

// AddSteps adds one or more steps to the StepRunner.
func (r *StepRunner) AddSteps(steps ...Step) *StepRunner {
	r.steps = append(r.steps, steps...)
	return r
}

type StepResult struct {
	executable string
	args       []string
	output     []byte
	err        error
	duration   time.Duration
}

func NewStepResult(executable string, args []string, output []byte, err error) StepResult {
	return StepResult{
		executable: executable,
		args:       args,
		output:     output,
		err:        err,
	}
}

func (le *StepResult) Executable() string {
	return le.executable
}

func (le *StepResult) Args() []string {
	return le.args
}

func (le *StepResult) CommandLine() string {
	return formatCmdLine(le.executable, le.args)
}

func (le *StepResult) Output() string {
	return string(le.output)
}

func (le *StepResult) String() string {
	return formatResult(le.executable, le.args, le.output, le.err)
}

func (le *StepResult) Error() error {
	return le.err
}

func (le *StepResult) Duration() time.Duration {
	return le.duration
}

// RunResult captures the outcome of running the steps.
type RunResult []StepResult

// Error returns the last error encountered, if any.
func (r RunResult) Error() error {
	if len(r) == 0 {
		return nil
	}
	return r[len(r)-1].Error()
}

// Run executes all added steps in sequence and returns a RunResult.
func (r *StepRunner) Run(ctx context.Context, cmdRunner *CommandRunner) RunResult {
	var log RunResult
	start := time.Now()
	for i, step := range r.steps {
		result, err := step.Run(ctx, cmdRunner)
		log = append(log, result)
		if err != nil {
			break
		}
		if r.options.timing {
			fmt.Fprintf(os.Stderr, "  step: %d: %v: %v\n", i, result.Duration(), result.CommandLine())
		}
	}
	if r.options.timing {
		fmt.Fprintf(os.Stderr, "total: %v\n", time.Since(start))
	}
	return log
}

// CommandRunnerOption configures a CommandRunner.
type CommandRunnerOption func(o *commandRunnerOptions)

type commandRunnerOptions struct {
	dryRun bool
	timing bool
	stdout io.Writer
	stderr io.Writer
}

// WithDryRun configures the CommandRunner to simulate command execution without actually running commands.
func WithDryRun(dryRun bool) CommandRunnerOption {
	return func(o *commandRunnerOptions) {
		o.dryRun = dryRun
	}
}

func WithCommandTiming(timing bool) CommandRunnerOption {
	return func(o *commandRunnerOptions) {
		o.timing = timing
	}
}

// CommandRunner executes system commands.
type CommandRunner struct {
	options commandRunnerOptions
}

// NewCommandRunner creates a new CommandRunner with the provided options.
func NewCommandRunner(opts ...CommandRunnerOption) *CommandRunner {
	var options commandRunnerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &CommandRunner{options: options}
}

// WithStdout configures the CommandRunner to write standard output to the provided io.Writer.
func WithStdout(w io.Writer) CommandRunnerOption {
	return func(o *commandRunnerOptions) {
		o.stdout = w
	}
}

// WithStderr configures the CommandRunner to write standard error to the provided io.Writer.
func WithStderr(w io.Writer) CommandRunnerOption {
	return func(o *commandRunnerOptions) {
		o.stderr = w
	}
}

func (r *CommandRunner) DryRun() bool {
	return r.options.dryRun
}

// formatCmdLine formats a command and its arguments into a single string.
func formatCmdLine(name string, args []string) string {
	var out strings.Builder
	out.WriteString(name)
	out.WriteRune(' ')
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			arg = fmt.Sprintf("%q", arg)
		}
		out.WriteString(arg)
		out.WriteRune(' ')
	}
	return out.String()
}

func formatResult(name string, args []string, input []byte, err error) string {
	scanner := bufio.NewScanner(bytes.NewReader(input))
	var out strings.Builder
	out.WriteString(formatCmdLine(name, args))
	if err != nil {
		out.WriteString(" : ")
		out.WriteString(err.Error())
	}
	out.WriteRune('\n')
	for scanner.Scan() {
		out.WriteString("  ")
		out.WriteString(scanner.Text())
		out.WriteRune('\n')
	}
	return out.String()
}

// Run executes the specified command with arguments and returns the combined output and any error encountered.
func (r *CommandRunner) Run(ctx context.Context, name string, args ...string) (StepResult, error) {
	if r.options.dryRun {
		return StepResult{executable: name, args: args}, nil
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = CWDFromContext(ctx)
	if r.options.stdout != nil {
		cmd.Stdout = r.options.stdout
	}
	if r.options.stderr != nil {
		cmd.Stderr = r.options.stderr
	}
	output, err := cmd.CombinedOutput()
	return StepResult{executable: name, args: args, output: output, duration: time.Since(start), err: err}, err
}

func (r *CommandRunner) WriteFile(ctx context.Context, path string, data []byte, perm uint32) (string, error) {
	if r.options.dryRun {
		return fmt.Sprintf("write %d bytes to %q with perm %o", len(data), path, perm), nil
	}
	cmd := exec.CommandContext(ctx, "tee", path)
	if r.options.stdout != nil {
		cmd.Stdout = r.options.stdout
	}
	if r.options.stderr != nil {
		cmd.Stderr = r.options.stderr
	}
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if err := exec.CommandContext(ctx, "chmod", fmt.Sprintf("%o", perm), path).Run(); err != nil { //nolint:gosec // G204
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %q with perm %o", len(data), path, perm), nil
}

// StepFunc is a helper to create Steps from functions.
func StepFunc(f func(context.Context, *CommandRunner) (StepResult, error)) Step {
	return stepFunc(f)
}

type stepFunc func(context.Context, *CommandRunner) (StepResult, error)

func (s stepFunc) Run(ctx context.Context, cmdRunner *CommandRunner) (StepResult, error) {
	return s(ctx, cmdRunner)
}

func NoopStep(detail string) Step {
	return StepFunc(func(_ context.Context, _ *CommandRunner) (StepResult, error) {
		return StepResult{executable: "noop: " + detail}, nil
	})
}

func ErrorStep(err error, cmd string, args ...string) Step {
	return StepFunc(func(_ context.Context, _ *CommandRunner) (StepResult, error) {
		return StepResult{executable: cmd, args: args, err: err}, err
	})
}

type cwdKey struct{}

// ContextWithCWD returns a new context with the specified current
// working directory. CommandRunner will use this directory for
// executing commands.
func ContextWithCWD(ctx context.Context, cwd string) context.Context {
	return context.WithValue(ctx, cwdKey{}, cwd)
}

var processCWD string

func init() {
	var err error
	processCWD, err = os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}
}

// CWDFromContext retrieves the current working directory from the context,
// as set by ContextWithCWD. If no directory has been set in the context
// the function returns the process's current working directory at the time
// that this package was initialized. Note that this may differ from the actual
// current working directory of the process if it has changed since the package
// was initialized or if another package that was initialized earlier has
// changed the current working directory of the process.
func CWDFromContext(ctx context.Context) string {
	cwd, ok := ctx.Value(cwdKey{}).(string)
	if !ok {
		return processCWD
	}
	return cwd
}
