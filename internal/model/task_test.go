package model

import (
	"errors"
	"testing"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

func TestTaskResult_ZeroValue(t *testing.T) {
	var result TaskResult

	if result.Name != "" {
		t.Errorf("expected empty Name, got %q", result.Name)
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Duration != 0 {
		t.Errorf("expected Duration to be 0, got %v", result.Duration)
	}
	if result.Error != nil {
		t.Errorf("expected Error to be nil, got %v", result.Error)
	}
	if result.TestCounts != nil {
		t.Errorf("expected TestCounts to be nil, got %v", result.TestCounts)
	}
}

func TestTaskResult_WithValues(t *testing.T) {
	testErr := errors.New("test error")
	testCounts := &testparser.TestCounts{Passed: 5, Failed: 1}

	result := TaskResult{
		Name:       "build:go",
		Success:    false,
		Duration:   2 * time.Second,
		Error:      testErr,
		TestCounts: testCounts,
	}

	if result.Name != "build:go" {
		t.Errorf("expected Name to be 'build:go', got %q", result.Name)
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Duration != 2*time.Second {
		t.Errorf("expected Duration to be 2s, got %v", result.Duration)
	}
	if result.Error != testErr {
		t.Errorf("expected Error to be testErr, got %v", result.Error)
	}
	if result.TestCounts != testCounts {
		t.Errorf("expected TestCounts to be testCounts, got %v", result.TestCounts)
	}
}

func TestTaskResult_SuccessfulTask(t *testing.T) {
	result := TaskResult{
		Name:     "test:rs",
		Success:  true,
		Duration: 500 * time.Millisecond,
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Error != nil {
		t.Errorf("expected Error to be nil for successful task, got %v", result.Error)
	}
}

func TestTaskRunSummary_ZeroValue(t *testing.T) {
	var summary TaskRunSummary

	if len(summary.Tasks) != 0 {
		t.Errorf("expected Tasks to be empty, got %d tasks", len(summary.Tasks))
	}
	if summary.TotalDuration != 0 {
		t.Errorf("expected TotalDuration to be 0, got %v", summary.TotalDuration)
	}
	if summary.Passed != 0 {
		t.Errorf("expected Passed to be 0, got %d", summary.Passed)
	}
	if summary.Failed != 0 {
		t.Errorf("expected Failed to be 0, got %d", summary.Failed)
	}
	if summary.TestCounts != nil {
		t.Errorf("expected TestCounts to be nil, got %v", summary.TestCounts)
	}
}

func TestTaskRunSummary_WithTasks(t *testing.T) {
	tasks := []TaskResult{
		{Name: "build:go", Success: true, Duration: 1 * time.Second},
		{Name: "build:rs", Success: true, Duration: 2 * time.Second},
		{Name: "build:py", Success: false, Duration: 500 * time.Millisecond, Error: errors.New("build failed")},
	}

	summary := TaskRunSummary{
		Tasks:         tasks,
		TotalDuration: 3500 * time.Millisecond,
		Passed:        2,
		Failed:        1,
	}

	if len(summary.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(summary.Tasks))
	}
	if summary.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.TotalDuration != 3500*time.Millisecond {
		t.Errorf("expected TotalDuration to be 3.5s, got %v", summary.TotalDuration)
	}
}

func TestTaskRunSummary_WithTestCounts(t *testing.T) {
	testCounts := &testparser.TestCounts{
		Passed:  10,
		Failed:  2,
		Skipped: 1,
	}

	summary := TaskRunSummary{
		Tasks: []TaskResult{
			{Name: "test:go", Success: true, Duration: 5 * time.Second, TestCounts: testCounts},
		},
		TotalDuration: 5 * time.Second,
		Passed:        1,
		Failed:        0,
		TestCounts:    testCounts,
	}

	if summary.TestCounts == nil {
		t.Fatal("expected TestCounts to be non-nil")
	}
	if summary.TestCounts.Passed != 10 {
		t.Errorf("expected TestCounts.Passed to be 10, got %d", summary.TestCounts.Passed)
	}
	if summary.TestCounts.Failed != 2 {
		t.Errorf("expected TestCounts.Failed to be 2, got %d", summary.TestCounts.Failed)
	}
	if summary.TestCounts.Skipped != 1 {
		t.Errorf("expected TestCounts.Skipped to be 1, got %d", summary.TestCounts.Skipped)
	}
}

func TestTaskRunSummary_AllPassed(t *testing.T) {
	summary := TaskRunSummary{
		Tasks: []TaskResult{
			{Name: "build", Success: true, Duration: 1 * time.Second},
			{Name: "test", Success: true, Duration: 2 * time.Second},
		},
		TotalDuration: 3 * time.Second,
		Passed:        2,
		Failed:        0,
	}

	if summary.Failed != 0 {
		t.Errorf("expected Failed to be 0, got %d", summary.Failed)
	}
	if summary.Passed != len(summary.Tasks) {
		t.Errorf("expected Passed to equal task count, got %d vs %d", summary.Passed, len(summary.Tasks))
	}
}

func TestTaskRunSummary_AllFailed(t *testing.T) {
	summary := TaskRunSummary{
		Tasks: []TaskResult{
			{Name: "build", Success: false, Duration: 1 * time.Second, Error: errors.New("fail1")},
			{Name: "test", Success: false, Duration: 1 * time.Second, Error: errors.New("fail2")},
		},
		TotalDuration: 2 * time.Second,
		Passed:        0,
		Failed:        2,
	}

	if summary.Passed != 0 {
		t.Errorf("expected Passed to be 0, got %d", summary.Passed)
	}
	if summary.Failed != len(summary.Tasks) {
		t.Errorf("expected Failed to equal task count, got %d vs %d", summary.Failed, len(summary.Tasks))
	}
}
