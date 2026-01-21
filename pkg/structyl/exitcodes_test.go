package structyl_test

import (
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/errors"
	"github.com/AndreyAkinshin/structyl/pkg/structyl"
)

// TestExitCodeValues verifies that exit code constants have the expected values
// as documented in docs/specs/error-handling.md.
func TestExitCodeValues(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"ExitSuccess", structyl.ExitSuccess, 0},
		{"ExitFailure", structyl.ExitFailure, 1},
		{"ExitConfigError", structyl.ExitConfigError, 2},
		{"ExitEnvError", structyl.ExitEnvError, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("structyl.%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestExitCodeConsistency verifies that public exit code constants match
// the internal errors package constants. This prevents drift between
// the public API and internal implementation.
func TestExitCodeConsistency(t *testing.T) {
	tests := []struct {
		name     string
		public   int
		internal int
	}{
		{"Success", structyl.ExitSuccess, errors.ExitSuccess},
		{"Failure/RuntimeError", structyl.ExitFailure, errors.ExitRuntimeError},
		{"ConfigError", structyl.ExitConfigError, errors.ExitConfigError},
		{"EnvError/EnvironmentError", structyl.ExitEnvError, errors.ExitEnvironmentError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.public != tt.internal {
				t.Errorf("exit code mismatch: structyl constant = %d, errors constant = %d",
					tt.public, tt.internal)
			}
		})
	}
}
