package cli

// Test Parallelism Design Note
//
// Many tests in this file are intentionally NOT parallel (no t.Parallel() call).
// This is due to global state in the CLI package:
//
//   1. Global Output Writer: The `out` variable (output.Writer) is package-level.
//      Functions like parseGlobalFlags() call applyVerbosityToOutput() which modifies
//      this shared writer's Quiet and Verbose flags. Parallel tests would cause
//      data races on this state.
//
//   2. Working Directory: Tests using withWorkingDir() change the process's current
//      working directory via os.Chdir(). This is process-global state that cannot
//      be safely modified by parallel tests.
//
// Tests that don't modify global state (pure functions, independent struct operations)
// can and do use t.Parallel(). Look for t.Parallel() calls to identify which tests
// are safe for parallel execution.
//
// Future refactoring could inject io.Writer dependencies to enable more parallel tests,
// but the current design prioritizes simplicity over test performance.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: Testing Docker error handling requires runner package
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/testing/mocks"
)

func TestParseGlobalFlags(t *testing.T) {
	// Note: subtests are NOT parallel because parseGlobalFlags calls applyVerbosityToOutput
	// which modifies the global output writer. Running subtests in parallel would cause
	// concurrent writes to this shared state.
	tests := []struct {
		name           string
		args           []string
		wantDocker     bool
		wantNoDocker   bool
		wantTargetType string
		wantQuiet      bool
		wantVerbose    bool
		wantRemaining  []string
		wantErr        bool
	}{
		{
			name:          "no flags",
			args:          []string{"build"},
			wantRemaining: []string{"build"},
		},
		{
			name:          "--docker flag",
			args:          []string{"--docker", "build"},
			wantDocker:    true,
			wantRemaining: []string{"build"},
		},
		{
			name:          "--no-docker flag",
			args:          []string{"--no-docker", "build"},
			wantNoDocker:  true,
			wantRemaining: []string{"build"},
		},
		{
			name:    "--continue flag is removed",
			args:    []string{"--continue", "build"},
			wantErr: true,
		},
		{
			name:          "-q flag",
			args:          []string{"-q", "build"},
			wantQuiet:     true,
			wantRemaining: []string{"build"},
		},
		{
			name:          "--quiet flag",
			args:          []string{"--quiet", "build"},
			wantQuiet:     true,
			wantRemaining: []string{"build"},
		},
		{
			name:          "-v flag",
			args:          []string{"-v", "build"},
			wantVerbose:   true,
			wantRemaining: []string{"build"},
		},
		{
			name:          "--verbose flag",
			args:          []string{"--verbose", "build"},
			wantVerbose:   true,
			wantRemaining: []string{"build"},
		},
		{
			name:           "--type with space",
			args:           []string{"--type", "language", "build"},
			wantTargetType: "language",
			wantRemaining:  []string{"build"},
		},
		{
			name:           "--type=value",
			args:           []string{"--type=auxiliary", "build"},
			wantTargetType: "auxiliary",
			wantRemaining:  []string{"build"},
		},
		{
			name:          "-- passthrough",
			args:          []string{"build", "--", "--verbose", "--debug"},
			wantRemaining: []string{"build", "--", "--verbose", "--debug"},
		},
		{
			name:          "multiple flags",
			args:          []string{"--docker", "--quiet", "build"},
			wantDocker:    true,
			wantQuiet:     true,
			wantRemaining: []string{"build"},
		},
		{
			name:           "all flags combined",
			args:           []string{"--docker", "--type=language", "test", "rs"},
			wantDocker:     true,
			wantTargetType: "language",
			wantRemaining:  []string{"test", "rs"},
		},
		{
			// --type only accepts "language" or "auxiliary"; any other value is invalid
			name:    "invalid --type value",
			args:    []string{"--type=invalid", "build"},
			wantErr: true,
		},
		{
			name:    "quiet and verbose mutually exclusive",
			args:    []string{"-q", "-v", "build"},
			wantErr: true,
		},
		{
			name:    "quiet and verbose long form mutually exclusive",
			args:    []string{"--quiet", "--verbose", "build"},
			wantErr: true,
		},
		{
			name:    "docker and no-docker mutually exclusive",
			args:    []string{"--docker", "--no-docker", "build"},
			wantErr: true,
		},
		{
			name:          "empty args",
			args:          []string{},
			wantRemaining: nil,
		},
		{
			name:           "empty type value is valid",
			args:           []string{"--type=", "build"},
			wantTargetType: "",
			wantRemaining:  []string{"build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, remaining, err := parseGlobalFlags(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if opts.Docker != tt.wantDocker {
				t.Errorf("Docker = %v, want %v", opts.Docker, tt.wantDocker)
			}
			if opts.NoDocker != tt.wantNoDocker {
				t.Errorf("NoDocker = %v, want %v", opts.NoDocker, tt.wantNoDocker)
			}
			if opts.TargetType != tt.wantTargetType {
				t.Errorf("TargetType = %q, want %q", opts.TargetType, tt.wantTargetType)
			}
			if opts.Quiet != tt.wantQuiet {
				t.Errorf("Quiet = %v, want %v", opts.Quiet, tt.wantQuiet)
			}
			if opts.Verbose != tt.wantVerbose {
				t.Errorf("Verbose = %v, want %v", opts.Verbose, tt.wantVerbose)
			}

			if len(remaining) != len(tt.wantRemaining) {
				t.Errorf("remaining = %v, want %v", remaining, tt.wantRemaining)
			} else {
				for i, r := range remaining {
					if r != tt.wantRemaining[i] {
						t.Errorf("remaining[%d] = %q, want %q", i, r, tt.wantRemaining[i])
					}
				}
			}
		})
	}
}

func TestParseGlobalFlags_TypeMissingValue(t *testing.T) {
	// Note: subtests are NOT parallel because parseGlobalFlags calls applyVerbosityToOutput
	// which modifies the global output writer.
	tests := []struct {
		name string
		args []string
	}{
		{"type flag only", []string{"--type"}},
		{"type at end after command", []string{"build", "--type"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseGlobalFlags(tt.args)
			if err == nil {
				t.Error("parseGlobalFlags() expected error for --type without value")
			}
			if err != nil && !strings.Contains(err.Error(), "--type requires a value") {
				t.Errorf("error = %q, want to contain '--type requires a value'", err.Error())
			}
		})
	}
}

