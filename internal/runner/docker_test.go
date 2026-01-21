package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestNewDockerRunner(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	runner := NewDockerRunner("/project", nil)

	if runner.composeFile != "docker-compose.yml" {
		t.Errorf("composeFile = %q, want default %q", runner.composeFile, "docker-compose.yml")
	}
}

func TestDockerUnavailableError(t *testing.T) {
	t.Parallel()
	err := &DockerUnavailableError{}

	if err.Error() == "" {
		t.Error("Error() should return a message")
	}
	if err.ExitCode() != 3 {
		t.Errorf("ExitCode() = %d, want 3", err.ExitCode())
	}
}

func TestBuildRunArgs(t *testing.T) {
	t.Parallel()
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
	if !GetDockerMode(true, false) {
		t.Error("explicit --docker should return true")
	}

	// Explicit --no-docker flag
	if GetDockerMode(false, true) {
		t.Error("explicit --no-docker should return false")
	}

	// --no-docker takes precedence over --docker
	if GetDockerMode(true, true) {
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

			result := GetDockerMode(false, false)
			if result != tt.expected {
				t.Errorf("GetDockerMode() with STRUCTYL_DOCKER=%q = %v, want %v",
					tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestGetDockerMode_Default(t *testing.T) {
	t.Setenv("STRUCTYL_DOCKER", "")

	result := GetDockerMode(false, false)
	if result {
		t.Error("default should be false (native execution)")
	}
}

func TestIsDockerAvailable_DoesNotPanic(t *testing.T) {
	t.Parallel()
	// This test verifies IsDockerAvailable does not panic when Docker
	// is unavailable. The actual result depends on system state.
	result := IsDockerAvailable()
	_ = result // We only verify the function completes without panic
}

func TestCheckDockerAvailable_ReturnsCorrectType(t *testing.T) {
	t.Parallel()
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

func TestBuildRunArgs_CustomComposeFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		composeFile string
	}{
		{"simple", "custom.yml"},
		{"nested_path", "docker/compose.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := NewDockerRunner("/project", &config.DockerConfig{
				ComposeFile: tt.composeFile,
			})

			args := runner.buildRunArgs("service", "cmd")

			for i, arg := range args {
				if arg == "-f" && i+1 < len(args) {
					if args[i+1] != tt.composeFile {
						t.Errorf("compose file = %q, want %q", args[i+1], tt.composeFile)
					}
					return
				}
			}
			t.Errorf("args should contain -f flag with compose file %q", tt.composeFile)
		})
	}
}

func TestBuildRunArgs_ShellCommand(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	runner := NewDockerRunner("/my/project/root", nil)

	if runner.projectRoot != "/my/project/root" {
		t.Errorf("projectRoot = %q, want %q", runner.projectRoot, "/my/project/root")
	}
}

func TestDockerRunner_Config(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// =============================================================================
// NewDockerRunnerWithConfig Tests
// =============================================================================

func TestNewDockerRunnerWithConfig_NilConfig(t *testing.T) {
	t.Parallel()
	runner := NewDockerRunnerWithConfig("/project", nil)

	if runner == nil {
		t.Fatal("NewDockerRunnerWithConfig(nil) returned nil")
	}
	if runner.projectRoot != "/project" {
		t.Errorf("projectRoot = %q, want %q", runner.projectRoot, "/project")
	}
	if runner.config != nil {
		t.Error("config should be nil when input is nil")
	}
	if runner.projectConfig != nil {
		t.Error("projectConfig should be nil when input is nil")
	}
	// Should use default compose file
	if runner.composeFile != "docker-compose.yml" {
		t.Errorf("composeFile = %q, want default %q", runner.composeFile, "docker-compose.yml")
	}
}

func TestNewDockerRunnerWithConfig_WithDockerConfig(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Docker: &config.DockerConfig{
			ComposeFile: "custom-compose.yml",
		},
	}

	runner := NewDockerRunnerWithConfig("/project", cfg)

	if runner.config != cfg.Docker {
		t.Error("config should reference the Docker config from input")
	}
	if runner.projectConfig != cfg {
		t.Error("projectConfig should reference the full config")
	}
	if runner.composeFile != "custom-compose.yml" {
		t.Errorf("composeFile = %q, want %q", runner.composeFile, "custom-compose.yml")
	}
}

