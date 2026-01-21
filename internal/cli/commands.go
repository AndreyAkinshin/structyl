package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	internalerrors "github.com/AndreyAkinshin/structyl/internal/errors"
	"github.com/AndreyAkinshin/structyl/internal/mise"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/release"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: Docker runner functionality still needed
	"github.com/AndreyAkinshin/structyl/internal/schema"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

var out = output.New()

// Help text alignment widths for consistent formatting.
// These values align the flag/command names with their descriptions.
const (
	helpFlagWidthShort      = 10 // Width for short flags like "-h, --help"
	helpFlagWidthLong       = 12 // Width for longer flags like "[services]"
	helpFlagWidthGlobal     = 14 // Width for global flags like "--type=<type>"
	helpSubcommandWidthSync = 6  // Width for "sync" subcommand in mise help
)

// applyVerbosityToOutput configures the output writer based on verbosity settings.
func applyVerbosityToOutput(opts *GlobalOptions) {
	out.SetQuiet(opts.Quiet)
	out.SetVerbose(opts.Verbose)
}

// loadProject loads the project configuration and handles errors uniformly.
// Returns the project and exit code 0 on success, or nil and appropriate exit code on failure.
// Exit codes: 1 for runtime errors, 2 for config errors (per errors package specification).
func loadProject() (*project.Project, int) {
	proj, err := project.LoadProject()
	if err != nil {
		out.ErrorPrefix("%v", err)
		return nil, internalerrors.GetExitCode(err)
	}
	return proj, 0
}

// loadProjectWithRegistry loads project and creates target registry in one step.
// Returns (nil, nil, exitCode) on failure. On success, returns (proj, registry, 0).
// This consolidates the common pattern of loading project then creating registry.
func loadProjectWithRegistry() (*project.Project, *target.Registry, int) {
	proj, exitCode := loadProject()
	if proj == nil {
		return nil, nil, exitCode
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return nil, nil, internalerrors.ExitConfigError
	}
	return proj, registry, 0
}

// printProjectWarnings outputs any warnings accumulated during project loading.
func printProjectWarnings(proj *project.Project) {
	for _, w := range proj.Warnings {
		out.WarningSimple("%s", w)
	}
}

// MiseRegenerateMode specifies when mise.toml should be regenerated.
type MiseRegenerateMode int

const (
	// MiseAutoRegenerate regenerates only if auto_generate is enabled or file is missing.
	MiseAutoRegenerate MiseRegenerateMode = iota
	// MiseForceRegenerate always regenerates mise.toml regardless of settings.
	MiseForceRegenerate
)

// ensureMiseConfig ensures mise.toml is up-to-date.
// If auto_generate is enabled, regenerates the file.
// If mode is MiseForceRegenerate, always regenerates.
func ensureMiseConfig(proj *project.Project, mode MiseRegenerateMode) error {
	autoGen := true // default to auto-generation so mise.toml stays in sync

	if proj.Config.Mise != nil && proj.Config.Mise.AutoGenerate != nil {
		autoGen = *proj.Config.Mise.AutoGenerate
	}

	miseTomlExists := mise.MiseTomlExists(proj.Root)
	forceRegenerate := mode == MiseForceRegenerate
	fileNeedsCreation := !miseTomlExists
	// Regenerate if: explicitly forced, auto-generation enabled, or file is missing
	shouldRegenerate := forceRegenerate || autoGen || fileNeedsCreation
	if shouldRegenerate {
		_, err := mise.WriteMiseTomlWithToolchains(proj.Root, proj.Config, proj.Toolchains, true)
		if err != nil {
			return fmt.Errorf("failed to generate mise.toml: %w", err)
		}
	}

	return nil
}

// formatMiseTaskName converts a structyl command and optional target to a mise task name.
// Examples:
//   - ("build", "") → "build"
//   - ("build", "go") → "build:go"
//   - ("ci", "rs") → "ci:rs"
func formatMiseTaskName(cmd string, target string) string {
	if target == "" {
		return cmd
	}
	return fmt.Sprintf("%s:%s", cmd, target)
}

