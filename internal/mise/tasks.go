package mise

import "github.com/AndreyAkinshin/structyl/internal/model"

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

// TaskResult is an alias for the shared model type.
type TaskResult = model.TaskResult

// TaskRunSummary is an alias for the shared model type.
type TaskRunSummary = model.TaskRunSummary