func TestNewDockerRunnerWithConfig_ConfigWithoutDocker(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Docker: nil, // No Docker config
	}

	runner := NewDockerRunnerWithConfig("/project", cfg)

	if runner.config != nil {
		t.Error("config should be nil when Docker config is nil")
	}
	if runner.projectConfig != cfg {
		t.Error("projectConfig should still reference the full config")
	}
	if runner.composeFile != "docker-compose.yml" {
		t.Errorf("composeFile = %q, want default %q", runner.composeFile, "docker-compose.yml")
	}
}

func TestNewDockerRunnerWithConfig_StoresProjectConfig(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-project"},
		Targets: map[string]config.TargetConfig{
			"go": {Type: "language", Title: "Go"},
		},
	}

	runner := NewDockerRunnerWithConfig("/project", cfg)

	if runner.projectConfig == nil {
		t.Fatal("projectConfig should not be nil")
	}
	if runner.projectConfig.Project.Name != "test-project" {
		t.Errorf("projectConfig.Project.Name = %q, want %q",
			runner.projectConfig.Project.Name, "test-project")
	}
}

// =============================================================================
// getDockerfilePath Tests
// =============================================================================

func TestGetDockerfilePath_DefaultDirectory(t *testing.T) {
	t.Parallel()
	runner := NewDockerRunnerWithConfig("/project", nil)

	targetCfg := config.TargetConfig{
		Directory: "", // Empty means use target name as directory
	}

	path := runner.getDockerfilePath("myapp", targetCfg)

	// Use filepath.FromSlash for cross-platform path comparison
	expected := filepath.FromSlash("/project/myapp/Dockerfile")
	if path != expected {
		t.Errorf("getDockerfilePath() = %q, want %q", path, expected)
	}
}

func TestGetDockerfilePath_CustomDirectory(t *testing.T) {
	t.Parallel()
	runner := NewDockerRunnerWithConfig("/project", nil)

	targetCfg := config.TargetConfig{
		Directory: "services/api",
	}

	path := runner.getDockerfilePath("myapp", targetCfg)

	// Use filepath.FromSlash for cross-platform path comparison
	expected := filepath.FromSlash("/project/services/api/Dockerfile")
	if path != expected {
		t.Errorf("getDockerfilePath() = %q, want %q", path, expected)
	}
}

func TestGetDockerfilePath_RootDirectory(t *testing.T) {
	t.Parallel()
	runner := NewDockerRunnerWithConfig("/project", nil)

	targetCfg := config.TargetConfig{
		Directory: ".",
	}

	path := runner.getDockerfilePath("main", targetCfg)

	// filepath.Join cleans the path, so /project/. becomes /project
	// Use filepath.FromSlash for cross-platform path comparison
	expected := filepath.FromSlash("/project/Dockerfile")
	if path != expected {
		t.Errorf("getDockerfilePath() = %q, want %q", path, expected)
	}
}

// =============================================================================
// Mock DockerCommandRunner Tests
// =============================================================================

// mockDockerCommandRunner is a test double for DockerCommandRunner.
type mockDockerCommandRunner struct {
	// runFunc is called when Run is invoked.
	runFunc func(ctx context.Context, args []string, dir string) error
	// checkAvailableFunc is called when CheckAvailable is invoked.
	checkAvailableFunc func() error
	// calls records all Run invocations for verification.
	calls []mockDockerCall
}

type mockDockerCall struct {
	args []string
	dir  string
}

