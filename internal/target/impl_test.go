package target

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

func TestTargetType_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		typ  TargetType
		want bool
	}{
		{TypeLanguage, true},
		{TypeAuxiliary, true},
		{"language", true},
		{"auxiliary", true},
		{"", false},
		{"unknown", false},
		{"Language", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			if got := tt.typ.IsValid(); got != tt.want {
				t.Errorf("TargetType(%q).IsValid() = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestParseTargetType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		wantTyp TargetType
		wantOk  bool
	}{
		{"language", TypeLanguage, true},
		{"auxiliary", TypeAuxiliary, true},
		{"", "", false},
		{"unknown", "", false},
		{"Language", "", false}, // case sensitive
		{"LANGUAGE", "", false},
		{"lang", "", false},
		{"aux", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotTyp, gotOk := ParseTargetType(tt.input)
			if gotTyp != tt.wantTyp || gotOk != tt.wantOk {
				t.Errorf("ParseTargetType(%q) = (%q, %v), want (%q, %v)",
					tt.input, gotTyp, gotOk, tt.wantTyp, tt.wantOk)
			}
		})
	}
}

func TestValidTargetTypes(t *testing.T) {
	t.Parallel()
	types := ValidTargetTypes()

	if len(types) != 2 {
		t.Errorf("ValidTargetTypes() returned %d types, want 2", len(types))
	}

	// Check that both valid types are present
	hasLanguage := false
	hasAuxiliary := false
	for _, typ := range types {
		if typ == "language" {
			hasLanguage = true
		}
		if typ == "auxiliary" {
			hasAuxiliary = true
		}
	}

	if !hasLanguage {
		t.Error("ValidTargetTypes() missing 'language'")
	}
	if !hasAuxiliary {
		t.Error("ValidTargetTypes() missing 'auxiliary'")
	}
}

// readTestOutput reads a file and decodes it to a string, handling Windows UTF-16 encoding.
// On Windows, PowerShell's redirection operator creates UTF-16 LE files with BOM.
func readTestOutput(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Check for UTF-16 LE BOM (0xFF 0xFE)
	if len(content) >= 2 && content[0] == 0xFF && content[1] == 0xFE {
		// Decode UTF-16 LE to string
		content = content[2:] // skip BOM
		if len(content)%2 != 0 {
			content = content[:len(content)-1] // truncate odd byte
		}

		u16s := make([]uint16, len(content)/2)
		for i := range u16s {
			u16s[i] = binary.LittleEndian.Uint16(content[i*2:])
		}
		return string(utf16.Decode(u16s)), nil
	}

	return string(content), nil
}

func TestSkipError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *SkipError
		contains []string
	}{
		{
			name: "disabled",
			err: &SkipError{
				Target:  "rs",
				Command: "test",
				Reason:  SkipReasonDisabled,
			},
			contains: []string{"rs", "test", "disabled"},
		},
		{
			name: "command_not_found",
			err: &SkipError{
				Target:  "go",
				Command: "lint",
				Reason:  SkipReasonCommandNotFound,
				Detail:  "golangci-lint",
			},
			contains: []string{"go", "lint", "golangci-lint", "not found"},
		},
		{
			name: "script_not_found",
			err: &SkipError{
				Target:  "ts",
				Command: "check",
				Reason:  SkipReasonScriptNotFound,
				Detail:  "lint",
			},
			contains: []string{"ts", "check", "lint", "package.json"},
		},
		{
			name: "unknown_reason",
			err: &SkipError{
				Target:  "test",
				Command: "cmd",
				Reason:  SkipReason("custom"),
			},
			contains: []string{"test", "cmd", "custom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(msg, substr) {
					t.Errorf("Error() = %q, want to contain %q", msg, substr)
				}
			}
		})
	}
}

func TestIsSkipError_WrappedError(t *testing.T) {
	skipErr := &SkipError{
		Target:  "rs",
		Command: "test",
		Reason:  SkipReasonDisabled,
	}

	// Direct SkipError
	if !IsSkipError(skipErr) {
		t.Error("IsSkipError(SkipError) = false, want true")
	}

	// Wrapped SkipError
	wrapped := fmt.Errorf("wrapped: %w", skipErr)
	if !IsSkipError(wrapped) {
		t.Error("IsSkipError(wrapped SkipError) = false, want true")
	}

	// Non-SkipError
	nonSkipErr := fmt.Errorf("regular error")
	if IsSkipError(nonSkipErr) {
		t.Error("IsSkipError(regular error) = true, want false")
	}

	// Nil
	if IsSkipError(nil) {
		t.Error("IsSkipError(nil) = true, want false")
	}
}

