// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"

	structylerrors "github.com/AndreyAkinshin/structyl/internal/errors"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

// out is the shared output writer for runner messages.
var out = output.New()

const (
	// minParallelWorkers ensures at least one worker to prevent semaphore deadlock,
	// even if runtime.NumCPU() returns 0 (which can happen in containerized or
	// restricted environments where CPU detection fails).
	minParallelWorkers = 1

	// maxParallelWorkers caps STRUCTYL_PARALLEL at 256 workers. Beyond this limit,
	// goroutine scheduling overhead typically outweighs parallelism benefits for
	// structyl's I/O-bound target execution (subprocess spawning, file system ops).
	// On typical systems (16-128 cores), 256 provides ample headroom without
	// risking scheduler thrashing or excessive memory usage from blocked goroutines.
	maxParallelWorkers = 256
)

// Runner orchestrates command execution across multiple targets.
// It handles dependency ordering for sequential runs and parallel
// execution via a worker pool. The Runner uses a target.Registry
// to resolve targets and their commands.
type Runner struct {
	registry *target.Registry
}

// RunOptions configures execution behavior.
//
// NOTE: This type is part of the internal Runner API. CLI commands use mise
// for orchestration and do not expose Continue or Parallel options to users.
// These fields exist for internal use (tests, direct API calls) only.
type RunOptions struct {
	Docker bool // Run in Docker container

	// Continue controls whether execution continues after a target fails.
	// INTERNAL USE ONLY: The CLI --continue flag was removed; mise backend
	// always stops on first failure. This field exists for direct Runner API
	// calls in tests.
	Continue bool

	// Parallel enables concurrent target execution with a worker pool.
	// INTERNAL USE ONLY: Does NOT respect depends_on ordering—targets may
	// execute before their dependencies complete. CLI commands use mise for
	// dependency-aware parallel execution.
	Parallel  bool
	Args      []string          // Arguments to pass to commands
	Env       map[string]string // Additional environment variables
	Verbosity target.Verbosity  // Output verbosity level
}

// New creates a new Runner.
func New(registry *target.Registry) *Runner {
	return &Runner{registry: registry}
}

// Run executes a command on a single target.
func (r *Runner) Run(ctx context.Context, targetName, cmd string, opts RunOptions) error {
	t, ok := r.registry.Get(targetName)
	if !ok {
		return structylerrors.NotFound("target", targetName)
	}

	execOpts := target.ExecOptions{
		Args:      opts.Args,
		Env:       opts.Env,
		Verbosity: opts.Verbosity,
	}

	return t.Execute(ctx, cmd, execOpts)
}

// RunAll executes a command on all targets in dependency order.
func (r *Runner) RunAll(ctx context.Context, cmd string, opts RunOptions) error {
	targets, err := r.registry.TopologicalOrder()
	if err != nil {
		return err
	}

	filtered := filterByCommand(targets, cmd)

	if len(filtered) == 0 {
		out.WarningSimple("no targets support command %q", cmd)
		return nil
	}

	if opts.Parallel {
		return r.runParallel(ctx, filtered, cmd, opts)
	}

	return r.runSequential(ctx, filtered, cmd, opts)
}

// RunTargets executes a command on specific targets in dependency order.
func (r *Runner) RunTargets(ctx context.Context, targetNames []string, cmd string, opts RunOptions) error {
	// Get targets in order
	allTargets, err := r.registry.TopologicalOrder()
	if err != nil {
		return err
	}

	// Filter to requested targets
	targetSet := make(map[string]bool)
	for _, name := range targetNames {
		targetSet[name] = true
	}

	var requestedTargets []target.Target
	for _, t := range allTargets {
		if targetSet[t.Name()] {
			requestedTargets = append(requestedTargets, t)
		}
	}

	filtered := filterByCommand(requestedTargets, cmd)

	if len(filtered) == 0 {
		out.WarningSimple("no targets support command %q", cmd)
		return nil
	}

	if opts.Parallel {
		return r.runParallel(ctx, filtered, cmd, opts)
	}

	return r.runSequential(ctx, filtered, cmd, opts)
}

// runSequential executes targets one at a time in order.
func (r *Runner) runSequential(ctx context.Context, targets []target.Target, cmd string, opts RunOptions) error {
	execOpts := target.ExecOptions{
		Args:      opts.Args,
		Env:       opts.Env,
		Verbosity: opts.Verbosity,
	}

	var errs []error
	for _, t := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := t.Execute(ctx, cmd, execOpts); err != nil {
			if shouldContinueAfterError(err) {
				continue
			}
			errs = append(errs, formatTargetError(t.Name(), cmd, err))
			if !opts.Continue {
				return errs[0]
			}
		}
	}

	if len(errs) > 0 {
		return combineErrors(errs)
	}
	return nil
}

