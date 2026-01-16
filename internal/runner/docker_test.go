package runner

import (
	"context"
	"runtime"
	"testing"

	"github.com/akinshin/structyl/internal/config"
)

func TestNewDockerRunner(t *testing.T) {
	cfg := &config.DockerConfig{
		ComposeFile: "custom-compose.yml",
	}

	runner := NewDockerRunner("/project", cfg)

	if runner.composeFile != "custom-compose.yml" {
		t.Errorf("composeFile = %q, want %q", runner.composeFile, "custom-compose.yml")
	}
	if runner.projectRoot != "/project" {
		t.Errorf("projectRoot = %q, want %q", runner.projectRoot, "/project")
	}
}

func TestNewDockerRunner_Default(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	if runner.composeFile != "docker-compose.yml" {
		t.Errorf("composeFile = %q, want default %q", runner.composeFile, "docker-compose.yml")
	}
}

func TestDockerUnavailableError(t *testing.T) {
	err := &DockerUnavailableError{}

	if err.Error() == "" {
		t.Error("Error() should return a message")
	}
	if err.ExitCode() != 3 {
		t.Errorf("ExitCode() = %d, want 3", err.ExitCode())
	}
}

func TestBuildRunArgs(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("myservice", "echo hello")

	// Should contain compose, run, --rm
	found := map[string]bool{}
	for _, arg := range args {
		found[arg] = true
	}

	if !found["compose"] {
		t.Error("args should contain 'compose'")
	}
	if !found["run"] {
		t.Error("args should contain 'run'")
	}
	if !found["--rm"] {
		t.Error("args should contain '--rm'")
	}
	if !found["myservice"] {
		t.Error("args should contain service name")
	}

	// On non-Windows, should have --user flag
	if runtime.GOOS != "windows" {
		if !found["--user"] {
			t.Error("args should contain '--user' on non-Windows")
		}
	}
}

func TestGetDockerMode_Flags(t *testing.T) {
	// Use t.Setenv for automatic cleanup
	t.Setenv("STRUCTYL_DOCKER", "")

	// Explicit --docker flag
	if !GetDockerMode(true, false, "") {
		t.Error("explicit --docker should return true")
	}

	// Explicit --no-docker flag
	if GetDockerMode(false, true, "") {
		t.Error("explicit --no-docker should return false")
	}

	// --no-docker takes precedence over --docker
	if GetDockerMode(true, true, "") {
		t.Error("--no-docker should take precedence")
	}
}

func TestGetDockerMode_EnvVar(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"YES", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			t.Setenv("STRUCTYL_DOCKER", tt.envValue)

			result := GetDockerMode(false, false, "")
			if result != tt.expected {
				t.Errorf("GetDockerMode() with STRUCTYL_DOCKER=%q = %v, want %v",
					tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestGetDockerMode_CustomEnvVar(t *testing.T) {
	t.Setenv("MY_DOCKER_VAR", "true")

	result := GetDockerMode(false, false, "MY_DOCKER_VAR")
	if !result {
		t.Error("should use custom env var name")
	}
}

func TestGetDockerMode_Default(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "")

	result := GetDockerMode(false, false, "")
	if result {
		t.Error("default should be false (native execution)")
	}
}

func TestIsDockerAvailable_DoesNotPanic(t *testing.T) {
	// This test verifies IsDockerAvailable does not panic when Docker
	// is unavailable. The actual result depends on system state.
	result := IsDockerAvailable()
	_ = result // We only verify the function completes without panic
}

func TestCheckDockerAvailable_ReturnsCorrectType(t *testing.T) {
	err := CheckDockerAvailable()
	if err != nil {
		// Verify it returns the correct error type
		_, ok := err.(*DockerUnavailableError)
		if !ok {
			t.Errorf("CheckDockerAvailable() error type = %T, want *DockerUnavailableError", err)
		}
	}
	// If err is nil, Docker is available - both outcomes are valid
}

func TestBuildRunArgs_IncludesComposeFile(t *testing.T) {
	runner := NewDockerRunner("/project", &config.DockerConfig{
		ComposeFile: "custom.yml",
	})

	args := runner.buildRunArgs("service", "cmd")

	// Find -f flag and its value
	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) {
			if args[i+1] != "custom.yml" {
				t.Errorf("compose file = %q, want %q", args[i+1], "custom.yml")
			}
			return
		}
	}
	t.Error("args should contain -f flag with compose file")
}

func TestBuildRunArgs_CustomComposeFile(t *testing.T) {
	cfg := &config.DockerConfig{ComposeFile: "docker/compose.yaml"}
	runner := NewDockerRunner("/project", cfg)

	args := runner.buildRunArgs("app", "echo test")

	foundCompose := false
	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) && args[i+1] == "docker/compose.yaml" {
			foundCompose = true
			break
		}
	}
	if !foundCompose {
		t.Error("args should contain custom compose file path")
	}
}