func TestNewTarget(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "C#",
		Toolchain: "dotnet",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, err := NewTarget("cs", cfg, "/project", "", resolver)
	if err != nil {
		t.Fatalf("NewTarget() error = %v", err)
	}

	if target.Name() != "cs" {
		t.Errorf("Name() = %q, want %q", target.Name(), "cs")
	}
	if target.Title() != "C#" {
		t.Errorf("Title() = %q, want %q", target.Title(), "C#")
	}
	if target.Type() != TypeLanguage {
		t.Errorf("Type() = %q, want %q", target.Type(), TypeLanguage)
	}
}

func TestNewTarget_DefaultDirectory(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "C#",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("cs", cfg, "/project", "", resolver)

	if target.Directory() != "cs" {
		t.Errorf("Directory() = %q, want %q (default)", target.Directory(), "cs")
	}
	if target.Cwd() != "cs" {
		t.Errorf("Cwd() = %q, want %q (default)", target.Cwd(), "cs")
	}
}

func TestNewTarget_CustomDirectory(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "C#",
		Directory: "src/csharp",
		Cwd:       "src/csharp/main",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("cs", cfg, "/project", "", resolver)

	if target.Directory() != "src/csharp" {
		t.Errorf("Directory() = %q, want %q", target.Directory(), "src/csharp")
	}
	if target.Cwd() != "src/csharp/main" {
		t.Errorf("Cwd() = %q, want %q", target.Cwd(), "src/csharp/main")
	}
}

func TestNewTarget_InvalidType(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "invalid",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	_, err := NewTarget("test", cfg, "/project", "", resolver)
	if err == nil {
		t.Fatal("NewTarget() expected error for invalid type")
	}
	// Verify error message mentions the invalid type
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "type") {
		t.Errorf("error = %q, want to mention 'invalid' or 'type'", err.Error())
	}
}

func TestTarget_Commands(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Rust",
		Toolchain: "cargo",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "", resolver)

	commands := target.Commands()
	if len(commands) == 0 {
		t.Error("Commands() returned empty")
	}

	// Should have cargo commands
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		cmdMap[cmd] = true
	}

	expected := []string{"build", "test", "clean", "check:fix"}
	for _, cmd := range expected {
		if !cmdMap[cmd] {
			t.Errorf("Commands() missing %q", cmd)
		}
	}
}

func TestTarget_GetCommand(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Rust",
		Toolchain: "cargo",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "", resolver)

	cmd, ok := target.GetCommand("build")
	if !ok {
		t.Error("GetCommand(build) = not found")
	}
	if cmd != "cargo build" {
		t.Errorf("GetCommand(build) = %v, want 'cargo build'", cmd)
	}

	_, ok = target.GetCommand("nonexistent")
	if ok {
		t.Error("GetCommand(nonexistent) = found, want not found")
	}
}

func TestTarget_CommandOverrides(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Rust",
		Toolchain: "cargo",
		Commands: map[string]interface{}{
			"build": "cargo build --workspace",
			"demo":  "cargo run --example demo",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "", resolver)

	// Overridden command
	cmd, _ := target.GetCommand("build")
	if cmd != "cargo build --workspace" {
		t.Errorf("GetCommand(build) = %v, want 'cargo build --workspace'", cmd)
	}

	// Custom command
	cmd, _ = target.GetCommand("demo")
	if cmd != "cargo run --example demo" {
		t.Errorf("GetCommand(demo) = %v, want 'cargo run --example demo'", cmd)
	}

	// Inherited command
	cmd, _ = target.GetCommand("clean")
	if cmd != "cargo clean" {
		t.Errorf("GetCommand(clean) = %v, want 'cargo clean'", cmd)
	}
}

func TestTarget_DependsOn(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "auxiliary",
		Title:     "App",
		DependsOn: []string{"lib", "img"},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("app", cfg, "/project", "", resolver)

	deps := target.DependsOn()
	if len(deps) != 2 {
		t.Errorf("len(DependsOn()) = %d, want 2", len(deps))
	}
}

func TestTarget_DependsOn_Empty(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Lib",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("lib", cfg, "/project", "", resolver)

	deps := target.DependsOn()
	if deps == nil {
		t.Error("DependsOn() = nil, want empty slice")
	}
	if len(deps) != 0 {
		t.Errorf("len(DependsOn()) = %d, want 0", len(deps))
	}
}