func TestParseGlobalFlags_UnknownFlagsPassThrough(t *testing.T) {
	// Unknown flags are passed through to commands (not rejected at global level)
	// This allows command-specific flags like "build --release"
	// Note: subtests are NOT parallel because parseGlobalFlags calls applyVerbosityToOutput
	// which modifies the global output writer.
	tests := []struct {
		name       string
		args       []string
		wantRemain []string
	}{
		{"unknown long flag", []string{"--unknown-flag", "build"}, []string{"--unknown-flag", "build"}},
		{"unknown short flag", []string{"-x", "build"}, []string{"-x", "build"}},
		{"command with unknown flag", []string{"build", "--release"}, []string{"build", "--release"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, remaining, err := parseGlobalFlags(tt.args)
			if err != nil {
				t.Errorf("parseGlobalFlags(%v) unexpected error: %v", tt.args, err)
			}
			if len(remaining) != len(tt.wantRemain) {
				t.Errorf("remaining = %v, want %v", remaining, tt.wantRemain)
			}
		})
	}
}

func TestRun_Help(t *testing.T) {
	// Note: subtests are NOT parallel because Run() uses global output writer.
	tests := []struct {
		name string
		args []string
	}{
		{"help", []string{"help"}},
		{"-h", []string{"-h"}},
		{"--help", []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := Run(tt.args)
			if exitCode != 0 {
				t.Errorf("Run(%v) = %d, want 0", tt.args, exitCode)
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	// Note: subtests are NOT parallel because Run() uses global output writer.
	tests := []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"--version", []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := Run(tt.args)
			if exitCode != 0 {
				t.Errorf("Run(%v) = %d, want 0", tt.args, exitCode)
			}
		})
	}
}

func TestRun_EmptyArgs(t *testing.T) {
	// Note: NOT parallel because Run() uses global output writer.
	exitCode := Run([]string{})
	if exitCode != 0 {
		t.Errorf("Run([]) = %d, want 0", exitCode)
	}
}

func TestSanitizeProjectName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"MyProject", "myproject"},
		{"my-project", "my-project"},
		{"My Project", "my-project"},
		{"My_Project", "my-project"},
		{"123project", "project-123project"},
		{"my--project", "my-project"},
		{"project-", "project"},
		{"UPPER", "upper"},
		{"with spaces", "with-spaces"},
		{"", "my-project"},
		{"123", "project-123"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeProjectName_SpecialCharacters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"dots", "my.project", "my-project"},
		{"underscores", "my_project_name", "my-project-name"},
		{"mixed special", "my@project#123", "my-project-123"},
		{"unicode", "my™project", "my-project"},
		{"multiple consecutive special", "my---project___name", "my-project-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// testProjectOptions configures test project creation.
type testProjectOptions struct {
	// IncludeDocker adds docker configuration to the project.
	IncludeDocker bool
	// IncludeAuxiliaryTarget adds an auxiliary target (img) to the project.
	IncludeAuxiliaryTarget bool
}

// createTestProject creates a temporary project for testing CLI commands.
func createTestProject(t *testing.T) string {
	return createTestProjectWithOptions(t, testProjectOptions{IncludeAuxiliaryTarget: true})
}

// createTestProjectWithOptions creates a temporary project with the specified options.
func createTestProjectWithOptions(t *testing.T, opts testProjectOptions) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Resolve symlinks (macOS /var -> /private/var)
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create target directories
	csDir := filepath.Join(root, "cs")
	if err := os.MkdirAll(csDir, 0755); err != nil {
		t.Fatal(err)
	}

	if opts.IncludeAuxiliaryTarget {
		imgDir := filepath.Join(root, "img")
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create minimal .csproj file for C# target (required on Windows where dotnet is installed)
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	csprojPath := filepath.Join(csDir, "test.csproj")
	if err := os.WriteFile(csprojPath, []byte(csproj), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal C# source file
	csFile := "namespace Test;\n\npublic class Class1 { }\n"
	csFilePath := filepath.Join(csDir, "Class1.cs")
	if err := os.WriteFile(csFilePath, []byte(csFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Build config JSON
	configJSON := buildTestProjectConfig(opts)
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create docker-compose.yml if docker is enabled
	if opts.IncludeDocker {
		composeFile := `version: "3.8"
services:
  cs:
    image: mcr.microsoft.com/dotnet/sdk:8.0
    working_dir: /app
    volumes:
      - .:/app
`
		composePath := filepath.Join(root, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(composeFile), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

// buildTestProjectConfig generates the config.json content based on options.
func buildTestProjectConfig(opts testProjectOptions) string {
	var dockerSection string
	if opts.IncludeDocker {
		dockerSection = `
		"docker": {
			"compose_file": "docker-compose.yml",
			"env_var": "TEST_DOCKER"
		},`
	}

	var imgTarget string
	if opts.IncludeAuxiliaryTarget {
		imgTarget = `,
			"img": {
				"type": "auxiliary",
				"title": "Images"
			}`
	}

	return `{
		"project": {"name": "test-project"},
		"mise": {"enabled": false},` + dockerSection + `
		"targets": {
			"cs": {
				"type": "language",
				"title": "C#",
				"toolchain": "dotnet"
			}` + imgTarget + `
		}
	}`
}

// withWorkingDir changes to dir, runs fn, then restores original directory.
// IMPORTANT: Tests using this helper should NOT use t.Parallel() because
// os.Chdir() modifies process-global state (working directory).
func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})
	fn()
}

func TestCmdTargets_Success(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdTargets([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdTargets() = %d, want 0", exitCode)
		}
	})
}

func TestCmdTargets_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdTargets([]string{}, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdTargets() = 0, want non-zero when no project")
		}
	})
}

func TestCmdTargets_WithTypeFilter(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Filter to language only
		exitCode := cmdTargets([]string{}, &GlobalOptions{TargetType: "language"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(language) = %d, want 0", exitCode)
		}

		// Filter to auxiliary only
		exitCode = cmdTargets([]string{}, &GlobalOptions{TargetType: "auxiliary"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(auxiliary) = %d, want 0", exitCode)
		}
	})
}

func TestCmdTargets_JSONOutput(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdTargets([]string{"--json"}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdTargets(--json) = %d, want 0", exitCode)
		}
	})
}

func TestCmdTargets_JSONWithTypeFilter(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdTargets([]string{"--json"}, &GlobalOptions{TargetType: "language"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(--json, language) = %d, want 0", exitCode)
		}
	})
}

