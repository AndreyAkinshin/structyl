// Package cli provides command-line interface functionality for structyl.
package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: intentionally using deprecated package for backwards compatibility
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// Version is set at build time.
var Version = "dev"

// wantsHelp checks if args contain -h or --help flag.
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

	// Handle help and version first
	switch cmd {
	case "-h", "--help", "help":
		printUsage()
		return 0
	case "--version", "version":
		fmt.Printf("structyl %s\n", Version)
		return 0
	}

	// Parse global flags
	opts, remaining, err := parseGlobalFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
		return 2
	}

	// Re-extract command after flag parsing
	if len(remaining) == 0 {
		printUsage()
		return 0
	}
	cmd = remaining[0]
	cmdArgs := remaining[1:]

	// Route to command handler
	switch cmd {
	// Project initialization (creates new project)
	case "init":
		return cmdInit(cmdArgs)

	// Deprecated: "new" is now "init"
	case "new":
		fmt.Fprintln(os.Stderr, "warning: 'structyl new' is deprecated, use 'structyl init'")
		return cmdInit(cmdArgs)

	// Release command
	case "release":
		return cmdRelease(cmdArgs, opts)

	// CI commands
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
		return cmdUpgrade(cmdArgs)
	case "completion":
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
	Docker          bool
	NoDocker        bool
	ContinueOnError bool
	TargetType      string
	Quiet           bool
	Verbose         bool
}

// parseGlobalFlags extracts global flags from arguments.
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
			opts.ContinueOnError = true
			i++
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

	// Validate target type
	if opts.TargetType != "" && opts.TargetType != "language" && opts.TargetType != "auxiliary" {
		return nil, nil, fmt.Errorf("invalid --type value %q (must be 'language' or 'auxiliary')", opts.TargetType)
	}

	// Validate mutual exclusivity of quiet and verbose
	if opts.Quiet && opts.Verbose {
		return nil, nil, fmt.Errorf("--quiet and --verbose are mutually exclusive")
	}

	return opts, remaining, nil
}

// isDockerMode determines if Docker mode should be used.
func isDockerMode(opts *GlobalOptions) bool {
	return runner.GetDockerMode(opts.Docker, opts.NoDocker, "")
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
	w.HelpCommand("targets", "List all configured targets", 10)
	w.HelpCommand("config", "Configuration utilities", 10)
	w.HelpCommand("upgrade", "Manage pinned CLI version", 10)
	w.HelpCommand("completion", "Generate shell completion (bash, zsh, fish)", 10)
	w.HelpCommand("version", "Show version information", 10)

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
	w.HelpCommand("targets", "List all configured targets", 10)
	w.HelpCommand("config", "Configuration utilities", 10)
	w.HelpCommand("upgrade", "Manage pinned CLI version", 10)
	w.HelpCommand("completion", "Generate shell completion (bash, zsh, fish)", 10)
	w.HelpCommand("version", "Show version information", 10)

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
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", 14)
	w.HelpFlag("-v, --verbose", "Maximum detail", 14)
	w.HelpFlag("--docker", "Run in Docker container", 14)
	w.HelpFlag("--no-docker", "Disable Docker mode", 14)
	w.HelpFlag("--continue", "Continue on error (don't fail-fast)", 14)
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", 14)
	w.HelpFlag("-h, --help", "Show this help", 14)
	w.HelpFlag("--version", "Show version", 14)

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
	var cmds []cmdFreq
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
