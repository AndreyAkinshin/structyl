//go:build windows

package target

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWindowsShellCommand_UsesSystemRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := buildWindowsShellCommand(ctx, "echo hello")

	// Verify it uses PowerShell
	if !strings.Contains(cmd.Path, "powershell.exe") {
		t.Errorf("cmd.Path = %q, want to contain powershell.exe", cmd.Path)
	}

	// Verify args include expected flags
	args := cmd.Args
	hasNoProfile := false
	hasNonInteractive := false
	hasCommand := false
	for i, arg := range args {
		if arg == "-NoProfile" {
			hasNoProfile = true
		}
		if arg == "-NonInteractive" {
			hasNonInteractive = true
		}
		if arg == "-Command" && i+1 < len(args) && args[i+1] == "echo hello" {
			hasCommand = true
		}
	}

	if !hasNoProfile {
		t.Error("command should include -NoProfile flag")
	}
	if !hasNonInteractive {
		t.Error("command should include -NonInteractive flag")
	}
	if !hasCommand {
		t.Error("command should include -Command with the cmd string")
	}
}

func TestBuildWindowsShellCommand_FallbackSystemRoot(t *testing.T) {
	// Save and clear SYSTEMROOT
	original := os.Getenv("SYSTEMROOT")
	defer os.Setenv("SYSTEMROOT", original)
	os.Setenv("SYSTEMROOT", "")

	ctx := context.Background()
	cmd := buildWindowsShellCommand(ctx, "echo test")

	// Should fall back to C:\Windows
	expected := filepath.Join(`C:\Windows`, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if cmd.Path != expected {
		t.Errorf("cmd.Path = %q, want %q (fallback when SYSTEMROOT empty)", cmd.Path, expected)
	}
}

func TestBuildWindowsShellCommand_CustomSystemRoot(t *testing.T) {
	// Save and set custom SYSTEMROOT
	original := os.Getenv("SYSTEMROOT")
	defer os.Setenv("SYSTEMROOT", original)
	os.Setenv("SYSTEMROOT", `D:\CustomWindows`)

	ctx := context.Background()
	cmd := buildWindowsShellCommand(ctx, "echo test")

	expected := filepath.Join(`D:\CustomWindows`, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if cmd.Path != expected {
		t.Errorf("cmd.Path = %q, want %q", cmd.Path, expected)
	}
}

func TestBuildShellCommand_OnWindows_DelegatesToWindowsFunc(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := buildShellCommand(ctx, "echo hello")

	// On Windows, should use PowerShell
	if !strings.Contains(cmd.Path, "powershell.exe") {
		t.Errorf("cmd.Path = %q, want PowerShell on Windows", cmd.Path)
	}
}
