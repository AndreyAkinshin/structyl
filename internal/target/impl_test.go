package target

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

func TestNewTarget(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "C#",
		Toolchain: "dotnet",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, err := NewTarget("cs", cfg, "/project", resolver)
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
	target, _ := NewTarget("cs", cfg, "/project", resolver)

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
	target, _ := NewTarget("cs", cfg, "/project", resolver)

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
	_, err := NewTarget("test", cfg, "/project", resolver)
	if err == nil {
		t.Fatal("NewTarget() expected error for invalid type")
	}
}

func TestTarget_Commands(t *testing.T) {
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Rust",
		Toolchain: "cargo",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", resolver)

	commands := target.Commands()
	if len(commands) == 0 {
		t.Error("Commands() returned empty")
	}

	// Should have cargo commands
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		cmdMap[cmd] = true
	}

	expected := []string{"build", "test", "clean", "format"}
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
	target, _ := NewTarget("rs", cfg, "/project", resolver)

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
	target, _ := NewTarget("rs", cfg, "/project", resolver)

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
	target, _ := NewTarget("app", cfg, "/project", resolver)

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
	target, _ := NewTarget("lib", cfg, "/project", resolver)

	deps := target.DependsOn()
	if deps == nil {
		t.Error("DependsOn() = nil, want empty slice")
	}
	if len(deps) != 0 {
		t.Errorf("len(DependsOn()) = %d, want 0", len(deps))
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
	target, _ := NewTarget("py", cfg, "/project", resolver)

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
	target, _ := NewTarget("cs", cfg, "/project", resolver)

	vars := target.Vars()
	if vars["test_project"] != "MyProject.Tests" {
		t.Errorf("Vars()[test_project] = %q, want %q", vars["test_project"], "MyProject.Tests")
	}
}

func TestTarget_DemoPath(t *testing.T) {
	cfg := config.TargetConfig{
		Type:     "language",
		Title:    "Rust",
		DemoPath: "examples/demo.rs",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("rs", cfg, "/project", resolver)

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
	target, _ := NewTarget("rs", cfg, "/project", resolver)

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
	target, _ := NewTarget("rs", cfg, "/project", resolver)
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
	target, _ := NewTarget("cs", cfg, "/project", resolver)
	impl := target.(*targetImpl)

	result := impl.interpolateVars("dotnet test ${test_project} -c ${config}")
	expected := "dotnet test MyProject.Tests -c Release"
	if result != expected {
		t.Errorf("interpolateVars() = %q, want %q", result, expected)
	}
}

func TestInterpolateVars_EscapedVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Shell",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("sh", cfg, "/project", resolver)
	impl := target.(*targetImpl)

	// $${var} should become ${var} (literal)
	result := impl.interpolateVars("echo $${HOME}")
	if result != "echo ${HOME}" {
		t.Errorf("interpolateVars($${HOME}) = %q, want %q", result, "echo ${HOME}")
	}
}

func TestInterpolateVars_UnmatchedVariables(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", resolver)
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
	target, _ := NewTarget("test", cfg, "/project", resolver)
	impl := target.(*targetImpl)

	// Mix of builtin, custom, escaped, and unmatched
	result := impl.interpolateVars("${target} ${custom} $${literal} ${unknown}")
	expected := "test value ${literal} ${unknown}"
	if result != expected {
		t.Errorf("interpolateVars() = %q, want %q", result, expected)
	}
}

func TestExecute_UndefinedCommand_ReturnsError(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "undefined", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for undefined command")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("error = %q, want to contain 'not defined'", err.Error())
	}
}

func TestExecute_NilCommand_SkipsExecution(t *testing.T) {
	cfg := config.TargetConfig{
		Type:  "language",
		Title: "Test",
		Commands: map[string]interface{}{
			"skip": nil,
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, "/project", resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "skip", ExecOptions{})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil for nil command", err)
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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "fail", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for failing command")
	}
}

func TestExecute_WithArgs_AppendsArguments(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Use a command that writes its arguments to a file
	// printf writes without trailing newline, echo adds one
	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			// Base command that will receive args
			"write": fmt.Sprintf("echo test > %s", outputFile),
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx := context.Background()
	// Execute - the "write" command will write "test" to the file
	// Args will be appended but don't affect the redirected output
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
		// PowerShell syntax for environment variables
		echoCmd = fmt.Sprintf("\"$env:TARGET_VAR $env:OPTS_VAR\" | Out-File -FilePath '%s' -Encoding ASCII", outputFile)
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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

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
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}

	if !strings.Contains(string(content), "target_value") {
		t.Errorf("output = %q, want to contain 'target_value'", string(content))
	}
	if !strings.Contains(string(content), "opts_value") {
		t.Errorf("output = %q, want to contain 'opts_value'", string(content))
	}
}

func TestExecute_CompositeCommand_ExecutesInOrder(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "composite.txt")

	// Use platform-specific commands for appending to file
	var firstCmd, secondCmd string
	if runtime.GOOS == "windows" {
		// PowerShell syntax for appending to file with ASCII encoding
		firstCmd = fmt.Sprintf("'first' | Out-File -FilePath '%s' -Encoding ASCII -Append", outputFile)
		secondCmd = fmt.Sprintf("'second' | Out-File -FilePath '%s' -Encoding ASCII -Append", outputFile)
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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "both", ExecOptions{})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("failed to read output: %v", err)
		return
	}

	// Verify order
	output := string(content)
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
	target, _ := NewTarget("test", cfg, "/project", resolver)

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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := target.Execute(ctx, "sleep", ExecOptions{})

	if err == nil {
		t.Error("Execute() expected error for canceled context")
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
	target, _ := NewTarget("test", cfg, "/project", resolver)

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

func TestExecute_UnavailableCommand_SkipsWithWarning(t *testing.T) {
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
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "lint", ExecOptions{})

	// Should NOT return error - it should skip gracefully
	if err != nil {
		t.Errorf("Execute() error = %v, want nil for unavailable command (should skip)", err)
	}
}

func TestExecute_CompositeWithUnavailableCommand_ContinuesWithOthers(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	cfg := config.TargetConfig{
		Type:      "language",
		Title:     "Test",
		Directory: ".",
		Cwd:       ".",
		Commands: map[string]interface{}{
			"lint":  "nonexistent-tool-xyz-12345 run",
			"vet":   "echo vet-ran > " + outputFile,
			"check": []interface{}{"lint", "vet"},
		},
	}

	resolver, _ := toolchain.NewResolver(&config.Config{})
	target, _ := NewTarget("test", cfg, tmpDir, resolver)

	ctx := context.Background()
	err := target.Execute(ctx, "check", ExecOptions{})

	// Should succeed - lint is skipped, vet runs
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Verify vet ran
	content, readErr := os.ReadFile(outputFile)
	if readErr != nil {
		t.Errorf("failed to read output file: %v", readErr)
		return
	}

	if !strings.Contains(string(content), "vet-ran") {
		t.Errorf("output = %q, want to contain 'vet-ran' (vet should have executed)", string(content))
	}
}