// runViaMise executes a command via mise.
// Mise handles dependency resolution and parallel execution internally,
// so we simply delegate to RunTask regardless of the task structure.
// The optional registry parameter enables helpful hints when a command fails because
// the user may have typed a target name instead of a command name.
func runViaMise(proj *project.Project, cmd string, targetName string, args []string, opts *GlobalOptions, registry *target.Registry) int {
	ctx := context.Background()

	task := formatMiseTaskName(cmd, targetName)

	executor := mise.NewExecutor(proj.Root)
	executor.SetVerbose(opts.Verbose)

	// Error details are output by mise directly to stderr; we only need the exit code.
	if err := executor.RunTask(ctx, task, args); err != nil {
		// If no target was specified and cmd matches a known target name,
		// the user likely typed "structyl cs" instead of "structyl build cs"
		if registry != nil && targetName == "" {
			if _, exists := registry.Get(cmd); exists {
				out.Hint("Did you mean 'structyl build %s'?", cmd)
			}
		}
		return internalerrors.ExitRuntimeError
	}
	return 0
}

// cmdUnified handles both target-specific and cross-target commands.
// The first argument is always the command. If a second argument matches a target name,
// it runs the command on that target. Otherwise, it runs on all targets that have it.
func cmdUnified(args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	if len(args) == 0 {
		out.ErrorPrefix("usage: structyl <command> [target] [args] or structyl <command> [args]")
		return internalerrors.ExitConfigError
	}

	// Check for help flag early (after command name)
	if len(args) > 1 && wantsHelp(args[1:]) {
		printUnifiedUsage(args[0])
		return 0
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	printProjectWarnings(proj)

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return internalerrors.ExitConfigError
	}

	cmd := args[0]
	remaining := args[1:]

	// Determine target name (if specified)
	targetName, cmdArgs := extractTargetArg(remaining, registry)

	// Check mise is installed
	if err := EnsureMise(true); err != nil {
		out.ErrorPrefix("%v", err)
		PrintMiseInstallInstructions()
		return internalerrors.ExitEnvironmentError
	}

	// Ensure mise.toml is up-to-date
	if err := ensureMiseConfig(proj, MiseAutoRegenerate); err != nil {
		out.ErrorPrefix("%v", err)
		return internalerrors.ExitRuntimeError
	}

	// If --type is specified and no specific target given, filter targets by type
	if opts.TargetType != "" && targetName == "" {
		targets := registry.ByType(target.TargetType(opts.TargetType))
		if len(targets) == 0 {
			out.WarningSimple("no targets of type %q found", opts.TargetType)
			return 0
		}
		// Run command for each matching target
		for _, t := range targets {
			if result := runViaMise(proj, cmd, t.Name(), cmdArgs, opts, registry); result != 0 {
				return result
			}
		}
		return 0
	}

	return runViaMise(proj, cmd, targetName, cmdArgs, opts, registry)
}

// cmdTargets lists all configured targets.
func cmdTargets(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printTargetsUsage()
		return 0
	}

	_, registry, exitCode := loadProjectWithRegistry()
	if registry == nil {
		return exitCode
	}

	var targets []target.Target
	if opts.TargetType != "" {
		targets = registry.ByType(target.TargetType(opts.TargetType))
		if len(targets) == 0 {
			out.WarningSimple("no targets of type %q found", opts.TargetType)
			return 0
		}
	} else {
		targets = registry.All()
	}

	for _, t := range targets {
		commands := strings.Join(t.Commands(), ", ")
		out.TargetInfo(t.Name(), string(t.Type()), t.Title())
		out.TargetDetail("commands", commands)
		if deps := t.DependsOn(); len(deps) > 0 {
			out.TargetDetail("depends_on", strings.Join(deps, ", "))
		}
	}

	return 0
}

