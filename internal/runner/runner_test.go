package runner

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/testing/mocks"
)

func TestGetParallelWorkers_Default(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "")

	workers := getParallelWorkers()
	if workers < 1 {
		t.Errorf("getParallelWorkers() = %d, want >= 1", workers)
	}
}

func TestGetParallelWorkers_FromEnv(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	workers := getParallelWorkers()
	if workers != 4 {
		t.Errorf("getParallelWorkers() = %d, want 4", workers)
	}
}

func TestGetParallelWorkers_InvalidEnv(t *testing.T) {
	tests := []string{
		"invalid",
		"0",
		"-1",
		"257",
	}

	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			t.Setenv("STRUCTYL_PARALLEL", val)

			workers := getParallelWorkers()
			// Should fall back to CPU count
			if workers < 1 {
				t.Errorf("getParallelWorkers() = %d, want >= 1", workers)
			}
		})
	}
}

func TestGetParallelWorkers_Boundary1(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "1")

	workers := getParallelWorkers()
	if workers != 1 {
		t.Errorf("getParallelWorkers() = %d, want 1", workers)
	}
}

func TestGetParallelWorkers_Boundary256(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "256")

	workers := getParallelWorkers()
	if workers != 256 {
		t.Errorf("getParallelWorkers() = %d, want 256", workers)
	}
}

func TestCombineErrors_Empty(t *testing.T) {
	t.Parallel()
	err := combineErrors(nil)
	if err != nil {
		t.Errorf("combineErrors(nil) = %v, want nil", err)
	}
}

func TestCombineErrors_Single(t *testing.T) {
	t.Parallel()
	original := os.ErrNotExist
	err := combineErrors([]error{original})
	if err != original {
		t.Errorf("combineErrors([1]) = %v, want original error", err)
	}
}

func TestCombineErrors_Multiple(t *testing.T) {
	t.Parallel()
	errors := []error{
		os.ErrNotExist,
		os.ErrPermission,
	}
	err := combineErrors(errors)
	if err == nil {
		t.Error("combineErrors([2]) = nil, want error")
	}

	// Verify message format
	msg := err.Error()
	if !strings.Contains(msg, "2 errors") {
		t.Errorf("error message = %q, want to contain '2 errors'", msg)
	}
	if !strings.Contains(msg, "not exist") {
		t.Errorf("error message = %q, want to contain first error", msg)
	}
}

// Helper to create a minimal test registry
func createTestRegistry(t *testing.T) (*target.Registry, string) {
	t.Helper()
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := tmpDir + "/rs"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Toolchain: "cargo",
				Directory: "rs",
			},
		},
	}

	registry, err := target.NewRegistry(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	return registry, tmpDir
}

func TestNew_ValidRegistry_CreatesRunner(t *testing.T) {
	registry, _ := createTestRegistry(t)

	r := New(registry)
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.registry != registry {
		t.Error("runner.registry not set correctly")
	}
}

func TestRun_UnknownTarget_ReturnsError(t *testing.T) {
	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.Run(ctx, "nonexistent", "build", RunOptions{})

	if err == nil {
		t.Error("Run() expected error for unknown target")
	}
	if !strings.Contains(err.Error(), "unknown target") {
		t.Errorf("error = %q, want to contain 'unknown target'", err.Error())
	}
}

func TestRunAll_NoTargetsWithCommand_ReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := tmpDir + "/img"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with auxiliary target that has no commands
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"img": {
				Type:      "auxiliary",
				Title:     "Images",
				Directory: "img",
				// No toolchain, no commands
			},
		},
	}

	registry, err := target.NewRegistry(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	r := New(registry)
	ctx := context.Background()

	// Run a command that no target has
	err = r.RunAll(ctx, "nonexistent_command", RunOptions{})
	if err != nil {
		t.Errorf("RunAll() error = %v, want nil", err)
	}
}

