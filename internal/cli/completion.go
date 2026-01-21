package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// cmdCompletion generates shell completion scripts.
func cmdCompletion(args []string) int {
	w := output.New()
	shell := ""
	alias := ""

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			printCompletionUsage()
			return 0
		case strings.HasPrefix(arg, "--alias="):
			alias = strings.TrimPrefix(arg, "--alias=")
		case arg == "--alias":
			w.ErrorPrefix("completion: --alias requires a value (--alias=<name>)")
			return 2
		case strings.HasPrefix(arg, "-"):
			w.ErrorPrefix("completion: unknown flag: %s", arg)
			printCompletionUsage()
			return 2
		default:
			if shell != "" {
				w.ErrorPrefix("completion: unexpected argument: %s", arg)
				return 2
			}
			shell = arg
		}
	}

	if shell == "" {
		w.ErrorPrefix("completion: shell required (bash, zsh, fish)")
		printCompletionUsage()
		return 2
	}

	// Use "structyl" as default command name
	cmdName := "structyl"
	if alias != "" {
		cmdName = alias
	}

	switch shell {
	case "bash":
		fmt.Print(generateBashCompletion(cmdName))
	case "zsh":
		fmt.Print(generateZshCompletion(cmdName))
	case "fish":
		fmt.Print(generateFishCompletion(cmdName))
	default:
		w.ErrorPrefix("completion: unsupported shell %q (use bash, zsh, or fish)", shell)
		return 2
	}

	return 0
}

// printCompletionUsage prints the help text for the completion command.
func printCompletionUsage() {
	w := output.New()

	w.HelpTitle("structyl completion - generate shell completion scripts")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl completion <shell> [--alias=<name>]")

	w.HelpSection("Arguments:")
	w.HelpFlag("<shell>", "Shell type: bash, zsh, or fish", 10)

	w.HelpSection("Options:")
	w.HelpFlag("--alias=<name>", "Generate completion for command alias", 14)
	w.HelpFlag("-h, --help", "Show this help", 14)

	w.HelpSection("Examples:")
	w.HelpExample("structyl completion bash", "Generate bash completion")
	w.HelpExample("structyl completion zsh", "Generate zsh completion")
	w.HelpExample("structyl completion fish", "Generate fish completion")
	w.HelpExample("structyl completion bash --alias=s", "Generate bash completion for alias 's'")

	w.HelpSection("Installation:")
	w.Println("  Bash:  eval \"$(structyl completion bash)\"")
	w.Println("  Zsh:   eval \"$(structyl completion zsh)\"")
	w.Println("  Fish:  structyl completion fish | source")
	w.Println("")
}

// builtinCommands returns the list of built-in CLI commands.
func builtinCommands() []string {
	return []string{
		"init",
		"ci",
		"ci:release",
		"release",
		"docker-build",
		"docker-clean",
		"dockerfile",
		"github",
		"mise",
		"targets",
		"config",
		"upgrade",
		"version",
		"help",
		"completion",
	}
}

// commonTargetCommands returns commands typically available on targets.
func commonTargetCommands() []string {
	defaults := toolchain.GetDefaultToolchains()
	commands := toolchain.GetStandardCommands(defaults)
	if len(commands) > 0 {
		sort.Strings(commands)
		return commands
	}
	// Fallback
	return []string{
		"bench", "build", "build:release", "check", "check:fix",
		"clean", "demo", "doc", "pack", "restore", "test", "test:coverage",
	}
}

// globalFlags returns the global CLI flags.
func globalFlags() []string {
	return []string{
		"--docker",
		"--no-docker",
		"--type",
		"--help",
		"--version",
	}
}