// cmdConfig handles configuration utilities.
func cmdConfig(args []string) int {
	if len(args) == 0 {
		out.ErrorPrefix("config: subcommand required (validate)")
		return internalerrors.ExitConfigError
	}

	switch args[0] {
	case "validate":
		return cmdConfigValidate()
	case "-h", "--help":
		printConfigUsage()
		return 0
	default:
		out.ErrorPrefix("config: unknown subcommand %q", args[0])
		return internalerrors.ExitConfigError
	}
}

func cmdConfigValidate() int {
	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Run JSON Schema validation on raw config file.
	// LoadProject performs Go struct parsing and semantic validation,
	// but schema validation catches additional issues like type mismatches
	// and constraint violations defined in the JSON Schema.
	configPath := proj.ConfigPath()
	configData, err := os.ReadFile(configPath)
	if err != nil {
		out.ErrorPrefix("failed to read config for schema validation: %v", err)
		return internalerrors.ExitRuntimeError
	}

	if err := schema.ValidateConfig(configData); err != nil {
		out.ErrorPrefix("schema validation failed: %v", err)
		return internalerrors.ExitConfigError
	}

	printProjectWarnings(proj)

	// Validate registry creation
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return internalerrors.ExitConfigError
	}

	// Count targets by type
	targets := registry.All()
	var langCount, auxCount int
	for _, t := range targets {
		if t.Type() == target.TypeLanguage {
			langCount++
		} else {
			auxCount++
		}
	}

	out.ValidationSuccess("Configuration is valid.")
	out.SummaryItem("Project", proj.Config.Project.Name)
	out.SummaryItem("Targets", fmt.Sprintf("%d (%d language, %d auxiliary)", len(targets), langCount, auxCount))
	if len(proj.Warnings) > 0 {
		out.SummaryItem("Warnings", fmt.Sprintf("%d", len(proj.Warnings)))
	}
	return 0
}

// cmdCI runs the CI pipeline.
func cmdCI(cmd string, args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printCIUsage(cmd)
		return 0
	}

	applyVerbosityToOutput(opts)

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Determine target name from args
	var targetName string
	var cmdArgs []string
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		// Registry failed to load - log warning and treat all args as command args
		out.WarningSimple("could not load target registry: %v", err)
		cmdArgs = args
	} else {
		targetName, cmdArgs = extractTargetArg(args, registry)
	}

	// Check mise is installed
	if err := EnsureMise(true); err != nil {
		out.ErrorPrefix("%v", err)
		PrintMiseInstallInstructions()
		return internalerrors.ExitEnvironmentError
	}

	// Ensure mise.toml is up-to-date
	if err := ensureMiseConfig(proj, MiseAutoRegenerate); err != nil {
		out.ErrorPrefix("%v", err)
		return internalerrors.ExitRuntimeError
	}

	return runViaMise(proj, cmd, targetName, cmdArgs, opts, registry)
}

// extractTargetArg extracts an optional target name from args if the first arg is a known target.
// Returns (targetName, remaining args). If registry is nil or first arg is not a target,
// returns empty targetName and all original args.
func extractTargetArg(args []string, registry *target.Registry) (string, []string) {
	if len(args) == 0 || registry == nil {
		return "", args
	}
	if _, ok := registry.Get(args[0]); ok {
		return args[0], args[1:]
	}
	return "", args
}

// cmdMise handles mise-related subcommands.
func cmdMise(args []string, opts *GlobalOptions) int {
	if len(args) == 0 {
		out.ErrorPrefix("mise: subcommand required (sync)")
		out.Println("usage: structyl mise sync")
		return internalerrors.ExitConfigError
	}

	// Check if first arg is a known subcommand - if so, route to it
	// Otherwise, check for help flag at mise level
	switch args[0] {
	case "sync":
		return cmdMiseSync(args[1:], opts)
	case "-h", "--help":
		printMiseUsage()
		return 0
	default:
		out.ErrorPrefix("mise: unknown subcommand %q", args[0])
		out.Println("usage: structyl mise sync")
		return internalerrors.ExitConfigError
	}
}

