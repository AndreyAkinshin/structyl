package mise

import (
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
			got := buildRunArgs(tt.task, tt.args, false)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildRunArgs(%q, %v, false) = %v, want %v", tt.task, tt.args, got, tt.expected)
			}
		})
	}
}

func TestBuildRunArgs_SkipDepsHasNoEffect(t *testing.T) {
	// Verify that skipDeps=true produces the same result as skipDeps=false
	// since mise doesn't support --no-deps flag
	argsWithSkip := buildRunArgs("test", []string{"--coverage"}, true)
	argsWithoutSkip := buildRunArgs("test", []string{"--coverage"}, false)

	if !reflect.DeepEqual(argsWithSkip, argsWithoutSkip) {
		t.Errorf("skipDeps should have no effect: with=%v, without=%v", argsWithSkip, argsWithoutSkip)
	}

	// Both should be: ["run", "test", "--coverage"]
	expected := []string{"run", "test", "--coverage"}
	if !reflect.DeepEqual(argsWithSkip, expected) {
		t.Errorf("args = %v, want %v", argsWithSkip, expected)
	}
}

func TestBuildRunArgs_TaskArgsAfterTaskName(t *testing.T) {
	// Verify task args come after task name
	args := buildRunArgs("build:rs", []string{"--release", "--target", "arm64"}, true)

	// Expected: ["run", "build:rs", "--release", "--target", "arm64"]
	taskIdx := -1
	for i, arg := range args {
		if arg == "build:rs" {
			taskIdx = i
			break
		}
	}

	if taskIdx == -1 {
		t.Fatal("task name not found")
	}

	// All args after taskIdx should be the task args
	taskArgs := args[taskIdx+1:]
	expected := []string{"--release", "--target", "arm64"}
	if !reflect.DeepEqual(taskArgs, expected) {
		t.Errorf("task args = %v, want %v", taskArgs, expected)
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

