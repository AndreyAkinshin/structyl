package mise

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// MiseConfig represents a mise.toml configuration.
type MiseConfig struct {
	Tools map[string]string
	Tasks map[string]MiseTask
}

// MiseTask represents a task in mise.toml.
type MiseTask struct {
	Description string
	Run         string            // Direct shell command
	Dir         string            // Working directory
	Env         map[string]string // Environment variables
	DependsOn   []string          // Task dependencies (run in parallel)
	RunSequence []RunStep         // Sequential run steps (run one by one)
}

// RunStep represents a step in a sequential run.
// Either Run (shell command), Task (single task), or Tasks (parallel tasks) should be set.
type RunStep struct {
	Run   string   // Direct shell command
	Task  string   // Single task to run
	Tasks []string // Multiple tasks to run in parallel
}

// GenerateMiseToml generates the content of a mise.toml file.
// Deprecated: Use GenerateMiseTomlWithToolchains for loaded toolchains configuration.
func GenerateMiseToml(cfg *config.Config) (string, error) {
	return GenerateMiseTomlWithToolchains(cfg, nil)
}

// GenerateMiseTomlWithToolchains generates the content of a mise.toml file
// using the loaded toolchains configuration.
func GenerateMiseTomlWithToolchains(cfg *config.Config, loaded *toolchain.ToolchainsFile) (string, error) {
	var b strings.Builder

	// Get all tools
	tools := GetAllToolsWithToolchains(cfg, loaded)
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
	tasks := generateTasksWithToolchains(cfg, loaded)
	if len(tasks) > 0 {
		writeTasks(&b, tasks)
	}

	return b.String(), nil
}

// getCommandsToGenerate returns the list of commands to generate mise tasks for.
// Uses loaded config if available, otherwise falls back to defaults.
func getCommandsToGenerate(loaded *toolchain.ToolchainsFile) []string {
	commands := toolchain.GetStandardCommands(loaded)
	if len(commands) > 0 {
		return commands
	}
	// Fallback defaults
	return []string{
		"clean", "restore", "build", "build:release", "test",
		"check", "check:fix", "bench", "demo", "doc", "pack",
	}
}

// getAggregateCommands returns the list of commands that get aggregate tasks.
// Uses loaded config if available, otherwise falls back to defaults.
func getAggregateCommands(loaded *toolchain.ToolchainsFile) []string {
	commands := toolchain.GetAggregateCommands(loaded)
	if len(commands) > 0 {
		return commands
	}
	// Fallback defaults
	return []string{
		"clean", "restore", "build", "build:release", "test",
		"check", "check:fix",
	}
}

// getCIPipeline returns the CI pipeline commands.
// Uses loaded config if available, otherwise falls back to defaults.
func getCIPipeline(loaded *toolchain.ToolchainsFile) []string {
	pipeline := toolchain.GetPipeline(loaded, "ci")
	if len(pipeline) > 0 {
		return pipeline
	}
	// Fallback defaults
	return []string{"clean", "restore", "check", "build", "test"}
}