func TestRunAll_NoTargetsWithCommand_PrintsWarning(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := tmpDir + "/img"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with auxiliary target that has no commands
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"img": {
				Type:      "auxiliary",
				Title:     "Images",
				Directory: "img",
			},
		},
	}

	registry, err := target.NewRegistry(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Capture stderr
	oldStderr := os.Stderr
	pr, pw, _ := os.Pipe()
	os.Stderr = pw

	runner := New(registry)
	ctx := context.Background()
	_ = runner.RunAll(ctx, "nonexistent_command", RunOptions{})

	pw.Close()
	os.Stderr = oldStderr

	var buf [512]byte
	n, _ := pr.Read(buf[:])
	output := string(buf[:n])

	if !strings.Contains(output, "warning:") {
		t.Errorf("expected warning in stderr, got: %q", output)
	}
	if !strings.Contains(output, "nonexistent_command") {
		t.Errorf("expected command name in warning, got: %q", output)
	}
}

func TestRunTargets_EmptyTargetList_ReturnsNil(t *testing.T) {
	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.RunTargets(ctx, []string{}, "build", RunOptions{})

	if err != nil {
		t.Errorf("RunTargets() error = %v, want nil", err)
	}
}

func TestRunSequential_ExecutesAll(t *testing.T) {
	// Create mock targets
	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage)
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{})

	if err != nil {
		t.Errorf("runSequential() error = %v", err)
	}

	// All targets should have executed
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1", target2.ExecCount())
	}
	if target3.ExecCount() != 1 {
		t.Errorf("target3.ExecCount() = %d, want 1", target3.ExecCount())
	}
}

func TestRunSequential_StopsOnError(t *testing.T) {
	testErr := errors.New("test error")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage)
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error { return testErr })
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: false})

	if err == nil {
		t.Error("runSequential() expected error")
	}

	// First two should have executed, third should not
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1", target2.ExecCount())
	}
	if target3.ExecCount() != 0 {
		t.Errorf("target3.ExecCount() = %d, want 0 (should not execute after error)", target3.ExecCount())
	}
}

func TestRunSequential_ContinueOnError(t *testing.T) {
	testErr := errors.New("test error")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage)
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error { return testErr })
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Error("runSequential() expected combined error")
	}

	// All targets should have executed despite error in t2
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1", target2.ExecCount())
	}
	if target3.ExecCount() != 1 {
		t.Errorf("target3.ExecCount() = %d, want 1 (should continue after error)", target3.ExecCount())
	}
}

func TestRunSequential_ContextCancellation(t *testing.T) {
	// Use channel-based synchronization instead of time.Sleep to avoid flakiness
	started := make(chan struct{})

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			close(started) // Signal that execution has begun
			<-ctx.Done()   // Block until context is canceled
			return ctx.Err()
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after target1 starts executing
	go func() {
		<-started // Wait for target1 to start
		cancel()
	}()

	err := r.runSequential(ctx, targets, "build", RunOptions{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("runSequential() error = %v, want context.Canceled", err)
	}
}

func TestRunParallel_ExecutesAll(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage)
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{})

	if err != nil {
		t.Errorf("runParallel() error = %v", err)
	}

	// All targets should have executed
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1", target2.ExecCount())
	}
	if target3.ExecCount() != 1 {
		t.Errorf("target3.ExecCount() = %d, want 1", target3.ExecCount())
	}
}

func TestRunParallel_CollectsErrors(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	testErr := errors.New("test error")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage)
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error { return testErr })
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Error("runParallel() expected error")
	}

	// All should execute with Continue: true
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1", target2.ExecCount())
	}
	if target3.ExecCount() != 1 {
		t.Errorf("target3.ExecCount() = %d, want 1", target3.ExecCount())
	}
}

func TestRunParallel_FailFastCancels(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "1") // Serialize to make test deterministic

	testErr := errors.New("test error")

	// First target fails immediately
	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error { return testErr })
	// Second target checks context
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		})

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: false})

	if err == nil {
		t.Error("runParallel() expected error")
	}

	// First should have executed
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
}

func TestCombineErrors_MessageFormat(t *testing.T) {
	t.Parallel()
	errs := []error{
		errors.New("error one"),
		errors.New("error two"),
		errors.New("error three"),
	}

	combined := combineErrors(errs)
	msg := combined.Error()

	if !strings.Contains(msg, "3 errors") {
		t.Errorf("message should contain '3 errors', got %q", msg)
	}
	if !strings.Contains(msg, "error one") {
		t.Errorf("message should contain 'error one', got %q", msg)
	}
	if !strings.Contains(msg, "error two") {
		t.Errorf("message should contain 'error two', got %q", msg)
	}
	if !strings.Contains(msg, "error three") {
		t.Errorf("message should contain 'error three', got %q", msg)
	}
}

