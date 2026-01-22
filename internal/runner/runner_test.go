package runner

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/testing/mocks"
)

// Note: getParallelWorkers tests use t.Setenv which modifies process-wide state.
// These tests cannot use t.Parallel() - they must run sequentially.
func TestGetParallelWorkers_Default(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "")

	workers := getParallelWorkers()
	expected := max(1, runtime.NumCPU())
	if workers != expected {
		t.Errorf("getParallelWorkers() = %d, want max(1, runtime.NumCPU()) = %d", workers, expected)
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
	// Note: t.Setenv modifies process state, so these tests cannot use t.Parallel()
	tests := []string{
		"invalid",
		"0",
		"-1",
		"257",
		" 4",  // leading whitespace - strconv.Atoi fails
		"4 ",  // trailing whitespace - strconv.Atoi fails
		"4.0", // float - strconv.Atoi fails
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

func TestGetParallelWorkers_LeadingZeros(t *testing.T) {
	// Leading zeros are valid for strconv.Atoi
	t.Setenv("STRUCTYL_PARALLEL", "007")

	workers := getParallelWorkers()
	if workers != 7 {
		t.Errorf("getParallelWorkers() = %d, want 7 (leading zeros accepted)", workers)
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
	tests := []struct {
		name     string
		errs     []error
		contains []string
	}{
		{
			name:     "two_errors",
			errs:     []error{os.ErrNotExist, os.ErrPermission},
			contains: []string{"not exist", "permission denied"},
		},
		{
			name:     "three_errors",
			errs:     []error{errors.New("error one"), errors.New("error two"), errors.New("error three")},
			contains: []string{"error one", "error two", "error three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			combined := combineErrors(tt.errs)
			if combined == nil {
				t.Fatal("combineErrors() = nil, want error")
			}

			msg := combined.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(msg, substr) {
					t.Errorf("message = %q, want to contain %q", msg, substr)
				}
			}
		})
	}
}

func TestCombineErrors_ErrorsAs(t *testing.T) {
	t.Parallel()
	// Create a custom error type to test errors.As extraction
	pathErr := &os.PathError{Op: "open", Path: "/test", Err: os.ErrNotExist}
	combined := combineErrors([]error{pathErr, os.ErrPermission})

	// errors.As should be able to extract the PathError type
	var extracted *os.PathError
	if !errors.As(combined, &extracted) {
		t.Error("errors.As(combined, *os.PathError) = false, want true")
	}
	if extracted.Path != "/test" {
		t.Errorf("extracted.Path = %q, want %q", extracted.Path, "/test")
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
	t.Parallel()
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
	t.Parallel()
	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.Run(ctx, "nonexistent", "build", RunOptions{})

	if err == nil {
		t.Error("Run() expected error for unknown target")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestRunAll_NoTargetsWithCommand_ReturnsNil(t *testing.T) {
	t.Parallel()
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

func TestRunTargets_EmptyTargetList_ReturnsNil(t *testing.T) {
	t.Parallel()
	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.RunTargets(ctx, []string{}, "build", RunOptions{})

	if err != nil {
		t.Errorf("RunTargets() error = %v, want nil", err)
	}
}

func TestRunSequential_ExecutesAll(t *testing.T) {
	t.Parallel()
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

	// Verify command string was passed correctly
	if target1.LastCommand() != "build" {
		t.Errorf("target1.LastCommand() = %q, want %q", target1.LastCommand(), "build")
	}
	if target2.LastCommand() != "build" {
		t.Errorf("target2.LastCommand() = %q, want %q", target2.LastCommand(), "build")
	}
	if target3.LastCommand() != "build" {
		t.Errorf("target3.LastCommand() = %q, want %q", target3.LastCommand(), "build")
	}
}

func TestRunSequential_StopsOnError(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestRunSequential_ContextDeadline(t *testing.T) {
	t.Parallel()
	// Use channel-based synchronization to ensure deadline triggers during execution
	started := make(chan struct{})

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			close(started) // Signal that execution has begun
			<-ctx.Done()   // Block until context deadline expires
			return ctx.Err()
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	// Use a short timeout that will expire while target1 is executing.
	// 100ms is short enough to test quickly but long enough to avoid flakiness on slow CI machines.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait for execution to start, then let the deadline expire
	go func() {
		<-started
		// The deadline will naturally expire after 100ms
	}()

	err := r.runSequential(ctx, targets, "build", RunOptions{})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("runSequential() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestRunParallel_ContextCancellation(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "2")

	// Use channel-based synchronization to ensure cancellation happens during execution.
	// target1 blocks until context is canceled; target2 should either:
	// - Not start (if cancellation propagates before scheduling), or
	// - Return context.Canceled (if already running)
	started := make(chan struct{})
	target1Done := make(chan struct{})

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			close(started) // Signal that execution has begun
			<-ctx.Done()   // Block until context is canceled
			close(target1Done)
			return ctx.Err()
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			// Check context before doing work
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

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after target1 starts executing
	go func() {
		<-started // Wait for target1 to start
		cancel()
	}()

	err := r.runParallel(ctx, targets, "build", RunOptions{})

	// Wait for target1 to complete to ensure clean goroutine shutdown
	<-target1Done

	// The error should indicate cancellation
	if err == nil {
		t.Fatal("runParallel() expected error when context is canceled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("runParallel() error = %v, want context.Canceled", err)
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

func TestCombineErrors_Unwrappable(t *testing.T) {
	t.Parallel()
	err1 := os.ErrNotExist
	err2 := os.ErrPermission

	combined := combineErrors([]error{err1, err2})

	// errors.Join produces an error that can be unwrapped with errors.Is
	if !errors.Is(combined, os.ErrNotExist) {
		t.Error("combined error should match os.ErrNotExist via errors.Is")
	}
	if !errors.Is(combined, os.ErrPermission) {
		t.Error("combined error should match os.ErrPermission via errors.Is")
	}
}

// =============================================================================
// Options Propagation Tests
// =============================================================================

func TestRunSequential_WithEnvOption_PassesToAllTargets(t *testing.T) {
	t.Parallel()
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

func TestRunSequential_WithVerbosityOption_PassesToAllTargets(t *testing.T) {
	t.Parallel()
	var receivedVerbosity []target.Verbosity

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			receivedVerbosity = append(receivedVerbosity, opts.Verbosity)
			return nil
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			receivedVerbosity = append(receivedVerbosity, opts.Verbosity)
			return nil
		})

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Verbosity: target.VerbosityVerbose})

	if err != nil {
		t.Errorf("runSequential() error = %v", err)
	}

	if len(receivedVerbosity) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(receivedVerbosity))
	}

	for i, received := range receivedVerbosity {
		if received != target.VerbosityVerbose {
			t.Errorf("target %d: Verbosity = %v, want VerbosityVerbose", i, received)
		}
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

	// All target names should be mentioned in the combined error
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
	t.Parallel()
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

	// Should have both error messages in the combined error
	if !strings.Contains(errMsg, "t1 failed") {
		t.Errorf("error should mention 't1 failed', got %q", errMsg)
	}
	if !strings.Contains(errMsg, "t2 failed") {
		t.Errorf("error should mention 't2 failed', got %q", errMsg)
	}
}

// =============================================================================
// SkipError Handling Tests
// =============================================================================

func TestRunSequential_SkipError_ContinuesWithoutFailure(t *testing.T) {
	t.Parallel()

	// SkipError should be logged but not treated as a failure
	skipErr := &target.SkipError{
		Target:  "t1",
		Command: "build",
		Reason:  target.SkipReasonDisabled,
	}

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return skipErr
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{})

	// Should return nil because SkipErrors are not failures
	if err != nil {
		t.Errorf("runSequential() error = %v, want nil (SkipError should not cause failure)", err)
	}

	// All targets should still execute
	if target1.ExecCount() != 1 {
		t.Errorf("target1.ExecCount() = %d, want 1", target1.ExecCount())
	}
	if target2.ExecCount() != 1 {
		t.Errorf("target2.ExecCount() = %d, want 1 (should continue after SkipError)", target2.ExecCount())
	}
	if target3.ExecCount() != 1 {
		t.Errorf("target3.ExecCount() = %d, want 1 (should continue after SkipError)", target3.ExecCount())
	}
}

func TestRunParallel_SkipError_ContinuesWithoutFailure(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	// SkipError should be logged but not treated as a failure in parallel mode
	skipErr := &target.SkipError{
		Target:  "t1",
		Command: "build",
		Reason:  target.SkipReasonCommandNotFound,
		Detail:  "golangci-lint",
	}

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return skipErr
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage)
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{})

	// Should return nil because SkipErrors are not failures
	if err != nil {
		t.Errorf("runParallel() error = %v, want nil (SkipError should not cause failure)", err)
	}

	// All targets should still execute
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

func TestRunSequential_MixedErrors_SkipErrorNotInCombined(t *testing.T) {
	t.Parallel()

	// When we have both SkipErrors and real errors, SkipErrors should not be
	// included in the combined error, but real errors should be.
	skipErr := &target.SkipError{
		Target:  "t1",
		Command: "build",
		Reason:  target.SkipReasonDisabled,
	}
	realErr := errors.New("t2 failed")

	target1 := mocks.NewTarget("t1").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return skipErr
		})
	target2 := mocks.NewTarget("t2").WithType(target.TypeLanguage).
		WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
			return realErr
		})
	target3 := mocks.NewTarget("t3").WithType(target.TypeLanguage)

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: true})

	// Should return error because t2 failed with a real error
	if err == nil {
		t.Fatal("runSequential() expected error for real failure")
	}

	errMsg := err.Error()

	// Real error should be in the message
	if !strings.Contains(errMsg, "t2 failed") {
		t.Errorf("error = %q, want to contain 't2 failed'", errMsg)
	}

	// SkipError reason should NOT be in the combined error message
	// (it was logged separately, not collected into errs slice)
	if strings.Contains(errMsg, "disabled") {
		t.Errorf("error = %q, should not contain SkipError details (disabled)", errMsg)
	}

	// All three targets should have executed with Continue: true
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

