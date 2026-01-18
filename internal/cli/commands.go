package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/AndreyAkinshin/structyl/internal/mise"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/release"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: intentionally using deprecated package for backwards compatibility
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// out is the shared output writer for CLI commands.
var out = output.New()

// getVerbosity converts GlobalOptions to target.Verbosity.
func getVerbosity(opts *GlobalOptions) target.Verbosity {
	if opts.Quiet {
		return target.VerbosityQuiet
	}
	if opts.Verbose {
		return target.VerbosityVerbose
	}
	return target.VerbosityDefault
}

// applyVerbosityToOutput configures the output writer based on verbosity settings.
func applyVerbosityToOutput(opts *GlobalOptions) {
	out.SetQuiet(opts.Quiet)
	out.SetVerbose(opts.Verbose)
}

// targetResult tracks the result of running a command on a single target.
type targetResult struct {
	name     string
	success  bool
	err      error
	duration time.Duration
}

// printCommandSummary prints a summary of command execution across multiple targets.
func printCommandSummary(cmd string, results []targetResult, totalDuration time.Duration) {
	out.SummaryHeader(cmd + " Summary")

	// Print detailed target listing
	out.SummarySectionLabel("Targets:")
	for _, r := range results {
		var errMsg string
		if r.err != nil {
			errMsg = r.err.Error()
		}
		out.SummaryAction(r.name, r.success, runner.FormatDuration(r.duration), errMsg)
	}
	out.Println("")

	// Count results
	var passed, failed int
	var failedNames []string
	for _, r := range results {
		if r.success {
			passed++
		} else {
			failed++
			failedNames = append(failedNames, r.name)
		}
	}

	// Print summary details
	out.SummaryItem("Command", cmd)
	out.SummaryItem("Targets", fmt.Sprintf("%d", len(results)))
	out.SummaryPassed("Passed", fmt.Sprintf("%d", passed))
	if failed > 0 {
		out.SummaryFailed("Failed", fmt.Sprintf("%d (%s)", failed, strings.Join(failedNames, ", ")))
	}
	out.SummaryItem("Duration", runner.FormatDuration(totalDuration))

	// Final message
	if failed == 0 {
		out.FinalSuccess("All %d targets completed %s successfully.", len(results), cmd)
	} else {
		out.FinalFailure("%d of %d targets failed.", failed, len(results))
	}
}

// loadProject loads the project configuration and handles errors uniformly.
// Returns the project and exit code 0 on success, or nil and exit code 1 on failure.
func loadProject() (*project.Project, int) {
	proj, err := project.LoadProject()
	if err != nil {
		out.ErrorPrefix("%v", err)
		return nil, 1
	}
	return proj, 0
}

// isMiseEnabled checks if mise integration is enabled for the project.
// Returns true by default (mise.enabled defaults to true).
func isMiseEnabled(proj *project.Project) bool {
	if proj.Config.Mise == nil {
		return true // Default to enabled
	}
	// If Mise config exists but Enabled is not set, default to true
	// Only return false if explicitly set to false
	return proj.Config.Mise.Enabled || proj.Config.Mise == nil
}

