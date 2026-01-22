package mise

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/output"
)

func TestBuildRunArgs_WithoutSkipDeps(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		args     []string
		expected []string
	}{
		{
			name:     "simple task",
			task:     "build",
			args:     nil,
			expected: []string{"run", "build"},
		},
		{
			name:     "task with args",
			task:     "test",
			args:     []string{"--verbose"},
			expected: []string{"run", "test", "--verbose"},
		},
		{
			name:     "namespaced task",
			task:     "test:go",
			args:     nil,
			expected: []string{"run", "test:go"},
		},
		{
			name:     "namespaced task with multiple args",
			task:     "build:rs",
			args:     []string{"--release", "--target", "x86_64"},
			expected: []string{"run", "build:rs", "--release", "--target", "x86_64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRunArgs(tt.task, tt.args)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildRunArgs(%q, %v) = %v, want %v", tt.task, tt.args, got, tt.expected)
			}
		})
	}
}

// Tests for Executor basic methods

func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
	}{
		{"with current dir", "."},
		{"with absolute path", "/tmp/project"},
		{"with relative path", "../other-project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExecutor(tt.projectRoot)

			if e == nil {
				t.Fatal("NewExecutor() returned nil")
			}
			if e.projectRoot != tt.projectRoot {
				t.Errorf("NewExecutor().projectRoot = %q, want %q", e.projectRoot, tt.projectRoot)
			}
			if e.verbose {
				t.Error("NewExecutor().verbose should be false by default")
			}
		})
	}
}

func TestExecutor_SetVerbose(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"enable verbose", true},
		{"disable verbose", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExecutor(".")
			e.SetVerbose(tt.verbose)

			if e.verbose != tt.verbose {
				t.Errorf("SetVerbose(%v): verbose = %v, want %v", tt.verbose, e.verbose, tt.verbose)
			}
		})
	}
}

func TestExecutor_SetVerbose_Toggle(t *testing.T) {
	e := NewExecutor(".")

	// Initially false
	if e.verbose {
		t.Error("initial verbose should be false")
	}

	// Enable
	e.SetVerbose(true)
	if !e.verbose {
		t.Error("verbose should be true after SetVerbose(true)")
	}

	// Disable
	e.SetVerbose(false)
	if e.verbose {
		t.Error("verbose should be false after SetVerbose(false)")
	}
}

// =============================================================================
// Dependency Resolution Tests
// =============================================================================

