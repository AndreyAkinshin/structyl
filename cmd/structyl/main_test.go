// Package main tests for the structyl CLI entry point.
//
// Note on coverage: The cmd/structyl package contains only main(), which calls
// cli.Run() and then os.Exit(). Since main() cannot be tested directly without
// process termination, we use two approaches:
//
//  1. Direct cli.Run() calls via TestCliRun_* — provides import path coverage
//  2. Subprocess tests via exec.Command — verifies end-to-end binary behavior
//
// The actual CLI logic resides in internal/cli with comprehensive test coverage.
// This package is a thin entry point that delegates to cli.Run().
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

// TestCliRun_Help verifies cli.Run returns success for help flag.
// This provides coverage by testing the same code path as main().
func TestCliRun_Help(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"--help"})
	if exitCode != 0 {
		t.Errorf("cli.Run([--help]) = %d, want 0", exitCode)
	}
}

// TestCliRun_Version verifies cli.Run returns success for version flag.
func TestCliRun_Version(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"--version"})
	if exitCode != 0 {
		t.Errorf("cli.Run([--version]) = %d, want 0", exitCode)
	}
}

// TestCliRun_NoArgs verifies cli.Run handles no arguments gracefully.
func TestCliRun_NoArgs(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{})
	if exitCode != 0 {
		t.Errorf("cli.Run([]) = %d, want 0", exitCode)
	}
}
