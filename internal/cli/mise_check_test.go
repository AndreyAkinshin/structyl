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

// TestCheckMise_ReturnsStatus verifies that CheckMise returns a valid status.
// Note: This test depends on the actual system state - if mise is installed,
// it will return installed=true; otherwise installed=false.
func TestCheckMise_ReturnsStatus(t *testing.T) {
	t.Parallel()
	status := CheckMise()
	// We can't assert on specific values since they depend on system state,
	// but we can verify the function returns without panic and the status
	// has consistent state (if installed, version/path should be populated).
	if status.Installed {
		// If mise reports as installed, Path should be non-empty
		if status.Path == "" {
			t.Error("CheckMise() returned Installed=true but Path is empty")
		}
	}
}

// TestEnsureMise_NonInteractive_ReturnsMiseNotInstalled verifies that
// EnsureMise returns the appropriate error in non-interactive mode when
// mise is not installed. This test skips if mise IS installed on the system.
func TestEnsureMise_NonInteractive_ReturnsMiseNotInstalled(t *testing.T) {
	t.Parallel()
	// First check if mise is installed
	status := CheckMise()
	if status.Installed {
		t.Skip("mise is installed, cannot test not-installed branch")
	}

	// In non-interactive mode, should return error immediately
	err := EnsureMise(false)
	if err == nil {
		t.Error("EnsureMise(false) should return error when mise not installed")
		return
	}

	expectedMsg := "mise is not installed. Install it from https://mise.jdx.dev"
	if err.Error() != expectedMsg {
		t.Errorf("EnsureMise(false) error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestEnsureMise_Interactive_WhenInstalled verifies EnsureMise returns nil
// when mise is already installed (regardless of interactive mode).
func TestEnsureMise_Interactive_WhenInstalled(t *testing.T) {
	t.Parallel()
	status := CheckMise()
	if !status.Installed {
		t.Skip("mise is not installed, cannot test installed branch")
	}

	// When mise is installed, should return nil regardless of interactive mode
	err := EnsureMise(false)
	if err != nil {
		t.Errorf("EnsureMise(false) with mise installed = %v, want nil", err)
	}

	err = EnsureMise(true)
	if err != nil {
		t.Errorf("EnsureMise(true) with mise installed = %v, want nil", err)
	}
}