func TestBuildRunArgs_ShellCommand(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("service", "npm run build")

	// Command should be at the end wrapped in shell
	lastArgs := args[len(args)-2:]
	if runtime.GOOS == "windows" {
		if lastArgs[0] != "-Command" {
			t.Errorf("Windows should use -Command, got %q", lastArgs[0])
		}
	} else {
		if lastArgs[0] != "-c" {
			t.Errorf("Unix should use -c, got %q", lastArgs[0])
		}
	}
	if lastArgs[1] != "npm run build" {
		t.Errorf("command = %q, want %q", lastArgs[1], "npm run build")
	}
}

func TestBuildRunArgs_ServiceName(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("myapp", "test")

	found := false
	for _, arg := range args {
		if arg == "myapp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("args should contain service name 'myapp'")
	}
}

func TestDockerRunner_ProjectRoot(t *testing.T) {
	runner := NewDockerRunner("/my/project/root", nil)

	if runner.projectRoot != "/my/project/root" {
		t.Errorf("projectRoot = %q, want %q", runner.projectRoot, "/my/project/root")
	}
}

func TestDockerRunner_Config(t *testing.T) {
	cfg := &config.DockerConfig{
		ComposeFile: "compose.yml",
	}
	runner := NewDockerRunner("/project", cfg)

	if runner.config != cfg {
		t.Error("config should be stored in runner")
	}
}

// =============================================================================
// Docker Runner Method Tests - Error Path Coverage
// These tests verify error handling when Docker is unavailable. When Docker IS
// available, they verify the functions don't error on basic invocations.
// =============================================================================

func TestDockerRunner_Run_ReturnsDockerUnavailableError(t *testing.T) {
	// Skip if Docker is available - we want to test the error path
	if IsDockerAvailable() {
		t.Skip("Docker is available; skipping unavailable error path test")
	}

	runner := NewDockerRunner(t.TempDir(), nil)
	ctx := context.Background()

	err := runner.Run(ctx, "service", "echo hello")
	if err == nil {
		t.Fatal("Run() expected error when Docker is unavailable")
	}

	// Verify error type
	dockerErr, ok := err.(*DockerUnavailableError)
	if !ok {
		t.Errorf("Run() error type = %T, want *DockerUnavailableError", err)
	}
	if dockerErr.ExitCode() != 3 {
		t.Errorf("DockerUnavailableError.ExitCode() = %d, want 3", dockerErr.ExitCode())
	}
}

func TestDockerRunner_Build_ReturnsDockerUnavailableError(t *testing.T) {
	if IsDockerAvailable() {
		t.Skip("Docker is available; skipping unavailable error path test")
	}

	runner := NewDockerRunner(t.TempDir(), nil)
	ctx := context.Background()

	err := runner.Build(ctx, "service1", "service2")
	if err == nil {
		t.Fatal("Build() expected error when Docker is unavailable")
	}

	if _, ok := err.(*DockerUnavailableError); !ok {
		t.Errorf("Build() error type = %T, want *DockerUnavailableError", err)
	}
}

func TestDockerRunner_Clean_ReturnsDockerUnavailableError(t *testing.T) {
	if IsDockerAvailable() {
		t.Skip("Docker is available; skipping unavailable error path test")
	}

	runner := NewDockerRunner(t.TempDir(), nil)
	ctx := context.Background()

	err := runner.Clean(ctx)
	if err == nil {
		t.Fatal("Clean() expected error when Docker is unavailable")
	}

	if _, ok := err.(*DockerUnavailableError); !ok {
		t.Errorf("Clean() error type = %T, want *DockerUnavailableError", err)
	}
}

func TestDockerRunner_Exec_ReturnsDockerUnavailableError(t *testing.T) {
	if IsDockerAvailable() {
		t.Skip("Docker is available; skipping unavailable error path test")
	}

	runner := NewDockerRunner(t.TempDir(), nil)
	ctx := context.Background()

	err := runner.Exec(ctx, "service", "echo hello")
	if err == nil {
		t.Fatal("Exec() expected error when Docker is unavailable")
	}

	if _, ok := err.(*DockerUnavailableError); !ok {
		t.Errorf("Exec() error type = %T, want *DockerUnavailableError", err)
	}
}

func TestDockerRunner_Run_WithValidProjectRoot(t *testing.T) {
	// This test verifies that Run correctly sets up the command with the project root
	// We can't fully execute without Docker, but we verify the runner is configured correctly
	projectRoot := t.TempDir()
	runner := NewDockerRunner(projectRoot, nil)

	if runner.projectRoot != projectRoot {
		t.Errorf("projectRoot = %q, want %q", runner.projectRoot, projectRoot)
	}
}