func TestResolveTaskDependencies_SingleTask_NoDeps(t *testing.T) {
	tasks := []MiseTaskMeta{
		{Name: "build", Depends: nil},
	}

	result, err := resolveTaskDependenciesFromSlice(tasks, "build")
	if err != nil {
		t.Fatalf("resolveTaskDependenciesFromSlice() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].Name != "build" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "build")
	}
}

func TestResolveTaskDependencies_LinearChain(t *testing.T) {
	// A -> B -> C (C depends on B, B depends on A)
	tasks := []MiseTaskMeta{
		{Name: "A", Depends: nil},
		{Name: "B", Depends: []string{"A"}},
		{Name: "C", Depends: []string{"B"}},
	}

	result, err := resolveTaskDependenciesFromSlice(tasks, "C")
	if err != nil {
		t.Fatalf("resolveTaskDependenciesFromSlice() error = %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	// Order should be: A, B, C (dependencies first)
	names := make([]string, len(result))
	for i, r := range result {
		names[i] = r.Name
	}
	expected := []string{"A", "B", "C"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("task order = %v, want %v", names, expected)
	}
}

func TestResolveTaskDependencies_DiamondDependency(t *testing.T) {
	// Diamond: D depends on B and C, both B and C depend on A
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	tasks := []MiseTaskMeta{
		{Name: "A", Depends: nil},
		{Name: "B", Depends: []string{"A"}},
		{Name: "C", Depends: []string{"A"}},
		{Name: "D", Depends: []string{"B", "C"}},
	}

	result, err := resolveTaskDependenciesFromSlice(tasks, "D")
	if err != nil {
		t.Fatalf("resolveTaskDependenciesFromSlice() error = %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("len(result) = %d, want 4", len(result))
	}

	// A must come before B and C; B and C must come before D
	positions := make(map[string]int)
	for i, r := range result {
		positions[r.Name] = i
	}

	if positions["A"] >= positions["B"] {
		t.Errorf("A (pos %d) should come before B (pos %d)", positions["A"], positions["B"])
	}
	if positions["A"] >= positions["C"] {
		t.Errorf("A (pos %d) should come before C (pos %d)", positions["A"], positions["C"])
	}
	if positions["B"] >= positions["D"] {
		t.Errorf("B (pos %d) should come before D (pos %d)", positions["B"], positions["D"])
	}
	if positions["C"] >= positions["D"] {
		t.Errorf("C (pos %d) should come before D (pos %d)", positions["C"], positions["D"])
	}
}

func TestResolveTaskDependencies_CircularDependency(t *testing.T) {
	// A -> B -> C -> A (cycle)
	tasks := []MiseTaskMeta{
		{Name: "A", Depends: []string{"C"}},
		{Name: "B", Depends: []string{"A"}},
		{Name: "C", Depends: []string{"B"}},
	}

	_, err := resolveTaskDependenciesFromSlice(tasks, "A")
	if err == nil {
		t.Error("resolveTaskDependenciesFromSlice() expected error for circular dependency")
	}
	if err != nil && !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error = %q, want to contain 'circular dependency'", err.Error())
	}
}

func TestResolveTaskDependencies_SelfDependency(t *testing.T) {
	// A depends on itself
	tasks := []MiseTaskMeta{
		{Name: "A", Depends: []string{"A"}},
	}

	_, err := resolveTaskDependenciesFromSlice(tasks, "A")
	if err == nil {
		t.Error("resolveTaskDependenciesFromSlice() expected error for self dependency")
	}
	if err != nil && !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error = %q, want to contain 'circular dependency'", err.Error())
	}
}

func TestResolveTaskDependencies_TaskNotFound(t *testing.T) {
	tasks := []MiseTaskMeta{
		{Name: "build", Depends: nil},
	}

	_, err := resolveTaskDependenciesFromSlice(tasks, "nonexistent")
	if err == nil {
		t.Error("resolveTaskDependenciesFromSlice() expected error for missing task")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestResolveTaskDependencies_MissingDependency(t *testing.T) {
	tasks := []MiseTaskMeta{
		{Name: "build", Depends: []string{"missing-dep"}},
	}

	_, err := resolveTaskDependenciesFromSlice(tasks, "build")
	if err == nil {
		t.Error("resolveTaskDependenciesFromSlice() expected error for missing dependency")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestResolveTaskDependencies_EmptyTaskList(t *testing.T) {
	tasks := []MiseTaskMeta{}

	_, err := resolveTaskDependenciesFromSlice(tasks, "build")
	if err == nil {
		t.Error("resolveTaskDependenciesFromSlice() expected error for empty task list")
	}
}

func TestResolveTaskDependencies_MultipleDependencies(t *testing.T) {
	// D depends on A, B, and C (all independent)
	tasks := []MiseTaskMeta{
		{Name: "A", Depends: nil},
		{Name: "B", Depends: nil},
		{Name: "C", Depends: nil},
		{Name: "D", Depends: []string{"A", "B", "C"}},
	}

	result, err := resolveTaskDependenciesFromSlice(tasks, "D")
	if err != nil {
		t.Fatalf("resolveTaskDependenciesFromSlice() error = %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("len(result) = %d, want 4", len(result))
	}

	// D must be last
	if result[len(result)-1].Name != "D" {
		t.Errorf("last task = %q, want D", result[len(result)-1].Name)
	}
}

// =============================================================================
// taskExistsInJSON Tests
// =============================================================================

func TestTaskExistsInJSON_Found(t *testing.T) {
	json := []byte(`[{"name":"build"},{"name":"test"},{"name":"clean"}]`)

	if !taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = false for existing task 'build'")
	}
	if !taskExistsInJSON(json, "test") {
		t.Error("taskExistsInJSON() = false for existing task 'test'")
	}
	if !taskExistsInJSON(json, "clean") {
		t.Error("taskExistsInJSON() = false for existing task 'clean'")
	}
}

func TestTaskExistsInJSON_NotFound(t *testing.T) {
	json := []byte(`[{"name":"build"},{"name":"test"}]`)

	if taskExistsInJSON(json, "nonexistent") {
		t.Error("taskExistsInJSON() = true for nonexistent task")
	}
}

func TestTaskExistsInJSON_EmptyArray(t *testing.T) {
	json := []byte(`[]`)

	if taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = true for empty array")
	}
}

func TestTaskExistsInJSON_InvalidJSON(t *testing.T) {
	json := []byte(`not valid json`)

	if taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = true for invalid JSON")
	}
}

func TestTaskExistsInJSON_NamespacedTask(t *testing.T) {
	json := []byte(`[{"name":"build:go"},{"name":"build:rs"},{"name":"test:go"}]`)

	if !taskExistsInJSON(json, "build:go") {
		t.Error("taskExistsInJSON() = false for namespaced task 'build:go'")
	}
	if taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = true for partial match 'build'")
	}
}

func TestTaskExistsInJSON_WithExtraFields(t *testing.T) {
	// Mise JSON output includes extra fields; verify they don't break parsing
	json := []byte(`[{"name":"build","source":"mise.toml","depends":["restore"],"description":"Build the project"}]`)

	if !taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = false with extra fields")
	}
}

func TestTaskExistsInJSON_SpacesInJSON(t *testing.T) {
	// JSON with formatting/spaces
	json := []byte(`[
  { "name": "build" },
  { "name": "test" }
]`)

	if !taskExistsInJSON(json, "build") {
		t.Error("taskExistsInJSON() = false with formatted JSON")
	}
	if !taskExistsInJSON(json, "test") {
		t.Error("taskExistsInJSON() = false with formatted JSON")
	}
}

// =============================================================================
// Mock CommandRunner Tests
// =============================================================================

// mockCommandRunner is a test double for CommandRunner.
type mockCommandRunner struct {
	// runFunc is called when Run is invoked.
	runFunc func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error
	// outputFunc is called when Output is invoked.
	outputFunc func(ctx context.Context, name string, args []string, dir string) ([]byte, error)
	// calls records all method invocations for verification.
	calls []mockCall
}

type mockCall struct {
	method string
	name   string
	args   []string
	dir    string
}

func (m *mockCommandRunner) Run(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.calls = append(m.calls, mockCall{method: "Run", name: name, args: args, dir: dir})
	if m.runFunc != nil {
		return m.runFunc(ctx, name, args, dir, env, stdin, stdout, stderr)
	}
	return nil
}

func (m *mockCommandRunner) Output(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{method: "Output", name: name, args: args, dir: dir})
	if m.outputFunc != nil {
		return m.outputFunc(ctx, name, args, dir)
	}
	return nil, nil
}

func TestExecutor_RunTask_WithMock(t *testing.T) {
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	err := e.RunTask(context.Background(), "build", []string{"--release"})

	if err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.method != "Run" {
		t.Errorf("method = %q, want Run", call.method)
	}
	if call.name != "mise" {
		t.Errorf("name = %q, want mise", call.name)
	}
	expectedArgs := []string{"run", "build", "--release"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
	if call.dir != "/project" {
		t.Errorf("dir = %q, want /project", call.dir)
	}
}

func TestExecutor_RunTask_Error(t *testing.T) {
	expectedErr := errors.New("command failed")
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return expectedErr
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	err := e.RunTask(context.Background(), "build", nil)

	if err != expectedErr {
		t.Errorf("RunTask() error = %v, want %v", err, expectedErr)
	}
}

func TestExecutor_TaskExists_WithMock(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		task     string
		expected bool
	}{
		{
			name:     "task exists",
			json:     `[{"name":"build"},{"name":"test"}]`,
			task:     "build",
			expected: true,
		},
		{
			name:     "task not found",
			json:     `[{"name":"build"},{"name":"test"}]`,
			task:     "clean",
			expected: false,
		},
		{
			name:     "empty list",
			json:     `[]`,
			task:     "build",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCommandRunner{
				outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
					return []byte(tt.json), nil
				},
			}

			e := NewExecutorWithRunner("/project", mock)
			got := e.TaskExists(tt.task)

			if got != tt.expected {
				t.Errorf("TaskExists(%q) = %v, want %v", tt.task, got, tt.expected)
			}
		})
	}
}