func TestTarget_DependsOn_Immutable(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "auxiliary",
		Title:     "App",
		DependsOn: []string{"lib", "img"},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("app", cfg, "/project", "", resolver)

	// Get dependencies and modify the returned slice
	deps := target.DependsOn()
	deps[0] = "modified"

	// Verify the original is unchanged
	original := target.DependsOn()
	if original[0] != "lib" {
		t.Errorf("DependsOn() was mutated: got %q, want %q", original[0], "lib")
	}
}

func TestTarget_Env(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Python",
		Env: map[string]string{
			"PYTHONPATH": "src",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("py", cfg, "/project", "", resolver)

	env := target.Env()
	if env["PYTHONPATH"] != "src" {
		t.Errorf("Env()[PYTHONPATH] = %q, want %q", env["PYTHONPATH"], "src")
	}
}

func TestTarget_Vars(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "C#",
		Vars: map[string]string{
			"test_project": "MyProject.Tests",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("cs", cfg, "/project", "", resolver)

	vars := target.Vars()
	if vars["test_project"] != "MyProject.Tests" {
		t.Errorf("Vars()[test_project] = %q, want %q", vars["test_project"], "MyProject.Tests")
	}
}

func TestTarget_Env_ReturnsCopy(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Python",
		Env: map[string]string{
			"KEY": "original",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("py", cfg, "/project", "", resolver)

	// Get env and mutate it
	env := target.Env()
	env["KEY"] = "mutated"
	env["NEW_KEY"] = "new_value"

	// Get env again - should reflect original state
	env2 := target.Env()
	if env2["KEY"] != "original" {
		t.Errorf("Env() returned mutable map: KEY = %q, want %q", env2["KEY"], "original")
	}
	if _, exists := env2["NEW_KEY"]; exists {
		t.Error("Env() returned mutable map: NEW_KEY should not exist")
	}
}

func TestTarget_Vars_ReturnsCopy(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "C#",
		Vars: map[string]string{
			"key": "original",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("cs", cfg, "/project", "", resolver)

	// Get vars and mutate it
	vars := target.Vars()
	vars["key"] = "mutated"
	vars["new_key"] = "new_value"

	// Get vars again - should reflect original state
	vars2 := target.Vars()
	if vars2["key"] != "original" {
		t.Errorf("Vars() returned mutable map: key = %q, want %q", vars2["key"], "original")
	}
	if _, exists := vars2["new_key"]; exists {
		t.Error("Vars() returned mutable map: new_key should not exist")
	}
}

func TestTarget_DemoPath(t *testing.T) {
	cfg := config.TargetConfig{
		Type:     "language",
		Title:    "Rust",
		DemoPath: "examples/demo.rs",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "", resolver)

	if target.DemoPath() != "examples/demo.rs" {
		t.Errorf("DemoPath() = %q, want %q", target.DemoPath(), "examples/demo.rs")
	}
}

func TestTarget_DemoPath_Empty(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Rust",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "", resolver)

	if target.DemoPath() != "" {
		t.Errorf("DemoPath() = %q, want empty", target.DemoPath())
	}
}

func TestInterpolateVars_BuiltinVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Rust",
		Directory: "src/rust",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", "1.2.3", resolver)
	impl := target.(*targetImpl)

	// Test ${target} builtin
	result := impl.interpolateVars("echo ${target}")
	if result != "echo rs" {
		t.Errorf("interpolateVars(${target}) = %q, want %q", result, "echo rs")
	}

	// Test ${target_dir} builtin
	result = impl.interpolateVars("cd ${target_dir}")
	if result != "cd src/rust" {
		t.Errorf("interpolateVars(${target_dir}) = %q, want %q", result, "cd src/rust")
	}

	// Test ${root} builtin
	result = impl.interpolateVars("cd ${root}")
	if result != "cd /project" {
		t.Errorf("interpolateVars(${root}) = %q, want %q", result, "cd /project")
	}

	// Test ${version} builtin
	result = impl.interpolateVars("echo v${version}")
	if result != "echo v1.2.3" {
		t.Errorf("interpolateVars(${version}) = %q, want %q", result, "echo v1.2.3")
	}
}

func TestInterpolateVars_EmptyVersion(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	// Empty version should interpolate to empty string (preserves ${version} pattern would be confusing)
	result := impl.interpolateVars("echo version=${version}")
	if result != "echo version=" {
		t.Errorf("interpolateVars(${version}) with empty version = %q, want %q", result, "echo version=")
	}
}

func TestInterpolateVars_CustomVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "C#",
		Vars: map[string]string{
			"test_project": "MyProject.Tests",
			"config":       "Release",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("cs", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	result := impl.interpolateVars("dotnet test ${test_project} -c ${config}")
	expected := "dotnet test MyProject.Tests -c Release"
	if result != expected {
		t.Errorf("interpolateVars() = %q, want %q", result, expected)
	}
}

func TestInterpolateVars_EscapeSequences(t *testing.T) {
	// Consolidated test for all escape sequence behaviors.
	// $${var} should become ${var} (literal), allowing shell variables to pass through.
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic escape: $${var} → ${var}
		{"basic_escape", "echo $${HOME}", "echo ${HOME}"},
		{"escape_at_start", "$${VAR}", "${VAR}"},
		{"escape_with_prefix", "prefix $${VAR}", "prefix ${VAR}"},

		// Multiple escapes in one string
		{"multiple_escapes", "$${HOME}:$${PATH}:${target}", "${HOME}:${PATH}:test"},

		// Nested escape: $$${target} - the $${target} is detected and escaped
		{"nested_escape", "echo $$${target}", "echo $${target}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := impl.interpolateVars(tt.input)
			if result != tt.expected {
				t.Errorf("interpolateVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInterpolateVars_UnmatchedVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	// Unmatched variables should be preserved as-is
	result := impl.interpolateVars("echo ${unknown_var}")
	if result != "echo ${unknown_var}" {
		t.Errorf("interpolateVars(${unknown_var}) = %q, want preserved", result)
	}
}

func TestInterpolateVars_MixedVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: "mydir",
		Vars: map[string]string{
			"custom": "value",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	// Mix of builtin, custom, escaped, and unmatched
	result := impl.interpolateVars("${target} ${custom} $${literal} ${unknown}")
	expected := "test value ${literal} ${unknown}"
	if result != expected {
		t.Errorf("interpolateVars() = %q, want %q", result, expected)
	}
}

func TestResolveCommandVariant(t *testing.T) {
	// Use a target with explicit verbose/quiet variants to test the resolution logic
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			"test":         "go test ./...",
			"test:verbose": "go test -v ./...",
			"clean":        "go clean",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	tests := []struct {
		name      string
		cmd       string
		verbosity Verbosity
		expected  string
	}{
		{"default verbosity returns original", "test", VerbosityDefault, "test"},
		{"verbose with variant returns variant", "test", VerbosityVerbose, "test:verbose"},
		{"quiet without variant returns original", "clean", VerbosityQuiet, "clean"},
		{"verbose without variant returns original", "clean", VerbosityVerbose, "clean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := impl.resolveCommandVariant(tt.cmd, tt.verbosity)
			if result != tt.expected {
				t.Errorf("resolveCommandVariant(%q, %v) = %q, want %q", tt.cmd, tt.verbosity, result, tt.expected)
			}
		})
	}
}

func TestResolveCommandVariant_WithQuietVariant(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			"build":       "go build ./...",
			"build:quiet": "go build -q ./...",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)
	impl := target.(*targetImpl)

	// Should resolve to quiet variant when it exists
	result := impl.resolveCommandVariant("build", VerbosityQuiet)
	if result != "build:quiet" {
		t.Errorf("resolveCommandVariant(build, Quiet) = %q, want %q", result, "build:quiet")
	}

	// Default should return original
	result = impl.resolveCommandVariant("build", VerbosityDefault)
	if result != "build" {
		t.Errorf("resolveCommandVariant(build, Default) = %q, want %q", result, "build")
	}
}

func TestExecute_WithVerbosity_ResolvesVariant(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"test":         "echo normal > " + outputFile,
			"test:verbose": "echo verbose > " + outputFile,
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()

	// Execute with verbose - should use test:verbose
	err := target.Execute(ctx, "test", ExecOptions{Verbosity: VerbosityVerbose})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	content, err := readTestOutput(outputFile)
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}
	if !strings.Contains(content, "verbose") {
		t.Errorf("output = %q, want to contain 'verbose' (should have used test:verbose)", content)
	}
}

func TestExecute_UndefinedCommand_ReturnsError(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "undefined", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for undefined command")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("error = %q, want to contain 'not defined'", err.Error())
	}
}

func TestExecute_NilCommand_ReturnsSkipError(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			"skip": nil,
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "skip", ExecOptions{})

	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError for nil command", err)
	}
	skipErr, _ := err.(*SkipError)
	if skipErr.Reason != SkipReasonDisabled {
		t.Errorf("SkipError.Reason = %q, want %q", skipErr.Reason, SkipReasonDisabled)
	}
}

func TestExecute_StringCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"echo": "echo hello",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "echo", ExecOptions{})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
}