// ensureMiseConfig ensures mise.toml is up-to-date.
// If auto_generate is enabled, regenerates the file.
// If force is true, always regenerates.
func ensureMiseConfig(proj *project.Project, force bool) error {
	// Check if auto-generate is enabled
	autoGen := true // default
	if proj.Config.Mise != nil {
		autoGen = proj.Config.Mise.AutoGenerate
	}

	if force || autoGen || !mise.MiseTomlExists(proj.Root) {
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
func runViaMise(proj *project.Project, cmd string, targetName string, args []string, opts *GlobalOptions) int {
	ctx := context.Background()

	// Format task name
	task := formatMiseTaskName(cmd, targetName)

	// Create executor
	executor := mise.NewExecutor(proj.Root)
	executor.SetVerbose(opts.Verbose)

	// Try to resolve dependencies to check if this is an aggregate task
	tasks, err := executor.ResolveTaskDependencies(ctx, task)
	if err != nil {
		// Fall back to direct execution if dependency resolution fails
		if err := executor.RunTask(ctx, task, args); err != nil {
			return 1
		}
		return 0
	}

	// If single task with no dependencies, use direct execution (no tracking overhead)
	if len(tasks) == 1 {
		if err := executor.RunTask(ctx, task, args); err != nil {
			return 1
		}
		return 0
	}

	// Multiple tasks means root task has dependencies.
	// Run ONLY the root task directly - mise will handle parallel execution of dependencies.
	// Running individual dependency tasks first would cause duplicate execution
	// (once by us sequentially, once by mise in parallel when running root task).
	if err := executor.RunTask(ctx, task, args); err != nil {
		return 1
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
		return 2
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

	// Print warnings
	for _, w := range proj.Warnings {
		out.WarningSimple("%s", w)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return 2
	}

	// First argument is always the command
	cmd := args[0]
	remaining := args[1:]

	// Determine target name (if specified)
	var targetName string
	var cmdArgs []string

	if len(remaining) > 0 {
		if _, ok := registry.Get(remaining[0]); ok {
			// It's a target
			targetName = remaining[0]
			cmdArgs = remaining[1:]
		} else {
			cmdArgs = remaining
		}
	}

	// Use mise if enabled and not in Docker mode
	if isMiseEnabled(proj) && !isDockerMode(opts) {
		// Check mise is installed
		if err := EnsureMise(true); err != nil {
			out.ErrorPrefix("%v", err)
			PrintMiseInstallInstructions()
			return 1
		}

		// Ensure mise.toml is up-to-date
		if err := ensureMiseConfig(proj, false); err != nil {
			out.ErrorPrefix("%v", err)
			return 1
		}

		return runViaMise(proj, cmd, targetName, cmdArgs, opts)
	}

	// Fallback to direct execution (Docker mode or mise disabled)
	if targetName != "" {
		if t, ok := registry.Get(targetName); ok {
			return runTargetCommand(t, cmd, cmdArgs, opts)
		}
	}

	// No target specified - run command on all targets
	return runCommandOnAllTargets(registry, cmd, cmdArgs, opts)
}

// runTargetCommand executes a command on a specific target.
func runTargetCommand(t target.Target, cmd string, args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	if _, ok := t.GetCommand(cmd); !ok {
		out.ErrorPrefix("[%s] command %q not defined", t.Name(), cmd)
		return 1
	}

	ctx := context.Background()
	execOpts := target.ExecOptions{
		Docker:    isDockerMode(opts),
		Args:      args,
		Verbosity: getVerbosity(opts),
	}

	out.TargetStart(t.Name(), cmd)
	if err := t.Execute(ctx, cmd, execOpts); err != nil {
		out.TargetFailed(t.Name(), cmd, err)
		return 1
	}

	out.TargetSuccess(t.Name(), cmd)
	return 0
}

// runCommandOnAllTargets executes a command on all targets that have it defined.
func runCommandOnAllTargets(registry *target.Registry, cmd string, args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	// Get targets in dependency order
	targets, err := registry.TopologicalOrder()
	if err != nil {
		out.ErrorPrefix("%v", err)
		return 2
	}

	// Filter by type if specified
	if opts.TargetType != "" {
		targets = filterTargetsByType(targets, target.TargetType(opts.TargetType))
	}

	// For test command, filter to language targets only by default
	if cmd == "test" && opts.TargetType == "" {
		targets = filterTargetsByType(targets, target.TypeLanguage)
	}

	// Count targets with this command
	var targetsWithCommand []target.Target
	for _, t := range targets {
		if _, ok := t.GetCommand(cmd); ok {
			targetsWithCommand = append(targetsWithCommand, t)
		}
	}

	if len(targetsWithCommand) == 0 {
		out.ErrorPrefix("unknown command %q (no target defines it)", cmd)
		return 1
	}

	// Execute command for each target
	ctx := context.Background()
	execOpts := target.ExecOptions{
		Docker:    isDockerMode(opts),
		Args:      args,
		Verbosity: getVerbosity(opts),
	}

	startTime := time.Now()
	var results []targetResult
	hasError := false

	for _, t := range targetsWithCommand {
		targetStart := time.Now()
		out.TargetStart(t.Name(), cmd)

		err := t.Execute(ctx, cmd, execOpts)
		targetDuration := time.Since(targetStart)

		if err != nil {
			out.TargetFailed(t.Name(), cmd, err)
			results = append(results, targetResult{
				name:     t.Name(),
				success:  false,
				err:      err,
				duration: targetDuration,
			})
			hasError = true
			if !opts.ContinueOnError {
				// Print summary even on early exit if we ran multiple targets
				if len(results) > 1 {
					printCommandSummary(cmd, results, time.Since(startTime))
				}
				return 1
			}
		} else {
			out.TargetSuccess(t.Name(), cmd)
			results = append(results, targetResult{
				name:     t.Name(),
				success:  true,
				duration: targetDuration,
			})
		}
	}

	// Print summary if multiple targets were run
	if len(results) > 1 {
		printCommandSummary(cmd, results, time.Since(startTime))
	}

	if hasError {
		return 1
	}
	return 0
}

// cmdMeta executes a command across all targets (used by cmdCI).
func cmdMeta(cmd string, args []string, opts *GlobalOptions) int {
	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return 2
	}

	return runCommandOnAllTargets(registry, cmd, args, opts)
}

// cmdTargets lists all configured targets.
func cmdTargets(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printTargetsUsage()
		return 0
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return 2
	}

	targets := registry.All()
	if opts.TargetType != "" {
		targets = filterTargetsByType(targets, target.TargetType(opts.TargetType))
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
		return 2
	}

	switch args[0] {
	case "validate":
		return cmdConfigValidate()
	case "-h", "--help":
		printConfigUsage()
		return 0
	default:
		out.ErrorPrefix("config: unknown subcommand %q", args[0])
		return 2
	}
}

func cmdConfigValidate() int {
	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Print warnings
	for _, w := range proj.Warnings {
		out.WarningSimple("%s", w)
	}

	// Validate registry creation
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		out.ErrorPrefix("%v", err)
		return 2
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
	if len(args) > 0 {
		registry, err := target.NewRegistry(proj.Config, proj.Root)
		if err == nil {
			if _, ok := registry.Get(args[0]); ok {
				targetName = args[0]
				cmdArgs = args[1:]
			} else {
				cmdArgs = args
			}
		}
	}

	// Use mise if enabled and not in Docker mode
	if isMiseEnabled(proj) && !isDockerMode(opts) {
		// Check mise is installed
		if err := EnsureMise(true); err != nil {
			out.ErrorPrefix("%v", err)
			PrintMiseInstallInstructions()
			return 1
		}

		// Ensure mise.toml is up-to-date
		if err := ensureMiseConfig(proj, false); err != nil {
			out.ErrorPrefix("%v", err)
			return 1
		}

		return runViaMise(proj, cmd, targetName, cmdArgs, opts)
	}

	// Fallback to direct execution (Docker mode or mise disabled)
	// Get CI pipeline from config
	defaults := toolchain.GetDefaultToolchains()
	pipelineName := "ci"
	if cmd == "ci:release" {
		pipelineName = "ci:release"
	}
	commands := toolchain.GetPipeline(defaults, pipelineName)
	if len(commands) == 0 {
		// Fallback defaults
		if cmd == "ci:release" {
			commands = []string{"clean", "restore", "check", "build:release", "test"}
		} else {
			commands = []string{"clean", "restore", "check", "build", "test"}
		}
	}

	startTime := time.Now()
	ciResult := &runner.CIResult{
		StartTime:     startTime,
		PhaseResults:  make([]runner.PhaseResult, 0, len(commands)),
		TargetResults: make(map[string]runner.TargetResult),
		Success:       true,
	}

	for _, c := range commands {
		phaseStart := time.Now()
		out.PhaseHeader(c)

		exitCode := cmdMeta(c, args, opts)
		phaseDuration := time.Since(phaseStart)

		phaseResult := runner.PhaseResult{
			Name:      c,
			StartTime: phaseStart,
			EndTime:   time.Now(),
			Duration:  phaseDuration,
			Success:   exitCode == 0,
		}
		ciResult.PhaseResults = append(ciResult.PhaseResults, phaseResult)

		if exitCode != 0 {
			ciResult.Success = false
			ciResult.EndTime = time.Now()
			ciResult.Duration = time.Since(startTime)
			runner.PrintCISummary(ciResult, out)
			return exitCode
		}
	}

	ciResult.EndTime = time.Now()
	ciResult.Duration = time.Since(startTime)
	runner.PrintCISummary(ciResult, out)

	return 0
}

// filterTargetsByType returns only targets of the specified type.
func filterTargetsByType(targets []target.Target, targetType target.TargetType) []target.Target {
	var filtered []target.Target
	for _, t := range targets {
		if t.Type() == targetType {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// cmdMise handles mise-related subcommands.
func cmdMise(args []string, opts *GlobalOptions) int {
	if len(args) == 0 {
		out.ErrorPrefix("mise: subcommand required (sync)")
		out.Println("usage: structyl mise sync [--force]")
		return 2
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
		out.Println("usage: structyl mise sync [--force]")
		return 2
	}
}

// cmdMiseSync regenerates the mise.toml file.
func cmdMiseSync(args []string, opts *GlobalOptions) int {
	if wantsHelp(args) {
		printMiseSyncUsage()
		return 0
	}

	// Parse flags
	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			if strings.HasPrefix(arg, "-") {
				out.ErrorPrefix("mise sync: unknown option %q", arg)
				return 2
			}
		}
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Generate mise.toml using loaded toolchains
	created, err := mise.WriteMiseTomlWithToolchains(proj.Root, proj.Config, proj.Toolchains, force || true) // Always force on explicit sync
	if err != nil {
		out.ErrorPrefix("mise sync: %v", err)
		return 1
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

// cmdDockerBuild builds Docker images for services.
func cmdDockerBuild(args []string, opts *GlobalOptions) int {
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

	// Check Docker availability
	if err := runner.CheckDockerAvailable(); err != nil {
		out.ErrorPrefix("%v", err)
		if dockerErr, ok := err.(*runner.DockerUnavailableError); ok {
			return dockerErr.ExitCode()
		}
		return 1
	}

	ctx := context.Background()
	if err := dockerRunner.Build(ctx, args...); err != nil {
		out.ErrorPrefix("docker-build failed: %v", err)
		return 1
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

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	dockerRunner := runner.NewDockerRunner(proj.Root, proj.Config.Docker)

	// Check Docker availability
	if err := runner.CheckDockerAvailable(); err != nil {
		out.ErrorPrefix("%v", err)
		if dockerErr, ok := err.(*runner.DockerUnavailableError); ok {
			return dockerErr.ExitCode()
		}
		return 1
	}

	ctx := context.Background()
	if err := dockerRunner.Clean(ctx); err != nil {
		out.ErrorPrefix("docker-clean failed: %v", err)
		return 1
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
		return 2
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
		return 1
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
	w.HelpFlag("[target]", "Target name to run command on (optional)", 10)

	w.HelpSection("Global Options:")
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", 14)
	w.HelpFlag("-v, --verbose", "Maximum detail", 14)
	w.HelpFlag("--docker", "Run in Docker container", 14)
	w.HelpFlag("--no-docker", "Disable Docker mode", 14)
	w.HelpFlag("--continue", "Continue on error (don't fail-fast)", 14)
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", 14)
	w.HelpFlag("-h, --help", "Show this help", 14)

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
	w.HelpFlag("<version>", "Version number (e.g., 1.2.3)", 10)

	w.HelpSection("Options:")
	w.HelpFlag("--push", "Push to remote with tags after commit", 10)
	w.HelpFlag("--dry-run", "Print what would be done without making changes", 10)
	w.HelpFlag("--force", "Force release with uncommitted changes", 10)
	w.HelpFlag("-h, --help", "Show this help", 10)

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
	w.HelpFlag("[target]", "Target name to run CI on (optional)", 10)

	w.HelpSection("Global Options:")
	w.HelpFlag("-q, --quiet", "Minimal output (errors only)", 14)
	w.HelpFlag("-v, --verbose", "Maximum detail", 14)
	w.HelpFlag("--docker", "Run in Docker container", 14)
	w.HelpFlag("--no-docker", "Disable Docker mode", 14)
	w.HelpFlag("--continue", "Continue on error (don't fail-fast)", 14)
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", 14)
	w.HelpFlag("-h, --help", "Show this help", 14)

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
	w.HelpCommand("validate", "Validate the project configuration", 10)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", 10)

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
	w.HelpCommand("sync", "Regenerate mise.toml from project configuration", 6)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", 10)

	w.HelpSection("Examples:")
	w.HelpExample("structyl mise sync", "Regenerate mise.toml")
	w.HelpExample("structyl mise sync --force", "Force regenerate mise.toml")
	w.Println("")
}

// printMiseSyncUsage prints the help text for the mise sync command.
func printMiseSyncUsage() {
	w := output.New()

	w.HelpTitle("structyl mise sync - regenerate mise.toml")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl mise sync [--force]")

	w.HelpSection("Description:")
	w.Println("  Regenerates the mise.toml file from project configuration.")
	w.Println("  This file defines tasks and tools for the mise task runner.")

	w.HelpSection("Options:")
	w.HelpFlag("--force", "Force regeneration even if file exists", 10)
	w.HelpFlag("-h, --help", "Show this help", 10)

	w.HelpSection("Examples:")
	w.HelpExample("structyl mise sync", "Regenerate mise.toml")
	w.HelpExample("structyl mise sync --force", "Force regenerate mise.toml")
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
	w.HelpFlag("[services]", "Service names to build (optional, builds all if omitted)", 12)

	w.HelpSection("Options:")
	w.HelpFlag("-h, --help", "Show this help", 10)

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
	w.HelpFlag("-h, --help", "Show this help", 10)

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
	w.HelpFlag("--type=<type>", "Filter targets by type (language or auxiliary)", 14)
	w.HelpFlag("-h, --help", "Show this help", 14)

	w.HelpSection("Examples:")
	w.HelpExample("structyl targets", "List all targets")
	w.HelpExample("structyl targets --type=language", "List only language targets")
	w.Println("")
}
