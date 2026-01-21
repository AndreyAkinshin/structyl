package mise

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
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