func TestExecute_StringCommand_Failure(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"fail": "exit 1",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "fail", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for failing command")
	}
}

func TestExecute_WithArgs_ExecutesSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	// This test verifies that Execute succeeds when args are provided.
	// The args are passed to the shell command (appended after the base command).
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"write": fmt.Sprintf("echo test > %s", outputFile),
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	// Execute with args - verifies execution completes without error
	err := target.Execute(ctx, "write", ExecOptions{
		Args: []string{"arg1", "arg2"},
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify the command executed (file was created)
	if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
		t.Error("output file was not created - command did not execute")
	}
}

func TestExecute_WithEnv_SetsEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "env_output.txt")

	// Use platform-specific command syntax
	var echoCmd string
	if runtime.GOOS == "windows" {
		// PowerShell: pipe to Out-File for reliable file writing
		echoCmd = fmt.Sprintf(`"$env:TARGET_VAR $env:OPTS_VAR" | Out-File -FilePath '%s' -Encoding utf8`, outputFile)
	} else {
		echoCmd = "echo $TARGET_VAR $OPTS_VAR > " + outputFile
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Env: map[string]string{
			"TARGET_VAR": "target_value",
		},
		Commands: map[string]interface{}{
			"env": echoCmd,
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "env", ExecOptions{
		Env: map[string]string{
			"OPTS_VAR": "opts_value",
		},
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}

	// Verify output file contains both env vars
	content, err := readTestOutput(outputFile)
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}

	if !strings.Contains(content, "target_value") {
		t.Errorf("output = %q, want to contain 'target_value'", content)
	}
	if !strings.Contains(content, "opts_value") {
		t.Errorf("output = %q, want to contain 'opts_value'", content)
	}
}

