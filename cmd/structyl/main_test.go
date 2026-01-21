// Package main tests for the structyl CLI entry point.
package main

import (
	"os/exec"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/cli"
)

// TestMain_BuildVerification verifies the binary builds successfully.
// This is a smoke test to ensure the package compiles without errors.
func TestMain_BuildVerification(t *testing.T) {
	t.Parallel()

	// Attempt to build the package
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build main package: %v", err)
	}
}

// TestMain_HelpFlag verifies the --help flag works correctly.
func TestMain_HelpFlag(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "run", ".", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// --help should exit with code 0
		t.Fatalf("--help failed: %v\noutput: %s", err, out)
	}

	// Verify output contains expected text
	output := string(out)
	if len(output) == 0 {
		t.Error("--help produced empty output")
	}
}

// TestMain_VersionFlag verifies the --version flag works correctly.
func TestMain_VersionFlag(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "run", ".", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\noutput: %s", err, out)
	}

	output := string(out)
	if len(output) == 0 {
		t.Error("--version produced empty output")
	}
}

// TestCLI_RunDirect tests cli.Run directly to achieve coverage.
// This exercises the same code path as main() without calling os.Exit.
func TestCLI_RunDirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{"help", []string{"--help"}, 0},
		{"version", []string{"--version"}, 0},
		{"no_args", []string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exitCode := cli.Run(tt.args)
			if exitCode != tt.wantExit {
				t.Errorf("cli.Run(%v) = %d, want %d", tt.args, exitCode, tt.wantExit)
			}
		})
	}
}