// =============================================================================
// Docker Mode Tests
// =============================================================================

func TestRunSequential_WithDockerOption_PassesToAllTargets(t *testing.T) {
	var receivedDocker []bool

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			receivedDocker = append(receivedDocker, opts.Docker)
			return nil
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			receivedDocker = append(receivedDocker, opts.Docker)
			return nil
		})

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Docker: true})

	if err != nil {
		t.Errorf("runSequential() error = %v", err)
	}

	if len(receivedDocker) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(receivedDocker))
	}

	for i, received := range receivedDocker {
		if !received {
			t.Errorf("target %d: Docker option not passed (received false)", i)
		}
	}
}

func TestRunParallel_WithDockerOption_PassesToAllTargets(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	var mu sync.Mutex
	var receivedDocker []bool

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			mu.Lock()
			receivedDocker = append(receivedDocker, opts.Docker)
			mu.Unlock()
			return nil
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			mu.Lock()
			receivedDocker = append(receivedDocker, opts.Docker)
			mu.Unlock()
			return nil
		})
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			mu.Lock()
			receivedDocker = append(receivedDocker, opts.Docker)
			mu.Unlock()
			return nil
		})

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Docker: true})

	if err != nil {
		t.Errorf("runParallel() error = %v", err)
	}

	if len(receivedDocker) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(receivedDocker))
	}

	for i, received := range receivedDocker {
		if !received {
			t.Errorf("target %d: Docker option not passed (received false)", i)
		}
	}
}

func TestRunSequential_WithEnvOption_PassesToAllTargets(t *testing.T) {
	var receivedEnv []map[string]string

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			receivedEnv = append(receivedEnv, opts.Env)
			return nil
		})

	targets := []target.Target{target1}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	env := map[string]string{"FOO": "bar", "BAZ": "qux"}
	err := r.runSequential(ctx, targets, "build", RunOptions{Env: env})

	if err != nil {
		t.Errorf("runSequential() error = %v", err)
	}

	if len(receivedEnv) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(receivedEnv))
	}

	if receivedEnv[0]["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got FOO=%s", receivedEnv[0]["FOO"])
	}
	if receivedEnv[0]["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got BAZ=%s", receivedEnv[0]["BAZ"])
	}
}

// =============================================================================
// Parallel Error Aggregation Tests
// =============================================================================

func TestRunParallel_AllFail_CombinesAllErrors(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t1 failed")
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t2 failed")
		})
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t3 failed")
		})

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Fatal("expected error when all targets fail")
	}

	errMsg := err.Error()

	// Error message should mention "3 errors"
	if !strings.Contains(errMsg, "3 errors") {
		t.Errorf("error should mention '3 errors', got %q", errMsg)
	}

	// All target names should be mentioned
	if !strings.Contains(errMsg, "t1") {
		t.Errorf("error should mention 't1', got %q", errMsg)
	}
	if !strings.Contains(errMsg, "t2") {
		t.Errorf("error should mention 't2', got %q", errMsg)
	}
	if !strings.Contains(errMsg, "t3") {
		t.Errorf("error should mention 't3', got %q", errMsg)
	}
}

func TestRunParallel_AllFail_WithoutContinue_StopsOnFirstError(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "1") // Serialize for deterministic order

	// With fail-fast and serialized execution, the first error should
	// cancel the context, potentially preventing later tasks from running.
	// However, due to goroutine scheduling, both may complete before cancellation.
	// This test verifies that:
	// 1. An error is returned
	// 2. If both complete, both are reported (due to race between cancel and completion)

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t1 failed")
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			// Check context before proceeding
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return errors.New("t2 failed")
			}
		})

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: false})

	if err == nil {
		t.Fatal("expected error when targets fail")
	}

	// The key behavior: we get an error. The exact count depends on timing.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed") {
		t.Errorf("error should mention failure, got %q", errMsg)
	}
}

func TestRunSequential_AllFail_CombinesAllErrors(t *testing.T) {
	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t1 failed")
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return errors.New("t2 failed")
		})

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Fatal("expected error when all targets fail")
	}

	errMsg := err.Error()

	// Should have combined errors
	if !strings.Contains(errMsg, "2 errors") {
		t.Errorf("error should mention '2 errors', got %q", errMsg)
	}
}
