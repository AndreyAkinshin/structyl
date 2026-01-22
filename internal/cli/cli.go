// Package cli provides command-line interface functionality for structyl.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/errors"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// Version is set at build time.
var Version = "dev"

// wantsHelp returns true if args contain -h or --help before any -- separator.
// Arguments after -- are passed through to commands, so help flags there are ignored.
func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
		if arg == "--" {
			return false
		}
	}
	return false
}

// Run executes the CLI with the given arguments and returns an exit code.
func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}

	cmd := args[0]

	switch cmd {
	case "-h", "--help", "help":
		printUsage()
		return 0
	case "--version", "version":
		fmt.Printf("structyl %s\n", Version)
		return 0
	}

	opts, remaining, err := parseGlobalFlags(args)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return errors.ExitConfigError
	}

	// Re-extract command after flag parsing
	if len(remaining) == 0 {
		printUsage()
		return 0
	}
	cmd = remaining[0]
	cmdArgs := remaining[1:]

	initUpdateCheck(opts.Quiet)

	// Show notification at the end of the run (unless skipped)
	defer showUpdateNotification()

	// Route to command handler
	switch cmd {
	// Project initialization (creates new project)
	case "init":
		return cmdInit(cmdArgs)

	// Deprecated: "new" is now "init". Hidden from help text intentionally.
	// Emits deprecation warning but still works for backward compatibility.
	// Scheduled for removal in v2.0.0.
	case "new":
		out.WarningSimple("'structyl new' is deprecated and will be removed in v2.0.0; use 'structyl init'")
		return cmdInit(cmdArgs)

	// Release command
	case "release":
		return cmdRelease(cmdArgs, opts)

	// CI commands require explicit routing because they wrap mise tasks with pre/post
	// validation. Custom ci:* variants (e.g., ci:custom) defined in config go through
	// cmdUnified which delegates to mise directly without the CI wrapper logic.
	case "ci", "ci:release":
		return cmdCI(cmd, cmdArgs, opts)

	// Docker commands
	case "docker-build":
		return cmdDockerBuild(cmdArgs, opts)
	case "docker-clean":
		return cmdDockerClean(cmdArgs, opts)

	// Generation commands (mise-based)
	case "dockerfile":
		return cmdDockerfile(cmdArgs, opts)
	case "github":
		return cmdGitHub(cmdArgs, opts)

	// Mise commands
	case "mise":
		return cmdMise(cmdArgs, opts)

	// Test utilities
	case "test-summary":
		return cmdTestSummary(cmdArgs)

	// Utility commands
	case "targets":
		return cmdTargets(cmdArgs, opts)
	case "config":
		return cmdConfig(cmdArgs)
	case "upgrade":
		skipUpdateNotification()
		return cmdUpgrade(cmdArgs)
	case "completion":
		skipUpdateNotification()
		return cmdCompletion(cmdArgs)

	default:
		// Unified command handling:
		// - First arg is the command
		// - If second arg matches a target name: run command on that target
		// - Otherwise: run command on all targets that have it
		return cmdUnified(remaining, opts)
	}
}

// GlobalOptions holds parsed global flags.
type GlobalOptions struct {
	Docker     bool
	NoDocker   bool
	TargetType string
	Quiet      bool
	Verbose    bool
}

// parseGlobalFlags manually parses global flags from arguments.
//
// Manual parsing is used instead of stdlib flag package because:
// - Flags can appear anywhere in the argument list, not just before the command
// - Pass-through arguments after -- must be preserved verbatim
// - Custom error messages with usage hints are needed
// - Flag package doesn't support these use cases cleanly
func parseGlobalFlags(args []string) (*GlobalOptions, []string, error) {
	opts := &GlobalOptions{}
	var remaining []string

	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--docker":
			opts.Docker = true
			i++
		case arg == "--no-docker":
			opts.NoDocker = true
			i++
		case arg == "--continue":
			return nil, nil, fmt.Errorf("--continue flag has been removed; multi-target operations now stop on first failure")
		case arg == "-q" || arg == "--quiet":
			opts.Quiet = true
			i++
		case arg == "-v" || arg == "--verbose":
			opts.Verbose = true
			i++
		case arg == "--type":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("--type requires a value")
			}
			opts.TargetType = args[i+1]
			i += 2
		case strings.HasPrefix(arg, "--type="):
			opts.TargetType = strings.TrimPrefix(arg, "--type=")
			i++
		case arg == "--":
			// Everything after -- is passed through
			remaining = append(remaining, args[i:]...)
			i = len(args)
		default:
			remaining = append(remaining, arg)
			i++
		}
	}

	if err := validateGlobalOptions(opts); err != nil {
		return nil, nil, err
	}

	// Apply verbosity settings to global output writer.
	// This ensures all commands use consistent verbosity regardless of
	// whether they explicitly call applyVerbosityToOutput.
	applyVerbosityToOutput(opts)

	return opts, remaining, nil
}

