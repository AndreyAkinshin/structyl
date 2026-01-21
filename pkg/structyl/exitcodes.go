// Package structyl provides public constants and utilities for external tools
// integrating with Structyl.
package structyl

// Exit codes returned by the structyl CLI.
// These constants allow external tools to check exit codes symbolically
// rather than using magic numbers.
const (
	// ExitSuccess indicates the command completed successfully.
	ExitSuccess = 0

	// ExitFailure indicates a runtime failure (build failed, test failed, etc.).
	ExitFailure = 1

	// ExitConfigError indicates a configuration error (invalid config, validation failure, etc.).
	ExitConfigError = 2

	// ExitEnvError indicates an environment error (Docker unavailable, missing dependency, etc.).
	ExitEnvError = 3
)