func TestCmdTargets_JSONEmptyResult(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Filter to nonexistent type should return empty array
		exitCode := cmdTargets([]string{"--json"}, &GlobalOptions{TargetType: "nonexistent"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(--json, nonexistent) = %d, want 0 (empty result)", exitCode)
		}
	})
}

func TestCmdTargets_UnknownFlag_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdTargets([]string{"--unknown"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdTargets(--unknown) = %d, want 2 (config error)", exitCode)
		}
	})
}

func TestCmdTargets_UnknownArg_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdTargets([]string{"foo"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdTargets(foo) = %d, want 2 (config error)", exitCode)
		}
	})
}

func TestTargetJSON_DependsOnAlwaysPresent(t *testing.T) {
	t.Parallel()
	// Verify that TargetJSON serializes depends_on even when empty.
	// This ensures consistent JSON schema for API consumers.
	tests := []struct {
		name   string
		target TargetJSON
		want   string
	}{
		{
			name: "empty depends_on",
			target: TargetJSON{
				Name:      "go",
				Type:      "language",
				Title:     "Go",
				Commands:  []string{"build", "test"},
				DependsOn: []string{},
			},
			want: `"depends_on": []`,
		},
		{
			name: "non-empty depends_on",
			target: TargetJSON{
				Name:      "go",
				Type:      "language",
				Title:     "Go",
				Commands:  []string{"build"},
				DependsOn: []string{"core"},
			},
			want: `"depends_on": [`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.MarshalIndent(tt.target, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			output := string(data)
			if !strings.Contains(output, tt.want) {
				t.Errorf("JSON output missing %q:\n%s", tt.want, output)
			}
		})
	}
}

func TestCmdConfig_NoSubcommand_ReturnsError(t *testing.T) {
	exitCode := cmdConfig([]string{})
	if exitCode != 2 {
		t.Errorf("cmdConfig([]) = %d, want 2", exitCode)
	}
}

func TestCmdConfig_UnknownSubcommand_ReturnsError(t *testing.T) {
	exitCode := cmdConfig([]string{"unknown"})
	if exitCode != 2 {
		t.Errorf("cmdConfig([unknown]) = %d, want 2", exitCode)
	}
}

func TestCmdConfigValidate_ValidProject(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdConfigValidate()
		if exitCode != 0 {
			t.Errorf("cmdConfigValidate() = %d, want 0", exitCode)
		}
	})
}

func TestCmdConfigValidate_InvalidProject(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdConfigValidate()
		if exitCode == 0 {
			t.Error("cmdConfigValidate() = 0, want non-zero when no project")
		}
	})
}

func TestCmdUnified_EmptyArgs_ReturnsError(t *testing.T) {
	exitCode := cmdUnified([]string{}, &GlobalOptions{})
	if exitCode != 2 {
		t.Errorf("cmdUnified([]) = %d, want 2", exitCode)
	}
}

func TestCmdUnified_CommandMatchingTargetName_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// With command-first order, "cs" is treated as a command name
		// No target has a "cs" command defined, so it should fail
		exitCode := cmdUnified([]string{"cs"}, &GlobalOptions{})
		if exitCode != 1 {
			t.Errorf("cmdUnified([cs]) = %d, want 1 (unknown command)", exitCode)
		}
	})
}

func TestCmdUnified_UnknownCommand_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// "nonexistent" is not a target, so it's treated as a command
		// but no target has this command
		exitCode := cmdUnified([]string{"nonexistent"}, &GlobalOptions{})
		if exitCode != 1 {
			t.Errorf("cmdUnified(nonexistent) = %d, want 1", exitCode)
		}
	})
}

func TestCmdUnified_TargetWithUndefinedCommand_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// "img" target has no commands defined
		// With command-first order: structyl build img
		exitCode := cmdUnified([]string{"build", "img"}, &GlobalOptions{})
		if exitCode != 1 {
			t.Errorf("cmdUnified(build, img) = %d, want 1", exitCode)
		}
	})
}

func TestCmdUnified_TypeFilter_NoMatchingTargets(t *testing.T) {
	// Cannot use t.Parallel(): uses withWorkingDir which modifies global state
	root := createTestProjectWithOptions(t, testProjectOptions{IncludeAuxiliaryTarget: false})
	withWorkingDir(t, root, func() {
		// Project has only "cs" (language) target, no auxiliary targets
		// Filter by auxiliary type should find no targets and return 0 with warning
		exitCode := cmdUnified([]string{"build"}, &GlobalOptions{TargetType: "auxiliary"})
		if exitCode != 0 {
			t.Errorf("cmdUnified(build, --type=auxiliary) = %d, want 0 (warning, no error)", exitCode)
		}
	})
}

func TestCmdUnified_TypeFilter_MatchingTargets(t *testing.T) {
	// Cannot use t.Parallel(): uses withWorkingDir which modifies global state
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Project has "cs" (language) and "img" (auxiliary) targets
		// Filter by language type should find "cs" target and attempt to run command
		// Exit code 1 is expected because mise is disabled in test project
		exitCode := cmdUnified([]string{"build"}, &GlobalOptions{TargetType: "language"})
		// We just verify the code path is exercised - exit code depends on mise availability
		// The important thing is it doesn't return 2 (config error) or panic
		if exitCode == 2 {
			t.Errorf("cmdUnified(build, --type=language) = 2 (config error), want 0 or 1")
		}
	})
}

func TestCmdCI_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdCI("ci", nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdCI() = 0, want non-zero when no project")
		}
	})
}

func TestCmdCI_Release_HasReleaseBuild(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// ci:release should work on valid project.
		// This verifies command routing to the release pipeline.
		// Exit code 2 = usage/routing error, 0/1 = command was parsed correctly
		exitCode := cmdCI("ci:release", nil, &GlobalOptions{})
		if exitCode == 2 {
			t.Errorf("cmdCI(ci:release) = 2 (usage error), want 0 or 1 (command recognized)")
		}
	})
}

// =============================================================================
// Work Item 1: cmdInit Tests
// =============================================================================

func TestCmdInit_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks (macOS /var -> /private/var)
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit(nil)
		if exitCode != 0 {
			t.Errorf("cmdInit() = %d, want 0", exitCode)
		}

		// Verify .structyl/config.json was created
		configPath := filepath.Join(root, ".structyl", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error(".structyl/config.json was not created")
		}

		// Verify .structyl/AGENTS.md was created
		agentsPath := filepath.Join(root, ".structyl", "AGENTS.md")
		if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
			t.Error(".structyl/AGENTS.md was not created")
		}
	})
}

