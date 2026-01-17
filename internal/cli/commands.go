package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/release"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: intentionally using deprecated package for backwards compatibility
	"github.com/AndreyAkinshin/structyl/internal/target"
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

// cmdUnified handles both target-specific and cross-target commands.
// The first argument is always the command. If a second argument matches a target name,
// it runs the command on that target. Otherwise, it runs on all targets that have it.
func cmdUnified(args []string, opts *GlobalOptions) int {
	applyVerbosityToOutput(opts)

	if len(args) == 0 {
		out.ErrorPrefix("usage: structyl <command> [target] [args] or structyl <command> [args]")
		return 2
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

	// Check if second argument is a target name
	if len(remaining) > 0 {
		if t, ok := registry.Get(remaining[0]); ok {
			// It's a target - run specific command on it
			return runTargetCommand(t, cmd, remaining[1:], opts)
		}
	}

	// No target specified - run command on all targets
	return runCommandOnAllTargets(registry, cmd, remaining, opts)
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
func cmdTargets(opts *GlobalOptions) int {
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
	applyVerbosityToOutput(opts)

	// CI pipeline: clean, restore, check, build, test
	commands := []string{"clean", "restore", "check", "build", "test"}

	if cmd == "ci:release" {
		commands = []string{"clean", "restore", "check", "build:release", "test"}
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

// cmdDockerBuild builds Docker images for services.
func cmdDockerBuild(args []string, opts *GlobalOptions) int {
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
func cmdDockerClean(opts *GlobalOptions) int {
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