// generateTasksWithToolchains creates mise tasks from project config
// using the loaded toolchains configuration.
func generateTasksWithToolchains(cfg *config.Config, loaded *toolchain.ToolchainsFile) map[string]MiseTask {
	tasks := make(map[string]MiseTask)

	// Get command lists from loaded config
	commandsToGenerate := getCommandsToGenerate(loaded)
	aggregateCommands := getAggregateCommands(loaded)
	ciPipeline := getCIPipeline(loaded)

	// Setup task for structyl
	tasks[TaskSetupStructyl] = MiseTask{
		Description: "Install structyl CLI",
		Run:         ".structyl/setup.sh",
	}

	// Collect target names sorted for deterministic output
	var targetNames []string
	for name := range cfg.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	// Track which commands have tasks for aggregate generation
	commandTargets := make(map[string][]string) // command -> []target

	// Generate per-command tasks for each target
	for _, targetName := range targetNames {
		targetCfg := cfg.Targets[targetName]

		// Get resolved commands for this target
		resolvedCommands := getResolvedCommandsForTargetWithToolchains(targetCfg, cfg, loaded)

		// Determine working directory
		dir := targetCfg.Directory
		if dir == "" {
			dir = targetCfg.Cwd
		}

		for _, cmdName := range commandsToGenerate {
			cmdDef, ok := resolvedCommands[cmdName]
			if !ok {
				continue
			}

			taskName := fmt.Sprintf("%s:%s", cmdName, targetName)

			// Handle different command definition types
			switch v := cmdDef.(type) {
			case string:
				// Direct shell command
				task := MiseTask{
					Description: fmt.Sprintf("%s for %s target", capitalize(cmdName), targetName),
					Run:         v,
				}
				if dir != "" {
					task.Dir = dir
				}
				if len(targetCfg.Env) > 0 {
					task.Env = targetCfg.Env
				}
				tasks[taskName] = task
				commandTargets[cmdName] = append(commandTargets[cmdName], targetName)

			case []interface{}:
				// List of shell commands to run sequentially
				var steps []RunStep
				for _, item := range v {
					if cmdStr, ok := item.(string); ok {
						steps = append(steps, RunStep{Run: cmdStr})
					}
				}
				if len(steps) > 0 {
					task := MiseTask{
						Description: fmt.Sprintf("%s for %s target", capitalize(cmdName), targetName),
						RunSequence: steps,
					}
					if dir != "" {
						task.Dir = dir
					}
					if len(targetCfg.Env) > 0 {
						task.Env = targetCfg.Env
					}
					tasks[taskName] = task
					commandTargets[cmdName] = append(commandTargets[cmdName], targetName)
				}

			case nil:
				// Command explicitly disabled, skip
				continue
			}
		}

		// Generate CI task for each target with sequential execution
		ciTaskName := fmt.Sprintf("ci:%s", targetName)
		var ciSteps []RunStep
		for _, cmd := range ciPipeline {
			depTask := fmt.Sprintf("%s:%s", cmd, targetName)
			if _, exists := tasks[depTask]; exists {
				ciSteps = append(ciSteps, RunStep{Task: depTask})
			}
		}
		if len(ciSteps) > 0 {
			tasks[ciTaskName] = MiseTask{
				Description: fmt.Sprintf("Run CI for %s target", targetName),
				RunSequence: ciSteps,
			}
		}
	}

	// Generate aggregate tasks (build, test, etc.) that depend on all target-specific tasks
	for _, cmdName := range aggregateCommands {
		targets := commandTargets[cmdName]
		if len(targets) == 0 {
			continue
		}

		var deps []string
		for _, targetName := range targets {
			deps = append(deps, fmt.Sprintf("%s:%s", cmdName, targetName))
		}

		tasks[cmdName] = MiseTask{
			Description: fmt.Sprintf("%s all targets", capitalize(cmdName)),
			DependsOn:   deps,
		}
	}

	// Generate main CI task
	var ciDeps []string
	for _, targetName := range targetNames {
		ciTaskName := fmt.Sprintf("ci:%s", targetName)
		if _, exists := tasks[ciTaskName]; exists {
			ciDeps = append(ciDeps, ciTaskName)
		}
	}
	if len(ciDeps) > 0 {
		tasks["ci"] = MiseTask{
			Description: "Run CI for all targets",
			DependsOn:   ciDeps,
		}
	}

	return tasks
}

// getResolvedCommandsForTargetWithToolchains resolves commands for a target using loaded toolchains,
// merging toolchain defaults with overrides.
// Resolution priority (highest to lowest):
//  1. Target-specific commands in config.json targets.X.commands
//  2. Custom toolchain commands in config.json toolchains.X
//  3. Loaded .structyl/toolchains.json overrides
//  4. Hardcoded Go defaults
func getResolvedCommandsForTargetWithToolchains(targetCfg config.TargetConfig, cfg *config.Config, loaded *toolchain.ToolchainsFile) map[string]interface{} {
	commands := make(map[string]interface{})

	// Get toolchain commands from loaded config (or fall back to builtins)
	if tc, ok := toolchain.GetFromConfig(targetCfg.Toolchain, loaded); ok {
		for k, v := range tc.Commands {
			commands[k] = v
		}
	}

	// Override with custom toolchain commands if defined in config.json
	if tcCfg, ok := cfg.Toolchains[targetCfg.Toolchain]; ok {
		// If extending, get base commands first
		if tcCfg.Extends != "" {
			if base, ok := toolchain.GetFromConfig(tcCfg.Extends, loaded); ok {
				for k, v := range base.Commands {
					commands[k] = v
				}
			}
		}
		// Apply custom commands from config.json
		for k, v := range tcCfg.Commands {
			commands[k] = v
		}
	}

	// Override with target-specific commands
	for k, v := range targetCfg.Commands {
		commands[k] = v
	}

	return commands
}