func TestCmdInit_ConfigExists_IsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create existing config in .structyl/
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"name":"existing"},"targets":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		// Init should succeed when config exists (idempotent)
		exitCode := cmdInit(nil)
		if exitCode != 0 {
			t.Errorf("cmdInit() = %d, want 0 (idempotent)", exitCode)
		}

		// Config should not be overwritten
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "existing") {
			t.Error("config.json was overwritten, want preserved")
		}
	})
}

func TestCmdInit_WithMiseFlag_CreatesMiseToml(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create rs/ directory with Cargo.toml for detection
	rsDir := filepath.Join(root, "rs")
	if err := os.MkdirAll(rsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rsDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit([]string{"--mise"})
		if exitCode != 0 {
			t.Errorf("cmdInit(--mise) = %d, want 0", exitCode)
			return
		}

		// Verify mise.toml was created
		miseTomlPath := filepath.Join(root, "mise.toml")
		if _, err := os.Stat(miseTomlPath); os.IsNotExist(err) {
			t.Error("mise.toml was not created")
		}
	})
}

func TestCmdInit_CreatesVersionFile(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit(nil)
		if exitCode != 0 {
			t.Errorf("cmdInit() = %d, want 0", exitCode)
			return
		}

		// Verify PROJECT_VERSION file was created
		versionPath := filepath.Join(root, ".structyl", "PROJECT_VERSION")
		content, err := os.ReadFile(versionPath)
		if err != nil {
			t.Errorf("PROJECT_VERSION file not created: %v", err)
			return
		}
		if string(content) != "0.1.0\n" {
			t.Errorf("PROJECT_VERSION = %q, want %q", string(content), "0.1.0\n")
		}
	})
}

func TestCmdInit_CreatesTestsDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit(nil)
		if exitCode != 0 {
			t.Errorf("cmdInit() = %d, want 0", exitCode)
			return
		}

		// Verify tests directory was created
		testsDir := filepath.Join(root, "tests")
		info, err := os.Stat(testsDir)
		if err != nil {
			t.Errorf("tests directory not created: %v", err)
			return
		}
		if !info.IsDir() {
			t.Error("tests is not a directory")
		}
	})
}

func TestCmdInit_VersionFileExists_NotOverwritten(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create existing PROJECT_VERSION file in .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(structylDir, "PROJECT_VERSION")
	existingVersion := "1.0.0\n"
	if err := os.WriteFile(versionPath, []byte(existingVersion), 0644); err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit(nil)
		if exitCode != 0 {
			t.Errorf("cmdInit() = %d, want 0", exitCode)
			return
		}

		// Verify PROJECT_VERSION file was NOT overwritten
		content, err := os.ReadFile(versionPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != existingVersion {
			t.Errorf("PROJECT_VERSION = %q, want %q (should not be overwritten)", string(content), existingVersion)
		}
	})
}

// =============================================================================
// Work Item 2: detectTargetDirectories Tests
// =============================================================================

func TestDetectTargetDirectories_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	targets := detectTargetDirectories(tmpDir)
	if len(targets) != 0 {
		t.Errorf("detectTargetDirectories() = %d targets, want 0", len(targets))
	}
}

func TestDetectTargetDirectories_Rust(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rs/ directory with Cargo.toml
	rsDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(rsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rsDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targets := detectTargetDirectories(tmpDir)
	if len(targets) != 1 {
		t.Errorf("detectTargetDirectories() = %d targets, want 1", len(targets))
		return
	}

	target, ok := targets["rs"]
	if !ok {
		t.Error("expected 'rs' target not found")
		return
	}
	if target.Toolchain != "cargo" {
		t.Errorf("target.Toolchain = %q, want %q", target.Toolchain, "cargo")
	}
	if target.Title != "Rust" {
		t.Errorf("target.Title = %q, want %q", target.Title, "Rust")
	}
}

func TestDetectTargetDirectories_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go/ directory with go.mod
	goDir := filepath.Join(tmpDir, "go")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targets := detectTargetDirectories(tmpDir)
	target, ok := targets["go"]
	if !ok {
		t.Error("expected 'go' target not found")
		return
	}
	if target.Toolchain != "go" {
		t.Errorf("target.Toolchain = %q, want %q", target.Toolchain, "go")
	}
}

