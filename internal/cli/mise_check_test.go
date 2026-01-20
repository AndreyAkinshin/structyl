package cli

import "testing"

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