// validateGlobalOptions checks that global options are valid.
func validateGlobalOptions(opts *GlobalOptions) error {
	// Validate target type
	if opts.TargetType != "" {
		if _, ok := target.ParseTargetType(opts.TargetType); !ok {
			return fmt.Errorf("invalid --type value %q\n  valid values: %s\n  example: structyl build --type=language",
				opts.TargetType, strings.Join(target.ValidTargetTypes(), ", "))
		}
	}

	// Validate mutual exclusivity of quiet and verbose
	if opts.Quiet && opts.Verbose {
		return fmt.Errorf("--quiet and --verbose are mutually exclusive")
	}

	// Validate mutual exclusivity of docker and no-docker.
	// Note: When both flags are absent, GetDockerMode() checks STRUCTYL_DOCKER env var.
	// See docs/specs/commands.md for full precedence rules.
	if opts.Docker && opts.NoDocker {
		return fmt.Errorf("--docker and --no-docker are mutually exclusive")
	}

	return nil
}

func printUsage() {
	w := output.New()

	w.HelpTitle("structyl - multi-language project orchestration")

	// Try to load project for context-aware help
	proj, _ := project.LoadProject()
	if proj != nil {
		printProjectHelp(w, proj)
	} else {
		printGenericHelp(w)
	}
}

func printProjectHelp(w *output.Writer, proj *project.Project) {
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		printGenericHelp(w)
		return
	}

	targets := registry.All()

	w.HelpSection("Usage:")
	w.HelpUsage("structyl <command> <target> [args]   Run command for a specific target")
	w.HelpUsage("structyl <command> [args]            Run command for all targets that have it")

	// Collect all unique commands across targets
	allCommands := collectAllCommands(targets)

	// Print common commands (available on all/most targets)
	if len(allCommands) > 0 {
		w.HelpSection("Commands:")
		for _, cmd := range allCommands {
			w.HelpCommand(cmd.name, cmd.description, cmd.width)
		}
	}

	// Print targets and their specific commands
	w.HelpSection("Targets:")
	maxNameLen := 0
	for _, t := range targets {
		if len(t.Name()) > maxNameLen {
			maxNameLen = len(t.Name())
		}
	}
	for _, t := range targets {
		cmds := t.Commands()
		cmdStr := strings.Join(cmds, ", ")
		w.HelpCommand(t.Name(), fmt.Sprintf("%s [%s]", t.Title(), cmdStr), maxNameLen)
	}

	w.HelpSection("CI/Release Commands:")
	w.HelpCommand("ci", "Run CI pipeline (clean, restore, check, build, test)", 15)
	w.HelpCommand("ci:release", "Run CI pipeline with release builds", 15)
	w.HelpCommand("release <ver>", "Create a release (set version, commit, optionally push)", 15)
	w.HelpSubCommand("--push", "Push to remote with tags", 10)
	w.HelpSubCommand("--dry-run", "Print what would be done", 10)
	w.HelpSubCommand("--force", "Force release with uncommitted changes", 10)

	w.HelpSection("Docker Commands:")
	w.HelpCommand("docker-build [services]", "Build Docker images for services", 22)
	w.HelpCommand("docker-clean", "Remove Docker containers and images", 22)

	w.HelpSection("Generation Commands:")
	w.HelpCommand("dockerfile", "Generate Dockerfiles with mise", 12)
	w.HelpCommand("github", "Generate GitHub Actions CI workflow", 12)

	w.HelpSection("Mise Commands:")
	w.HelpCommand("mise sync", "Regenerate mise.toml from config", 12)

	w.HelpSection("Utility Commands:")
	w.HelpCommand("targets", "List all configured targets", 16)
	w.HelpCommand("config validate", "Validate project configuration", 16)
	w.HelpCommand("upgrade", "Manage pinned CLI version", 16)
	w.HelpCommand("completion", "Generate shell completion (bash, zsh, fish)", 16)
	w.HelpCommand("version", "Show version information", 16)

	printGlobalFlags(w)
	printExamplesForProject(w, targets)
}

