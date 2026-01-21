package mise

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestGenerateMiseToml_Basic(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo", Title: "Rust"},
			"go": {Toolchain: "go", Title: "Go"},
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Check tools section
	if !strings.Contains(content, "[tools]") {
		t.Error("missing [tools] section")
	}
	if !strings.Contains(content, `rust = "stable"`) {
		t.Error("missing rust tool")
	}
	if !strings.Contains(content, `go = "1.22"`) {
		t.Error("missing go tool")
	}
	if !strings.Contains(content, `golangci-lint = "latest"`) {
		t.Error("missing golangci-lint tool")
	}

	// Check tasks section
	if !strings.Contains(content, `[tasks."setup:structyl"]`) {
		t.Error("missing setup:structyl task")
	}
	if !strings.Contains(content, `[tasks."ci:rs"]`) {
		t.Error("missing ci:rs task")
	}
	if !strings.Contains(content, `[tasks."ci:go"]`) {
		t.Error("missing ci:go task")
	}
	if !strings.Contains(content, `[tasks."ci"]`) {
		t.Error("missing main ci task")
	}

	// Check that check:go task is generated (unified static analysis command)
	if !strings.Contains(content, `[tasks."check:go"]`) {
		t.Error("missing check:go task")
	}
}

func TestGenerateMiseToml_Empty(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Should still have setup task
	if !strings.Contains(content, `[tasks."setup:structyl"]`) {
		t.Error("missing setup:structyl task")
	}
}

func TestGenerateMiseToml_UnsupportedToolchain(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"custom": {Toolchain: "make"}, // make has no mise mapping
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Should not have tools section (or empty tools)
	if strings.Contains(content, `make = `) {
		t.Error("should not contain make tool")
	}
}

func TestWriteMiseToml(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo"},
		},
	}

	// First write should create file
	created, err := WriteMiseToml(tmpDir, cfg, WriteIfMissing)
	if err != nil {
		t.Fatalf("WriteMiseToml() error = %v", err)
	}
	if !created {
		t.Error("WriteMiseToml() = false, want true (file should be created)")
	}

	// Second write with WriteIfMissing should not overwrite
	created, err = WriteMiseToml(tmpDir, cfg, WriteIfMissing)
	if err != nil {
		t.Fatalf("WriteMiseToml() error = %v", err)
	}
	if created {
		t.Error("WriteMiseToml() = true, want false (file exists)")
	}

	// Third write with WriteAlways should overwrite
	created, err = WriteMiseToml(tmpDir, cfg, WriteAlways)
	if err != nil {
		t.Fatalf("WriteMiseToml() error = %v", err)
	}
	if !created {
		t.Error("WriteMiseToml(WriteAlways) = false, want true")
	}

	// Verify file exists and has content
	content, err := os.ReadFile(filepath.Join(tmpDir, "mise.toml"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if !strings.Contains(string(content), "[tools]") {
		t.Error("file content missing [tools] section")
	}
}

func TestMiseTomlExists(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Should not exist initially
	if MiseTomlExists(tmpDir) {
		t.Error("MiseTomlExists() = true, want false")
	}

	// Create file
	err := os.WriteFile(filepath.Join(tmpDir, "mise.toml"), []byte("[tools]"), 0644)
	if err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Should exist now
	if !MiseTomlExists(tmpDir) {
		t.Error("MiseTomlExists() = false, want true")
	}
}

func TestGenerateMiseToml_TaskDependencies(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo", Directory: "rs"},
			"ts": {Toolchain: "npm", Directory: "ts"},
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Check that individual command tasks exist
	if !strings.Contains(content, `[tasks."build:rs"]`) {
		t.Error("missing build:rs task")
	}
	if !strings.Contains(content, `[tasks."test:rs"]`) {
		t.Error("missing test:rs task")
	}

	// Check that CI tasks exist and use sequential run format
	if !strings.Contains(content, `[tasks."ci:rs"]`) {
		t.Error("missing ci:rs task")
	}

	// Check that per-target CI tasks use sequential execution (run = [...])
	if !strings.Contains(content, `{ task = "clean:rs" }`) {
		t.Error("ci:rs should use sequential run format with clean:rs task")
	}
	if !strings.Contains(content, `{ task = "build:rs" }`) {
		t.Error("ci:rs should use sequential run format with build:rs task")
	}
	if !strings.Contains(content, `{ task = "test:rs" }`) {
		t.Error("ci:rs should use sequential run format with test:rs task")
	}

	// Check that main ci task depends on individual ci tasks (parallel execution across targets)
	if !strings.Contains(content, `depends = ["ci:rs", "ci:ts"]`) {
		t.Error("main ci task should depend on ci:rs and ci:ts")
	}

	// Check aggregate tasks
	if !strings.Contains(content, `[tasks."build"]`) {
		t.Error("missing aggregate build task")
	}
	if !strings.Contains(content, `[tasks."test"]`) {
		t.Error("missing aggregate test task")
	}
}

func TestGenerateMiseToml_WithEnvVars(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {
				Toolchain: "go",
				Directory: "go",
				Env: map[string]string{
					"CGO_ENABLED": "0",
					"GOOS":        "linux",
				},
			},
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Check that env vars are included in tasks
	if !strings.Contains(content, "env = {") {
		t.Error("missing env inline table")
	}
	if !strings.Contains(content, `CGO_ENABLED = "0"`) {
		t.Error("missing CGO_ENABLED env var")
	}
	if !strings.Contains(content, `GOOS = "linux"`) {
		t.Error("missing GOOS env var")
	}
}

func TestCapitalizeASCII(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "A"},
		{"hello", "Hello"},
		{"HELLO", "HELLO"},
		{"123", "123"},
		{"helloWorld", "HelloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalizeASCII(tt.input)
			if result != tt.expected {
				t.Errorf("capitalizeASCII(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWriteMiseToml_ReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows read-only directory permissions do not prevent file creation")
	}
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0444); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}
	t.Cleanup(func() {
		// Restore permissions for cleanup
		os.Chmod(readOnlyDir, 0755)
	})

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo"},
		},
	}

	_, err := WriteMiseToml(readOnlyDir, cfg, WriteAlways)
	if err == nil {
		t.Error("WriteMiseToml() expected error for read-only directory, got nil")
	}
}

