package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/release"
	"github.com/AndreyAkinshin/structyl/internal/runner"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

// out is the shared output writer for CLI commands.
var out = output.New()

// loadProject loads the project configuration and handles errors uniformly.
// Returns the project and exit code 0 on success, or nil and exit code 1 on failure.
func loadProject() (*project.Project, int) {
	proj, err := project.LoadProject()
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
		return nil, 1
	}
	return proj, 0
}

// cmdUnified handles both target-specific and cross-target commands.
// The first argument is always the command. If a second argument matches a target name,
// it runs the command on that target. Otherwise, it runs on all targets that have it.
func cmdUnified(args []string, opts *GlobalOptions) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "structyl: usage: structyl <command> [target] [args] or structyl <command> [args]\n")
		return 2
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// Print warnings
	for _, w := range proj.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
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
	if _, ok := t.GetCommand(cmd); !ok {
		fmt.Fprintf(os.Stderr, "structyl: [%s] command %q not defined\n", t.Name(), cmd)
		return 1
	}

	ctx := context.Background()
	execOpts := target.ExecOptions{
		Docker: isDockerMode(opts),
		Args:   args,
	}

	out.TargetStart(t.Name(), cmd)
	if err := t.Execute(ctx, cmd, execOpts); err != nil {
		out.TargetFailed(t.Name(), cmd, err)
		return 1
	}

	return 0
}

// runCommandOnAllTargets executes a command on all targets that have it defined.
func runCommandOnAllTargets(registry *target.Registry, cmd string, args []string, opts *GlobalOptions) int {
	// Get targets in dependency order
	targets, err := registry.TopologicalOrder()
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
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

	// Check if any target has this command
	hasCommand := false
	for _, t := range targets {
		if _, ok := t.GetCommand(cmd); ok {
			hasCommand = true
			break
		}
	}

	if !hasCommand {
		fmt.Fprintf(os.Stderr, "structyl: unknown command %q (no target defines it)\n", cmd)
		return 1
	}

	// Execute command for each target
	ctx := context.Background()
	execOpts := target.ExecOptions{
		Docker: isDockerMode(opts),
		Args:   args,
	}

	hasError := false
	for _, t := range targets {
		// Skip if target doesn't have this command
		if _, ok := t.GetCommand(cmd); !ok {
			continue
		}

		out.TargetStart(t.Name(), cmd)
		if err := t.Execute(ctx, cmd, execOpts); err != nil {
			out.TargetFailed(t.Name(), cmd, err)
			hasError = true
			if !opts.ContinueOnError {
				return 1
			}
		}
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
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
		return 2
	}

	targets := registry.All()
	if opts.TargetType != "" {
		targets = filterTargetsByType(targets, target.TargetType(opts.TargetType))
	}

	for _, t := range targets {
		commands := strings.Join(t.Commands(), ", ")
		fmt.Printf("%s (%s): %s\n", t.Name(), t.Type(), t.Title())
		fmt.Printf("  commands: %s\n", commands)
		if deps := t.DependsOn(); len(deps) > 0 {
			fmt.Printf("  depends_on: %s\n", strings.Join(deps, ", "))
		}
	}

	return 0
}

// cmdConfig handles configuration utilities.
func cmdConfig(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "structyl config: subcommand required (validate)\n")
		return 2
	}

	switch args[0] {
	case "validate":
		return cmdConfigValidate()
	default:
		fmt.Fprintf(os.Stderr, "structyl config: unknown subcommand %q\n", args[0])
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
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	// Validate registry creation
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
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

	fmt.Println("Configuration is valid.")
	fmt.Printf("  Project: %s\n", proj.Config.Project.Name)
	fmt.Printf("  Targets: %d (%d language, %d auxiliary)\n", len(targets), langCount, auxCount)
	if len(proj.Warnings) > 0 {
		fmt.Printf("  Warnings: %d\n", len(proj.Warnings))
	}
	return 0
}

// cmdCI runs the CI pipeline.
func cmdCI(cmd string, args []string, opts *GlobalOptions) int {
	// CI pipeline: clean, restore, check, build, test
	commands := []string{"clean", "restore", "check", "build", "test"}

	if cmd == "ci:release" {
		commands = []string{"clean", "restore", "check", "build:release", "test"}
	}

	for _, c := range commands {
		fmt.Printf("=== %s ===\n", c)
		if result := cmdMeta(c, args, opts); result != 0 {
			return result
		}
	}

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

	dockerRunner := runner.NewDockerRunner(proj.Root, proj.Config.Docker)

	// Check Docker availability
	if err := runner.CheckDockerAvailable(); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
		if dockerErr, ok := err.(*runner.DockerUnavailableError); ok {
			return dockerErr.ExitCode()
		}
		return 1
	}

	ctx := context.Background()
	if err := dockerRunner.Build(ctx, args...); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: docker-build failed: %v\n", err)
		return 1
	}

	fmt.Println("Docker images built successfully.")
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
		fmt.Fprintf(os.Stderr, "structyl: %v\n", err)
		if dockerErr, ok := err.(*runner.DockerUnavailableError); ok {
			return dockerErr.ExitCode()
		}
		return 1
	}

	ctx := context.Background()
	if err := dockerRunner.Clean(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: docker-clean failed: %v\n", err)
		return 1
	}

	fmt.Println("Docker containers and images cleaned successfully.")
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
		fmt.Fprintf(os.Stderr, "structyl release: version required\n")
		fmt.Fprintf(os.Stderr, "usage: structyl release <version> [--push] [--dry-run] [--force]\n")
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
		fmt.Fprintf(os.Stderr, "structyl release: %v\n", err)
		return 1
	}

	return 0
}