func TestExecutor_TaskExists_OutputError(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return nil, errors.New("mise not found")
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	got := e.TaskExists("build")

	if got != false {
		t.Error("TaskExists() = true when Output returns error, want false")
	}
}

func TestExecutor_GetTasksMeta_WithMock(t *testing.T) {
	jsonOutput := `[{"name":"build","depends":["restore"]},{"name":"restore","depends":[]}]`
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(jsonOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.GetTasksMeta(context.Background())

	if err != nil {
		t.Fatalf("GetTasksMeta() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	if tasks[0].Name != "build" {
		t.Errorf("tasks[0].Name = %q, want build", tasks[0].Name)
	}
	if len(tasks[0].Depends) != 1 || tasks[0].Depends[0] != "restore" {
		t.Errorf("tasks[0].Depends = %v, want [restore]", tasks[0].Depends)
	}
}

func TestExecutor_GetTasksMeta_Error(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return nil, errors.New("mise not found")
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.GetTasksMeta(context.Background())

	if err == nil {
		t.Error("GetTasksMeta() expected error")
	}
	if !strings.Contains(err.Error(), "failed to get mise tasks") {
		t.Errorf("error = %q, want to contain 'failed to get mise tasks'", err.Error())
	}
}

func TestExecutor_GetTasksMeta_InvalidJSON(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.GetTasksMeta(context.Background())

	if err == nil {
		t.Error("GetTasksMeta() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse mise tasks") {
		t.Errorf("error = %q, want to contain 'failed to parse mise tasks'", err.Error())
	}
}

func TestExecutor_Install_WithMock(t *testing.T) {
	mock := &mockCommandRunner{}

	e := NewExecutorWithRunner("/project", mock)
	err := e.Install(context.Background())

	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.method != "Run" {
		t.Errorf("method = %q, want Run", call.method)
	}
	expectedArgs := []string{"install"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

func TestExecutor_Trust_WithMock(t *testing.T) {
	mock := &mockCommandRunner{}

	e := NewExecutorWithRunner("/project", mock)
	err := e.Trust(context.Background())

	if err != nil {
		t.Fatalf("Trust() error = %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.method != "Run" {
		t.Errorf("method = %q, want Run", call.method)
	}
	expectedArgs := []string{"trust"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

// =============================================================================
// RunTaskWithCapture Tests
// =============================================================================

func TestExecutor_RunTaskWithCapture_Success(t *testing.T) {
	expectedOutput := "build output\nline 2\n"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			// Write to stdout (which is captured via MultiWriter)
			stdout.Write([]byte(expectedOutput))
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	output, err := e.RunTaskWithCapture(context.Background(), "build", nil)

	if err != nil {
		t.Fatalf("RunTaskWithCapture() error = %v", err)
	}
	if output != expectedOutput {
		t.Errorf("output = %q, want %q", output, expectedOutput)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	expectedArgs := []string{"run", "build"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

func TestExecutor_RunTaskWithCapture_WithArgs(t *testing.T) {
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.RunTaskWithCapture(context.Background(), "test", []string{"--verbose", "--coverage"})

	if err != nil {
		t.Fatalf("RunTaskWithCapture() error = %v", err)
	}
	call := mock.calls[0]
	expectedArgs := []string{"run", "test", "--verbose", "--coverage"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

func TestExecutor_RunTaskWithCapture_Error(t *testing.T) {
	expectedErr := errors.New("task failed")
	outputBeforeError := "partial output\n"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			stdout.Write([]byte(outputBeforeError))
			return expectedErr
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	output, err := e.RunTaskWithCapture(context.Background(), "build", nil)

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	// Output should still be captured even on error
	if output != outputBeforeError {
		t.Errorf("output = %q, want %q", output, outputBeforeError)
	}
}

func TestExecutor_RunTaskWithCapture_CapturesStderr(t *testing.T) {
	stdoutContent := "stdout line\n"
	stderrContent := "stderr line\n"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			stdout.Write([]byte(stdoutContent))
			stderr.Write([]byte(stderrContent))
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	output, err := e.RunTaskWithCapture(context.Background(), "build", nil)

	if err != nil {
		t.Fatalf("RunTaskWithCapture() error = %v", err)
	}
	// Both stdout and stderr should be captured
	if !strings.Contains(output, stdoutContent) {
		t.Errorf("output missing stdout: %q", output)
	}
	if !strings.Contains(output, stderrContent) {
		t.Errorf("output missing stderr: %q", output)
	}
}

// =============================================================================
// RunTaskOutput Tests
// =============================================================================

func TestExecutor_RunTaskOutput_Success(t *testing.T) {
	expectedOutput := "task output\n"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			stdout.Write([]byte(expectedOutput))
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	output, err := e.RunTaskOutput(context.Background(), "build", nil)

	if err != nil {
		t.Fatalf("RunTaskOutput() error = %v", err)
	}
	if output != expectedOutput {
		t.Errorf("output = %q, want %q", output, expectedOutput)
	}
}

func TestExecutor_RunTaskOutput_WithArgs(t *testing.T) {
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.RunTaskOutput(context.Background(), "test", []string{"--json"})

	if err != nil {
		t.Fatalf("RunTaskOutput() error = %v", err)
	}
	call := mock.calls[0]
	expectedArgs := []string{"run", "test", "--json"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

func TestExecutor_RunTaskOutput_Error(t *testing.T) {
	stderrContent := "error details"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			stderr.Write([]byte(stderrContent))
			return errors.New("exit status 1")
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.RunTaskOutput(context.Background(), "build", nil)

	if err == nil {
		t.Fatal("RunTaskOutput() expected error")
	}
	if !strings.Contains(err.Error(), "mise run failed") {
		t.Errorf("error = %q, want to contain 'mise run failed'", err.Error())
	}
	if !strings.Contains(err.Error(), stderrContent) {
		t.Errorf("error = %q, want to contain stderr content %q", err.Error(), stderrContent)
	}
}

func TestExecutor_RunTaskOutput_ReturnsOnlyStdout(t *testing.T) {
	stdoutContent := "stdout only\n"
	stderrContent := "stderr content\n"
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			stdout.Write([]byte(stdoutContent))
			stderr.Write([]byte(stderrContent))
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	output, err := e.RunTaskOutput(context.Background(), "build", nil)

	if err != nil {
		t.Fatalf("RunTaskOutput() error = %v", err)
	}
	// Should only return stdout, not stderr
	if output != stdoutContent {
		t.Errorf("output = %q, want %q", output, stdoutContent)
	}
	if strings.Contains(output, stderrContent) {
		t.Errorf("output should not contain stderr: %q", output)
	}
}

// =============================================================================
// ListTasks Tests
// =============================================================================

func TestExecutor_ListTasks_Success(t *testing.T) {
	miseOutput := "build  Build the project\ntest   Run tests\nclean  Clean artifacts\n"
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(miseOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	expected := []string{"build", "test", "clean"}
	if !reflect.DeepEqual(tasks, expected) {
		t.Errorf("tasks = %v, want %v", tasks, expected)
	}
}

func TestExecutor_ListTasks_WithNamespacedTasks(t *testing.T) {
	miseOutput := "build:go   Build Go\nbuild:rs   Build Rust\ntest:go    Test Go\n"
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(miseOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	expected := []string{"build:go", "build:rs", "test:go"}
	if !reflect.DeepEqual(tasks, expected) {
		t.Errorf("tasks = %v, want %v", tasks, expected)
	}
}

func TestExecutor_ListTasks_Empty(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(""), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("tasks = %v, want empty", tasks)
	}
}

func TestExecutor_ListTasks_SkipsComments(t *testing.T) {
	miseOutput := "# This is a comment\nbuild  Build\n# Another comment\ntest   Test\n"
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(miseOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	expected := []string{"build", "test"}
	if !reflect.DeepEqual(tasks, expected) {
		t.Errorf("tasks = %v, want %v", tasks, expected)
	}
}

func TestExecutor_ListTasks_SkipsBlankLines(t *testing.T) {
	miseOutput := "build  Build\n\n   \ntest   Test\n"
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(miseOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	expected := []string{"build", "test"}
	if !reflect.DeepEqual(tasks, expected) {
		t.Errorf("tasks = %v, want %v", tasks, expected)
	}
}

func TestExecutor_ListTasks_Error(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return nil, errors.New("mise not found")
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.ListTasks(context.Background())

	if err == nil {
		t.Fatal("ListTasks() expected error")
	}
}

func TestExecutor_ListTasks_VerifiesArgs(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(""), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, _ = e.ListTasks(context.Background())

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.method != "Output" {
		t.Errorf("method = %q, want Output", call.method)
	}
	expectedArgs := []string{"tasks"}
	if !reflect.DeepEqual(call.args, expectedArgs) {
		t.Errorf("args = %v, want %v", call.args, expectedArgs)
	}
}

// =============================================================================
// ResolveTaskDependencies Tests (via Executor)
// =============================================================================

func TestExecutor_ResolveTaskDependencies_Success(t *testing.T) {
	jsonOutput := `[{"name":"build","depends":["restore"]},{"name":"restore","depends":[]}]`
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(jsonOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks, err := e.ResolveTaskDependencies(context.Background(), "build")

	if err != nil {
		t.Fatalf("ResolveTaskDependencies() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	// restore should come before build
	if tasks[0].Name != "restore" {
		t.Errorf("tasks[0].Name = %q, want restore", tasks[0].Name)
	}
	if tasks[1].Name != "build" {
		t.Errorf("tasks[1].Name = %q, want build", tasks[1].Name)
	}
}

func TestExecutor_ResolveTaskDependencies_GetTasksMetaError(t *testing.T) {
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return nil, errors.New("mise not found")
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.ResolveTaskDependencies(context.Background(), "build")

	if err == nil {
		t.Fatal("ResolveTaskDependencies() expected error")
	}
}

func TestExecutor_ResolveTaskDependencies_TaskNotFound(t *testing.T) {
	jsonOutput := `[{"name":"build","depends":[]}]`
	mock := &mockCommandRunner{
		outputFunc: func(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
			return []byte(jsonOutput), nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	_, err := e.ResolveTaskDependencies(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("ResolveTaskDependencies() expected error for missing task")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

// =============================================================================
// RunTasksWithTracking Tests
// =============================================================================

func TestExecutor_RunTasksWithTracking_AllSuccess(t *testing.T) {
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks := []MiseTaskMeta{
		{Name: "restore"},
		{Name: "build"},
		{Name: "test"},
	}

	// Use an output writer that discards output
	out := output.NewWithWriters(io.Discard, io.Discard, false)
	summary := e.RunTasksWithTracking(context.Background(), tasks, nil, false, out, nil)

	if summary.Passed != 3 {
		t.Errorf("Passed = %d, want 3", summary.Passed)
	}
	if summary.Failed != 0 {
		t.Errorf("Failed = %d, want 0", summary.Failed)
	}
	if len(summary.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(summary.Tasks))
	}
	for i, task := range summary.Tasks {
		if !task.Success {
			t.Errorf("task[%d] Success = false, want true", i)
		}
	}
}

func TestExecutor_RunTasksWithTracking_StopsOnFailure(t *testing.T) {
	callCount := 0
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			callCount++
			// Fail on second task
			if callCount == 2 {
				return errors.New("task failed")
			}
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks := []MiseTaskMeta{
		{Name: "restore"},
		{Name: "build"},
		{Name: "test"},
	}

	out := output.NewWithWriters(io.Discard, io.Discard, false)
	summary := e.RunTasksWithTracking(context.Background(), tasks, nil, false, out, nil)

	// Should stop after failure (continueOnError = false)
	if summary.Passed != 1 {
		t.Errorf("Passed = %d, want 1", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if len(summary.Tasks) != 2 {
		t.Errorf("len(Tasks) = %d, want 2 (stopped after failure)", len(summary.Tasks))
	}
}

func TestExecutor_RunTasksWithTracking_ContinueOnError(t *testing.T) {
	callCount := 0
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			callCount++
			// Fail on second task
			if callCount == 2 {
				return errors.New("task failed")
			}
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks := []MiseTaskMeta{
		{Name: "restore"},
		{Name: "build"},
		{Name: "test"},
	}

	out := output.NewWithWriters(io.Discard, io.Discard, false)
	summary := e.RunTasksWithTracking(context.Background(), tasks, nil, true, out, nil) // continueOnError = true

	// Should continue after failure
	if summary.Passed != 2 {
		t.Errorf("Passed = %d, want 2", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if len(summary.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(summary.Tasks))
	}
}

func TestExecutor_RunTasksWithTracking_EmptyTasks(t *testing.T) {
	mock := &mockCommandRunner{}

	e := NewExecutorWithRunner("/project", mock)
	out := output.NewWithWriters(io.Discard, io.Discard, false)
	summary := e.RunTasksWithTracking(context.Background(), []MiseTaskMeta{}, nil, false, out, nil)

	if summary.Passed != 0 {
		t.Errorf("Passed = %d, want 0", summary.Passed)
	}
	if summary.Failed != 0 {
		t.Errorf("Failed = %d, want 0", summary.Failed)
	}
	if len(summary.Tasks) != 0 {
		t.Errorf("len(Tasks) = %d, want 0", len(summary.Tasks))
	}
	if len(mock.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(mock.calls))
	}
}

func TestExecutor_RunTasksWithTracking_WithArgs(t *testing.T) {
	var capturedArgs [][]string
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			capturedArgs = append(capturedArgs, args)
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks := []MiseTaskMeta{
		{Name: "test"},
	}

	out := output.NewWithWriters(io.Discard, io.Discard, false)
	e.RunTasksWithTracking(context.Background(), tasks, []string{"--verbose"}, false, out, nil)

	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 call, got %d", len(capturedArgs))
	}
	expectedArgs := []string{"run", "test", "--verbose"}
	if !reflect.DeepEqual(capturedArgs[0], expectedArgs) {
		t.Errorf("args = %v, want %v", capturedArgs[0], expectedArgs)
	}
}

func TestExecutor_RunTasksWithTracking_RecordsDuration(t *testing.T) {
	mock := &mockCommandRunner{
		runFunc: func(ctx context.Context, name string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
			return nil
		},
	}

	e := NewExecutorWithRunner("/project", mock)
	tasks := []MiseTaskMeta{
		{Name: "build"},
	}

	out := output.NewWithWriters(io.Discard, io.Discard, false)
	summary := e.RunTasksWithTracking(context.Background(), tasks, nil, false, out, nil)

	if summary.TotalDuration <= 0 {
		t.Error("TotalDuration should be positive")
	}
	if len(summary.Tasks) == 1 && summary.Tasks[0].Duration <= 0 {
		t.Error("task Duration should be positive")
	}
}