// capitalize returns a string with the first letter capitalized.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
		if task.Dir != "" {
			fmt.Fprintf(b, "dir = %q\n", task.Dir)
		}
		if len(task.Env) > 0 {
			// Write env as inline table
			b.WriteString("env = { ")
			var envPairs []string
			// Sort env keys for deterministic output
			envKeys := make([]string, 0, len(task.Env))
			for k := range task.Env {
				envKeys = append(envKeys, k)
			}
			sort.Strings(envKeys)
			for _, k := range envKeys {
				envPairs = append(envPairs, fmt.Sprintf("%s = %q", k, task.Env[k]))
			}
			b.WriteString(strings.Join(envPairs, ", "))
			b.WriteString(" }\n")
		}
		if len(task.DependsOn) > 0 {
			// Format depends as TOML array
			deps := make([]string, len(task.DependsOn))
			for i, dep := range task.DependsOn {
				deps[i] = fmt.Sprintf("%q", dep)
			}
			fmt.Fprintf(b, "depends = [%s]\n", strings.Join(deps, ", "))
		}
		if len(task.RunSequence) > 0 {
			// Format run as array of sequential steps
			b.WriteString("run = [\n")
			for _, step := range task.RunSequence {
				if step.Run != "" {
					// Direct shell command
					fmt.Fprintf(b, "    %q,\n", step.Run)
				} else if step.Task != "" {
					fmt.Fprintf(b, "    { task = %q },\n", step.Task)
				} else if len(step.Tasks) > 0 {
					tasks := make([]string, len(step.Tasks))
					for i, t := range step.Tasks {
						tasks[i] = fmt.Sprintf("%q", t)
					}
					fmt.Fprintf(b, "    { tasks = [%s] },\n", strings.Join(tasks, ", "))
				}
			}
			b.WriteString("]\n")
		} else if task.Run != "" {
			fmt.Fprintf(b, "run = %q\n", task.Run)
		}
		b.WriteString("\n")
	}
}

// WriteMiseToml generates and writes a mise.toml file to the project root.
// Returns true if a new file was created, false if it already exists.
// Use force=true to overwrite an existing file.
// Deprecated: Use WriteMiseTomlWithToolchains for loaded toolchains configuration.
func WriteMiseToml(projectRoot string, cfg *config.Config, force bool) (bool, error) {
	return WriteMiseTomlWithToolchains(projectRoot, cfg, nil, force)
}

// WriteMiseTomlWithToolchains generates and writes a mise.toml file to the project root
// using the loaded toolchains configuration.
// Returns true if a new file was created, false if it already exists.
// Use force=true to overwrite an existing file.
func WriteMiseTomlWithToolchains(projectRoot string, cfg *config.Config, loaded *toolchain.ToolchainsFile, force bool) (bool, error) {
	outputPath := filepath.Join(projectRoot, "mise.toml")

	// Check if file already exists
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return false, nil
		}
	}

	content, err := GenerateMiseTomlWithToolchains(cfg, loaded)
	if err != nil {
		return false, fmt.Errorf("failed to generate mise.toml: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write mise.toml: %w", err)
	}

	return true, nil
}

// MiseTomlExists checks if a mise.toml file exists in the project root.
func MiseTomlExists(projectRoot string) bool {
	path := filepath.Join(projectRoot, "mise.toml")
	_, err := os.Stat(path)
	return err == nil
}
