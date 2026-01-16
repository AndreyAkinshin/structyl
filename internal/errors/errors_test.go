package errors

import (
	"errors"
	"testing"
)

func TestStructylError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *StructylError
		expected string
	}{
		{
			name:     "message only",
			err:      &StructylError{Message: "something failed"},
			expected: "something failed",
		},
		{
			name:     "with target",
			err:      &StructylError{Target: "rs", Message: "build failed"},
			expected: "[rs] build failed",
		},
		{
			name:     "with target and command",
			err:      &StructylError{Target: "rs", Command: "build", Message: "compilation error"},
			expected: "[rs] build: compilation error",
		},
		{
			name:     "command without target not included",
			err:      &StructylError{Command: "build", Message: "something failed"},
			expected: "something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStructylError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &StructylError{
		Message: "wrapper",
		Cause:   cause,
	}

	if got := err.Unwrap(); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}

	// Test nil cause
	errNoCause := &StructylError{Message: "no cause"}
	if got := errNoCause.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

func TestStructylError_ExitCode(t *testing.T) {
	tests := []struct {
		name     string
		kind     ErrorKind
		expected int
	}{
		{"runtime", KindRuntime, ExitRuntimeError},
		{"config", KindConfig, ExitConfigError},
		{"validation", KindValidation, ExitConfigError},
		{"not found", KindNotFound, ExitRuntimeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &StructylError{Kind: tt.kind}
			if got := err.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New("test error")

	if err.Kind != KindRuntime {
		t.Errorf("Kind = %v, want %v", err.Kind, KindRuntime)
	}
	if err.Message != "test error" {
		t.Errorf("Message = %q, want %q", err.Message, "test error")
	}
}

func TestNewf(t *testing.T) {
	err := Newf("error %d: %s", 42, "details")

	if err.Kind != KindRuntime {
		t.Errorf("Kind = %v, want %v", err.Kind, KindRuntime)
	}
	if err.Message != "error 42: details" {
		t.Errorf("Message = %q, want %q", err.Message, "error 42: details")
	}
}

func TestConfig(t *testing.T) {
	err := Config("invalid config")

	if err.Kind != KindConfig {
		t.Errorf("Kind = %v, want %v", err.Kind, KindConfig)
	}
	if err.Message != "invalid config" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid config")
	}
	if err.ExitCode() != ExitConfigError {
		t.Errorf("ExitCode() = %d, want %d", err.ExitCode(), ExitConfigError)
	}
}

func TestConfigf(t *testing.T) {
	err := Configf("field %q: %s", "name", "is required")

	if err.Kind != KindConfig {
		t.Errorf("Kind = %v, want %v", err.Kind, KindConfig)
	}
	expected := `field "name": is required`
	if err.Message != expected {
		t.Errorf("Message = %q, want %q", err.Message, expected)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, "wrapped message")

	if err.Kind != KindRuntime {
		t.Errorf("Kind = %v, want %v", err.Kind, KindRuntime)
	}
	if err.Message != "wrapped message" {
		t.Errorf("Message = %q, want %q", err.Message, "wrapped message")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return original cause")
	}
}

func TestTargetError(t *testing.T) {
	err := TargetError("rs", "build", "compilation failed")

	if err.Kind != KindRuntime {
		t.Errorf("Kind = %v, want %v", err.Kind, KindRuntime)
	}
	if err.Target != "rs" {
		t.Errorf("Target = %q, want %q", err.Target, "rs")
	}
	if err.Command != "build" {
		t.Errorf("Command = %q, want %q", err.Command, "build")
	}
	if err.Message != "compilation failed" {
		t.Errorf("Message = %q, want %q", err.Message, "compilation failed")
	}

	// Verify formatted error message
	expected := "[rs] build: compilation failed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("target", "nonexistent")

	if err.Kind != KindNotFound {
		t.Errorf("Kind = %v, want %v", err.Kind, KindNotFound)
	}
	expected := "target not found: nonexistent"
	if err.Message != expected {
		t.Errorf("Message = %q, want %q", err.Message, expected)
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, ExitSuccess},
		{"StructylError runtime", New("runtime"), ExitRuntimeError},
		{"StructylError config", Config("config"), ExitConfigError},
		{"StructylError validation", &StructylError{Kind: KindValidation}, ExitConfigError},
		{"generic error", errors.New("generic"), ExitRuntimeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetExitCode(tt.err); got != tt.expected {
				t.Errorf("GetExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestErrorKindConstants(t *testing.T) {
	// Verify error kinds have distinct values
	kinds := []ErrorKind{KindRuntime, KindConfig, KindNotFound, KindValidation}
	seen := make(map[ErrorKind]bool)

	for _, k := range kinds {
		if seen[k] {
			t.Errorf("Duplicate ErrorKind value: %v", k)
		}
		seen[k] = true
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify exit codes match specification
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitRuntimeError != 1 {
		t.Errorf("ExitRuntimeError = %d, want 1", ExitRuntimeError)
	}
	if ExitConfigError != 2 {
		t.Errorf("ExitConfigError = %d, want 2", ExitConfigError)
	}
}