// warnIfHasDependencies emits a warning if any target has dependencies.
// Parallel mode doesn't respect depends_on ordering, so users should be aware.
func warnIfHasDependencies(targets []target.Target) {
	for _, t := range targets {
		if len(t.DependsOn()) > 0 {
			out.WarningSimple("parallel mode does not respect depends_on ordering; targets may execute before dependencies complete")
			return
		}
	}
}

// runParallel executes targets concurrently using a bounded worker pool.
//
// NOTE: This function does NOT respect depends_on ordering. All targets are
// dispatched to available workers immediately. For dependency-aware execution,
// use mise's built-in task runner which topologically sorts targets.
//
// Worker count is controlled by STRUCTYL_PARALLEL (default: runtime.NumCPU()).
func (r *Runner) runParallel(ctx context.Context, targets []target.Target, cmd string, opts RunOptions) error {
	warnIfHasDependencies(targets)
	workers := getParallelWorkers()

	// Create cancellable context for fail-fast
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	// Bounded parallelism via semaphore pattern: channel capacity limits concurrent
	// goroutines. Each worker acquires a slot (send to channel) before executing
	// and releases it (receive from channel) when done.
	sem := make(chan struct{}, workers)

	execOpts := target.ExecOptions{
		Args:      opts.Args,
		Env:       opts.Env,
		Verbosity: opts.Verbosity,
	}

	// Process targets concurrently with worker pool limiting.
	// KNOWN LIMITATION: TopologicalOrder() ensures targets are scheduled in valid
	// dependency order, but this loop launches all goroutines immediately. The
	// semaphore limits concurrency but not dependency completion order. This means
	// targets with depends_on may execute before their dependencies complete.
	// See AGENTS.md "Known Limitations" for workarounds and fix requirements.
	for _, t := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			err := t.Execute(ctx, cmd, execOpts)

			// Mutex scope: The lock protects the shared errs slice. The cancel()
			// call inside the lock ensures atomic "append error + trigger cancel"
			// semantics, preventing a race where multiple goroutines could append
			// errors after cancellation was triggered. While cancel() itself is
			// thread-safe, keeping it under the lock provides clearer invariants.
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				if shouldContinueAfterError(err) {
					return
				}
				errs = append(errs, formatTargetError(t.Name(), cmd, err))
				if !opts.Continue {
					cancel()
				}
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return combineErrors(errs)
	}
	return nil
}

// getParallelWorkers returns the number of parallel workers to use.
// Invalid STRUCTYL_PARALLEL values (non-numeric, <1, >256) log a warning
// and fall back to runtime.NumCPU(). The result is always at least 1
// to prevent blocking on semaphore acquisition.
func getParallelWorkers() int {
	env := os.Getenv("STRUCTYL_PARALLEL")
	if env == "" {
		return max(minParallelWorkers, runtime.NumCPU())
	}

	n, err := strconv.Atoi(env)
	if err != nil {
		out.WarningSimple("invalid STRUCTYL_PARALLEL value %q (not a number), using default", env)
		return max(minParallelWorkers, runtime.NumCPU())
	}

	if n < minParallelWorkers || n > maxParallelWorkers {
		out.WarningSimple("STRUCTYL_PARALLEL=%d out of range [%d-%d], using default", n, minParallelWorkers, maxParallelWorkers)
		return max(minParallelWorkers, runtime.NumCPU())
	}

	return n
}

// shouldContinueAfterError checks if execution should continue after an error.
// Returns true for skip errors (execution continues), false for real errors (halt).
//
// For skip errors, logs a warning with the skip reason before returning.
// Per docs/specs/commands.md, disabled commands produce warnings, not info.
func shouldContinueAfterError(err error) bool {
	if target.IsSkipError(err) {
		out.Warning("%s", err.Error())
		return true
	}
	return false
}

// combineErrors combines multiple errors into one.
// Returns an error that supports errors.Is and errors.As for each individual error.
func combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return errors.Join(errs...)
}

// filterByCommand returns targets that have the specified command defined.
// Preserves the input slice order.
func filterByCommand(targets []target.Target, cmd string) []target.Target {
	var filtered []target.Target
	for _, t := range targets {
		if _, ok := t.GetCommand(cmd); ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// formatTargetError formats a target execution error with consistent messaging.
// Format matches docs/specs/error-handling.md grammar: [target] command: message
func formatTargetError(targetName, cmd string, err error) error {
	return fmt.Errorf("[%s] %s: %w", targetName, cmd, err)
}