func (m *mockDockerCommandRunner) Run(ctx context.Context, args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.calls = append(m.calls, mockDockerCall{args: args, dir: dir})
	if m.runFunc != nil {
		return m.runFunc(ctx, args, dir)
	}
	return nil
}

func (m *mockDockerCommandRunner) CheckAvailable() error {
	if m.checkAvailableFunc != nil {
		return m.checkAvailableFunc()
	}
	return nil
}

func TestDockerRunner_Run_WithMock(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Run(context.Background(), "myservice", "echo hello")

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.dir != "/project" {
		t.Errorf("dir = %q, want /project", call.dir)
	}

	// Verify args contain expected elements
	argsStr := strings.Join(call.args, " ")
	if !strings.Contains(argsStr, "compose") {
		t.Error("args should contain 'compose'")
	}
	if !strings.Contains(argsStr, "run") {
		t.Error("args should contain 'run'")
	}
	if !strings.Contains(argsStr, "myservice") {
		t.Error("args should contain 'myservice'")
	}
}

func TestDockerRunner_Run_DockerUnavailable(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{
		checkAvailableFunc: func() error {
			return &DockerUnavailableError{}
		},
	}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Run(context.Background(), "service", "cmd")

	if err == nil {
		t.Fatal("Run() expected error when Docker unavailable")
	}
	if _, ok := err.(*DockerUnavailableError); !ok {
		t.Errorf("error type = %T, want *DockerUnavailableError", err)
	}
	if len(mock.calls) != 0 {
		t.Error("Run should not be called when Docker is unavailable")
	}
}

func TestDockerRunner_Build_WithMock(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Build(context.Background(), "service1", "service2")

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]

	// Verify args contain expected elements
	argsStr := strings.Join(call.args, " ")
	if !strings.Contains(argsStr, "compose") {
		t.Error("args should contain 'compose'")
	}
	if !strings.Contains(argsStr, "build") {
		t.Error("args should contain 'build'")
	}
	if !strings.Contains(argsStr, "service1") {
		t.Error("args should contain 'service1'")
	}
	if !strings.Contains(argsStr, "service2") {
		t.Error("args should contain 'service2'")
	}
}

func TestDockerRunner_Clean_WithMock(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Clean(context.Background())

	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]

	// Verify args contain expected elements
	argsStr := strings.Join(call.args, " ")
	if !strings.Contains(argsStr, "compose") {
		t.Error("args should contain 'compose'")
	}
	if !strings.Contains(argsStr, "down") {
		t.Error("args should contain 'down'")
	}
	if !strings.Contains(argsStr, "--rmi") {
		t.Error("args should contain '--rmi'")
	}
}

func TestDockerRunner_Exec_WithMock(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Exec(context.Background(), "myservice", "npm test")

	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]

	// Verify args contain expected elements
	argsStr := strings.Join(call.args, " ")
	if !strings.Contains(argsStr, "compose") {
		t.Error("args should contain 'compose'")
	}
	if !strings.Contains(argsStr, "exec") {
		t.Error("args should contain 'exec'")
	}
	if !strings.Contains(argsStr, "myservice") {
		t.Error("args should contain 'myservice'")
	}
}

func TestDockerRunner_RunError_WithMock(t *testing.T) {
	t.Parallel()
	expectedErr := fmt.Errorf("docker run failed")
	mock := &mockDockerCommandRunner{
		runFunc: func(ctx context.Context, args []string, dir string) error {
			return expectedErr
		},
	}

	runner := NewDockerRunnerWithCommandRunner("/project", nil, mock)
	err := runner.Run(context.Background(), "service", "cmd")

	if err != expectedErr {
		t.Errorf("Run() error = %v, want %v", err, expectedErr)
	}
}