// cmdMiseSync regenerates the mise.toml file.
func cmdMiseSync(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printMiseSyncUsage()
		return 0
	}

	// Parse flags
	for _, arg := range args {
		if arg == "--force" {
			out.ErrorPrefix("--force flag has been removed: mise sync always regenerates")
			return internalerrors.ExitConfigError
		}
		if strings.HasPrefix(arg, "-") {
			out.ErrorPrefix("mise sync: unknown option %q", arg)
			return internalerrors.ExitConfigError
		}
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Generate mise.toml using loaded toolchains (always regenerates)
	created, err := mise.WriteMiseTomlWithToolchains(proj.Root, proj.Config, proj.Toolchains, true)
	if err != nil {
		out.ErrorPrefix("mise sync: %v", err)
		return internalerrors.ExitRuntimeError
	}

	if created {
		out.Success("Generated mise.toml")
	} else {
		out.Info("mise.toml is up to date")
	}

	// Print summary using loaded toolchains
	tools := mise.GetAllToolsWithToolchains(proj.Config, proj.Toolchains)
	if len(tools) > 0 {
		out.HelpSection("Tools:")
		for name, version := range tools {
			out.Println("  %s = %s", name, version)
		}
	}

	return 0
}

// handleDockerError logs a Docker availability error and returns the appropriate exit code.
func handleDockerError(err error) int {
	out.ErrorPrefix("%v", err)
	var dockerErr *runner.DockerUnavailableError
	if errors.As(err, &dockerErr) {
		return dockerErr.ExitCode()
	}
	return internalerrors.ExitRuntimeError
}

// cmdDockerBuild builds Docker images for services.
func cmdDockerBuild(args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	if wantsHelp(args) {
		printDockerBuildUsage()
		return 0
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Use the full config runner to support per-target Dockerfiles
	dockerRunner := runner.NewDockerRunnerWithConfig(proj.Root, proj.Config)

	if err := runner.CheckDockerAvailable(); err != nil {
		return handleDockerError(err)
	}

	ctx := context.Background()
	if err := dockerRunner.Build(ctx, args...); err != nil {
		out.ErrorPrefix("docker-build failed: %v", err)
		return internalerrors.ExitRuntimeError
	}

	out.Success("Docker images built successfully.")
	return 0
}

// cmdDockerClean removes Docker containers and images.
func cmdDockerClean(args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	if wantsHelp(args) {
		printDockerCleanUsage()
		return 0
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	dockerRunner := runner.NewDockerRunner(proj.Root, proj.Config.Docker)

	if err := runner.CheckDockerAvailable(); err != nil {
		return handleDockerError(err)
	}

	ctx := context.Background()
	if err := dockerRunner.Clean(ctx); err != nil {
		out.ErrorPrefix("docker-clean failed: %v", err)
		return internalerrors.ExitRuntimeError
	}

	out.Success("Docker containers and images cleaned successfully.")
	return 0
}

// cmdRelease performs the release workflow.
func cmdRelease(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printReleaseUsage()
		return 0
	}

	// Parse release-specific flags
	releaseOpts := release.Options{}
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--push":
			releaseOpts.Push = true
		case "--dry-run":
			releaseOpts.DryRun = true
		case "--force":
			releaseOpts.Force = true
		default:
			remaining = append(remaining, arg)
		}
	}

	if len(remaining) == 0 {
		out.ErrorPrefix("release: version required")
		out.Errorln("usage: structyl release <version> [--push] [--dry-run] [--force]")
		return internalerrors.ExitConfigError
	}

	releaseOpts.Version = remaining[0]

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	releaser := release.NewReleaser(proj.Root, proj.Config)

	ctx := context.Background()
	if err := releaser.Release(ctx, releaseOpts); err != nil {
		out.ErrorPrefix("release: %v", err)
		return internalerrors.ExitRuntimeError
	}

	return 0
}

