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

// RunTask executes a mise task by name.
func (e *Executor) RunTask(ctx context.Context, task string, args []string) error {
	return e.runTask(ctx, task, args)
}

// buildRunArgs constructs the command arguments for mise run.
func buildRunArgs(task string, args []string) []string {
	cmdArgs := []string{"run", task}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

// runTask executes a mise task.
func (e *Executor) runTask(ctx context.Context, task string, args []string) error {
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
	return e.runTaskWithCapture(ctx, task, args)
}

// runTaskWithCapture executes a mise task, streaming output while capturing it.
func (e *Executor) runTaskWithCapture(ctx context.Context, task string, args []string) (string, error) {
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
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// TaskExists checks if a mise task exists.
func (e *Executor) TaskExists(task string) bool {
	cmd := exec.Command("mise", "tasks", "--json")
	cmd.Dir = e.projectRoot

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Simple check - look for task name in output
	return strings.Contains(string(output), fmt.Sprintf(`"name":"%s"`, task)) ||
		strings.Contains(string(output), fmt.Sprintf(`"name": "%s"`, task))
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
	// Build task map for lookup
	taskMap := make(map[string]MiseTaskMeta)
	for _, t := range allTasks {
		taskMap[t.Name] = t
	}

	// Check if the task exists
	rootTask, exists := taskMap[taskName]
	if !exists {
		return nil, fmt.Errorf("task %q not found", taskName)
	}

	// Topological sort with cycle detection
	var result []MiseTaskMeta
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected involving task %q", name)
		}
		if visited[name] {
			return nil
		}

		inStack[name] = true

		task, exists := taskMap[name]
		if !exists {
			return fmt.Errorf("dependency task %q not found", name)
		}

		// Visit dependencies first
		for _, dep := range task.Depends {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visited[name] = true
		inStack[name] = false
		result = append(result, task)

		return nil
	}

	if err := visit(taskName); err != nil {
		return nil, err
	}

	// If the root task has no dependencies and is a leaf task, return just itself
	if len(rootTask.Depends) == 0 {
		return []MiseTaskMeta{rootTask}, nil
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
			taskOutput, err = e.runTaskWithCapture(ctx, task.Name, args)
		} else {
			// Use regular execution for non-test tasks
			err = e.runTask(ctx, task.Name, args)
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