func TestGenerateMiseToml_AllToolchains(t *testing.T) {
	t.Parallel()

	// Test that all supported toolchains generate valid mise.toml
	toolchains := []string{
		"cargo", "go", "npm", "dotnet", "gradle", "maven", "uv",
	}

	for _, tc := range toolchains {
		t.Run(tc, func(t *testing.T) {
			cfg := &config.Config{
				Targets: map[string]config.TargetConfig{
					"target": {Toolchain: tc, Title: "Test"},
				},
			}

			content, err := GenerateMiseToml(cfg)
			if err != nil {
				t.Fatalf("GenerateMiseToml() error = %v", err)
			}

			// All should generate a setup task
			if !strings.Contains(content, `[tasks."setup:structyl"]`) {
				t.Error("missing setup:structyl task")
			}

			// CI task should be generated
			if !strings.Contains(content, `[tasks."ci:target"]`) {
				t.Error("missing ci:target task")
			}
		})
	}
}

func TestGenerateMiseToml_TargetWithManyCommands(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {
				Toolchain: "cargo",
				Title:     "Rust",
				Directory: "rs",
				Commands: map[string]interface{}{
					"build":         "cargo build",
					"build:release": "cargo build --release",
					"test":          "cargo test",
					"check":         []interface{}{"cargo fmt --check", "cargo clippy"},
					"bench":         "cargo bench",
				},
			},
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Verify tasks are generated
	expectedTasks := []string{
		`[tasks."build:rs"]`,
		`[tasks."build:release:rs"]`,
		`[tasks."test:rs"]`,
		`[tasks."check:rs"]`,
		`[tasks."bench:rs"]`,
	}

	for _, task := range expectedTasks {
		if !strings.Contains(content, task) {
			t.Errorf("missing task: %s", task)
		}
	}
}

func TestGenerateMiseToml_DisabledCommands(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {
				Toolchain: "cargo",
				Title:     "Rust",
				Commands: map[string]interface{}{
					"build": "cargo build",
					"test":  nil, // Explicitly disabled
				},
			},
		},
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		t.Fatalf("GenerateMiseToml() error = %v", err)
	}

	// Build should exist
	if !strings.Contains(content, `[tasks."build:rs"]`) {
		t.Error("missing build:rs task")
	}

	// Test should NOT exist (disabled)
	if strings.Contains(content, `[tasks."test:rs"]`) {
		t.Error("test:rs task should not exist (disabled)")
	}
}

func TestWriteTasks_SequenceWithParallelSubtasks(t *testing.T) {
	t.Parallel()

	// Create tasks with RunSequence containing Tasks (parallel subtasks)
	tasks := map[string]MiseTask{
		"deploy": {
			Description: "Deploy to all environments",
			RunSequence: []RunStep{
				{Run: "echo 'Starting deploy'"},                  // Shell command
				{Task: "build"},                                  // Single task
				{Tasks: []string{"deploy:staging", "deploy:qa"}}, // Parallel tasks
				{Run: "echo 'Deploy complete'"},                  // Final shell command
			},
		},
	}

	var b strings.Builder
	writeTasks(&b, tasks)
	content := b.String()

	// Verify run array structure
	if !strings.Contains(content, `[tasks."deploy"]`) {
		t.Error("missing deploy task")
	}
	if !strings.Contains(content, "run = [") {
		t.Error("missing run array")
	}

	// Verify shell command in sequence
	if !strings.Contains(content, `"echo 'Starting deploy'"`) {
		t.Error("missing shell command in sequence")
	}

	// Verify single task reference
	if !strings.Contains(content, `{ task = "build" }`) {
		t.Error("missing single task reference in sequence")
	}

	// Verify parallel tasks array - this exercises the RunStep.Tasks branch
	if !strings.Contains(content, `{ tasks = ["deploy:staging", "deploy:qa"] }`) {
		t.Error("missing parallel tasks array in sequence")
	}

	// Verify final shell command
	if !strings.Contains(content, `"echo 'Deploy complete'"`) {
		t.Error("missing final shell command in sequence")
	}
}

