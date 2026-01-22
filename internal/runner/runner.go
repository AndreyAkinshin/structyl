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

var out = output.New()

// parallelDepsWarningOnce ensures the parallel mode dependency warning is only shown once per process.
// This prevents repetitive warnings when runParallel is called multiple times.
var parallelDepsWarningOnce sync.Once

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
	allTargets, err := r.registry.TopologicalOrder()
	if err != nil {
		return err
	}

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
		// Early exit if context is canceled before starting the next target
		if ctx.Err() != nil {
			return ctx.Err()
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

	return combineErrors(errs)
}

// hasDependencies returns true if any target has depends_on configured.
func hasDependencies(targets []target.Target) bool {
	for _, t := range targets {
		if len(t.DependsOn()) > 0 {
			return true
		}
	}
	return false
}

// runParallel executes targets concurrently using a bounded worker pool.
//
// # Pattern
//
// Uses a channel-as-semaphore pattern for bounded concurrency. Workers are limited
// to STRUCTYL_PARALLEL (default: runtime.NumCPU()) concurrent goroutines.
//
// # Known Limitation
//
// This function does NOT respect depends_on ordering. All targets are dispatched
// to workers immediately regardless of dependencies. Topological ordering ensures
// dependencies are scheduled first, but the semaphore doesn't block on dependency
// completion.
//
// For dependency-aware execution, use STRUCTYL_PARALLEL=1 or mise's task runner.
// See docs/specs/targets.md#known-limitation-parallel-execution-and-dependencies.
func (r *Runner) runParallel(ctx context.Context, targets []target.Target, cmd string, opts RunOptions) error {
	if hasDependencies(targets) {
		parallelDepsWarningOnce.Do(func() {
			out.WarningSimple("parallel mode does not respect depends_on ordering; targets may execute before dependencies complete")
		})
	}
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
	//
	// KNOWN LIMITATION: TopologicalOrder() ensures targets are scheduled in valid
	// dependency order, but this loop launches all goroutines immediately. The
	// semaphore limits concurrency but not dependency completion order. This means
	// targets with depends_on may execute before their dependencies complete.
	//
	// For dependency-aware parallel execution, use mise (the default backend) or
	// set STRUCTYL_PARALLEL=1 for sequential execution that respects ordering.
	// See AGENTS.md "Known Limitations" for full details.
	for _, t := range targets {
		wg.Add(1)
		go func(t target.Target) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			err := t.Execute(ctx, cmd, execOpts)

			// Determine action under lock, execute cancel outside lock.
			// This keeps the critical section minimal while preserving atomicity
			// of the "check error + append + decide to cancel" sequence.
			var shouldCancel bool
			mu.Lock()
			if err != nil && !shouldContinueAfterError(err) {
				errs = append(errs, formatTargetError(t.Name(), cmd, err))
				shouldCancel = !opts.Continue
			}
			mu.Unlock()

			if shouldCancel {
				cancel()
			}
		}(t)
	}

	wg.Wait()

	if len(errs) > 0 {
		return combineErrors(errs)
	}
	return nil
}

// defaultWorkerCount returns the default number of parallel workers based on CPU count.
// Always returns at least minParallelWorkers to prevent semaphore deadlock.
func defaultWorkerCount() int {
	return max(minParallelWorkers, runtime.NumCPU())
}

// getParallelWorkers returns the number of parallel workers to use.
// Invalid STRUCTYL_PARALLEL values (non-numeric, <1, >256) log a warning
// and fall back to runtime.NumCPU(). The result is always at least 1
// to prevent blocking on semaphore acquisition.
func getParallelWorkers() int {
	env := os.Getenv("STRUCTYL_PARALLEL")
	if env == "" {
		return defaultWorkerCount()
	}

	n, err := strconv.Atoi(env)
	if err != nil {
		out.WarningSimple("invalid STRUCTYL_PARALLEL value %q (not a number), using default", env)
		return defaultWorkerCount()
	}

	if n < minParallelWorkers || n > maxParallelWorkers {
		out.WarningSimple("STRUCTYL_PARALLEL=%d out of range [%d-%d], using default", n, minParallelWorkers, maxParallelWorkers)
		return defaultWorkerCount()
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
