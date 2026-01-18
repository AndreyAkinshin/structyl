// Package errors provides structured error types and exit codes for Structyl.
package errors

import (
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
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if se, ok := err.(*StructylError); ok {
		return se.ExitCode()
	}
	return ExitRuntimeError
}
