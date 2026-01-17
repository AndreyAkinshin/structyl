package mise

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// MiseConfig represents a .mise.toml configuration.
type MiseConfig struct {
	Tools map[string]string
	Tasks map[string]MiseTask
}

// MiseTask represents a task in .mise.toml.
type MiseTask struct {
	Description string
	Run         string
	DependsOn   []string
}

// GenerateMiseToml generates the content of a .mise.toml file.
func GenerateMiseToml(cfg *config.Config) (string, error) {
	var b strings.Builder

	// Get all tools
	tools := GetAllToolsFromConfig(cfg)
	sortedTools := GetToolsSorted(tools)

	// Write tools section
	if len(sortedTools) > 0 {
		b.WriteString("[tools]\n")
		for _, pair := range sortedTools {
			b.WriteString(fmt.Sprintf("%s = %q\n", pair[0], pair[1]))
		}
		b.WriteString("\n")
	}

	// Generate tasks section
	tasks := generateTasks(cfg)
	if len(tasks) > 0 {
		writeTasks(&b, tasks)
	}

	return b.String(), nil
}

// generateTasks creates mise tasks from project config.
func generateTasks(cfg *config.Config) map[string]MiseTask {
	tasks := make(map[string]MiseTask)

	// Setup task for structyl
	tasks["setup:structyl"] = MiseTask{
		Description: "Install structyl CLI",
		Run:         ".structyl/setup.sh",
	}

	// Collect target names sorted for deterministic output
	var targetNames []string
	for name := range cfg.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	// Generate CI task for each target
	var ciDeps []string
	for _, name := range targetNames {
		taskName := fmt.Sprintf("ci:%s", name)
		tasks[taskName] = MiseTask{
			Description: fmt.Sprintf("Run CI for %s target", name),
			Run:         fmt.Sprintf("structyl ci %s", name),
			DependsOn:   []string{"setup:structyl"},
		}
		ciDeps = append(ciDeps, taskName)
	}

	// Main CI task that runs all targets
	if len(ciDeps) > 0 {
		tasks["ci"] = MiseTask{
			Description: "Run CI for all targets",
			DependsOn:   ciDeps,
		}
	}

	return tasks
}

// writeTasks writes the tasks section to the builder.
func writeTasks(b *strings.Builder, tasks map[string]MiseTask) {
	// Sort task names for deterministic output
	var taskNames []string
	for name := range tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	for _, name := range taskNames {
		task := tasks[name]
		fmt.Fprintf(b, "[tasks.%q]\n", name)
		if task.Description != "" {
			fmt.Fprintf(b, "description = %q\n", task.Description)
		}
		if len(task.DependsOn) > 0 {
			// Format depends as TOML array
			deps := make([]string, len(task.DependsOn))
			for i, dep := range task.DependsOn {
				deps[i] = fmt.Sprintf("%q", dep)
			}
			fmt.Fprintf(b, "depends = [%s]\n", strings.Join(deps, ", "))
		}
		if task.Run != "" {
			fmt.Fprintf(b, "run = %q\n", task.Run)
		}
		b.WriteString("\n")
	}
}

// WriteMiseToml generates and writes a .mise.toml file to the project root.
// Returns true if a new file was created, false if it already exists.
// Use force=true to overwrite an existing file.
func WriteMiseToml(projectRoot string, cfg *config.Config, force bool) (bool, error) {
	outputPath := filepath.Join(projectRoot, ".mise.toml")

	// Check if file already exists
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return false, nil
		}
	}

	content, err := GenerateMiseToml(cfg)
	if err != nil {
		return false, fmt.Errorf("failed to generate .mise.toml: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write .mise.toml: %w", err)
	}

	return true, nil
}

// MiseTomlExists checks if a .mise.toml file exists in the project root.
func MiseTomlExists(projectRoot string) bool {
	path := filepath.Join(projectRoot, ".mise.toml")
	_, err := os.Stat(path)
	return err == nil
}
