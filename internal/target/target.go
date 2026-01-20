// Package target provides the Target interface and registry for build targets.
package target

import "context"

// Verbosity represents the output verbosity level.
type Verbosity int

const (
	// VerbosityDefault is the normal output level.
	VerbosityDefault Verbosity = iota
	// VerbosityQuiet suppresses info messages (errors only).
	VerbosityQuiet
	// VerbosityVerbose shows maximum detail.
	VerbosityVerbose
)

// TargetType represents the type of build target.
type TargetType string

const (
	// TypeLanguage represents a language implementation target.
	TypeLanguage TargetType = "language"
	// TypeAuxiliary represents an auxiliary build target.
	TypeAuxiliary TargetType = "auxiliary"
)

// Target represents a build target (language or auxiliary).
type Target interface {
	// Identification
	Name() string      // Short name (e.g., "cs", "py")
	Title() string     // Display name (e.g., "C#", "Python")
	Type() TargetType  // "language" or "auxiliary"
	Directory() string // Target directory path (relative to root)
	Cwd() string       // Working directory for commands

	// Capabilities
	Commands() []string  // Available commands (including variants like "build:release")
	DependsOn() []string // Dependency targets

	// Configuration
	//
	// GetCommand returns the command definition for the given name.
	// The returned interface{} can be:
	//   - string: a shell command to execute
	//   - []string or []interface{}: a list of sub-command names to execute in sequence
	//   - nil: command is explicitly disabled (Execute returns SkipError)
	// Returns (nil, false) if the command is not defined.
	GetCommand(name string) (interface{}, bool)
	Env() map[string]string  // Environment variables
	Vars() map[string]string // Variables for interpolation
	DemoPath() string        // Path to demo file for documentation

	// Execution
	//
	// Execute runs the specified command. Returns nil on success.
	//
	// Error types:
	//   - *SkipError: command was skipped (disabled, missing executable, missing npm script).
	//     Use IsSkipError() to detect. Runner layer decides whether to continue.
	//   - context.Canceled/context.DeadlineExceeded: context was canceled or timed out.
	//   - Other errors: command execution failed (non-zero exit, missing command definition).
	//
	// For composite commands ([]string), sub-commands execute sequentially.
	// If any sub-command fails, execution stops and the error is returned.
	//
	// Context cancellation:
	//   - Uses exec.CommandContext which sends SIGKILL (Unix) or TerminateProcess (Windows)
	//     when the context is canceled or times out.
	//   - Child processes are terminated immediately; no graceful shutdown period.
	//   - Partial stdout/stderr output may be available depending on buffering.
	//   - Returns context.Canceled or context.DeadlineExceeded as appropriate.
	Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

// ExecOptions contains options for command execution.
type ExecOptions struct {
	Docker    bool              // Run in Docker container
	Args      []string          // Additional arguments
	Env       map[string]string // Additional environment variables
	Verbosity Verbosity         // Output verbosity level
}