func TestExecute_CompositeCommand_ExecutesInOrder(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "composite.txt")

	// Use platform-specific commands for appending to file
	var firstCmd, secondCmd string
	if runtime.GOOS == "windows" {
		// PowerShell: pipe to Out-File for reliable file writing
		firstCmd = fmt.Sprintf(`'first' | Out-File -FilePath '%s' -Encoding utf8`, outputFile)
		secondCmd = fmt.Sprintf(`'second' | Out-File -FilePath '%s' -Append -Encoding utf8`, outputFile)
	} else {
		firstCmd = "echo first >> " + outputFile
		secondCmd = "echo second >> " + outputFile
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"first":  firstCmd,
			"second": secondCmd,
			"both":   []interface{}{"first", "second"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "both", ExecOptions{})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}

	output, err := readTestOutput(outputFile)
	if err != nil {
		t.Errorf("failed to read output: %v", err)
		return
	}

	// Verify order
	firstIdx := strings.Index(output, "first")
	secondIdx := strings.Index(output, "second")

	if firstIdx == -1 || secondIdx == -1 {
		t.Errorf("output = %q, want both 'first' and 'second'", output)
		return
	}

	if firstIdx >= secondIdx {
		t.Errorf("output = %q, want 'first' before 'second'", output)
	}
}

func TestExecute_InvalidCommandType_ReturnsError(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			"invalid": 123, // Invalid type
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "invalid", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for invalid command type")
	}
	if !strings.Contains(err.Error(), "invalid command") {
		t.Errorf("error = %q, want to contain 'invalid command'", err.Error())
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"sleep": "sleep 10",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := target.Execute(ctx, "sleep", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for canceled context")
	}
}

func TestExecute_ContextTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"sleep": "sleep 10",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	// Context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := target.Execute(ctx, "sleep", ExecOptions{})
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Execute() expected error for timed out context")
	}

	// Should have returned quickly due to timeout, not after 10 seconds
	if elapsed > 2*time.Second {
		t.Errorf("Execute() took %v, expected to abort quickly on timeout", elapsed)
	}
}