func printGenericHelp(w *output.Writer) {
	w.HelpSection("Usage:")
	w.HelpUsage("structyl <command> <target> [args]   Run command for a specific target")
	w.HelpUsage("structyl <command> [args]            Run command for all targets that have it")

	w.HelpSection("Project Setup:")
	w.HelpCommand("init", "Initialize a new structyl project", 10)

	// Use loaded descriptions from defaults
	defaults := toolchain.GetDefaultToolchains()
	getDesc := func(cmd string) string {
		if desc := toolchain.GetCommandDescription(defaults, cmd); desc != "" {
			return desc
		}
		return fmt.Sprintf("Run %s", cmd)
	}

	w.HelpSection("Common Commands:")
	w.HelpCommand("build", getDesc("build"), 10)
	w.HelpCommand("test", getDesc("test"), 10)
	w.HelpCommand("clean", getDesc("clean"), 10)
	w.HelpCommand("restore", getDesc("restore"), 10)
	w.HelpCommand("check", getDesc("check"), 10)

	w.HelpSection("CI/Release Commands:")
	w.HelpCommand("ci", "Run CI pipeline (clean, restore, check, build, test)", 15)
	w.HelpCommand("ci:release", "Run CI pipeline with release builds", 15)
	w.HelpCommand("release <ver>", "Create a release (set version, commit, optionally push)", 15)

	w.HelpSection("Docker Commands:")
	w.HelpCommand("docker-build [services]", "Build Docker images for services", 22)
	w.HelpCommand("docker-clean", "Remove Docker containers and images", 22)

	w.HelpSection("Generation Commands:")
	w.HelpCommand("dockerfile", "Generate Dockerfiles with mise", 12)
	w.HelpCommand("github", "Generate GitHub Actions CI workflow", 12)

	w.HelpSection("Mise Commands:")
	w.HelpCommand("mise sync", "Regenerate mise.toml from config", 12)

	w.HelpSection("Utility Commands:")
	w.HelpCommand("targets", "List all configured targets", 16)
	w.HelpCommand("config validate", "Validate project configuration", 16)
	w.HelpCommand("upgrade", "Manage pinned CLI version", 16)
	w.HelpCommand("completion", "Generate shell completion (bash, zsh, fish)", 16)
	w.HelpCommand("version", "Show version information", 16)

	printGlobalFlags(w)

	w.HelpSection("Examples:")
	w.HelpExample("structyl init", "Initialize new project")
	w.HelpExample("structyl build", "Build all targets")
	w.HelpExample("structyl build rs", "Build Rust target")
	w.HelpExample("structyl test cs --filter=X", "Run C# tests with filter")
	w.Println("")
}

func printGlobalFlags(w *output.Writer) {
	w.HelpSection("Global Flags:")
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", widthFlagWithValue)
	w.HelpFlag("-v, --verbose", "Maximum detail", widthFlagWithValue)
	w.HelpFlag("--docker", "Run in Docker container", widthFlagWithValue)
	w.HelpFlag("--no-docker", "Disable Docker mode", widthFlagWithValue)
	w.HelpFlag("--type=<type>", "Filter targets by type (language, auxiliary)", widthFlagWithValue)
	w.HelpFlag("-h, --help", "Show this help", widthFlagWithValue)
	w.HelpFlag("--version", "Show version", widthFlagWithValue)

	w.HelpSection("Environment:")
	w.HelpEnvVar("STRUCTYL_DOCKER=1", "Auto-enable Docker mode", 18)
}

func printExamplesForProject(w *output.Writer, targets []target.Target) {
	w.HelpSection("Examples:")

	// Show examples using actual target names
	if len(targets) > 0 {
		first := targets[0]
		cmds := first.Commands()
		if len(cmds) > 0 {
			// Example: build all targets
			w.HelpExample(fmt.Sprintf("structyl %s", cmds[0]), fmt.Sprintf("%s all targets", cmds[0]))
			// Example: target-specific command
			w.HelpExample(fmt.Sprintf("structyl %s %s", cmds[0], first.Name()), fmt.Sprintf("%s %s target", cmds[0], first.Title()))
		}
		if len(targets) > 1 {
			second := targets[1]
			secondCmds := second.Commands()
			if len(secondCmds) > 0 {
				w.HelpExample(fmt.Sprintf("structyl %s %s", secondCmds[0], second.Name()), fmt.Sprintf("%s %s target", secondCmds[0], second.Title()))
			}
		}
	}

	w.HelpExample("structyl --docker build", "Build all in Docker")
	w.HelpExample("structyl release 1.2.3 --push", "Create and push release")
	w.Println("")
}

type commandInfo struct {
	name        string
	description string
	width       int
}

func collectAllCommands(targets []target.Target) []commandInfo {
	// Count how many targets have each command
	cmdCount := make(map[string]int)
	for _, t := range targets {
		for _, cmd := range t.Commands() {
			cmdCount[cmd]++
		}
	}

	// Get commands sorted by frequency (most common first)
	type cmdFreq struct {
		name  string
		count int
	}
	cmds := make([]cmdFreq, 0, len(cmdCount))
	for cmd, count := range cmdCount {
		cmds = append(cmds, cmdFreq{cmd, count})
	}
	sort.Slice(cmds, func(i, j int) bool {
		if cmds[i].count != cmds[j].count {
			return cmds[i].count > cmds[j].count
		}
		return cmds[i].name < cmds[j].name
	})

	// Get descriptions from loaded defaults
	defaults := toolchain.GetDefaultToolchains()

	maxNameLen := 0
	for _, c := range cmds {
		if len(c.name) > maxNameLen {
			maxNameLen = len(c.name)
		}
	}

	var result []commandInfo
	for _, c := range cmds {
		desc := toolchain.GetCommandDescription(defaults, c.name)
		if desc == "" {
			desc = fmt.Sprintf("Run %s", c.name)
		}
		// Add target count info
		if c.count < len(targets) {
			desc = fmt.Sprintf("%s (%d/%d targets)", desc, c.count, len(targets))
		}
		result = append(result, commandInfo{c.name, desc, maxNameLen})
	}

	return result
}
