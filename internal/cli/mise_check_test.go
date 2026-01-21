package cli

import (
	"errors"
	"testing"
)

func TestMiseStatus_Default(t *testing.T) {
	status := MiseStatus{}

	if status.Installed {
		t.Error("default MiseStatus.Installed should be false")
	}
	if status.Version != "" {
		t.Errorf("default MiseStatus.Version = %q, want empty", status.Version)
	}
	if status.Path != "" {
		t.Errorf("default MiseStatus.Path = %q, want empty", status.Path)
	}
}

func TestMiseStatus_Installed(t *testing.T) {
	status := MiseStatus{
		Installed: true,
		Version:   "2024.1.0",
		Path:      "/usr/local/bin/mise",
	}

	if !status.Installed {
		t.Error("MiseStatus.Installed should be true")
	}
	if status.Version != "2024.1.0" {
		t.Errorf("MiseStatus.Version = %q, want %q", status.Version, "2024.1.0")
	}
	if status.Path != "/usr/local/bin/mise" {
		t.Errorf("MiseStatus.Path = %q, want %q", status.Path, "/usr/local/bin/mise")
	}
}

// Note: CheckMise, EnsureMise, and InstallMise require external commands
// (exec.LookPath, curl, etc.) and are tested via integration tests.
// Unit tests would require mocking the exec package which adds complexity
// for little benefit since the actual behavior depends on the system state.

func TestErrMiseNotInstalled(t *testing.T) {
	t.Parallel()
	err := errMiseNotInstalled()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if msg != "mise is not installed. Install it from https://mise.jdx.dev" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestErrMiseRequired(t *testing.T) {
	t.Parallel()
	err := errMiseRequired()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if msg != "mise is required. Install it from https://mise.jdx.dev" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestErrInstallMise(t *testing.T) {
	t.Parallel()
	cause := errors.New("network error")
	err := errInstallMise(cause)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if msg != "failed to install mise: network error" {
		t.Errorf("unexpected error message: %q", msg)
	}
	// Verify error wrapping
	if !errors.Is(err, cause) {
		t.Error("expected error to wrap the cause")
	}
}

func TestErrMiseNotInPath(t *testing.T) {
	t.Parallel()
	err := errMiseNotInPath()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if msg != "mise installed but not in PATH" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestPrintMiseInstallInstructions(t *testing.T) {
	t.Parallel()
	// Verify function executes without panic.
	// The function writes to stdout which we don't capture here,
	// but the primary goal is ensuring no runtime errors.
	PrintMiseInstallInstructions()
}