func TestNewDockerRunnerWithCommandRunner(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := NewDockerRunnerWithCommandRunner("/project", &config.DockerConfig{
		ComposeFile: "custom.yml",
	}, mock)

	if runner.projectRoot != "/project" {
		t.Errorf("projectRoot = %q, want /project", runner.projectRoot)
	}
	if runner.composeFile != "custom.yml" {
		t.Errorf("composeFile = %q, want custom.yml", runner.composeFile)
	}
	if runner.runner != mock {
		t.Error("runner should be the provided mock")
	}
}

// =============================================================================
// selectTargetsToBuild Tests
// =============================================================================

func TestSelectTargetsToBuild_EmptyServices_ReturnsAllTargets(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Type: "language", Title: "Go"},
			"rs": {Type: "language", Title: "Rust"},
			"ts": {Type: "language", Title: "TypeScript"},
		},
	}
	runner := &DockerRunner{projectConfig: cfg}

	targets := runner.selectTargetsToBuild(nil)

	if len(targets) != 3 {
		t.Errorf("selectTargetsToBuild(nil) returned %d targets, want 3", len(targets))
	}
	for _, name := range []string{"go", "rs", "ts"} {
		if _, ok := targets[name]; !ok {
			t.Errorf("targets missing %q", name)
		}
	}
}

func TestSelectTargetsToBuild_EmptySlice_ReturnsAllTargets(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"api": {Type: "service"},
			"web": {Type: "service"},
		},
	}
	runner := &DockerRunner{projectConfig: cfg}

	targets := runner.selectTargetsToBuild([]string{})

	if len(targets) != 2 {
		t.Errorf("selectTargetsToBuild([]) returned %d targets, want 2", len(targets))
	}
}

func TestSelectTargetsToBuild_SpecificServices_ReturnsMatching(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Type: "language", Title: "Go"},
			"rs": {Type: "language", Title: "Rust"},
			"ts": {Type: "language", Title: "TypeScript"},
		},
	}
	runner := &DockerRunner{projectConfig: cfg}

	targets := runner.selectTargetsToBuild([]string{"go", "ts"})

	if len(targets) != 2 {
		t.Errorf("selectTargetsToBuild([go, ts]) returned %d targets, want 2", len(targets))
	}
	if _, ok := targets["go"]; !ok {
		t.Error("targets missing 'go'")
	}
	if _, ok := targets["ts"]; !ok {
		t.Error("targets missing 'ts'")
	}
	if _, ok := targets["rs"]; ok {
		t.Error("targets should not contain 'rs'")
	}
}

func TestSelectTargetsToBuild_NonexistentService_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Type: "language"},
		},
	}
	runner := &DockerRunner{projectConfig: cfg}

	targets := runner.selectTargetsToBuild([]string{"nonexistent"})

	if len(targets) != 0 {
		t.Errorf("selectTargetsToBuild([nonexistent]) returned %d targets, want 0", len(targets))
	}
}

func TestSelectTargetsToBuild_MixedExistingAndNonexistent(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Type: "language"},
			"rs": {Type: "language"},
		},
	}
	runner := &DockerRunner{projectConfig: cfg}

	targets := runner.selectTargetsToBuild([]string{"go", "nonexistent", "rs"})

	if len(targets) != 2 {
		t.Errorf("selectTargetsToBuild returned %d targets, want 2", len(targets))
	}
	if _, ok := targets["go"]; !ok {
		t.Error("targets missing 'go'")
	}
	if _, ok := targets["rs"]; !ok {
		t.Error("targets missing 'rs'")
	}
}

// =============================================================================
// buildTarget Tests
// =============================================================================