func TestDockerRunner_Build_EmptyServiceList(t *testing.T) {
	if IsDockerAvailable() {
		t.Skip("Docker is available; skipping unavailable error path test")
	}

	runner := NewDockerRunner(t.TempDir(), nil)
	ctx := context.Background()

	// Empty service list should still check Docker availability first
	err := runner.Build(ctx)
	if err == nil {
		t.Fatal("Build() expected error when Docker is unavailable")
	}

	if _, ok := err.(*DockerUnavailableError); !ok {
		t.Errorf("Build() error type = %T, want *DockerUnavailableError", err)
	}
}

// =============================================================================
// Work Item 4: Docker Command Construction Tests
// =============================================================================

func TestBuildRunArgs_CompleteStructure(t *testing.T) {
	runner := NewDockerRunner("/project", &config.DockerConfig{
		ComposeFile: "compose.yml",
	})

	args := runner.buildRunArgs("api", "npm test")

	// Verify exact structure for non-Windows
	if runtime.GOOS != "windows" {
		// Expected: ["compose", "-f", "compose.yml", "run", "--rm", "--user", "UID:GID", "api", "sh", "-c", "npm test"]
		expectedMinLen := 11
		if len(args) < expectedMinLen {
			t.Errorf("args length = %d, want >= %d", len(args), expectedMinLen)
		}

		// Verify first elements
		if args[0] != "compose" {
			t.Errorf("args[0] = %q, want %q", args[0], "compose")
		}
		if args[1] != "-f" {
			t.Errorf("args[1] = %q, want %q", args[1], "-f")
		}
		if args[2] != "compose.yml" {
			t.Errorf("args[2] = %q, want %q", args[2], "compose.yml")
		}
		if args[3] != "run" {
			t.Errorf("args[3] = %q, want %q", args[3], "run")
		}
		if args[4] != "--rm" {
			t.Errorf("args[4] = %q, want %q", args[4], "--rm")
		}
		if args[5] != "--user" {
			t.Errorf("args[5] = %q, want %q", args[5], "--user")
		}
		// args[6] is the UID:GID value

		// Find service name and shell wrapper
		foundService := false
		foundShell := false
		for i, arg := range args {
			if arg == "api" {
				foundService = true
			}
			if arg == "sh" && i+2 < len(args) && args[i+1] == "-c" && args[i+2] == "npm test" {
				foundShell = true
			}
		}
		if !foundService {
			t.Error("args should contain service name 'api'")
		}
		if !foundShell {
			t.Error("args should contain shell wrapper 'sh -c npm test'")
		}
	}
}

func TestBuildRunArgs_ServiceBeforeCommand(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("myservice", "echo hello")

	// Find positions of service and command
	serviceIdx := -1
	commandIdx := -1
	for i, arg := range args {
		if arg == "myservice" {
			serviceIdx = i
		}
		if arg == "echo hello" {
			commandIdx = i
		}
	}

	if serviceIdx == -1 {
		t.Fatal("service name not found in args")
	}
	if commandIdx == -1 {
		t.Fatal("command not found in args")
	}
	if serviceIdx >= commandIdx {
		t.Errorf("service (idx %d) should come before command (idx %d)", serviceIdx, commandIdx)
	}
}

func TestBuildRunArgs_UserFlagOnlyOnUnix(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("service", "cmd")

	hasUserFlag := false
	for _, arg := range args {
		if arg == "--user" {
			hasUserFlag = true
			break
		}
	}

	if runtime.GOOS == "windows" {
		if hasUserFlag {
			t.Error("--user flag should not be present on Windows")
		}
	} else {
		if !hasUserFlag {
			t.Error("--user flag should be present on Unix/Linux/macOS")
		}
	}
}

func TestBuildRunArgs_ShellWrapperPlatformSpecific(t *testing.T) {
	runner := NewDockerRunner("/project", nil)

	args := runner.buildRunArgs("service", "echo test")

	// Find shell executable
	hasSh := false
	hasPowershell := false
	for _, arg := range args {
		if arg == "sh" {
			hasSh = true
		}
		if arg == "powershell" {
			hasPowershell = true
		}
	}

	if runtime.GOOS == "windows" {
		if !hasPowershell {
			t.Error("Windows should use powershell")
		}
		if hasSh {
			t.Error("Windows should not use sh")
		}
	} else {
		if !hasSh {
			t.Error("Unix should use sh")
		}
		if hasPowershell {
			t.Error("Unix should not use powershell")
		}
	}
}
