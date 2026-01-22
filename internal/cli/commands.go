package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/AndreyAkinshin/structyl/internal/config"
	internalerrors "github.com/AndreyAkinshin/structyl/internal/errors"
	"github.com/AndreyAkinshin/structyl/internal/mise"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/release"

	// nolint:staticcheck // SA1019: runner package is deprecated but still required for Docker
	// functionality (DockerRunner, DockerUnavailableError, CheckDockerAvailable). These types
	// are used by docker-build/docker-clean commands and will be removed with the runner package.
	"github.com/AndreyAkinshin/structyl/internal/runner"
	"github.com/AndreyAkinshin/structyl/internal/schema"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

var out = output.New()

// Help text alignment widths for consistent formatting.
// These values align the flag/command names with their descriptions.
// Each width accommodates the longest string in its category plus padding.
const (
	widthFlagShort      = 10 // "-h, --help"
	widthArgPlaceholder = 12 // "[services]"
	widthFlagWithValue  = 14 // "--type=<type>"
	widthSubcommand     = 6  // "sync"
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

// isMiseAutoGenerateEnabled returns true if auto_generate is enabled.
// Auto-generation is enabled by default (true) when:
// - mise config is nil (not specified)
// - mise.auto_generate is nil (not specified)
// - mise.auto_generate is explicitly true
func isMiseAutoGenerateEnabled(cfg *config.Config) bool {
	const defaultAutoGenerate = true
	if cfg.Mise == nil || cfg.Mise.AutoGenerate == nil {
		return defaultAutoGenerate
	}
	return *cfg.Mise.AutoGenerate
}

// ensureMiseConfig ensures mise.toml is up-to-date.
// Regenerates when: forced, auto_generate enabled (default true), or file missing.
func ensureMiseConfig(proj *project.Project, mode MiseRegenerateMode) error {
	switch mode {
	case MiseForceRegenerate:
		return writeMiseConfig(proj)
	case MiseAutoRegenerate:
		if !mise.MiseTomlExists(proj.Root) || isMiseAutoGenerateEnabled(proj.Config) {
			return writeMiseConfig(proj)
		}
		return nil
	default:
		panic(fmt.Sprintf("BUG: invalid MiseRegenerateMode: %d", mode))
	}
}

// writeMiseConfig writes the mise.toml file from project configuration.
func writeMiseConfig(proj *project.Project) error {
	_, err := mise.WriteMiseTomlWithToolchains(proj.Root, proj.Config, proj.Toolchains, mise.WriteAlways)
	if err != nil {
		return fmt.Errorf("failed to generate mise.toml: %w", err)
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
//
// The registry parameter is optional (may be nil). When provided, it enables
// typo correction hints on failure—if the user typed a target name instead
// of a command name (e.g., "structyl cs" instead of "structyl build cs"),
// the hint suggests the correct syntax.
func runViaMise(proj *project.Project, cmd string, targetName string, args []string, opts *GlobalOptions, registry *target.Registry) int {
	ctx := context.Background()

	task := formatMiseTaskName(cmd, targetName)

	executor := mise.NewExecutor(proj.Root)
	executor.SetVerbose(opts.Verbose)

	if err := executor.RunTask(ctx, task, args); err != nil {
		maybeHintTypoCorrection(cmd, targetName, registry)
		return internalerrors.ExitRuntimeError
	}
	return 0
}

// maybeHintTypoCorrection suggests a correction when the user may have typed
// a target name instead of a command name (e.g., "structyl cs" instead of
// "structyl build cs").
func maybeHintTypoCorrection(cmd, targetName string, registry *target.Registry) {
	if registry == nil || targetName != "" {
		return
	}
	if _, exists := registry.Get(cmd); exists {
		out.Hint("Did you mean 'structyl build %s'?", cmd)
	}
}

// cmdUnified handles both target-specific and cross-target commands.
// The first argument is always the command. If a second argument matches a target name,
// it runs the command on that target. Otherwise, it runs on all targets that have it.
func cmdUnified(args []string, opts *GlobalOptions) int {
	if len(args) == 0 {
		out.ErrorPrefix("usage: structyl <command> [target] [args] or structyl <command> [args]")
		return internalerrors.ExitConfigError
	}

	// Check for help flag early (after command name)
	if len(args) > 1 && wantsHelp(args[1:]) {
		printUnifiedUsage(args[0])
		return 0
	}

	proj, registry, exitCode := loadProjectWithRegistry()
	if proj == nil {
		return exitCode
	}

	printProjectWarnings(proj)

	cmd := args[0]
	remaining := args[1:]

	// Determine target name (if specified)
	targetName, cmdArgs := extractTargetArg(remaining, registry)

	// Check mise is installed
	if err := EnsureMise(true); err != nil {
		out.ErrorPrefix("%v", err)
		PrintMiseInstallInstructions()
		return internalerrors.ExitEnvError
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

	// Parse --json flag
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "-") {
				out.ErrorPrefix("targets: unknown option %q", arg)
				return internalerrors.ExitConfigError
			}
			out.ErrorPrefix("targets: unexpected argument %q", arg)
			return internalerrors.ExitConfigError
		}
	}

	_, registry, exitCode := loadProjectWithRegistry()
	if registry == nil {
		return exitCode
	}

	var targets []target.Target
	if opts.TargetType != "" {
		targets = registry.ByType(target.TargetType(opts.TargetType))
		if len(targets) == 0 {
			if jsonOutput {
				fmt.Println("[]")
				return 0
			}
			out.WarningSimple("no targets of type %q found", opts.TargetType)
			return 0
		}
	} else {
		targets = registry.All()
	}

	if jsonOutput {
		return printTargetsJSON(targets)
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

// TargetJSON represents a target in JSON output format.
// This structure is stable and part of the public CLI API.
// All fields are always present in the JSON output for consistent parsing.
type TargetJSON struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Title     string   `json:"title"`      // Always present (required in config schema)
	Commands  []string `json:"commands"`   // Always present (may be empty array)
	DependsOn []string `json:"depends_on"` // Always present (may be empty array)
}

// nonNilStrings ensures a string slice is never nil (returns empty slice instead).
// This guarantees consistent JSON serialization: Go's encoding/json marshals
// nil slices as null but non-nil empty slices as []. For machine-readable
// output (--json flag), [] is more predictable for consumers parsing the output.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// targetToJSON converts a Target to its JSON representation.
func targetToJSON(t target.Target) TargetJSON {
	return TargetJSON{
		Name:      t.Name(),
		Type:      string(t.Type()),
		Title:     t.Title(),
		Commands:  nonNilStrings(t.Commands()),
		DependsOn: nonNilStrings(t.DependsOn()),
	}
}

// printTargetsJSON outputs targets in machine-readable JSON format.
func printTargetsJSON(targets []target.Target) int {
	result := make([]TargetJSON, 0, len(targets))
	for _, t := range targets {
		result = append(result, targetToJSON(t))
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		out.ErrorPrefix("failed to marshal targets to JSON: %v", err)
		return internalerrors.ExitRuntimeError
	}
	fmt.Println(string(data))
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
		return internalerrors.ExitConfigError
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

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Determine target name from args.
	// CI command tolerates registry load failures (as a warning) because it can still
	// run mise tasks directly. This differs from standard commands (cmdUnified) which
	// require the registry to resolve target-specific commands.
	var targetName string
	var cmdArgs []string
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.WarningSimple("could not load target registry: %v", err)
		cmdArgs = args
	} else {
		targetName, cmdArgs = extractTargetArg(args, registry)
	}

	// Check mise is installed
	if err := EnsureMise(true); err != nil {
		out.ErrorPrefix("%v", err)
		PrintMiseInstallInstructions()
		return internalerrors.ExitEnvError
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
//
// # Heuristic Behavior
//
// This function uses a heuristic: if the first argument matches a registered target name,
// it is interpreted as the target. Otherwise, it's treated as a command argument.
//
// This enables convenient syntax like "structyl build go" to build the "go" target.
// However, it means that if a user wants to pass an argument that happens to match
// a target name (e.g., passing "go" as an argument), they should use the explicit
// separator: "structyl build -- go".
//
// The target interpretation always wins when ambiguous. This is intentional to
// support the common case of targeting a specific implementation.
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
	created, err := mise.WriteMiseTomlWithToolchains(proj.Root, proj.Config, proj.Toolchains, mise.WriteAlways)
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

// prepareDockerCommand handles common setup for Docker commands: project
// loading and Docker availability check. Returns the project on success, or nil
// with an exit code on failure.
func prepareDockerCommand(opts *GlobalOptions) (*project.Project, int) {
	proj, exitCode := loadProject()
	if proj == nil {
		return nil, exitCode
	}

	if err := runner.CheckDockerAvailable(); err != nil {
		return nil, handleDockerError(err)
	}

	return proj, 0
}

// cmdDockerBuild builds Docker images for services.
func cmdDockerBuild(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printDockerBuildUsage()
		return 0
	}

	proj, exitCode := prepareDockerCommand(opts)
	if proj == nil {
		return exitCode
	}

	// Use the full config runner to support per-target Dockerfiles
	dockerRunner := runner.NewDockerRunnerWithConfig(proj.Root, proj.Config)

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
	if wantsHelp(args) {
		printDockerCleanUsage()
		return 0
	}

	proj, exitCode := prepareDockerCommand(opts)
	if proj == nil {
		return exitCode
	}

	dockerRunner := runner.NewDockerRunner(proj.Root, proj.Config.Docker)

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

	for _, arg := range args {
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
	defaults := toolchain.GetDefaultToolchains()
	desc := toolchain.GetCommandDescription(defaults, cmd)
	if desc == "" {
		desc = fmt.Sprintf("run %s", cmd)
	} else {
		// Convert to lowercase for help text
		desc = strings.ToLower(desc)
	}

	out.HelpTitle(fmt.Sprintf("structyl %s - %s", cmd, desc))

	out.HelpSection("Usage:")
	out.HelpUsage(fmt.Sprintf("structyl %s [target] [options]", cmd))

	out.HelpSection("Description:")
	out.Println("  When a target is specified, runs %s on that target only.", cmd)
	out.Println("  Without a target, runs %s on all targets that have it defined.", cmd)

	out.HelpSection("Arguments:")
	out.HelpFlag("[target]", "Target name to run command on (optional)", widthFlagShort)

	out.HelpSection("Global Options:")
	out.HelpFlag("-q, --quiet", "Minimal output (errors only)", widthFlagWithValue)
	out.HelpFlag("-v, --verbose", "Maximum detail", widthFlagWithValue)
	out.HelpFlag("--docker", "Run in Docker container", widthFlagWithValue)
	out.HelpFlag("--no-docker", "Disable Docker mode", widthFlagWithValue)
	out.Println("                  (precedence: --no-docker > --docker > STRUCTYL_DOCKER > default)")
	out.HelpFlag("--type=<type>", `Filter targets by type ("language" or "auxiliary")`, widthFlagWithValue)
	out.HelpFlag("-h, --help", "Show this help", widthFlagWithValue)

	out.HelpSection("Examples:")
	titleCase := cases.Title(language.English)
	out.HelpExample(fmt.Sprintf("structyl %s", cmd), fmt.Sprintf("%s all targets", titleCase.String(cmd)))
	out.HelpExample(fmt.Sprintf("structyl %s go", cmd), fmt.Sprintf("%s Go target only", titleCase.String(cmd)))
	out.HelpExample(fmt.Sprintf("structyl %s --docker", cmd), fmt.Sprintf("%s all targets in Docker", titleCase.String(cmd)))
	out.Println("")
}

// printReleaseUsage prints the help text for the release command.
func printReleaseUsage() {
	out.HelpTitle("structyl release - create a release")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl release <version> [options]")

	out.HelpSection("Description:")
	out.Println("  Creates a release by setting the version across all targets,")
	out.Println("  committing the changes, and optionally pushing to the remote.")

	out.HelpSection("Arguments:")
	out.HelpFlag("<version>", "Semantic version (X.Y.Z or X.Y.Z-prerelease)", widthFlagShort)

	out.HelpSection("Options:")
	out.HelpFlag("--push", "Push to remote with tags after commit", widthFlagShort)
	out.HelpFlag("--dry-run", "Print what would be done without making changes", widthFlagShort)
	out.HelpFlag("--force", "Force release with uncommitted changes", widthFlagShort)
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl release 1.2.3", "Create release 1.2.3")
	out.HelpExample("structyl release 1.2.3 --push", "Create and push release 1.2.3")
	out.HelpExample("structyl release 1.2.3 --dry-run", "Preview release without changes")
	out.Println("")
}

// printCIUsage prints the help text for the ci and ci:release commands.
func printCIUsage(cmd string) {
	if cmd == "ci:release" {
		out.HelpTitle("structyl ci:release - run CI pipeline with release builds")
	} else {
		out.HelpTitle("structyl ci - run CI pipeline")
	}

	out.HelpSection("Usage:")
	out.HelpUsage(fmt.Sprintf("structyl %s [target] [options]", cmd))

	out.HelpSection("Description:")
	if cmd == "ci:release" {
		out.Println("  Runs the CI pipeline with release builds: clean, restore, check,")
		out.Println("  build:release, test. When a target is specified, runs only for")
		out.Println("  that target.")
	} else {
		out.Println("  Runs the CI pipeline: clean, restore, check, build, test.")
		out.Println("  When a target is specified, runs only for that target.")
	}

	out.HelpSection("Arguments:")
	out.HelpFlag("[target]", "Target name to run CI on (optional)", widthFlagShort)

	out.HelpSection("Global Options:")
	out.HelpFlag("-q, --quiet", "Minimal output (errors only)", widthFlagWithValue)
	out.HelpFlag("-v, --verbose", "Maximum detail", widthFlagWithValue)
	out.HelpFlag("--docker", "Run in Docker container", widthFlagWithValue)
	out.HelpFlag("--no-docker", "Disable Docker mode", widthFlagWithValue)
	out.Println("                  (precedence: --no-docker > --docker > STRUCTYL_DOCKER > default)")
	out.HelpFlag("--type=<type>", `Filter targets by type ("language" or "auxiliary")`, widthFlagWithValue)
	out.HelpFlag("-h, --help", "Show this help", widthFlagWithValue)

	out.HelpSection("Examples:")
	out.HelpExample(fmt.Sprintf("structyl %s", cmd), "Run CI on all targets")
	out.HelpExample(fmt.Sprintf("structyl %s go", cmd), "Run CI on Go target only")
	out.HelpExample(fmt.Sprintf("structyl %s --docker", cmd), "Run CI in Docker")
	out.Println("")
}

// printConfigUsage prints the help text for the config command.
func printConfigUsage() {
	out.HelpTitle("structyl config - configuration utilities")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl config <subcommand>")

	out.HelpSection("Subcommands:")
	out.HelpCommand("validate", "Validate the project configuration", widthFlagShort)

	out.HelpSection("Options:")
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl config validate", "Validate project configuration")
	out.Println("")
}

// printMiseUsage prints the help text for the mise command.
func printMiseUsage() {
	out.HelpTitle("structyl mise - mise integration commands")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl mise <subcommand>")

	out.HelpSection("Subcommands:")
	out.HelpCommand("sync", "Regenerate mise.toml from project configuration", widthSubcommand)

	out.HelpSection("Options:")
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl mise sync", "Regenerate mise.toml")
	out.Println("")
}

// printMiseSyncUsage prints the help text for the mise sync command.
func printMiseSyncUsage() {
	out.HelpTitle("structyl mise sync - regenerate mise.toml")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl mise sync")

	out.HelpSection("Description:")
	out.Println("  Regenerates the mise.toml file from project configuration.")
	out.Println("  This file defines tasks and tools for the mise task runner.")
	out.Println("  Always regenerates the file (implicit force mode).")

	out.HelpSection("Options:")
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl mise sync", "Regenerate mise.toml")
	out.Println("")
}

// printDockerBuildUsage prints the help text for the docker-build command.
func printDockerBuildUsage() {
	out.HelpTitle("structyl docker-build - build Docker images")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl docker-build [services...]")

	out.HelpSection("Description:")
	out.Println("  Builds Docker images for the specified services (or all services")
	out.Println("  if none specified). Uses docker compose build under the hood.")

	out.HelpSection("Arguments:")
	out.HelpFlag("[services]", "Service names to build (optional, builds all if omitted)", widthArgPlaceholder)

	out.HelpSection("Options:")
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl docker-build", "Build all Docker images")
	out.HelpExample("structyl docker-build api", "Build only the 'api' service")
	out.Println("")
}

// printDockerCleanUsage prints the help text for the docker-clean command.
func printDockerCleanUsage() {
	out.HelpTitle("structyl docker-clean - remove Docker containers and images")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl docker-clean")

	out.HelpSection("Description:")
	out.Println("  Removes Docker containers and images associated with the project.")
	out.Println("  Uses docker compose down --rmi all under the hood.")

	out.HelpSection("Options:")
	out.HelpFlag("-h, --help", "Show this help", widthFlagShort)

	out.HelpSection("Examples:")
	out.HelpExample("structyl docker-clean", "Remove all Docker containers and images")
	out.Println("")
}

// printTargetsUsage prints the help text for the targets command.
func printTargetsUsage() {
	out.HelpTitle("structyl targets - list configured targets")

	out.HelpSection("Usage:")
	out.HelpUsage("structyl targets [options]")

	out.HelpSection("Description:")
	out.Println("  Lists all configured targets in the project with their type,")
	out.Println("  title, available commands, and dependencies.")

	out.HelpSection("Options:")
	out.HelpFlag("--json", "Output in machine-readable JSON format", widthFlagWithValue)
	out.HelpFlag("--type=<type>", `Filter targets by type ("language" or "auxiliary")`, widthFlagWithValue)
	out.HelpFlag("-h, --help", "Show this help", widthFlagWithValue)

	out.HelpSection("Examples:")
	out.HelpExample("structyl targets", "List all targets")
	out.HelpExample("structyl targets --json", "List targets as JSON")
	out.HelpExample("structyl targets --type=language", "List only language targets")
	out.Println("")
}