func generateBashCompletion(cmdName string) string {
	commands := append(builtinCommands(), commonTargetCommands()...)
	flags := globalFlags()

	// Generate function name from command (replace - with _)
	funcName := "_" + strings.ReplaceAll(cmdName, "-", "_") + "_completions"

	var aliasNote string
	if cmdName == "structyl" {
		aliasNote = `
# Alias support:
# If you use an alias (e.g., alias st="structyl"), add completion for it:
#   complete -F _structyl_completions st
# Or generate completion directly for your alias:
#   eval "$(structyl completion bash --alias=st)"
`
	} else {
		aliasNote = fmt.Sprintf(`
# This completion is generated for the alias "%s"
# Make sure you have the alias defined: alias %s="structyl"
`, cmdName, cmdName)
	}

	return fmt.Sprintf(`# structyl bash completion
# Add to ~/.bashrc: eval "$(structyl completion bash)"
%s
%s() {
    local cur prev words cword
    _init_completion || return

    local commands="%s"
    local flags="%s"
    local config_subcommands="validate"
    local completion_shells="bash zsh fish"

    case "${prev}" in
        %s)
            COMPREPLY=($(compgen -W "${commands} ${flags}" -- "${cur}"))
            return
            ;;
        config)
            COMPREPLY=($(compgen -W "${config_subcommands}" -- "${cur}"))
            return
            ;;
        completion)
            COMPREPLY=($(compgen -W "${completion_shells}" -- "${cur}"))
            return
            ;;
        --type)
            COMPREPLY=($(compgen -W "language auxiliary" -- "${cur}"))
            return
            ;;
    esac

    # Complete flags if current word starts with -
    if [[ "${cur}" == -* ]]; then
        COMPREPLY=($(compgen -W "${flags}" -- "${cur}"))
        return
    fi

    # Try to get dynamic target names if in a structyl project
    local targets
    if targets=$(structyl targets 2>/dev/null | awk '{print $1}'); then
        COMPREPLY=($(compgen -W "${targets} ${commands} ${flags}" -- "${cur}"))
    else
        COMPREPLY=($(compgen -W "${commands} ${flags}" -- "${cur}"))
    fi
}

complete -F %s %s
`, aliasNote, funcName, strings.Join(commands, " "), strings.Join(flags, " "), cmdName, funcName, cmdName)
}

func generateZshCompletion(cmdName string) string {
	// Generate function name from command (replace - with _)
	funcName := "_" + strings.ReplaceAll(cmdName, "-", "_")

	var aliasNote string
	if cmdName == "structyl" {
		aliasNote = `
# Alias support:
# If you use an alias (e.g., alias st="structyl"), add completion for it:
#   compdef _structyl st
# Or generate completion directly for your alias:
#   eval "$(structyl completion zsh --alias=st)"
`
	} else {
		aliasNote = fmt.Sprintf(`
# This completion is generated for the alias "%s"
# Make sure you have the alias defined: alias %s="structyl"
`, cmdName, cmdName)
	}

	return fmt.Sprintf(`#compdef %s
# structyl zsh completion
# Add to ~/.zshrc: eval "$(structyl completion zsh)"
%s
%s() {
    local -a commands flags target_commands config_subcommands completion_shells

    commands=(
        'init:Initialize a new structyl project'
        'ci:Run CI pipeline'
        'ci\:release:Run CI pipeline with release builds'
        'release:Create a release'
        'docker-build:Build Docker images'
        'docker-clean:Remove Docker containers and images'
        'dockerfile:Generate Dockerfiles with mise'
        'github:Generate GitHub Actions CI workflow'
        'mise:Mise integration commands'
        'targets:List all configured targets'
        'config:Configuration utilities'
        'upgrade:Manage pinned CLI version'
        'version:Show version information'
        'help:Show help'
        'completion:Generate shell completion'
    )

    target_commands=(
        'build:Build targets'
        'build\:release:Build targets in release mode'
        'test:Run tests'
        'test\:coverage:Run tests with coverage'
        'clean:Clean build artifacts'
        'restore:Restore dependencies'
        'check:Run static analysis'
        'format:Format code'
        'format-check:Check code formatting'
        'lint:Run linter'
        'bench:Run benchmarks'
        'demo:Run demos'
        'doc:Generate documentation'
        'pack:Create package'
    )

    flags=(
        '--docker[Run in Docker container]'
        '--no-docker[Disable Docker mode]'
        '--type=[Filter targets by type]:type:(language auxiliary)'
        '--help[Show help]'
        '--version[Show version]'
    )

    config_subcommands=(
        'validate:Validate configuration'
    )

    completion_shells=(
        'bash:Generate bash completion'
        'zsh:Generate zsh completion'
        'fish:Generate fish completion'
    )

    # Get dynamic targets (used in multiple places)
    local -a targets
    targets=(${(f)"$(structyl targets 2>/dev/null | awk '{print $1}')"})

    # Determine current position
    local cur_pos=$((CURRENT - 1))

    if (( cur_pos == 1 )); then
        # First argument: show all commands, target commands, dynamic targets, and flags
        _describe -t commands 'command' commands
        _describe -t target-commands 'target command' target_commands
        if [[ ${#targets[@]} -gt 0 && -n "${targets[1]}" ]]; then
            _describe -t targets 'target' targets
        fi
        _arguments -s $flags[@]
        return
    fi

    # Second+ argument: context-sensitive completion based on first word
    case "${words[2]}" in
        config)
            _describe -t config-subcommands 'config subcommand' config_subcommands
            ;;
        completion)
            _describe -t shells 'shell' completion_shells
            ;;
        build|build\:release|test|test\:coverage|clean|restore|check|format|format-check|lint|bench|demo|doc|pack)
            # Target commands: show dynamic targets as arguments
            if [[ ${#targets[@]} -gt 0 && -n "${targets[1]}" ]]; then
                _describe -t targets 'target' targets
            fi
            _arguments -s $flags[@]
            ;;
        *)
            # Unknown command or builtin without subcommands: just show flags
            _arguments -s $flags[@]
            ;;
    esac
}

compdef %s %s
`, cmdName, aliasNote, funcName, funcName, cmdName)
}

