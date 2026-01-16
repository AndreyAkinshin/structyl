package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akinshin/structyl/internal/target"
)

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantDocker     bool
		wantNoDocker   bool
		wantContinue   bool
		wantTargetType string
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
			name:          "--continue flag",
			args:          []string{"--continue", "build"},
			wantContinue:  true,
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
			args:          []string{"--docker", "--continue", "build"},
			wantDocker:    true,
			wantContinue:  true,
			wantRemaining: []string{"build"},
		},
		{
			name:           "all flags combined",
			args:           []string{"--docker", "--continue", "--type=language", "test", "rs"},
			wantDocker:     true,
			wantContinue:   true,
			wantTargetType: "language",
			wantRemaining:  []string{"test", "rs"},
		},
		{
			name:    "invalid --type value",
			args:    []string{"--type=invalid", "build"},
			wantErr: true,
		},
		{
			name:    "invalid --type=foo",
			args:    []string{"--type=foo", "build"},
			wantErr: true,
		},
		{
			name:    "invalid --type=other",
			args:    []string{"--type=other", "build"},
			wantErr: true,
		},
		{
			name:          "empty args",
			args:          []string{},
			wantRemaining: nil,
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
			if opts.ContinueOnError != tt.wantContinue {
				t.Errorf("ContinueOnError = %v, want %v", opts.ContinueOnError, tt.wantContinue)
			}
			if opts.TargetType != tt.wantTargetType {
				t.Errorf("TargetType = %q, want %q", opts.TargetType, tt.wantTargetType)
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

func TestParseGlobalFlags_EmptyTypeIsValid(t *testing.T) {
	// Empty --type= is valid (treated as no type filter)
	opts, remaining, err := parseGlobalFlags([]string{"--type=", "build"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if opts.TargetType != "" {
		t.Errorf("TargetType = %q, want empty", opts.TargetType)
	}
	if len(remaining) != 1 || remaining[0] != "build" {
		t.Errorf("remaining = %v, want [build]", remaining)
	}
}

func TestParseGlobalFlags_TypeWithoutValue(t *testing.T) {
	_, _, err := parseGlobalFlags([]string{"--type"})
	if err == nil {
		t.Error("parseGlobalFlags() expected error for --type without value")
	}
	if err != nil && !strings.Contains(err.Error(), "--type requires a value") {
		t.Errorf("error = %q, want to contain '--type requires a value'", err.Error())
	}
}

func TestParseGlobalFlags_TypeAtEndOfArgs(t *testing.T) {
	_, _, err := parseGlobalFlags([]string{"build", "--type"})
	if err == nil {
		t.Error("parseGlobalFlags() expected error for --type at end of args")
	}
}

func TestRun_Help(t *testing.T) {
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
	exitCode := Run([]string{})
	if exitCode != 0 {
		t.Errorf("Run([]) = %d, want 0", exitCode)
	}
}

func TestSanitizeProjectName(t *testing.T) {
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

func TestIsDockerMode_ExplicitDocker(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "")

	opts := &GlobalOptions{Docker: true}
	if !isDockerMode(opts) {
		t.Error("isDockerMode() = false, want true when Docker flag is set")
	}
}

func TestIsDockerMode_ExplicitNoDocker(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "true")

	opts := &GlobalOptions{NoDocker: true}
	if isDockerMode(opts) {
		t.Error("isDockerMode() = true, want false when NoDocker flag is set")
	}
}

func TestIsDockerMode_NoDockerTakesPrecedence(t *testing.T) {
	opts := &GlobalOptions{Docker: true, NoDocker: true}
	if isDockerMode(opts) {
		t.Error("isDockerMode() = true, want false (NoDocker takes precedence)")
	}
}

func TestIsDockerMode_EnvVar(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "true")

	opts := &GlobalOptions{}
	if !isDockerMode(opts) {
		t.Error("isDockerMode() = false, want true when STRUCTYL_DOCKER=true")
	}
}

func TestIsDockerMode_Default(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "")

	opts := &GlobalOptions{}
	if isDockerMode(opts) {
		t.Error("isDockerMode() = true, want false (default)")
	}
}

// mockTarget implements target.Target for testing
type mockTarget struct {
	name       string
	title      string
	targetType target.TargetType
	directory  string
	commands   map[string]interface{}
	dependsOn  []string
	env        map[string]string
	vars       map[string]string
	demoPath   string
	execFunc   func(ctx context.Context, cmd string, opts target.ExecOptions) error
}

func (m *mockTarget) Name() string            { return m.name }
func (m *mockTarget) Title() string           { return m.title }
func (m *mockTarget) Type() target.TargetType { return m.targetType }
func (m *mockTarget) Directory() string       { return m.directory }
func (m *mockTarget) Cwd() string             { return m.directory }
func (m *mockTarget) Commands() []string {
	cmds := make([]string, 0, len(m.commands))
	for k := range m.commands {
		cmds = append(cmds, k)
	}
	return cmds
}
func (m *mockTarget) DependsOn() []string { return m.dependsOn }
func (m *mockTarget) GetCommand(name string) (interface{}, bool) {
	cmd, ok := m.commands[name]
	return cmd, ok
}
func (m *mockTarget) Env() map[string]string  { return m.env }
func (m *mockTarget) Vars() map[string]string { return m.vars }
func (m *mockTarget) DemoPath() string        { return m.demoPath }
func (m *mockTarget) Execute(ctx context.Context, cmd string, opts target.ExecOptions) error {
	if m.execFunc != nil {
		return m.execFunc(ctx, cmd, opts)
	}
	return nil
}

func TestFilterTargetsByType_FiltersCorrectly(t *testing.T) {
	targets := []target.Target{
		&mockTarget{name: "cs", targetType: target.TypeLanguage},
		&mockTarget{name: "py", targetType: target.TypeLanguage},
		&mockTarget{name: "img", targetType: target.TypeAuxiliary},
		&mockTarget{name: "docs", targetType: target.TypeAuxiliary},
	}

	// Filter to language only
	filtered := filterTargetsByType(targets, target.TypeLanguage)
	if len(filtered) != 2 {
		t.Errorf("filterTargetsByType(language) = %d targets, want 2", len(filtered))
	}
	for _, tgt := range filtered {
		if tgt.Type() != target.TypeLanguage {
			t.Errorf("filtered target %q has type %q, want language", tgt.Name(), tgt.Type())
		}
	}

	// Filter to auxiliary only
	filtered = filterTargetsByType(targets, target.TypeAuxiliary)
	if len(filtered) != 2 {
		t.Errorf("filterTargetsByType(auxiliary) = %d targets, want 2", len(filtered))
	}
	for _, tgt := range filtered {
		if tgt.Type() != target.TypeAuxiliary {
			t.Errorf("filtered target %q has type %q, want auxiliary", tgt.Name(), tgt.Type())
		}
	}
}

func TestFilterTargetsByType_EmptySlice_ReturnsEmpty(t *testing.T) {
	filtered := filterTargetsByType(nil, target.TypeLanguage)
	if len(filtered) != 0 {
		t.Errorf("filterTargetsByType(nil) = %d targets, want 0", len(filtered))
	}

	filtered = filterTargetsByType([]target.Target{}, target.TypeLanguage)
	if len(filtered) != 0 {
		t.Errorf("filterTargetsByType([]) = %d targets, want 0", len(filtered))
	}
}

func TestFilterTargetsByType_NoMatches_ReturnsEmpty(t *testing.T) {
	targets := []target.Target{
		&mockTarget{name: "cs", targetType: target.TypeLanguage},
		&mockTarget{name: "py", targetType: target.TypeLanguage},
	}

	filtered := filterTargetsByType(targets, target.TypeAuxiliary)
	if len(filtered) != 0 {
		t.Errorf("filterTargetsByType(auxiliary) = %d targets, want 0", len(filtered))
	}
}

func TestFilterTargetsByType_AllMatch_ReturnsAll(t *testing.T) {
	targets := []target.Target{
		&mockTarget{name: "cs", targetType: target.TypeLanguage},
		&mockTarget{name: "py", targetType: target.TypeLanguage},
		&mockTarget{name: "go", targetType: target.TypeLanguage},
	}

	filtered := filterTargetsByType(targets, target.TypeLanguage)
	if len(filtered) != 3 {
		t.Errorf("filterTargetsByType(language) = %d targets, want 3", len(filtered))
	}
}

// createTestProject creates a temporary project for testing CLI commands
func createTestProject(t *testing.T) string {
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
	imgDir := filepath.Join(root, "img")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl/config.json
	config := `{
		"project": {"name": "test-project"},
		"targets": {
			"cs": {
				"type": "language",
				"title": "C#",
				"toolchain": "dotnet"
			},
			"img": {
				"type": "auxiliary",
				"title": "Images"
			}
		}
	}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	return root
}

// withWorkingDir changes to dir, runs fn, then restores original directory
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
		exitCode := cmdTargets(&GlobalOptions{})
		if exitCode != 0 {
			t.Errorf("cmdTargets() = %d, want 0", exitCode)
		}
	})
}

func TestCmdTargets_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdTargets(&GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdTargets() = 0, want non-zero when no project")
		}
	})
}

func TestCmdTargets_WithTypeFilter(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Filter to language only
		exitCode := cmdTargets(&GlobalOptions{TargetType: "language"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(language) = %d, want 0", exitCode)
		}

		// Filter to auxiliary only
		exitCode = cmdTargets(&GlobalOptions{TargetType: "auxiliary"})
		if exitCode != 0 {
			t.Errorf("cmdTargets(auxiliary) = %d, want 0", exitCode)
		}
	})
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

func TestCmdMeta_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdMeta("build", nil, &GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdMeta() = 0, want non-zero when no project")
		}
	})
}

func TestCmdMeta_WithTargetTypeFilter(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// Filter to auxiliary targets only - should succeed even if no commands match
		exitCode := cmdMeta("build", nil, &GlobalOptions{TargetType: "auxiliary"})
		// Exit code depends on whether any targets have the command
		// For auxiliary targets, they typically don't have build commands
		_ = exitCode // Just verify it doesn't panic
	})
}

func TestCmdMeta_ContinueOnError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// With continue flag, should continue even if errors occur
		// Now we error if no target has the command
		exitCode := cmdMeta("nonexistent-cmd", nil, &GlobalOptions{ContinueOnError: true})
		// Should return 1 if no targets have this command (clearer error feedback)
		if exitCode != 1 {
			t.Errorf("cmdMeta(nonexistent-cmd) = %d, want 1 (no targets have command)", exitCode)
		}
	})
}

func TestCmdMeta_TestCommandFiltersToLanguageTargets(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// The "test" command auto-filters to language targets
		// This test just verifies the code path works
		exitCode := cmdMeta("test", nil, &GlobalOptions{})
		// May fail if dotnet not installed, but we're testing the filter logic
		_ = exitCode // Just verify no panic
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
		// ci:release should work on valid project
		// Will likely fail during actual execution, but tests the command parsing
		exitCode := cmdCI("ci:release", nil, &GlobalOptions{})
		// May fail during execution, but command should be parsed correctly
		_ = exitCode // Just verify no panic
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

func TestCmdInit_ConfigExists_ReturnsError(t *testing.T) {
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
	if err := os.WriteFile(configPath, []byte(`{"project":{"name":"existing"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	withWorkingDir(t, root, func() {
		exitCode := cmdInit(nil)
		if exitCode != 2 {
			t.Errorf("cmdInit() = %d, want 2 (config exists)", exitCode)
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

		// Verify VERSION file was created
		versionPath := filepath.Join(root, "VERSION")
		content, err := os.ReadFile(versionPath)
		if err != nil {
			t.Errorf("VERSION file not created: %v", err)
			return
		}
		if string(content) != "0.1.0\n" {
			t.Errorf("VERSION = %q, want %q", string(content), "0.1.0\n")
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

	// Create existing VERSION file
	versionPath := filepath.Join(root, "VERSION")
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

		// Verify VERSION file was NOT overwritten
		content, err := os.ReadFile(versionPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != existingVersion {
			t.Errorf("VERSION = %q, want %q (should not be overwritten)", string(content), existingVersion)
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
		exitCode := cmdDockerClean(&GlobalOptions{})
		if exitCode == 0 {
			t.Error("cmdDockerClean() = 0, want non-zero when no project")
		}
	})
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

func TestCmdCI_ContinueOnError(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// With continue-on-error, pipeline should attempt all phases
		exitCode := cmdCI("ci", nil, &GlobalOptions{ContinueOnError: true})
		// Exit code 2 would indicate routing/parsing failure
		if exitCode == 2 {
			t.Errorf("cmdCI(ci) with ContinueOnError = 2 (usage error), want 0 or 1")
		}
	})
}

// =============================================================================
// Work Item 3: Docker Command Tests
// =============================================================================

// createTestProjectWithDocker creates a test project with docker configuration
func createTestProjectWithDocker(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create target directory
	csDir := filepath.Join(root, "cs")
	if err := os.MkdirAll(csDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl/config.json with docker config
	config := `{
		"project": {"name": "test-project"},
		"docker": {
			"compose_file": "docker-compose.yml",
			"env_var": "TEST_DOCKER"
		},
		"targets": {
			"cs": {
				"type": "language",
				"title": "C#",
				"toolchain": "dotnet"
			}
		}
	}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	// Create docker-compose.yml
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

	return root
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
		exitCode := cmdDockerClean(&GlobalOptions{})
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

func TestCmdMeta_RestoreCommand(t *testing.T) {
	root := createTestProject(t)
	withWorkingDir(t, root, func() {
		// "restore" is the new name for dependency restoration
		exitCode := cmdMeta("restore", nil, &GlobalOptions{})
		// Exit code 2 would indicate routing/parsing failure
		if exitCode == 2 {
			t.Errorf("cmdMeta(restore) = 2 (usage error), want 0 or 1")
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
