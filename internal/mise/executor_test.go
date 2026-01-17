package mise

import (
	"reflect"
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

func TestBuildRunArgs_WithSkipDeps(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		args     []string
		expected []string
	}{
		{
			name:     "simple task with skip deps",
			task:     "build",
			args:     nil,
			expected: []string{"run", "--no-deps", "build"},
		},
		{
			name:     "task with args and skip deps",
			task:     "test",
			args:     []string{"--verbose"},
			expected: []string{"run", "--no-deps", "test", "--verbose"},
		},
		{
			name:     "namespaced task with skip deps",
			task:     "test:go",
			args:     nil,
			expected: []string{"run", "--no-deps", "test:go"},
		},
		{
			name:     "aggregate task with skip deps",
			task:     "test",
			args:     nil,
			expected: []string{"run", "--no-deps", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRunArgs(tt.task, tt.args, true)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildRunArgs(%q, %v, true) = %v, want %v", tt.task, tt.args, got, tt.expected)
			}
		})
	}
}

func TestBuildRunArgs_NoDepsBeforeTaskName(t *testing.T) {
	// Verify --no-deps comes before task name (mise requirement)
	args := buildRunArgs("test", []string{"--coverage"}, true)

	// Find position of --no-deps and task name
	noDepsIdx := -1
	taskIdx := -1
	for i, arg := range args {
		if arg == "--no-deps" {
			noDepsIdx = i
		}
		if arg == "test" {
			taskIdx = i
		}
	}

	if noDepsIdx == -1 {
		t.Error("--no-deps not found in args")
	}
	if taskIdx == -1 {
		t.Error("task name not found in args")
	}
	if noDepsIdx >= taskIdx {
		t.Errorf("--no-deps (index %d) should come before task name (index %d)", noDepsIdx, taskIdx)
	}
}

func TestBuildRunArgs_TaskArgsAfterTaskName(t *testing.T) {
	// Verify task args come after task name
	args := buildRunArgs("build:rs", []string{"--release", "--target", "arm64"}, true)

	// Expected: ["run", "--no-deps", "build:rs", "--release", "--target", "arm64"]
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
