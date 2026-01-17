package mise

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	cmdArgs := []string{"run", task}
	cmdArgs = append(cmdArgs, args...)

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
