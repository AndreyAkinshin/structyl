// Package errors provides structured error types and exit codes for Structyl.
//
// # Error Types
//
// Structyl uses two distinct error types for different purposes:
//
//   - StructylError: Runtime, configuration, validation, and environment errors
//     that represent actual failures. Use GetExitCode() to determine the
//     appropriate exit code. StructylError implements Unwrap() for error chain
//     inspection with errors.Is() and errors.As().
//
//   - target.SkipError: Indicates a command was skipped (not failed). Skip
//     scenarios include disabled commands (nil in config), missing executables,
//     and missing npm scripts. Use target.IsSkipError() to detect. Skip errors
//     are informational and are logged as warnings rather than causing command
//     failure. They are NOT included in combined error results.
//
// The Runner layer handles both types: StructylError causes immediate failure
// (fail-fast), while SkipError is logged and execution continues to the next
// target.
package errors

import (
	"errors"
	"fmt"
)

// Exit codes as defined in the specification.
const (
	ExitSuccess          = 0 // Success
	ExitRuntimeError     = 1 // Runtime error (command failed, etc.)
	ExitConfigError      = 2 // Configuration error (invalid config, etc.)
	ExitEnvironmentError = 3 // Environment error (Docker not available, missing dependency, etc.)
)

// ErrorKind represents the type of error.
type ErrorKind int

const (
	KindRuntime ErrorKind = iota
	KindConfig
	KindNotFound
	KindValidation
	KindEnvironment
)

// StructylError is the base error type for Structyl.
type StructylError struct {
	Kind    ErrorKind
	Message string
	Target  string // Target name if applicable
	Command string // Command name if applicable
	Cause   error  // Underlying error
}

func (e *StructylError) Error() string {
	if e.Target != "" && e.Command != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Target, e.Command, e.Message)
	}
	if e.Target != "" {
		return fmt.Sprintf("[%s] %s", e.Target, e.Message)
	}
	return e.Message
}

func (e *StructylError) Unwrap() error {
	return e.Cause
}

// ExitCode returns the appropriate exit code for this error.
func (e *StructylError) ExitCode() int {
	switch e.Kind {
	case KindConfig, KindValidation:
		return ExitConfigError
	case KindEnvironment:
		return ExitEnvironmentError
	default:
		return ExitRuntimeError
	}
}

// New creates a new runtime error.
func New(message string) *StructylError {
	return &StructylError{
		Kind:    KindRuntime,
		Message: message,
	}
}

// Newf creates a new runtime error with formatting.
func Newf(format string, args ...interface{}) *StructylError {
	return New(fmt.Sprintf(format, args...))
}

// Config creates a new configuration error.
func Config(message string) *StructylError {
	return &StructylError{
		Kind:    KindConfig,
		Message: message,
	}
}

// Configf creates a new configuration error with formatting.
func Configf(format string, args ...interface{}) *StructylError {
	return Config(fmt.Sprintf(format, args...))
}

// Validation creates a new validation error.
// Validation errors are distinct from configuration errors: configuration
// errors indicate the configuration file is malformed or has invalid syntax,
// while validation errors indicate the configuration is syntactically valid
// but contains semantic errors (e.g., invalid version format, pattern not found).
func Validation(message string) *StructylError {
	return &StructylError{
		Kind:    KindValidation,
		Message: message,
	}
}

// Validationf creates a new validation error with formatting.
func Validationf(format string, args ...interface{}) *StructylError {
	return Validation(fmt.Sprintf(format, args...))
}

// Environment creates a new environment error.
func Environment(message string) *StructylError {
	return &StructylError{
		Kind:    KindEnvironment,
		Message: message,
	}
}

// Environmentf creates a new environment error with formatting.
func Environmentf(format string, args ...interface{}) *StructylError {
	return Environment(fmt.Sprintf(format, args...))
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) *StructylError {
	return &StructylError{
		Kind:    KindRuntime,
		Message: message,
		Cause:   err,
	}
}

// Wrapf wraps an error with additional context using a format string.
func Wrapf(err error, format string, args ...interface{}) *StructylError {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// TargetError creates an error for a specific target.
func TargetError(target, command, message string) *StructylError {
	return &StructylError{
		Kind:    KindRuntime,
		Target:  target,
		Command: command,
		Message: message,
	}
}

// NotFound creates a not found error.
func NotFound(what, name string) *StructylError {
	return &StructylError{
		Kind:    KindNotFound,
		Message: fmt.Sprintf("%s not found: %s", what, name),
	}
}

// GetExitCode returns the exit code for an error.
// It uses errors.As to unwrap error chains, allowing it to find StructylError
// even when wrapped by fmt.Errorf or other wrappers.
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var se *StructylError
	if errors.As(err, &se) {
		return se.ExitCode()
	}
	return ExitRuntimeError
}
