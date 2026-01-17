package runner

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

// mockTarget implements target.Target for testing runner execution
type mockTarget struct {
	name       string
	targetType target.TargetType
	commands   map[string]bool
	execFunc   func(ctx context.Context, cmd string) error
	execCount  int32
	mu         sync.Mutex
	execOrder  []string
	directory  string // optional custom directory
}

func (m *mockTarget) Name() string            { return m.name }
func (m *mockTarget) Title() string           { return m.name }
func (m *mockTarget) Type() target.TargetType { return m.targetType }
func (m *mockTarget) Directory() string {
	if m.directory != "" {
		return m.directory
	}
	return m.name
}
func (m *mockTarget) Cwd() string             { return m.name }
func (m *mockTarget) Commands() []string {
	if m.commands == nil {
		return nil
	}
	cmds := make([]string, 0, len(m.commands))
	for k := range m.commands {
		cmds = append(cmds, k)
	}
	return cmds
}
func (m *mockTarget) DependsOn() []string     { return nil }
func (m *mockTarget) Env() map[string]string  { return nil }
func (m *mockTarget) Vars() map[string]string { return nil }
func (m *mockTarget) DemoPath() string        { return "" }
func (m *mockTarget) GetCommand(name string) (interface{}, bool) {
	if m.commands == nil {
		return "cmd", true
	}
	_, ok := m.commands[name]
	return "cmd", ok
}

func (m *mockTarget) Execute(ctx context.Context, cmd string, opts target.ExecOptions) error {
	atomic.AddInt32(&m.execCount, 1)
	m.mu.Lock()
	m.execOrder = append(m.execOrder, m.name)
	m.mu.Unlock()

	if m.execFunc != nil {
		return m.execFunc(ctx, cmd)
	}
	return nil
}

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
	err := combineErrors(nil)
	if err != nil {
		t.Errorf("combineErrors(nil) = %v, want nil", err)
	}
}

func TestCombineErrors_Single(t *testing.T) {
	original := os.ErrNotExist
	err := combineErrors([]error{original})
	if err != original {
		t.Errorf("combineErrors([1]) = %v, want original error", err)
	}
}

func TestCombineErrors_Multiple(t *testing.T) {
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
	target1 := &mockTarget{name: "t1", targetType: target.TypeLanguage}
	target2 := &mockTarget{name: "t2", targetType: target.TypeLanguage}
	target3 := &mockTarget{name: "t3", targetType: target.TypeLanguage}

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{})

	if err != nil {
		t.Errorf("runSequential() error = %v", err)
	}

	// All targets should have executed
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
	if target2.execCount != 1 {
		t.Errorf("target2.execCount = %d, want 1", target2.execCount)
	}
	if target3.execCount != 1 {
		t.Errorf("target3.execCount = %d, want 1", target3.execCount)
	}
}

func TestRunSequential_StopsOnError(t *testing.T) {
	testErr := errors.New("test error")

	target1 := &mockTarget{name: "t1", targetType: target.TypeLanguage}
	target2 := &mockTarget{
		name:       "t2",
		targetType: target.TypeLanguage,
		execFunc:   func(ctx context.Context, cmd string) error { return testErr },
	}
	target3 := &mockTarget{name: "t3", targetType: target.TypeLanguage}

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: false})

	if err == nil {
		t.Error("runSequential() expected error")
	}

	// First two should have executed, third should not
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
	if target2.execCount != 1 {
		t.Errorf("target2.execCount = %d, want 1", target2.execCount)
	}
	if target3.execCount != 0 {
		t.Errorf("target3.execCount = %d, want 0 (should not execute after error)", target3.execCount)
	}
}

func TestRunSequential_ContinueOnError(t *testing.T) {
	testErr := errors.New("test error")

	target1 := &mockTarget{name: "t1", targetType: target.TypeLanguage}
	target2 := &mockTarget{
		name:       "t2",
		targetType: target.TypeLanguage,
		execFunc:   func(ctx context.Context, cmd string) error { return testErr },
	}
	target3 := &mockTarget{name: "t3", targetType: target.TypeLanguage}

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runSequential(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Error("runSequential() expected combined error")
	}

	// All targets should have executed despite error in t2
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
	if target2.execCount != 1 {
		t.Errorf("target2.execCount = %d, want 1", target2.execCount)
	}
	if target3.execCount != 1 {
		t.Errorf("target3.execCount = %d, want 1 (should continue after error)", target3.execCount)
	}
}

func TestRunSequential_ContextCancellation(t *testing.T) {
	// Use channel-based synchronization instead of time.Sleep to avoid flakiness
	started := make(chan struct{})

	target1 := &mockTarget{
		name:       "t1",
		targetType: target.TypeLanguage,
		execFunc: func(ctx context.Context, cmd string) error {
			close(started) // Signal that execution has begun
			<-ctx.Done()   // Block until context is canceled
			return ctx.Err()
		},
	}
	target2 := &mockTarget{name: "t2", targetType: target.TypeLanguage}

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

	target1 := &mockTarget{name: "t1", targetType: target.TypeLanguage}
	target2 := &mockTarget{name: "t2", targetType: target.TypeLanguage}
	target3 := &mockTarget{name: "t3", targetType: target.TypeLanguage}

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{})

	if err != nil {
		t.Errorf("runParallel() error = %v", err)
	}

	// All targets should have executed
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
	if target2.execCount != 1 {
		t.Errorf("target2.execCount = %d, want 1", target2.execCount)
	}
	if target3.execCount != 1 {
		t.Errorf("target3.execCount = %d, want 1", target3.execCount)
	}
}

func TestRunParallel_CollectsErrors(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "4")

	testErr := errors.New("test error")

	target1 := &mockTarget{name: "t1", targetType: target.TypeLanguage}
	target2 := &mockTarget{
		name:       "t2",
		targetType: target.TypeLanguage,
		execFunc:   func(ctx context.Context, cmd string) error { return testErr },
	}
	target3 := &mockTarget{name: "t3", targetType: target.TypeLanguage}

	targets := []target.Target{target1, target2, target3}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: true})

	if err == nil {
		t.Error("runParallel() expected error")
	}

	// All should execute with Continue: true
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
	if target2.execCount != 1 {
		t.Errorf("target2.execCount = %d, want 1", target2.execCount)
	}
	if target3.execCount != 1 {
		t.Errorf("target3.execCount = %d, want 1", target3.execCount)
	}
}

func TestRunParallel_FailFastCancels(t *testing.T) {
	t.Setenv("STRUCTYL_PARALLEL", "1") // Serialize to make test deterministic

	testErr := errors.New("test error")

	// First target fails immediately
	target1 := &mockTarget{
		name:       "t1",
		targetType: target.TypeLanguage,
		execFunc:   func(ctx context.Context, cmd string) error { return testErr },
	}
	// Second target checks context
	target2 := &mockTarget{
		name:       "t2",
		targetType: target.TypeLanguage,
		execFunc: func(ctx context.Context, cmd string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}

	targets := []target.Target{target1, target2}

	registry, _ := createTestRegistry(t)
	r := New(registry)

	ctx := context.Background()
	err := r.runParallel(ctx, targets, "build", RunOptions{Continue: false})

	if err == nil {
		t.Error("runParallel() expected error")
	}

	// First should have executed
	if target1.execCount != 1 {
		t.Errorf("target1.execCount = %d, want 1", target1.execCount)
	}
}

func TestCombineErrors_MessageFormat(t *testing.T) {
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