// printUnifiedUsage prints the help text for unified commands (build, test, etc.).
func printUnifiedUsage(cmd string) {
	w := output.New()

	defaults := toolchain.GetDefaultToolchains()
	desc := toolchain.GetCommandDescription(defaults, cmd)
	if desc == "" {
		desc = fmt.Sprintf("run %s", cmd)
	} else {
		// Convert to lowercase for help text
		desc = strings.ToLower(desc)
	}

	w.HelpTitle(fmt.Sprintf("structyl %s - %s", cmd, desc))

	w.HelpSection("Usage:")
	w.HelpUsage(fmt.Sprintf("structyl %s [target] [options]", cmd))

	w.HelpSection("Description:")
	w.Println("  When a target is specified, runs %s on that target only.", cmd)
	w.Println("  Without a target, runs %s on all targets that have it defined.", cmd)

	w.HelpSection("Arguments:")
	w.HelpFlag("[target]", "Target name to run command on (optional)", helpFlagWidthShort)

	w.HelpSection("Global Options:")
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", helpFlagWidthGlobal)
	w.HelpFlag("-v, --verbose", "Maximum detail", helpFlagWidthGlobal)
	w.HelpFlag("--docker", "Run in Docker container", helpFlagWidthGlobal)
	w.HelpFlag("--no-docker", "Disable Docker mode", helpFlagWidthGlobal)
	w.Println("                  (precedence: --no-docker > --docker > STRUCTYL_DOCKER > default)")
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", helpFlagWidthGlobal)
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthGlobal)

	w.HelpSection("Examples:")
	titleCase := cases.Title(language.English)
	w.HelpExample(fmt.Sprintf("structyl %s", cmd), fmt.Sprintf("%s all targets", titleCase.String(cmd)))
	w.HelpExample(fmt.Sprintf("structyl %s go", cmd), fmt.Sprintf("%s Go target only", titleCase.String(cmd)))
	w.HelpExample(fmt.Sprintf("structyl %s --docker", cmd), fmt.Sprintf("%s all targets in Docker", titleCase.String(cmd)))
	w.Println("")
}

// printReleaseUsage prints the help text for the release command.
func printReleaseUsage() {
	w := output.New()

	w.HelpTitle("structyl release - create a release")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl release <version> [options]")

	w.HelpSection("Description:")
	w.Println("  Creates a release by setting the version across all targets,")
	w.Println("  committing the changes, and optionally pushing to the remote.")

	w.HelpSection("Arguments:")
	w.HelpFlag("<version>", "Version number (e.g., 1.2.3)", helpFlagWidthShort)

	w.HelpSection("Options:")
	w.HelpFlag("--push", "Push to remote with tags after commit", helpFlagWidthShort)
	w.HelpFlag("--dry-run", "Print what would be done without making changes", helpFlagWidthShort)
	w.HelpFlag("--force", "Force release with uncommitted changes", helpFlagWidthShort)
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl release 1.2.3", "Create release 1.2.3")
	w.HelpExample("structyl release 1.2.3 --push", "Create and push release 1.2.3")
	w.HelpExample("structyl release 1.2.3 --dry-run", "Preview release without changes")
	w.Println("")
}

// printCIUsage prints the help text for the ci and ci:release commands.
func printCIUsage(cmd string) {
	w := output.New()

	if cmd == "ci:release" {
		w.HelpTitle("structyl ci:release - run CI pipeline with release builds")
	} else {
		w.HelpTitle("structyl ci - run CI pipeline")
	}

	w.HelpSection("Usage:")
	w.HelpUsage(fmt.Sprintf("structyl %s [target] [options]", cmd))

	w.HelpSection("Description:")
	if cmd == "ci:release" {
		w.Println("  Runs the CI pipeline with release builds: clean, restore, check,")
		w.Println("  build:release, test. When a target is specified, runs only for")
		w.Println("  that target.")
	} else {
		w.Println("  Runs the CI pipeline: clean, restore, check, build, test.")
		w.Println("  When a target is specified, runs only for that target.")
	}

	w.HelpSection("Arguments:")
	w.HelpFlag("[target]", "Target name to run CI on (optional)", helpFlagWidthShort)

	w.HelpSection("Global Options:")
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", helpFlagWidthGlobal)
	w.HelpFlag("-v, --verbose", "Maximum detail", helpFlagWidthGlobal)
	w.HelpFlag("--docker", "Run in Docker container", helpFlagWidthGlobal)
	w.HelpFlag("--no-docker", "Disable Docker mode", helpFlagWidthGlobal)
	w.Println("                  (precedence: --no-docker > --docker > STRUCTYL_DOCKER > default)")
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", helpFlagWidthGlobal)
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthGlobal)

	w.HelpSection("Examples:")
	w.HelpExample(fmt.Sprintf("structyl %s", cmd), "Run CI on all targets")
	w.HelpExample(fmt.Sprintf("structyl %s go", cmd), "Run CI on Go target only")
	w.HelpExample(fmt.Sprintf("structyl %s --docker", cmd), "Run CI in Docker")
	w.Println("")
}