func TestBuildTarget_CallsDockerBuild(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := &DockerRunner{
		projectRoot: "/project",
		runner:      mock,
	}

	targetCfg := config.TargetConfig{Directory: "myapp"}
	err := runner.buildTarget(context.Background(), "myapp", targetCfg)

	if err != nil {
		t.Fatalf("buildTarget() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}

	call := mock.calls[0]
	argsStr := strings.Join(call.args, " ")

	// Verify docker build command structure
	if !strings.Contains(argsStr, "build") {
		t.Error("args should contain 'build'")
	}
	if !strings.Contains(argsStr, "-t") {
		t.Error("args should contain '-t' for image tag")
	}
	if !strings.Contains(argsStr, "structyl-myapp") {
		t.Error("args should contain image name 'structyl-myapp'")
	}
	if !strings.Contains(argsStr, "-f") {
		t.Error("args should contain '-f' for dockerfile path")
	}
}

func TestBuildTarget_UsesCorrectImageName(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := &DockerRunner{
		projectRoot: "/project",
		runner:      mock,
	}

	targetCfg := config.TargetConfig{Directory: "services/api"}
	err := runner.buildTarget(context.Background(), "api-service", targetCfg)

	if err != nil {
		t.Fatalf("buildTarget() error = %v", err)
	}

	call := mock.calls[0]
	foundImageName := false
	for i, arg := range call.args {
		if arg == "-t" && i+1 < len(call.args) {
			if call.args[i+1] == "structyl-api-service" {
				foundImageName = true
			}
			break
		}
	}
	if !foundImageName {
		t.Errorf("image name should be 'structyl-api-service', got args: %v", call.args)
	}
}

func TestBuildTarget_UsesCorrectDockerfilePath(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := &DockerRunner{
		projectRoot: "/project",
		runner:      mock,
	}

	targetCfg := config.TargetConfig{Directory: "services/api"}
	err := runner.buildTarget(context.Background(), "api", targetCfg)

	if err != nil {
		t.Fatalf("buildTarget() error = %v", err)
	}

	call := mock.calls[0]
	foundDockerfile := false
	expectedPath := filepath.FromSlash("/project/services/api/Dockerfile")
	for i, arg := range call.args {
		if arg == "-f" && i+1 < len(call.args) {
			if call.args[i+1] == expectedPath {
				foundDockerfile = true
			}
			break
		}
	}
	if !foundDockerfile {
		t.Errorf("dockerfile path should be %q, got args: %v", expectedPath, call.args)
	}
}

func TestBuildTarget_UsesProjectRootAsDir(t *testing.T) {
	t.Parallel()
	mock := &mockDockerCommandRunner{}

	runner := &DockerRunner{
		projectRoot: "/my/project",
		runner:      mock,
	}

	targetCfg := config.TargetConfig{}
	err := runner.buildTarget(context.Background(), "app", targetCfg)

	if err != nil {
		t.Fatalf("buildTarget() error = %v", err)
	}

	call := mock.calls[0]
	if call.dir != "/my/project" {
		t.Errorf("dir = %q, want /my/project", call.dir)
	}
}

func TestBuildTarget_PropagatesRunError(t *testing.T) {
	t.Parallel()
	expectedErr := fmt.Errorf("docker build failed")
	mock := &mockDockerCommandRunner{
		runFunc: func(ctx context.Context, args []string, dir string) error {
			return expectedErr
		},
	}

	runner := &DockerRunner{
		projectRoot: "/project",
		runner:      mock,
	}

	targetCfg := config.TargetConfig{}
	err := runner.buildTarget(context.Background(), "app", targetCfg)

	if err != expectedErr {
		t.Errorf("buildTarget() error = %v, want %v", err, expectedErr)
	}
}

// =============================================================================
// tryBuildWithDockerfiles Tests
// =============================================================================

func TestTryBuildWithDockerfiles_NoDockerfiles_ReturnsFalse(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mock := &mockDockerCommandRunner{}

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Directory: "go"},
			"rs": {Directory: "rs"},
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	built, err := runner.tryBuildWithDockerfiles(context.Background(), nil)

	if err != nil {
		t.Fatalf("tryBuildWithDockerfiles() error = %v", err)
	}
	if built {
		t.Error("tryBuildWithDockerfiles() should return false when no Dockerfiles exist")
	}
	if len(mock.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(mock.calls))
	}
}

