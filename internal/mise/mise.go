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
	Run         string            // Direct shell command
	Dir         string            // Working directory
	Env         map[string]string // Environment variables
	DependsOn   []string          // Task dependencies
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

// commandsToGenerate lists the commands to generate mise tasks for.
var commandsToGenerate = []string{
	"clean", "restore", "build", "build:release", "test", "check",
	"lint", "format", "format-check", "bench", "demo", "doc", "pack",
}

// aggregateCommands are commands that get aggregate tasks across all targets.
var aggregateCommands = []string{
	"clean", "restore", "build", "build:release", "test", "check",
	"lint", "format", "format-check",
}

// generateTasks creates mise tasks from project config.
// Generates individual tasks for each command of each target with direct shell commands.
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

	// Track which commands have tasks for aggregate generation
	commandTargets := make(map[string][]string) // command -> []target

	// Generate per-command tasks for each target
	for _, targetName := range targetNames {
		targetCfg := cfg.Targets[targetName]

		// Get resolved commands for this target
		resolvedCommands := getResolvedCommandsForTarget(targetCfg, cfg)

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
				// List of other commands to depend on
				var deps []string
				for _, dep := range v {
					if depStr, ok := dep.(string); ok {
						deps = append(deps, fmt.Sprintf("%s:%s", depStr, targetName))
					}
				}
				if len(deps) > 0 {
					tasks[taskName] = MiseTask{
						Description: fmt.Sprintf("%s for %s target", capitalize(cmdName), targetName),
						DependsOn:   deps,
					}
					commandTargets[cmdName] = append(commandTargets[cmdName], targetName)
				}

			case nil:
				// Command explicitly disabled, skip
				continue
			}
		}

		// Generate CI task for each target
		ciTaskName := fmt.Sprintf("ci:%s", targetName)
		ciDeps := []string{}
		for _, cmd := range []string{"clean", "restore", "check", "build", "test"} {
			depTask := fmt.Sprintf("%s:%s", cmd, targetName)
			if _, exists := tasks[depTask]; exists {
				ciDeps = append(ciDeps, depTask)
			}
		}
		if len(ciDeps) > 0 {
			tasks[ciTaskName] = MiseTask{
				Description: fmt.Sprintf("Run CI for %s target", targetName),
				DependsOn:   ciDeps,
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

// getResolvedCommandsForTarget resolves commands for a target, merging toolchain defaults with overrides.
func getResolvedCommandsForTarget(targetCfg config.TargetConfig, cfg *config.Config) map[string]interface{} {
	commands := make(map[string]interface{})

	// Get built-in toolchain commands
	if tc := getBuiltinToolchain(targetCfg.Toolchain); tc != nil {
		for k, v := range tc {
			commands[k] = v
		}
	}

	// Override with custom toolchain commands if defined
	if tcCfg, ok := cfg.Toolchains[targetCfg.Toolchain]; ok {
		// If extending, get base commands first
		if tcCfg.Extends != "" {
			if base := getBuiltinToolchain(tcCfg.Extends); base != nil {
				for k, v := range base {
					commands[k] = v
				}
			}
		}
		// Apply custom commands
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
		if task.Run != "" {
			fmt.Fprintf(b, "run = %q\n", task.Run)
		}
		b.WriteString("\n")
	}
}

// builtinToolchainCommands maps toolchain names to their command definitions.
// This mirrors the toolchain package but is local to avoid import cycles.
var builtinToolchainCommands = map[string]map[string]interface{}{
	"cargo": {
		"clean":         "cargo clean",
		"restore":       nil,
		"build":         "cargo build",
		"build:release": "cargo build --release",
		"test":          "cargo test",
		"check":         []interface{}{"lint", "format-check"},
		"lint":          "cargo clippy -- -D warnings",
		"format":        "cargo fmt",
		"format-check":  "cargo fmt --check",
		"bench":         "cargo bench",
		"pack":          "cargo package",
		"doc":           "cargo doc --no-deps",
		"demo":          "cargo run --example demo",
	},
	"dotnet": {
		"clean":         "dotnet clean",
		"restore":       "dotnet restore",
		"build":         "dotnet build",
		"build:release": "dotnet build -c Release",
		"test":          "dotnet test",
		"check":         "dotnet format --verify-no-changes",
		"lint":          "dotnet format --verify-no-changes",
		"format":        "dotnet format",
		"format-check":  "dotnet format --verify-no-changes",
		"bench":         nil,
		"pack":          "dotnet pack",
		"doc":           nil,
		"demo":          "dotnet run --project Demo",
	},
	"go": {
		"clean":         "go clean",
		"restore":       "go mod download",
		"build":         "go build ./...",
		"test":          "go test ./...",
		"check":         []interface{}{"lint"},
		"lint":          "golangci-lint run --out-format=colored-line-number",
		"format":        "go fmt ./...",
		"format-check":  `test -z "$(gofmt -l .)"`,
		"bench":         "go test -bench=. ./...",
		"pack":          nil,
		"doc":           "go doc ./...",
		"demo":          "go run ./cmd/demo",
	},
	"npm": {
		"clean":        "npm run clean",
		"restore":      "npm ci",
		"build":        "npm run build",
		"test":         "npm test",
		"check":        []interface{}{"lint", "format-check"},
		"lint":         "npm run lint",
		"format":       "npm run format",
		"format-check": "npm run format:check",
		"bench":        nil,
		"pack":         "npm pack",
		"doc":          nil,
		"demo":         "npm run demo",
	},
	"pnpm": {
		"clean":        "pnpm run clean",
		"restore":      "pnpm install --frozen-lockfile",
		"build":        "pnpm build",
		"test":         "pnpm test",
		"check":        []interface{}{"lint", "format-check"},
		"lint":         "pnpm lint",
		"format":       "pnpm format",
		"format-check": "pnpm format:check",
		"bench":        nil,
		"pack":         "pnpm pack",
		"doc":          nil,
		"demo":         "pnpm run demo",
	},
	"yarn": {
		"clean":        "yarn clean",
		"restore":      "yarn install --frozen-lockfile",
		"build":        "yarn build",
		"test":         "yarn test",
		"check":        []interface{}{"lint", "format-check"},
		"lint":         "yarn lint",
		"format":       "yarn format",
		"format-check": "yarn format:check",
		"bench":        nil,
		"pack":         "yarn pack",
		"doc":          nil,
		"demo":         "yarn run demo",
	},
	"bun": {
		"clean":        "bun run clean",
		"restore":      "bun install --frozen-lockfile",
		"build":        "bun run build",
		"test":         "bun test",
		"check":        []interface{}{"lint", "format-check"},
		"lint":         "bun run lint",
		"format":       "bun run format",
		"format-check": "bun run format:check",
		"bench":        nil,
		"pack":         "bun pm pack",
		"doc":          nil,
		"demo":         "bun run demo",
	},
	"python": {
		"clean":        "rm -rf dist/ build/ *.egg-info **/__pycache__/",
		"restore":      "pip install -e .",
		"build":        "python -m build",
		"test":         "pytest",
		"check":        []interface{}{"lint"},
		"lint":         "ruff check .",
		"format":       "ruff format .",
		"format-check": "ruff format --check .",
		"bench":        nil,
		"pack":         "python -m build",
		"doc":          nil,
		"demo":         "python demo.py",
	},
	"uv": {
		"clean":        "rm -rf dist/ build/ *.egg-info .venv/",
		"restore":      "uv sync",
		"build":        "uv build",
		"test":         "uv run pytest",
		"check":        []interface{}{"lint"},
		"lint":         "uv run ruff check .",
		"format":       "uv run ruff format .",
		"format-check": "uv run ruff format --check .",
		"bench":        nil,
		"pack":         "uv build",
		"doc":          nil,
		"demo":         "uv run python demo.py",
	},
	"poetry": {
		"clean":        "rm -rf dist/",
		"restore":      "poetry install",
		"build":        "poetry build",
		"test":         "poetry run pytest",
		"check":        []interface{}{"lint"},
		"lint":         "poetry run ruff check .",
		"format":       "poetry run ruff format .",
		"format-check": "poetry run ruff format --check .",
		"bench":        nil,
		"pack":         "poetry build",
		"doc":          nil,
		"demo":         "poetry run python demo.py",
	},
	"gradle": {
		"clean":        "gradle clean",
		"restore":      nil,
		"build":        "gradle build -x test",
		"test":         "gradle test",
		"check":        "gradle check -x test",
		"lint":         "gradle check -x test",
		"format":       "gradle spotlessApply",
		"format-check": "gradle spotlessCheck",
		"bench":        nil,
		"pack":         "gradle jar",
		"doc":          "gradle javadoc",
		"demo":         "gradle run",
	},
	"maven": {
		"clean":        "mvn clean",
		"restore":      "mvn dependency:resolve",
		"build":        "mvn compile",
		"test":         "mvn test",
		"check":        "mvn verify -DskipTests",
		"lint":         "mvn checkstyle:check",
		"format":       "mvn spotless:apply",
		"format-check": "mvn spotless:check",
		"bench":        nil,
		"pack":         "mvn package -DskipTests",
		"doc":          "mvn javadoc:javadoc",
		"demo":         "mvn exec:java",
	},
	"make": {
		"clean":         "make clean",
		"restore":       nil,
		"build":         "make",
		"build:release": "make release",
		"test":          "make test",
		"check":         "make check",
		"lint":          "make lint",
		"format":        "make format",
		"format-check":  nil,
		"bench":         "make bench",
		"pack":          "make dist",
		"doc":           "make doc",
		"demo":          "make demo",
	},
	"swift": {
		"clean":         "swift package clean",
		"restore":       "swift package resolve",
		"build":         "swift build",
		"build:release": "swift build -c release",
		"test":          "swift test",
		"check":         nil,
		"lint":          "swiftlint",
		"format":        "swiftformat .",
		"format-check":  "swiftformat --lint .",
		"bench":         nil,
		"pack":          nil,
		"doc":           nil,
		"demo":          "swift run Demo",
	},
	"deno": {
		"clean":        nil,
		"restore":      "deno install",
		"build":        nil,
		"test":         "deno test",
		"check":        []interface{}{"lint"},
		"lint":         "deno lint",
		"format":       "deno fmt",
		"format-check": "deno fmt --check",
		"bench":        "deno bench",
		"doc":          "deno doc",
		"demo":         "deno run demo.ts",
	},
	"bundler": {
		"clean":        "bundle clean",
		"restore":      "bundle install",
		"build":        "bundle exec rake build",
		"test":         "bundle exec rake test",
		"check":        []interface{}{"lint"},
		"lint":         "bundle exec rubocop",
		"format":       "bundle exec rubocop -a",
		"format-check": "bundle exec rubocop --format offenses --fail-level convention",
		"bench":        nil,
		"pack":         "gem build *.gemspec",
		"doc":          "bundle exec yard doc",
		"demo":         "bundle exec ruby demo.rb",
	},
	"zig": {
		"clean":         nil,
		"restore":       nil,
		"build":         "zig build",
		"build:release": "zig build -Doptimize=ReleaseFast",
		"test":          "zig build test",
		"check":         nil,
		"lint":          nil,
		"format":        "zig fmt .",
		"format-check":  "zig fmt --check .",
		"bench":         nil,
		"doc":           nil,
		"demo":          "zig build run",
	},
}

// getBuiltinToolchain returns the command map for a built-in toolchain.
func getBuiltinToolchain(name string) map[string]interface{} {
	return builtinToolchainCommands[name]
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