// =============================================================================
// filterByCommand Tests
// =============================================================================

func TestFilterByCommand_FiltersTargetsWithCommand(t *testing.T) {
	t.Parallel()

	// Create targets with different command sets
	target1 := mocks.NewTarget("t1").
		WithType(target.TypeLanguage).
		WithCommand("build", "cargo build").
		WithCommand("test", "cargo test")
	target2 := mocks.NewTarget("t2").
		WithType(target.TypeLanguage).
		WithCommand("build", "go build").
		WithCommand("lint", "golangci-lint run")
	target3 := mocks.NewTarget("t3").
		WithType(target.TypeAuxiliary) // No commands

	targets := []target.Target{target1, target2, target3}

	// Filter for "build" - should include t1 and t2
	filtered := filterByCommand(targets, "build")
	if len(filtered) != 2 {
		t.Errorf("filterByCommand(build) = %d targets, want 2", len(filtered))
	}
	for _, f := range filtered {
		if f.Name() == "t3" {
			t.Error("filterByCommand(build) should not include t3 (no build command)")
		}
	}
}

func TestFilterByCommand_EmptyResult(t *testing.T) {
	t.Parallel()

	target1 := mocks.NewTarget("t1").
		WithType(target.TypeLanguage).
		WithCommand("build", "cargo build")
	target2 := mocks.NewTarget("t2").
		WithType(target.TypeLanguage).
		WithCommand("test", "cargo test")

	targets := []target.Target{target1, target2}

	// Filter for "lint" - no targets have this command
	filtered := filterByCommand(targets, "lint")
	if len(filtered) != 0 {
		t.Errorf("filterByCommand(lint) = %d targets, want 0", len(filtered))
	}
}