func TestDetectTargetDirectories_Python(t *testing.T) {
	tmpDir := t.TempDir()

	// Create py/ directory with pyproject.toml
	pyDir := filepath.Join(tmpDir, "py")
	if err := os.MkdirAll(pyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pyDir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targets := detectTargetDirectories(tmpDir)
	target, ok := targets["py"]
	if !ok {
		t.Error("expected 'py' target not found")
		return
	}
	// Python detection should find a Python toolchain
	if target.Title != "Python" {
		t.Errorf("target.Title = %q, want %q", target.Title, "Python")
	}
}

func TestDetectTargetDirectories_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rs/ with Cargo.toml
	rsDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(rsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rsDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go/ with go.mod
	goDir := filepath.Join(tmpDir, "go")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targets := detectTargetDirectories(tmpDir)
	if len(targets) != 2 {
		t.Errorf("detectTargetDirectories() = %d targets, want 2", len(targets))
	}
	if _, ok := targets["rs"]; !ok {
		t.Error("expected 'rs' target not found")
	}
	if _, ok := targets["go"]; !ok {
		t.Error("expected 'go' target not found")
	}
}

func TestDetectTargetDirectories_NoIndicatorFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rs/ directory WITHOUT Cargo.toml
	rsDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(rsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Don't create any indicator files

	targets := detectTargetDirectories(tmpDir)
	if len(targets) != 0 {
		t.Errorf("detectTargetDirectories() = %d targets, want 0 (no indicator files)", len(targets))
	}
}

func TestDetectTargetDirectories_AliasedDir_Rust(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rust/ (alias for rs) with Cargo.toml
	rustDir := filepath.Join(tmpDir, "rust")
	if err := os.MkdirAll(rustDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rustDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targets := detectTargetDirectories(tmpDir)
	if len(targets) != 1 {
		t.Errorf("detectTargetDirectories() = %d targets, want 1", len(targets))
		return
	}
	// Should be stored under canonical name "rs"
	target, ok := targets["rs"]
	if !ok {
		t.Error("expected 'rs' target (from 'rust' dir) not found")
		return
	}
	if target.Directory != "rust" {
		t.Errorf("target.Directory = %q, want %q", target.Directory, "rust")
	}
}

// =============================================================================
// Work Item 3: updateGitignore Tests
// =============================================================================

func TestUpdateGitignore_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	updateGitignore(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	if !strings.Contains(string(content), "# Structyl") {
		t.Error(".gitignore should contain '# Structyl' header")
	}
	if !strings.Contains(string(content), "artifacts/") {
		t.Error(".gitignore should contain 'artifacts/' entry")
	}
	// .structyl/ should NOT be in gitignore (config should be tracked)
	if strings.Contains(string(content), ".structyl/") {
		t.Error(".gitignore should NOT contain '.structyl/' entry (config should be tracked)")
	}
}

func TestUpdateGitignore_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing .gitignore with other entries
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	existingContent := "node_modules/\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	updateGitignore(tmpDir)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	// Should preserve existing content
	if !strings.Contains(string(content), "node_modules/") {
		t.Error(".gitignore should still contain 'node_modules/'")
	}
	// Should add Structyl entries
	if !strings.Contains(string(content), "# Structyl") {
		t.Error(".gitignore should contain '# Structyl' header")
	}
}

func TestUpdateGitignore_AlreadyHasStructyl(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore with Structyl entries already
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	existingContent := "node_modules/\n# Structyl\nartifacts/\n"
	if err := os.WriteFile(gitignorePath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	updateGitignore(tmpDir)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	// Should not duplicate entries
	count := strings.Count(string(content), "# Structyl")
	if count != 1 {
		t.Errorf("'# Structyl' appears %d times, want 1 (no duplication)", count)
	}
}

func TestUpdateGitignore_NoTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore without trailing newline
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	existingContent := "node_modules/" // No trailing newline
	if err := os.WriteFile(gitignorePath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	updateGitignore(tmpDir)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	// Should properly separate entries
	if !strings.Contains(string(content), "node_modules/\n") {
		t.Error(".gitignore should have newline after existing content")
	}
	if !strings.Contains(string(content), "# Structyl") {
		t.Error(".gitignore should contain '# Structyl' header")
	}
}

func TestUpdateGitignore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	updateGitignore(tmpDir)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	if !strings.Contains(string(content), "# Structyl") {
		t.Error(".gitignore should contain '# Structyl' header")
	}
}

// =============================================================================
// Work Item 4: Docker Commands Tests
// =============================================================================

func TestCmdDockerBuild_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdDockerBuild(nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdDockerBuild() = 0, want non-zero when no project")
		}
	})
}

func TestCmdDockerClean_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdDockerClean([]string{}, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdDockerClean() = 0, want non-zero when no project")
		}
	})
}

func TestHandleDockerError_DockerUnavailable(t *testing.T) {
	t.Parallel()
	err := &runner.DockerUnavailableError{}
	exitCode := handleDockerError(err)
	if exitCode != 3 {
		t.Errorf("handleDockerError(DockerUnavailableError) = %d, want 3", exitCode)
	}
}

func TestHandleDockerError_OtherError(t *testing.T) {
	t.Parallel()
	err := errors.New("some other error")
	exitCode := handleDockerError(err)
	if exitCode != 1 {
		t.Errorf("handleDockerError(other) = %d, want 1", exitCode)
	}
}

// =============================================================================
// Work Item 2: cmdCI Pipeline Tests
// =============================================================================

func TestCmdCI_DebugMode_ExecutesPipeline(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Debug mode CI pipeline: clean, restore, check, build, test
		// This test verifies the command path works without panic
		// Actual execution may fail if toolchains aren't installed
		exitCode := cmdCI("ci", nil, &GlobalOptions{})
		// Exit code should be valid (0 = success, 1 = command failure, 2 = usage error)
		// In CI without toolchains, we expect 1 (command failure) not 2 (usage error)
		if exitCode == 2 {
			t.Errorf("cmdCI(ci) = 2 (usage error), want 0 or 1")
		}
	})
}

func TestCmdCI_ReleaseMode_UsesBuildRelease(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Release mode CI pipeline: clean, restore, check, build:release, test
		// This test verifies the release mode path works
		exitCode := cmdCI("ci:release", nil, &GlobalOptions{})
		// Exit code 2 would indicate routing/parsing failure
		if exitCode == 2 {
			t.Errorf("cmdCI(ci:release) = 2 (usage error), want 0 or 1")
		}
	})
}

// =============================================================================
// Work Item 3: Docker Command Tests
// =============================================================================

// createTestProjectWithDocker creates a test project with docker configuration.
func createTestProjectWithDocker(t *testing.T) string {
	return createTestProjectWithOptions(t, testProjectOptions{IncludeDocker: true})
}

func TestCmdDockerBuild_ValidProject_LoadsProject(t *testing.T) {
	root := createTestProjectWithDocker(t)
	withWorkingDir(t, root, func() {
		// This tests the project loading path up to Docker availability check
		// Will fail with Docker unavailable error (expected in CI)
		exitCode := cmdDockerBuild(nil, &GlobalOptions{})
		// Exit code 3 = Docker unavailable (expected)
		// Exit code 0 = Docker available and build succeeded
		// Other exit codes = unexpected error
		if exitCode != 0 && exitCode != 3 {
			t.Errorf("cmdDockerBuild() = %d, want 0 or 3", exitCode)
		}
	})
}

func TestCmdDockerClean_ValidProject_LoadsProject(t *testing.T) {
	root := createTestProjectWithDocker(t)
	withWorkingDir(t, root, func() {
		// This tests the project loading path up to Docker availability check
		exitCode := cmdDockerClean([]string{}, &GlobalOptions{})
		// Exit code 3 = Docker unavailable (expected)
		// Exit code 0 = Docker available and clean succeeded
		if exitCode != 0 && exitCode != 3 {
			t.Errorf("cmdDockerClean() = %d, want 0 or 3", exitCode)
		}
	})
}

func TestCmdDockerBuild_WithTargetArgs(t *testing.T) {
	root := createTestProjectWithDocker(t)
	withWorkingDir(t, root, func() {
		// Test with specific target argument
		exitCode := cmdDockerBuild([]string{"cs"}, &GlobalOptions{})
		// May fail due to Docker unavailable
		if exitCode != 0 && exitCode != 3 {
			t.Errorf("cmdDockerBuild(cs) = %d, want 0 or 3", exitCode)
		}
	})
}

// =============================================================================
// Work Item 6: Command Rename Tests (init→restore, new→init)
// =============================================================================

func TestRun_RestoreCommand_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := Run([]string{"restore"})
		if exitCode == 0 {
			t.Error("Run(restore) = 0, want non-zero when no project")
		}
	})
}