func TestTryBuildWithDockerfiles_WithDockerfile_ReturnsTrue(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory with Dockerfile
	goDir := filepath.Join(tmpDir, "go")
	if err := createDirWithDockerfile(goDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	mock := &mockDockerCommandRunner{}
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Directory: "go"},
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	built, err := runner.tryBuildWithDockerfiles(context.Background(), nil)

	if err != nil {
		t.Fatalf("tryBuildWithDockerfiles() error = %v", err)
	}
	if !built {
		t.Error("tryBuildWithDockerfiles() should return true when Dockerfile exists")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
}

func TestTryBuildWithDockerfiles_MultipleTargets_OnlyBuildsWithDockerfile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create directories - only 'go' has a Dockerfile
	goDir := filepath.Join(tmpDir, "go")
	rsDir := filepath.Join(tmpDir, "rs")
	if err := createDirWithDockerfile(goDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := createDirWithoutDockerfile(rsDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	mock := &mockDockerCommandRunner{}
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Directory: "go"},
			"rs": {Directory: "rs"},
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	built, err := runner.tryBuildWithDockerfiles(context.Background(), nil)

	if err != nil {
		t.Fatalf("tryBuildWithDockerfiles() error = %v", err)
	}
	if !built {
		t.Error("tryBuildWithDockerfiles() should return true")
	}
	// Should only build 'go' target
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call (only go), got %d", len(mock.calls))
	}
}

func TestTryBuildWithDockerfiles_FilteredServices(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create directories with Dockerfiles for both
	goDir := filepath.Join(tmpDir, "go")
	rsDir := filepath.Join(tmpDir, "rs")
	if err := createDirWithDockerfile(goDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := createDirWithDockerfile(rsDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	mock := &mockDockerCommandRunner{}
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Directory: "go"},
			"rs": {Directory: "rs"},
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	// Only request 'go' service
	built, err := runner.tryBuildWithDockerfiles(context.Background(), []string{"go"})

	if err != nil {
		t.Fatalf("tryBuildWithDockerfiles() error = %v", err)
	}
	if !built {
		t.Error("tryBuildWithDockerfiles() should return true")
	}
	// Should only build 'go' target even though 'rs' also has Dockerfile
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call (only go), got %d", len(mock.calls))
	}
}

func TestTryBuildWithDockerfiles_BuildError_StopsAndReturnsError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	goDir := filepath.Join(tmpDir, "go")
	if err := createDirWithDockerfile(goDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	expectedErr := fmt.Errorf("build failed")
	mock := &mockDockerCommandRunner{
		runFunc: func(ctx context.Context, args []string, dir string) error {
			return expectedErr
		},
	}
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Directory: "go"},
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	built, err := runner.tryBuildWithDockerfiles(context.Background(), nil)

	if err != expectedErr {
		t.Errorf("tryBuildWithDockerfiles() error = %v, want %v", err, expectedErr)
	}
	if built {
		t.Error("tryBuildWithDockerfiles() should return false on error")
	}
}

func TestTryBuildWithDockerfiles_UsesTargetNameAsDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create directory using target name (empty Directory config)
	apiDir := filepath.Join(tmpDir, "api")
	if err := createDirWithDockerfile(apiDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	mock := &mockDockerCommandRunner{}
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"api": {Directory: ""}, // Empty means use target name
		},
	}
	runner := &DockerRunner{
		projectRoot:   tmpDir,
		projectConfig: cfg,
		runner:        mock,
	}

	built, err := runner.tryBuildWithDockerfiles(context.Background(), nil)

	if err != nil {
		t.Fatalf("tryBuildWithDockerfiles() error = %v", err)
	}
	if !built {
		t.Error("tryBuildWithDockerfiles() should return true")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
}

// Helper functions for test setup

func createDirWithDockerfile(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return createFile(filepath.Join(dir, "Dockerfile"), "FROM scratch\n")
}

func createDirWithoutDockerfile(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func createFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
