package mise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/testparser"
	"github.com/AndreyAkinshin/structyl/internal/topsort"
)

// Executor handles mise task execution.
type Executor struct {
	projectRoot string
	verbose     bool
}

// NewExecutor creates a new mise executor.
func NewExecutor(projectRoot string) *Executor {
	return &Executor{
		projectRoot: projectRoot,
	}
}

// SetVerbose enables verbose output.
func (e *Executor) SetVerbose(v bool) {
	e.verbose = v
}

// buildRunArgs constructs the command arguments for mise run.
func buildRunArgs(task string, args []string) []string {
	cmdArgs := []string{"run", task}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

// RunTask executes a mise task by name.
// Errors from mise (including non-zero exit) are returned as-is.
// Mise outputs its own diagnostics to stderr.
func (e *Executor) RunTask(ctx context.Context, task string, args []string) error {
	cmdArgs := buildRunArgs(task, args)

	cmd := exec.CommandContext(ctx, "mise", cmdArgs...)
	cmd.Dir = e.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Pass through environment
	cmd.Env = os.Environ()

	if e.verbose {
		fmt.Printf("Running: mise %s\n", strings.Join(cmdArgs, " "))
	}

	return cmd.Run()
}

// RunTaskWithCapture executes a mise task, streaming output while capturing it.
// Returns the combined stdout+stderr output and any execution error.
func (e *Executor) RunTaskWithCapture(ctx context.Context, task string, args []string) (string, error) {
	cmdArgs := buildRunArgs(task, args)

	cmd := exec.CommandContext(ctx, "mise", cmdArgs...)
	cmd.Dir = e.projectRoot
	cmd.Stdin = os.Stdin

	// Pass through environment
	cmd.Env = os.Environ()

	// Create a buffer to capture output while also streaming to stdout/stderr
	var capturedOutput bytes.Buffer

	// Use MultiWriter to both capture and stream output
	cmd.Stdout = io.MultiWriter(os.Stdout, &capturedOutput)
	cmd.Stderr = io.MultiWriter(os.Stderr, &capturedOutput)

	if e.verbose {
		fmt.Printf("Running: mise %s\n", strings.Join(cmdArgs, " "))
	}

	err := cmd.Run()
	return capturedOutput.String(), err
}

// RunTaskOutput executes a mise task and returns the output.
func (e *Executor) RunTaskOutput(ctx context.Context, task string, args []string) (string, error) {
	cmdArgs := []string{"run", task}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "mise", cmdArgs...)
	cmd.Dir = e.projectRoot
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("mise run failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// TaskExists checks if a mise task exists.
func (e *Executor) TaskExists(task string) bool {
	cmd := exec.Command("mise", "tasks", "--json")
	cmd.Dir = e.projectRoot

	output, err := cmd.Output()
	if err != nil {
		if e.verbose {
			fmt.Fprintf(os.Stderr, "[debug] TaskExists: failed to list mise tasks: %v\n", err)
		}
		return false
	}

	return taskExistsInJSON(output, task)
}

// taskExistsInJSON checks if a task name exists in mise JSON output.
// Separated for testability without calling mise.
func taskExistsInJSON(jsonData []byte, task string) bool {
	var tasks []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(jsonData, &tasks); err != nil {
		return false
	}
	for _, t := range tasks {
		if t.Name == task {
			return true
		}
	}
	return false
}

// ListTasks returns a list of available mise tasks.
func (e *Executor) ListTasks(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "mise", "tasks")
	cmd.Dir = e.projectRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tasks []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// First word is the task name
			parts := strings.Fields(line)
			if len(parts) > 0 {
				tasks = append(tasks, parts[0])
			}
		}
	}

	return tasks, nil
}

// Install runs mise install to ensure all tools are available.
func (e *Executor) Install(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "mise", "install")
	cmd.Dir = e.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}

// Trust marks the current directory as trusted for mise.
func (e *Executor) Trust(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "mise", "trust")
	cmd.Dir = e.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}