func TestFilterByCommand_EmptyInput(t *testing.T) {
	t.Parallel()

	filtered := filterByCommand(nil, "build")
	if filtered != nil {
		t.Errorf("filterByCommand(nil) = %v, want nil", filtered)
	}

	filtered = filterByCommand([]target.Target{}, "build")
	if filtered != nil {
		t.Errorf("filterByCommand([]) = %v, want nil", filtered)
	}
}

func TestFilterByCommand_AllMatch(t *testing.T) {
	t.Parallel()

	target1 := mocks.NewTarget("t1").
		WithType(target.TypeLanguage).
		WithCommand("ci", "cargo build && cargo test")
	target2 := mocks.NewTarget("t2").
		WithType(target.TypeLanguage).
		WithCommand("ci", "go build && go test")

	targets := []target.Target{target1, target2}

	filtered := filterByCommand(targets, "ci")
	if len(filtered) != 2 {
		t.Errorf("filterByCommand(ci) = %d targets, want 2", len(filtered))
	}
}

// =============================================================================
// shouldContinueAfterError Tests
// =============================================================================

func TestShouldContinueAfterError_SkipError_ReturnsTrue(t *testing.T) {
	t.Parallel()

	skipErr := &target.SkipError{
		Target:  "test-target",
		Command: "lint",
		Reason:  target.SkipReasonCommandNotFound,
		Detail:  "golangci-lint",
	}

	if !shouldContinueAfterError(skipErr) {
		t.Error("shouldContinueAfterError(SkipError) = false, want true")
	}
}

func TestShouldContinueAfterError_RegularError_ReturnsFalse(t *testing.T) {
	t.Parallel()

	regularErr := errors.New("build failed")

	if shouldContinueAfterError(regularErr) {
		t.Error("shouldContinueAfterError(regularError) = true, want false")
	}
}

func TestShouldContinueAfterError_WrappedSkipError_ReturnsTrue(t *testing.T) {
	t.Parallel()

	skipErr := &target.SkipError{
		Target:  "test-target",
		Command: "build",
		Reason:  target.SkipReasonDisabled,
	}
	// Wrap the SkipError
	wrappedErr := errors.Join(skipErr, errors.New("additional context"))

	// The inner SkipError should still be detected
	if !shouldContinueAfterError(wrappedErr) {
		t.Error("shouldContinueAfterError(wrappedSkipError) = false, want true")
	}
}

func TestShouldContinueAfterError_Nil_ReturnsFalse(t *testing.T) {
	t.Parallel()

	// Passing nil should not panic and should return false
	// (nil is not a SkipError)
	if shouldContinueAfterError(nil) {
		t.Error("shouldContinueAfterError(nil) = true, want false")
	}
}
