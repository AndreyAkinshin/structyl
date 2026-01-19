package mise

import (
	"time"

	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

// Task name constants for structyl-specific mise tasks.
const (
	// TaskSetupStructyl is the task that installs the structyl CLI.
	TaskSetupStructyl = "setup:structyl"
)

// MiseTaskMeta represents task metadata from `mise tasks --json`.
type MiseTaskMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Depends     []string `json:"depends"`
	Run         []string `json:"run"`
}

// TaskResult tracks execution result of a single task.
type TaskResult struct {
	Name       string
	Success    bool
	Duration   time.Duration
	Error      error
	TestCounts *testparser.TestCounts
}

// TaskRunSummary contains aggregated results.
type TaskRunSummary struct {
	Tasks         []TaskResult
	TotalDuration time.Duration
	Passed        int
	Failed        int
	TestCounts    *testparser.TestCounts // Aggregated test counts
}
