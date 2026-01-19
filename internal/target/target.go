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
	GetCommand(name string) (interface{}, bool) // Get command definition
	Env() map[string]string                     // Environment variables
	Vars() map[string]string                    // Variables for interpolation
	DemoPath() string                           // Path to demo file for documentation

	// Execution
	//
	// Execute runs the specified command. Returns nil on success.
	// Returns a SkipError if the command was skipped (disabled, missing executable,
	// or missing npm script). Callers should use IsSkipError() to distinguish
	// skip errors from execution failures and handle them appropriately.
	Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

// ExecOptions contains options for command execution.
type ExecOptions struct {
	Docker    bool              // Run in Docker container
	Args      []string          // Additional arguments
	Env       map[string]string // Additional environment variables
	Verbosity Verbosity         // Output verbosity level
}