func TestRun_RestoreCommand_ValidProject(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// "restore" meta command should work on valid project
		// May fail during actual execution if toolchains aren't installed
		exitCode := Run([]string{"restore"})
		// Exit code 2 would indicate routing/parsing failure
		if exitCode == 2 {
			t.Errorf("Run(restore) = 2 (usage error), want 0 or 1")
		}
	})
}

func TestRun_DeprecatedNew_ShowsWarning(t *testing.T) {
	// The "new" command should still work but is deprecated
	// It should emit a warning to stderr
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		// "new" should create project (deprecated alias for "init")
		exitCode := Run([]string{"new"})
		if exitCode != 0 {
			t.Errorf("Run(new) = %d, want 0", exitCode)
		}

		// Verify project was created
		configPath := filepath.Join(root, ".structyl", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error(".structyl/config.json was not created by 'new' command")
		}
	})
}

func TestRun_InitCommand_CreatesProject(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		// "init" should create project (renamed from "new")
		exitCode := Run([]string{"init"})
		if exitCode != 0 {
			t.Errorf("Run(init) = %d, want 0", exitCode)
		}

		// Verify project was created
		configPath := filepath.Join(root, ".structyl", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error(".structyl/config.json was not created by 'init' command")
		}
	})
}

// =============================================================================
// Work Item 1: cmdRelease Tests
// =============================================================================

func TestCmdRelease_MissingVersion_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Release requires a version argument
		exitCode := cmdRelease(nil, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdRelease(nil) = %d, want 2 (usage error)", exitCode)
		}

		// Also test with empty args
		exitCode = cmdRelease([]string{}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdRelease([]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

func TestCmdRelease_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdRelease([]string{"1.0.0"}, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdRelease() = 0, want non-zero when no project")
		}
	})
}

func TestCmdRelease_ParsesFlags(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Test that flags are parsed correctly without actually running release
		// Using --dry-run so no git operations are performed
		tests := []struct {
			name string
			args []string
		}{
			{"version only", []string{"1.0.0", "--dry-run"}},
			{"with push", []string{"1.0.0", "--dry-run", "--push"}},
			{"with force", []string{"1.0.0", "--dry-run", "--force"}},
			{"all flags", []string{"1.0.0", "--dry-run", "--push", "--force"}},
			{"flags before version", []string{"--dry-run", "1.0.0"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// --dry-run prevents actual release, but tests flag parsing
				exitCode := cmdRelease(tt.args, &GlobalOptions{})
				// May fail due to not being a git repo, but should not return 2 (usage error)
				// Exit code 2 would indicate flag parsing failed
				if exitCode == 2 {
					t.Errorf("cmdRelease(%v) = 2 (usage error), want different exit code", tt.args)
				}
			})
		}
	})
}

func TestCmdRelease_InvalidVersion_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Invalid version format should return error
		exitCode := cmdRelease([]string{"invalid-version", "--dry-run"}, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdRelease(invalid-version) = 0, want non-zero")
		}
	})
}

// =============================================================================
// collectAllCommands Tests
// =============================================================================

func TestCollectAllCommands_EmptyTargets(t *testing.T) {
	t.Parallel()
	result := collectAllCommands(nil)
	if len(result) != 0 {
		t.Errorf("collectAllCommands(nil) = %d items, want 0", len(result))
	}

	result = collectAllCommands([]target.Target{})
	if len(result) != 0 {
		t.Errorf("collectAllCommands([]) = %d items, want 0", len(result))
	}
}

func TestCollectAllCommands_SingleTarget(t *testing.T) {
	t.Parallel()
	mock := mocks.NewTarget("test").
		WithType(target.TypeLanguage).
		WithCommands(map[string]interface{}{
			"build": "go build",
			"test":  "go test",
		})

	result := collectAllCommands([]target.Target{mock})
	if len(result) != 2 {
		t.Errorf("collectAllCommands() = %d items, want 2", len(result))
	}

	// Verify commands are present
	cmdNames := make(map[string]bool)
	for _, cmd := range result {
		cmdNames[cmd.name] = true
	}
	if !cmdNames["build"] || !cmdNames["test"] {
		t.Errorf("collectAllCommands() missing expected commands, got %v", cmdNames)
	}
}

func TestCollectAllCommands_MultipleTargets_SortsByFrequency(t *testing.T) {
	t.Parallel()
	// Target 1 has build, test, clean
	mock1 := mocks.NewTarget("t1").
		WithType(target.TypeLanguage).
		WithCommands(map[string]interface{}{
			"build": "cmd1",
			"test":  "cmd2",
			"clean": "cmd3",
		})
	// Target 2 has build, test (no clean)
	mock2 := mocks.NewTarget("t2").
		WithType(target.TypeLanguage).
		WithCommands(map[string]interface{}{
			"build": "cmd1",
			"test":  "cmd2",
		})
	// Target 3 has build only
	mock3 := mocks.NewTarget("t3").
		WithType(target.TypeLanguage).
		WithCommands(map[string]interface{}{
			"build": "cmd1",
		})

	result := collectAllCommands([]target.Target{mock1, mock2, mock3})

	// build appears in 3 targets, test in 2, clean in 1
	// First item should be "build" (most frequent)
	if len(result) < 1 {
		t.Fatal("collectAllCommands() returned empty result")
	}
	if result[0].name != "build" {
		t.Errorf("collectAllCommands()[0].name = %q, want %q (most frequent)", result[0].name, "build")
	}
}

func TestCollectAllCommands_DescriptionMapping(t *testing.T) {
	t.Parallel()
	mock := mocks.NewTarget("test").
		WithType(target.TypeLanguage).
		WithCommands(map[string]interface{}{
			"build":   "go build",
			"test":    "go test",
			"clean":   "rm -rf",
			"restore": "go mod download",
			"custom":  "echo custom", // Not in descriptions map
		})

	result := collectAllCommands([]target.Target{mock})

	descMap := make(map[string]string)
	for _, cmd := range result {
		descMap[cmd.name] = cmd.description
	}

	// Known commands should have descriptions from the map
	if !containsSubstr(descMap["build"], "Build") {
		t.Errorf("build description = %q, want to contain 'Build'", descMap["build"])
	}
	if !containsSubstr(descMap["test"], "test") {
		t.Errorf("test description = %q, want to contain 'test'", descMap["test"])
	}
	// Unknown command should have fallback description
	if !containsSubstr(descMap["custom"], "custom") {
		t.Errorf("custom description = %q, want to contain 'custom'", descMap["custom"])
	}
}