func TestExecute_CompositeCommandWithInvalidItem_ReturnsError(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			// []interface{} with non-string element should error
			"bad": []interface{}{123},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "bad", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for composite command with invalid item")
	}
	if !strings.Contains(err.Error(), "invalid command list item") {
		t.Errorf("error = %q, want to contain 'invalid command list item'", err.Error())
	}
}

func TestExtractCommandName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"golangci-lint run", "golangci-lint"},
		{"go vet ./...", "go"},
		{"echo hello world", "echo"},
		{"npm", "npm"},
		{"", ""},
		{"   spaced   command", "spaced"},
		// Quoted expressions (PowerShell string output) should return empty
		{`"hello" | Out-File test.txt`, ""},
		{`'hello' | Out-File test.txt`, ""},
		{`"$env:VAR" > file.txt`, ""},
	}

	for _, tc := range tests {
		result := extractCommandName(tc.input)
		if result != tc.expected {
			t.Errorf("extractCommandName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestIsCommandAvailable(t *testing.T) {
	// Test with a command that should always exist
	if runtime.GOOS == "windows" {
		if !isCommandAvailable("cmd") {
			t.Error("isCommandAvailable(cmd) = false, want true on Windows")
		}
	} else {
		if !isCommandAvailable("sh") {
			t.Error("isCommandAvailable(sh) = false, want true on Unix")
		}
	}

	// Test with a command that should not exist
	if isCommandAvailable("nonexistent-command-xyz-12345") {
		t.Error("isCommandAvailable(nonexistent-command) = true, want false")
	}

	// Test shell builtins - should always return true
	builtins := []string{"exit", "test", "echo", "cd", "pwd", "true", "false"}
	for _, builtin := range builtins {
		if !isCommandAvailable(builtin) {
			t.Errorf("isCommandAvailable(%q) = false, want true for shell builtin", builtin)
		}
	}
}

func TestExecute_UnavailableCommand_ReturnsSkipError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"lint": "nonexistent-tool-xyz-12345 run",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "lint", ExecOptions{})

	// Should return SkipError, not nil, to distinguish from success
	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError for unavailable command", err)
	}
	skipErr, _ := err.(*SkipError)
	if skipErr.Reason != SkipReasonCommandNotFound {
		t.Errorf("SkipError.Reason = %q, want %q", skipErr.Reason, SkipReasonCommandNotFound)
	}
	if skipErr.Detail != "nonexistent-tool-xyz-12345" {
		t.Errorf("SkipError.Detail = %q, want %q", skipErr.Detail, "nonexistent-tool-xyz-12345")
	}
}

func TestExecute_CompositeWithUnavailableCommand_ReturnsSkipError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"lint":  "nonexistent-tool-xyz-12345 run",
			"vet":   "echo vet-ran",
			"check": []interface{}{"lint", "vet"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "check", ExecOptions{})

	// Composite commands propagate errors (including skip errors)
	// The runner layer decides whether to continue, not the target layer
	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError for composite with unavailable sub-command", err)
	}
}

