// Package model provides shared data types used across multiple internal packages.
// This package exists to break import cycles between packages like mise and output
// that need to share type definitions.
package model

import (
	"time"

	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

// TaskResult tracks execution result of a single task.
type TaskResult struct {
	Name       string
	Success    bool
	Duration   time.Duration
	Error      error
	TestCounts *testparser.TestCounts
}

// TaskRunSummary contains aggregated results from running multiple tasks.
type TaskRunSummary struct {
	Tasks         []TaskResult
	TotalDuration time.Duration
	Passed        int
	Failed        int
	TestCounts    *testparser.TestCounts // Aggregated test counts
}