// GetTasksMeta returns structured task metadata from mise.
func (e *Executor) GetTasksMeta(ctx context.Context) ([]MiseTaskMeta, error) {
	cmd := exec.CommandContext(ctx, "mise", "tasks", "--json")
	cmd.Dir = e.projectRoot
	cmd.Env = os.Environ()

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get mise tasks: %w", err)
	}

	var tasks []MiseTaskMeta
	if err := json.Unmarshal(output, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse mise tasks: %w", err)
	}

	return tasks, nil
}

// ResolveTaskDependencies returns tasks in topological order (dependencies first).
func (e *Executor) ResolveTaskDependencies(ctx context.Context, taskName string) ([]MiseTaskMeta, error) {
	allTasks, err := e.GetTasksMeta(ctx)
	if err != nil {
		return nil, err
	}

	return resolveTaskDependenciesFromSlice(allTasks, taskName)
}

// resolveTaskDependenciesFromSlice performs topological sort on tasks.
// This is an internal function that can be tested without calling mise.
func resolveTaskDependenciesFromSlice(allTasks []MiseTaskMeta, taskName string) ([]MiseTaskMeta, error) {
	// Build task map and graph
	taskMap := make(map[string]MiseTaskMeta)
	graph := make(topsort.Graph)
	for _, t := range allTasks {
		taskMap[t.Name] = t
		graph[t.Name] = t.Depends
	}

	// Check if the task exists
	if _, exists := taskMap[taskName]; !exists {
		return nil, fmt.Errorf("task %q not found", taskName)
	}

	// Use shared topological sort
	sortedNames, err := topsort.Sort(graph, []string{taskName})
	if err != nil {
		return nil, err
	}

	// Convert names back to task metadata
	result := make([]MiseTaskMeta, len(sortedNames))
	for i, name := range sortedNames {
		result[i] = taskMap[name]
	}

	return result, nil
}

// RunTasksWithTracking executes tasks individually with progress tracking.
// If parserRegistry is provided, test tasks will have their output parsed for test counts.
func (e *Executor) RunTasksWithTracking(ctx context.Context, tasks []MiseTaskMeta, args []string, continueOnError bool, out *output.Writer, parserRegistry *testparser.Registry) *TaskRunSummary {
	summary := &TaskRunSummary{
		Tasks:      make([]TaskResult, 0, len(tasks)),
		TestCounts: &testparser.TestCounts{},
	}

	startTime := time.Now()

	for _, task := range tasks {
		result := TaskResult{
			Name: task.Name,
		}

		out.TargetStart(task.Name, "run")
		taskStart := time.Now()

		// Determine if we should capture output for parsing
		var parser testparser.Parser
		if parserRegistry != nil {
			parser = parserRegistry.GetParserForTask(task.Name)
		}

		var err error
		var taskOutput string

		if parser != nil {
			// Use capture mode for test tasks to parse output
			taskOutput, err = e.RunTaskWithCapture(ctx, task.Name, args)
		} else {
			// Use regular execution for non-test tasks
			err = e.RunTask(ctx, task.Name, args)
		}

		result.Duration = time.Since(taskStart)

		if err != nil {
			result.Success = false
			result.Error = err
			out.TargetFailed(task.Name, "run", err)
			summary.Failed++
		} else {
			result.Success = true
			out.TargetSuccess(task.Name, "run")
			summary.Passed++
		}

		// Parse test output if parser available
		if parser != nil && taskOutput != "" {
			counts := parser.Parse(taskOutput)
			if counts.Parsed {
				result.TestCounts = &counts
				summary.TestCounts.Add(&counts)
			}
		}

		summary.Tasks = append(summary.Tasks, result)

		// Stop on first failure unless continue is set
		if !result.Success && !continueOnError {
			break
		}
	}

	summary.TotalDuration = time.Since(startTime)
	return summary
}