// printConfigUsage prints the help text for the config command.
func printConfigUsage() {
	w := output.New()

	w.HelpTitle("structyl config - configuration utilities")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl config <subcommand>")

	w.HelpSection("Subcommands:")
	w.HelpCommand("validate", "Validate the project configuration", helpFlagWidthShort)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl config validate", "Validate project configuration")
	w.Println("")
}

// printMiseUsage prints the help text for the mise command.
func printMiseUsage() {
	w := output.New()

	w.HelpTitle("structyl mise - mise integration commands")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl mise <subcommand>")

	w.HelpSection("Subcommands:")
	w.HelpCommand("sync", "Regenerate mise.toml from project configuration", helpSubcommandWidthSync)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl mise sync", "Regenerate mise.toml")
	w.Println("")
}

// printMiseSyncUsage prints the help text for the mise sync command.
func printMiseSyncUsage() {
	w := output.New()

	w.HelpTitle("structyl mise sync - regenerate mise.toml")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl mise sync")

	w.HelpSection("Description:")
	w.Println("  Regenerates the mise.toml file from project configuration.")
	w.Println("  This file defines tasks and tools for the mise task runner.")
	w.Println("  Always regenerates the file (implicit force mode).")

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl mise sync", "Regenerate mise.toml")
	w.Println("")
}

// printDockerBuildUsage prints the help text for the docker-build command.
func printDockerBuildUsage() {
	w := output.New()

	w.HelpTitle("structyl docker-build - build Docker images")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl docker-build [services...]")

	w.HelpSection("Description:")
	w.Println("  Builds Docker images for the specified services (or all services")
	w.Println("  if none specified). Uses docker compose build under the hood.")

	w.HelpSection("Arguments:")
	w.HelpFlag("[services]", "Service names to build (optional, builds all if omitted)", helpFlagWidthLong)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl docker-build", "Build all Docker images")
	w.HelpExample("structyl docker-build api", "Build only the 'api' service")
	w.Println("")
}

// printDockerCleanUsage prints the help text for the docker-clean command.
func printDockerCleanUsage() {
	w := output.New()

	w.HelpTitle("structyl docker-clean - remove Docker containers and images")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl docker-clean")

	w.HelpSection("Description:")
	w.Println("  Removes Docker containers and images associated with the project.")
	w.Println("  Uses docker compose down --rmi all under the hood.")

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthShort)

	w.HelpSection("Examples:")
	w.HelpExample("structyl docker-clean", "Remove all Docker containers and images")
	w.Println("")
}

// printTargetsUsage prints the help text for the targets command.
func printTargetsUsage() {
	w := output.New()

	w.HelpTitle("structyl targets - list configured targets")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl targets [options]")

	w.HelpSection("Description:")
	w.Println("  Lists all configured targets in the project with their type,")
	w.Println("  title, available commands, and dependencies.")

	w.HelpSection("Options:")
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", helpFlagWidthGlobal)
	w.HelpFlag("-h, --help", "Show this help", helpFlagWidthGlobal)

	w.HelpSection("Examples:")
	w.HelpExample("structyl targets", "List all targets")
	w.HelpExample("structyl targets --type=language", "List only language targets")
	w.Println("")
}
