// Package target provides the Target interface and registry for build targets.
package target

import "context"

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
	Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

// ExecOptions contains options for command execution.
type ExecOptions struct {
	Docker bool              // Run in Docker container
	Args   []string          // Additional arguments
	Env    map[string]string // Additional environment variables
}

// ExecResult contains the result of command execution.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}