func TestGetCommandsToGenerate_NilToolchains_ReturnsFallback(t *testing.T) {
	t.Parallel()
	// When loaded toolchains is nil, getCommandsToGenerate should return fallback defaults
	commands := getCommandsToGenerate(nil)

	if len(commands) == 0 {
		t.Fatal("getCommandsToGenerate(nil) returned empty slice, want fallback defaults")
	}

	// Verify fallback includes expected standard commands
	expected := map[string]bool{
		"clean":         true,
		"restore":       true,
		"build":         true,
		"build:release": true,
		"test":          true,
		"check":         true,
		"check:fix":     true,
		"bench":         true,
		"demo":          true,
		"doc":           true,
		"pack":          true,
	}

	for _, cmd := range commands {
		if expected[cmd] {
			delete(expected, cmd)
		}
	}

	if len(expected) > 0 {
		t.Errorf("missing expected commands in fallback: %v", expected)
	}
}

func TestGetAggregateCommands_NilToolchains_ReturnsFallback(t *testing.T) {
	t.Parallel()
	// When loaded toolchains is nil, getAggregateCommands should return fallback defaults
	commands := getAggregateCommands(nil)

	if len(commands) == 0 {
		t.Fatal("getAggregateCommands(nil) returned empty slice, want fallback defaults")
	}

	// Verify fallback includes expected aggregate commands
	expected := map[string]bool{
		"clean":         true,
		"restore":       true,
		"build":         true,
		"build:release": true,
		"test":          true,
		"check":         true,
		"check:fix":     true,
	}

	for _, cmd := range commands {
		if expected[cmd] {
			delete(expected, cmd)
		}
	}

	if len(expected) > 0 {
		t.Errorf("missing expected commands in fallback: %v", expected)
	}
}

func TestGetCIPipeline_NilToolchains_ReturnsFallback(t *testing.T) {
	t.Parallel()
	// When loaded toolchains is nil, getCIPipeline should return fallback defaults
	pipeline := getCIPipeline(nil)

	if len(pipeline) == 0 {
		t.Fatal("getCIPipeline(nil) returned empty slice, want fallback defaults")
	}

	// Verify fallback includes expected CI pipeline commands
	expected := []string{"clean", "restore", "check", "build", "test"}

	if len(pipeline) != len(expected) {
		t.Errorf("getCIPipeline(nil) length = %d, want %d", len(pipeline), len(expected))
	}

	for i, cmd := range expected {
		if i >= len(pipeline) || pipeline[i] != cmd {
			t.Errorf("getCIPipeline(nil)[%d] = %v, want %q", i, pipeline, cmd)
			break
		}
	}
}

func TestGetResolvedCommands_WithExtends(t *testing.T) {
	t.Parallel()

	// Test that custom toolchains extending built-in toolchains properly inherit
	// and override commands. This exercises the tcCfg.Extends branch in
	// getResolvedCommandsForTargetWithToolchains (lines 274-287).
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"custom": {
				Toolchain: "my-cargo",
				Title:     "Custom Rust",
			},
		},
		Toolchains: map[string]config.ToolchainConfig{
			"my-cargo": {
				Extends: "cargo", // Extends built-in cargo toolchain
				Commands: map[string]interface{}{
					"build": "cargo build --release --all-features", // Override
					"lint":  "custom-lint-tool",                     // Add new command
				},
			},
		},
	}

	targetCfg := cfg.Targets["custom"]
	commands := getResolvedCommandsForTargetWithToolchains(targetCfg, cfg, nil)

	// Verify inherited command from base cargo toolchain
	if testCmd, ok := commands["test"]; !ok || testCmd != "cargo test" {
		t.Errorf("expected test command from base cargo, got %v", commands["test"])
	}

	// Verify overridden command
	if buildCmd, ok := commands["build"]; !ok || buildCmd != "cargo build --release --all-features" {
		t.Errorf("expected overridden build command, got %v", commands["build"])
	}

	// Verify added command
	if lintCmd, ok := commands["lint"]; !ok || lintCmd != "custom-lint-tool" {
		t.Errorf("expected added lint command, got %v", commands["lint"])
	}

	// Verify other inherited commands from cargo
	if cleanCmd, ok := commands["clean"]; !ok || cleanCmd != "cargo clean" {
		t.Errorf("expected clean command from base cargo, got %v", commands["clean"])
	}
}
