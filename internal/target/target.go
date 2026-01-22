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

// IsValid returns true if the target type is a known valid value.
func (t TargetType) IsValid() bool {
	return t == TypeLanguage || t == TypeAuxiliary
}

// ParseTargetType parses a string into a TargetType.
// Returns the parsed type and true if valid, or empty string and false if invalid.
func ParseTargetType(s string) (TargetType, bool) {
	switch s {
	case string(TypeLanguage):
		return TypeLanguage, true
	case string(TypeAuxiliary):
		return TypeAuxiliary, true
	default:
		return "", false
	}
}

// ValidTargetTypes returns the valid target type values as strings.
// Useful for error messages.
func ValidTargetTypes() []string {
	return []string{string(TypeLanguage), string(TypeAuxiliary)}
}

// Target represents a build target (language or auxiliary).
//
// # Thread Safety
//
// Implementations must be safe for concurrent read access after construction.
// All getter methods (Name, Title, Commands, GetCommand, etc.) may be called
// concurrently from multiple goroutines.
//
// The Execute method may be called concurrently for different commands on the
// same target. However, calling Execute for the same command concurrently on
// the same target has undefined behavior (the underlying shell commands may
// conflict on shared resources like files or ports).
//
// Modification of a Target after construction is undefined behavior. Targets
// are expected to be immutable once created by the registry.
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
	//
	// Return values:
	//   - (string, true): shell command to execute
	//   - ([]interface{}, true): list of sub-command names to execute in sequence
	//   - (nil, true): command is explicitly disabled; Execute returns *SkipError
	//   - (nil, false): command is not defined; Execute returns error
	//
	// The distinction between (nil, true) and (nil, false) is important:
	//   - (nil, true) means the command is deliberately disabled (JSON null value)
	//   - (nil, false) means the command doesn't exist in the target's command map
	GetCommand(name string) (interface{}, bool)
	// Env returns environment variables to set when executing commands.
	// Returns nil if no environment variables are configured.
	// Note: empty maps are normalized to nil (cannot distinguish "not set" from "set to empty").
	Env() map[string]string
	// Vars returns variables available for interpolation in command strings.
	// Returns nil if no variables are configured.
	// Note: empty maps are normalized to nil (cannot distinguish "not set" from "set to empty").
	Vars() map[string]string
	DemoPath() string // Path to demo file for documentation

	// Execution
	//
	// Execute runs the specified command. Returns nil on success.
	//
	// Error types (exhaustive):
	//   - *SkipError: command was skipped (not failed). Use IsSkipError() to detect.
	//     Reasons: SkipReasonDisabled (command is nil), SkipReasonCommandNotFound
	//     (executable not in PATH), SkipReasonScriptNotFound (npm/pnpm/yarn/bun script missing).
	//     Runner layer logs these as warnings and continues execution.
	//   - *exec.ExitError: command executed but exited with non-zero status.
	//     The exit code is available via err.(*exec.ExitError).ExitCode().
	//   - context.Canceled: context was canceled before or during execution.
	//   - context.DeadlineExceeded: context deadline was exceeded.
	//   - fmt.Errorf (plain error): command definition error, e.g., command not defined
	//     for target, invalid command list item type, invalid command definition type.
	//
	// For composite commands ([]string), sub-commands execute sequentially.
	// If any sub-command fails, execution stops and the error is returned.
	//
	// Context cancellation:
	//   - Uses exec.CommandContext which sends SIGKILL (Unix) or TerminateProcess (Windows)
	//     when the context is canceled or times out.
	//   - Child processes are terminated immediately; no graceful shutdown period.
	//   - Partial stdout/stderr output may be available depending on buffering.
	Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

// ExecOptions contains options for command execution.
type ExecOptions struct {
	Args      []string          // Additional arguments
	Env       map[string]string // Additional environment variables
	Verbosity Verbosity         // Output verbosity level
}
