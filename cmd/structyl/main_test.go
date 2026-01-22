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

// TestCliRun_UnknownCommand verifies cli.Run returns non-zero for unknown commands.
// When run outside a project, unknown commands return exit code 1 (failure).
func TestCliRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"this-command-does-not-exist-xyz123"})
	if exitCode == 0 {
		t.Error("cli.Run([unknown]) = 0, want non-zero for unknown command")
	}
}

// TestCliRun_InvalidGlobalFlag verifies cli.Run returns exit code 2 for invalid flags.
func TestCliRun_InvalidGlobalFlag(t *testing.T) {
	t.Parallel()

	// --type with missing value should return exit code 2 (config error)
	exitCode := cli.Run([]string{"--type"})
	if exitCode != 2 {
		t.Errorf("cli.Run([--type]) = %d, want 2 (config error)", exitCode)
	}
}

// TestCliRun_MutuallyExclusiveFlags verifies cli.Run rejects conflicting flags.
func TestCliRun_MutuallyExclusiveFlags(t *testing.T) {
	t.Parallel()

	// --docker and --no-docker are mutually exclusive
	exitCode := cli.Run([]string{"--docker", "--no-docker", "build"})
	if exitCode != 2 {
		t.Errorf("cli.Run([--docker, --no-docker]) = %d, want 2", exitCode)
	}

	// --quiet and --verbose are mutually exclusive
	exitCode = cli.Run([]string{"--quiet", "--verbose", "build"})
	if exitCode != 2 {
		t.Errorf("cli.Run([--quiet, --verbose]) = %d, want 2", exitCode)
	}
}

// TestCliRun_ShortHelpFlag verifies -h works like --help.
func TestCliRun_ShortHelpFlag(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"-h"})
	if exitCode != 0 {
		t.Errorf("cli.Run([-h]) = %d, want 0", exitCode)
	}
}

// TestCliRun_HelpCommand verifies "help" command works.
func TestCliRun_HelpCommand(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"help"})
	if exitCode != 0 {
		t.Errorf("cli.Run([help]) = %d, want 0", exitCode)
	}
}

// TestCliRun_VersionCommand verifies "version" command works.
func TestCliRun_VersionCommand(t *testing.T) {
	t.Parallel()

	exitCode := cli.Run([]string{"version"})
	if exitCode != 0 {
		t.Errorf("cli.Run([version]) = %d, want 0", exitCode)
	}
}