func generateFishCompletion(cmdName string) string {
	var sb strings.Builder

	var aliasNote string
	if cmdName == "structyl" {
		aliasNote = `# Alias support:
# If you use an alias (e.g., alias st="structyl"), add completion for it:
#   complete -c st -w structyl
# Or generate completion directly for your alias:
#   structyl completion fish --alias=st | source
`
	} else {
		aliasNote = fmt.Sprintf(`# This completion is generated for the alias "%s"
# Make sure you have the alias defined: alias %s="structyl"
`, cmdName, cmdName)
	}

	sb.WriteString(fmt.Sprintf(`# structyl fish completion
# Add to config: structyl completion fish | source

%s
# Disable file completion by default
complete -c %s -f

`, aliasNote, cmdName))

	// Built-in commands
	commandDescs := map[string]string{
		"init":         "Initialize a new structyl project",
		"ci":           "Run CI pipeline",
		"ci:release":   "Run CI pipeline with release builds",
		"release":      "Create a release",
		"docker-build": "Build Docker images",
		"docker-clean": "Remove Docker containers and images",
		"dockerfile":   "Generate Dockerfiles with mise",
		"github":       "Generate GitHub Actions CI workflow",
		"mise":         "Mise integration commands",
		"targets":      "List all configured targets",
		"config":       "Configuration utilities",
		"upgrade":      "Manage pinned CLI version",
		"version":      "Show version information",
		"help":         "Show help",
		"completion":   "Generate shell completion",
	}

	for cmd, desc := range commandDescs {
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a '%s' -d '%s'\n", cmdName, cmd, desc))
	}

	sb.WriteString("\n# Target commands\n")
	defaults := toolchain.GetDefaultToolchains()
	for _, cmd := range commonTargetCommands() {
		desc := toolchain.GetCommandDescription(defaults, cmd)
		if desc == "" {
			desc = fmt.Sprintf("Run %s", cmd)
		}
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a '%s' -d '%s'\n", cmdName, cmd, desc))
	}

	sb.WriteString("\n# Global flags\n")
	sb.WriteString(fmt.Sprintf("complete -c %s -l docker -d 'Run in Docker container'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -l no-docker -d 'Disable Docker mode'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -l continue -d 'Continue on error'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -l type -d 'Filter targets by type' -xa 'language auxiliary'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -l help -d 'Show help'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -l version -d 'Show version'\n", cmdName))

	sb.WriteString("\n# config subcommands\n")
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from config' -a 'validate' -d 'Validate configuration'\n", cmdName))

	sb.WriteString("\n# completion subcommands\n")
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from completion' -a 'bash' -d 'Generate bash completion'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from completion' -a 'zsh' -d 'Generate zsh completion'\n", cmdName))
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from completion' -a 'fish' -d 'Generate fish completion'\n", cmdName))

	sb.WriteString("\n# Dynamic target completion\n")
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a '(structyl targets 2>/dev/null | string match -r \"^\\S+\")' -d 'Target'\n", cmdName))

	return sb.String()
}