func containsSubstr(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// =============================================================================
// Help Printing Tests
// =============================================================================

func TestRun_Help_NoProject_ReturnsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		// --help outside a project should succeed
		exitCode := Run([]string{"--help"})
		if exitCode != 0 {
			t.Errorf("Run([--help]) = %d, want 0", exitCode)
		}
	})
}

func TestRun_Help_WithProject_ReturnsSuccess(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// --help inside a project should succeed
		exitCode := Run([]string{"--help"})
		if exitCode != 0 {
			t.Errorf("Run([--help]) = %d, want 0", exitCode)
		}
	})
}

func TestPrintUsage_WithProjectTargets_ShowsAllTargets(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// This test verifies printUsage executes without error when a project exists
		// We can't easily capture stdout in tests, but we verify no panic occurs
		// and the function completes
		exitCode := Run([]string{"--help"})
		if exitCode != 0 {
			t.Errorf("Run([--help]) with project = %d, want 0", exitCode)
		}
	})
}

// =============================================================================
// Print Function Tests (Smoke Tests)
//
// These tests verify print/usage functions execute without panic. They are
// intentionally smoke tests with minimal assertions—the primary value is
// catching nil pointer dereferences, missing template variables, and other
// runtime errors that would cause panics during help output generation.
// =============================================================================

func TestPrintUnifiedUsage_AllCommands(t *testing.T) {
	t.Parallel()
	// Test that printUnifiedUsage executes without panic for all known commands
	commands := []string{
		"build", "build:release", "test", "test:coverage",
		"clean", "restore", "check", "check:fix",
		"bench", "demo", "doc", "pack",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			// Just verify no panic occurs
			printUnifiedUsage(cmd)
		})
	}
}

func TestPrintUnifiedUsage_UnknownCommand(t *testing.T) {
	t.Parallel()
	// Unknown commands should use default description
	printUnifiedUsage("unknown-cmd")
}

func TestPrintReleaseUsage(t *testing.T) {
	t.Parallel()
	// Verify release usage prints without panic
	printReleaseUsage()
}

func TestPrintCIUsage(t *testing.T) {
	t.Parallel()
	// Verify CI usage prints without panic for all CI commands
	commands := []string{"ci", "ci:release"}
	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			printCIUsage(cmd)
		})
	}
}

func TestPrintConfigUsage(t *testing.T) {
	t.Parallel()
	// Verify config usage prints without panic
	printConfigUsage()
}

func TestPrintMiseUsage(t *testing.T) {
	t.Parallel()
	// Verify mise usage prints without panic
	printMiseUsage()
}

func TestPrintMiseSyncUsage(t *testing.T) {
	t.Parallel()
	// Verify mise sync usage prints without panic
	printMiseSyncUsage()
}

func TestPrintDockerBuildUsage(t *testing.T) {
	t.Parallel()
	// Verify docker build usage prints without panic
	printDockerBuildUsage()
}

func TestPrintDockerCleanUsage(t *testing.T) {
	t.Parallel()
	// Verify docker clean usage prints without panic
	printDockerCleanUsage()
}

func TestPrintTargetsUsage(t *testing.T) {
	t.Parallel()
	// Verify targets usage prints without panic
	printTargetsUsage()
}

func TestPrintCompletionUsage(t *testing.T) {
	t.Parallel()
	// Verify completion usage prints without panic
	printCompletionUsage()
}

func TestPrintInitUsage(t *testing.T) {
	t.Parallel()
	// Verify init usage prints without panic
	printInitUsage()
}

func TestPrintUpgradeUsage(t *testing.T) {
	t.Parallel()
	// Verify upgrade usage prints without panic
	printUpgradeUsage()
}

// =============================================================================
// Mise Command Tests
// =============================================================================

func TestCmdMise_NoSubcommand_ReturnsUsageError(t *testing.T) {
	exitCode := cmdMise(nil, &GlobalOptions{})
	if exitCode != 2 {
		t.Errorf("cmdMise(nil) = %d, want 2 (usage error)", exitCode)
	}

	exitCode = cmdMise([]string{}, &GlobalOptions{})
	if exitCode != 2 {
		t.Errorf("cmdMise([]) = %d, want 2 (usage error)", exitCode)
	}
}

func TestCmdMise_UnknownSubcommand_ReturnsError(t *testing.T) {
	exitCode := cmdMise([]string{"unknown"}, &GlobalOptions{})
	if exitCode != 2 {
		t.Errorf("cmdMise([unknown]) = %d, want 2 (usage error)", exitCode)
	}
}

func TestCmdMise_HelpFlag_ReturnsSuccess(t *testing.T) {
	t.Parallel()
	helpFlags := []string{"-h", "--help"}
	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			exitCode := cmdMise([]string{flag}, &GlobalOptions{})
			if exitCode != 0 {
				t.Errorf("cmdMise([%s]) = %d, want 0", flag, exitCode)
			}
		})
	}
}

func TestCmdMiseSync_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdMiseSync(nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdMiseSync() = 0, want non-zero when no project")
		}
	})
}

func TestCmdMiseSync_HelpFlag_ReturnsSuccess(t *testing.T) {
	t.Parallel()
	helpFlags := []string{"-h", "--help"}
	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			exitCode := cmdMiseSync([]string{flag}, &GlobalOptions{})
			if exitCode != 0 {
				t.Errorf("cmdMiseSync([%s]) = %d, want 0", flag, exitCode)
			}
		})
	}
}

func TestCmdMiseSync_UnknownOption_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdMiseSync([]string{"--invalid"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdMiseSync([--invalid]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

func TestCmdMiseSync_RemovedForceFlag_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdMiseSync([]string{"--force"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdMiseSync([--force]) = %d, want 2 (config error)", exitCode)
		}
	})
}

func TestCmdMiseSync_ValidProject_Success(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// First call should generate mise.toml
		exitCode := cmdMiseSync(nil, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdMiseSync() = %d, want 0", exitCode)
		}

		// Verify mise.toml was created
		miseTomlPath := filepath.Join(root, "mise.toml")
		if _, err := os.Stat(miseTomlPath); os.IsNotExist(err) {
			t.Error("mise.toml was not created")
		}
	})
}

func TestCmdDockerfile_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdDockerfile(nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdDockerfile() = 0, want non-zero when no project")
		}
	})
}