func TestExecute_MissingNpmScript_ReturnsSkipError(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json without lint script
	packageJSON := `{"name": "test", "scripts": {"build": "echo building"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"lint": "npm run lint",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("ts", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err = target.Execute(ctx, "lint", ExecOptions{})

	// Should return SkipError to distinguish from success
	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError for missing npm script", err)
	}
	skipErr, _ := err.(*SkipError)
	if skipErr.Reason != SkipReasonScriptNotFound {
		t.Errorf("SkipError.Reason = %q, want %q", skipErr.Reason, SkipReasonScriptNotFound)
	}
	if skipErr.Detail != "lint" {
		t.Errorf("SkipError.Detail = %q, want %q", skipErr.Detail, "lint")
	}
}

func TestExecute_BasicCommand_WritesOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"echo": fmt.Sprintf("echo script-ran > %s", outputFile),
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("ts", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "echo", ExecOptions{})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	content, readErr := readTestOutput(outputFile)
	if readErr != nil {
		t.Errorf("failed to read output file: %v", readErr)
		return
	}

	if !strings.Contains(content, "script-ran") {
		t.Errorf("output = %q, want to contain 'script-ran'", content)
	}
}

func TestExecute_CompositeWithMissingNpmScript_ReturnsSkipError(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json with only build script, not lint
	packageJSON := `{"name": "test", "scripts": {"build": "echo building"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"lint":  "npm run lint",   // Missing script
			"build": "echo build-ran", // Would execute
			"check": []interface{}{"lint", "build"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("ts", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err = target.Execute(ctx, "check", ExecOptions{})

	// Composite commands propagate errors (including skip errors)
	// The runner layer decides whether to continue, not the target layer
	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError for composite with missing npm script", err)
	}
}

func TestExecute_NpmBuiltinCommand_AlwaysAvailable(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json without install script
	packageJSON := `{"name": "test", "scripts": {}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			// npm install is a builtin command, not a script
			// It should not be blocked even though there's no "install" script
			"install": "echo mock-npm-install",
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("ts", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err = target.Execute(ctx, "install", ExecOptions{})

	// Should succeed - "echo mock-npm-install" is not a package manager command
	// and should execute normally
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}

func TestExecute_CompositeCommand_ContextCancellationBetweenCommands(t *testing.T) {
	// Verify that context cancellation is checked between commands in a list
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "second_ran.txt")

	// Use platform-specific command syntax
	var createMarker string
	if runtime.GOOS == "windows" {
		createMarker = fmt.Sprintf(`'marker' | Out-File -FilePath '%s' -Encoding utf8`, markerFile)
	} else {
		createMarker = "echo marker > " + markerFile
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"first":  "echo first",
			"second": createMarker,
			"both":   []interface{}{"first", "second"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	// Cancel context before execution
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := target.Execute(ctx, "both", ExecOptions{})

	// Should return context.Canceled
	if err == nil {
		t.Error("Execute() expected error for canceled context")
	}
	if err != context.Canceled {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}

	// Verify "second" command did NOT run
	if _, statErr := os.Stat(markerFile); statErr == nil {
		t.Error("second command ran but should have been skipped due to context cancellation")
	}
}

func TestExecute_CompositeCommand_FirstSubCommandDisabled_StopsExecution(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "second_ran.txt")

	// Use platform-specific command syntax
	var createMarker string
	if runtime.GOOS == "windows" {
		createMarker = fmt.Sprintf(`'marker' | Out-File -FilePath '%s' -Encoding utf8`, markerFile)
	} else {
		createMarker = "echo marker > " + markerFile
	}

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"first":  nil, // Disabled - returns SkipError
			"second": createMarker,
			"both":   []interface{}{"first", "second"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, "", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "both", ExecOptions{})

	// Should return SkipError from "first" command
	if !IsSkipError(err) {
		t.Errorf("Execute() error = %v, want SkipError", err)
	}

	// Verify "second" command did NOT run by checking marker file doesn't exist
	if _, statErr := os.Stat(markerFile); statErr == nil {
		t.Error("second command ran but should have been skipped due to first command's SkipError")
	}
}

func TestFilterMiseEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty_input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no_mise_vars",
			input:    []string{"PATH=/usr/bin", "HOME=/home/user", "GOPATH=/go"},
			expected: []string{"PATH=/usr/bin", "HOME=/home/user", "GOPATH=/go"},
		},
		{
			name:     "only_mise_vars",
			input:    []string{"__MISE_WATCH=1", "__MISE_ORIG_PATH=/usr/bin", "MISE_SHELL=bash"},
			expected: []string{},
		},
		{
			name:     "mixed_vars",
			input:    []string{"PATH=/usr/bin", "__MISE_WATCH=1", "HOME=/home/user", "MISE_SHELL=bash", "GOPATH=/go"},
			expected: []string{"PATH=/usr/bin", "HOME=/home/user", "GOPATH=/go"},
		},
		{
			name:     "mise_prefix_only",
			input:    []string{"__MISE_WATCH=1", "__MISE_ORIG_PATH=/usr/bin", "__MISE_DIFF=..."},
			expected: []string{},
		},
		{
			name:     "mise_shell_only",
			input:    []string{"MISE_SHELL=zsh", "MISE_SHELL=bash"},
			expected: []string{},
		},
		{
			name:     "similar_but_not_mise_vars",
			input:    []string{"MISE_CONFIG=/config", "MISE=/mise", "_MISE_WATCH=1", "MISE_=empty"},
			expected: []string{"MISE_CONFIG=/config", "MISE=/mise", "_MISE_WATCH=1", "MISE_=empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterMiseEnv(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("filterMiseEnv() returned %d items, want %d", len(result), len(tt.expected))
				t.Errorf("got: %v", result)
				t.Errorf("want: %v", tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("filterMiseEnv()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}