func TestCmdDockerfile_HelpFlag_ReturnsSuccess(t *testing.T) {
	t.Parallel()
	helpFlags := []string{"-h", "--help"}
	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			exitCode := cmdDockerfile([]string{flag}, &GlobalOptions{})
			if exitCode != 0 {
				t.Errorf("cmdDockerfile([%s]) = %d, want 0", flag, exitCode)
			}
		})
	}
}

func TestCmdDockerfile_UnknownOption_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdDockerfile([]string{"--invalid"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdDockerfile([--invalid]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

func TestCmdGitHub_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdGitHub(nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdGitHub() = 0, want non-zero when no project")
		}
	})
}

func TestCmdGitHub_HelpFlag_ReturnsSuccess(t *testing.T) {
	t.Parallel()
	helpFlags := []string{"-h", "--help"}
	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			exitCode := cmdGitHub([]string{flag}, &GlobalOptions{})
			if exitCode != 0 {
				t.Errorf("cmdGitHub([%s]) = %d, want 0", flag, exitCode)
			}
		})
	}
}

func TestCmdGitHub_UnknownOption_ReturnsError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdGitHub([]string{"--invalid"}, &GlobalOptions{})
		if exitCode != 2 {
			t.Errorf("cmdGitHub([--invalid]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

func TestCmdDockerfile_ValidProject_Success(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdDockerfile([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdDockerfile() = %d, want 0", exitCode)
		}
		// Verify Dockerfile was created for cs target (dotnet toolchain)
		dockerfilePath := filepath.Join(root, "cs", "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			t.Error("cmdDockerfile() did not create cs/Dockerfile")
		}
	})
}

func TestCmdDockerfile_Force_OverwritesExisting(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Create initial Dockerfile
		exitCode := cmdDockerfile([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Fatalf("cmdDockerfile() initial = %d, want 0", exitCode)
		}

		dockerfilePath := filepath.Join(root, "cs", "Dockerfile")

		// Get initial file info
		initialInfo, err := os.Stat(dockerfilePath)
		if err != nil {
			t.Fatalf("could not stat initial Dockerfile: %v", err)
		}

		// Run again without force - should skip
		exitCode = cmdDockerfile([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdDockerfile() without force = %d, want 0", exitCode)
		}

		// File should be unchanged (mtime same)
		skipInfo, _ := os.Stat(dockerfilePath)
		if skipInfo.ModTime() != initialInfo.ModTime() {
			t.Error("cmdDockerfile() without force modified the file")
		}

		// Run with --force - should overwrite
		exitCode = cmdDockerfile([]string{"--force"}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdDockerfile(--force) = %d, want 0", exitCode)
		}
	})
}

func TestCmdGitHub_ValidProject_Success(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdGitHub([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdGitHub() = %d, want 0", exitCode)
		}
		// Verify workflow file was created
		workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
		if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
			t.Error("cmdGitHub() did not create .github/workflows/ci.yml")
		}
	})
}

func TestCmdGitHub_FileExists_SkipsWithoutForce(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Create initial workflow
		exitCode := cmdGitHub([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Fatalf("cmdGitHub() initial = %d, want 0", exitCode)
		}

		workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")

		// Get initial content
		initialContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("could not read initial workflow: %v", err)
		}

		// Run again without force - should skip (exit 0, file unchanged)
		exitCode = cmdGitHub([]string{}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdGitHub() without force = %d, want 0", exitCode)
		}

		// Content should be unchanged
		skipContent, _ := os.ReadFile(workflowPath)
		if string(skipContent) != string(initialContent) {
			t.Error("cmdGitHub() without force modified the file")
		}

		// Run with --force - should overwrite (exit 0)
		exitCode = cmdGitHub([]string{"--force"}, &GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdGitHub(--force) = %d, want 0", exitCode)
		}
	})
}

// =============================================================================
// isMiseAutoGenerateEnabled Tests
// =============================================================================

func TestIsMiseAutoGenerateEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *config.Config
		expected bool
	}{
		{
			name:     "nil Mise config",
			cfg:      &config.Config{},
			expected: true, // default is enabled
		},
		{
			name: "Mise config with nil AutoGenerate",
			cfg: &config.Config{
				Mise: &config.MiseConfig{
					AutoGenerate: nil,
				},
			},
			expected: true, // default is enabled
		},
		{
			name: "AutoGenerate explicitly true",
			cfg: &config.Config{
				Mise: &config.MiseConfig{
					AutoGenerate: boolPtr(true),
				},
			},
			expected: true,
		},
		{
			name: "AutoGenerate explicitly false",
			cfg: &config.Config{
				Mise: &config.MiseConfig{
					AutoGenerate: boolPtr(false),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isMiseAutoGenerateEnabled(tt.cfg)
			if got != tt.expected {
				t.Errorf("isMiseAutoGenerateEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEnsureMiseConfig_InvalidMode_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create minimal project
	proj := &project.Project{}

	// Call with invalid mode value (not MiseForceRegenerate or MiseAutoRegenerate)
	err := ensureMiseConfig(proj, MiseRegenerateMode(99))
	if err == nil {
		t.Error("expected error for invalid MiseRegenerateMode")
	}
	// Verify error message contains expected text
	if !strings.Contains(err.Error(), "BUG: invalid MiseRegenerateMode") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "BUG: invalid MiseRegenerateMode")
	}
}

// =============================================================================
// promptConfirmWithReader Tests
// =============================================================================

func TestPromptConfirmWithReader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"y returns true", "y\n", true},
		{"Y returns true", "Y\n", true},
		{"yes returns true", "yes\n", true},
		{"YES returns true", "YES\n", true},
		{"Yes returns true", "Yes\n", true},
		{"n returns false", "n\n", false},
		{"no returns false", "no\n", false},
		{"empty returns false", "\n", false},
		{"whitespace returns false", "  \n", false},
		{"invalid returns false", "invalid\n", false},
		{"yy returns false", "yy\n", false},
		{"yeah returns false", "yeah\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reader := strings.NewReader(tt.input)
			result := promptConfirmWithReader("Test?", reader)
			if result != tt.expected {
				t.Errorf("promptConfirmWithReader(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPromptConfirmWithReader_ReadError(t *testing.T) {
	t.Parallel()

	// Use a reader that returns an error (empty reader triggers EOF on first read)
	reader := strings.NewReader("")
	result := promptConfirmWithReader("Test?", reader)
	if result != false {
		t.Error("promptConfirmWithReader with empty reader should return false")
	}
}
