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

// maxParallelWorkers is the upper bound for parallel worker count.
// This prevents resource exhaustion from misconfigured STRUCTYL_PARALLEL values.
// The value 256 is chosen as a practical upper limit: beyond this, the overhead
// of goroutine scheduling and context switching typically outweighs parallelism benefits,
// and most build systems rarely have more than a few dozen independent targets.
const maxParallelWorkers = 256

// Runner orchestrates command execution across targets.
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

	// Filter to targets that have this command
	var filtered []target.Target
	for _, t := range targets {
		if _, ok := t.GetCommand(cmd); ok {
			filtered = append(filtered, t)
		}
	}

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

	var filtered []target.Target
	for _, t := range allTargets {
		if targetSet[t.Name()] {
			if _, ok := t.GetCommand(cmd); ok {
				filtered = append(filtered, t)
			}
		}
	}

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
			// Skip errors are logged as warnings but don't cause failure.
			// Per docs/specs/commands.md, disabled commands produce warnings, not info.
			if target.IsSkipError(err) {
				out.Warning("%s", err.Error())
				continue
			}
			errs = append(errs, fmt.Errorf("[%s] %s failed: %w", t.Name(), cmd, err))
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

// runParallel executes targets concurrently, respecting dependencies.
func (r *Runner) runParallel(ctx context.Context, targets []target.Target, cmd string, opts RunOptions) error {
	workers := getParallelWorkers()

	// Warn if any targets have dependencies - parallel mode doesn't respect them
	for _, t := range targets {
		if len(t.DependsOn()) > 0 {
			out.WarningSimple("parallel mode does not respect depends_on ordering; targets may execute before dependencies complete")
			break
		}
	}

	// Create cancellable context for fail-fast
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
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
		go func(t target.Target) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			err := t.Execute(ctx, cmd, execOpts)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				// Skip errors are logged as warnings but don't cause failure.
				// Per docs/specs/commands.md, disabled commands produce warnings, not info.
				if target.IsSkipError(err) {
					out.Warning("%s", err.Error())
					return
				}
				errs = append(errs, fmt.Errorf("[%s] %s failed: %w", t.Name(), cmd, err))
				if !opts.Continue {
					cancel()
				}
			}
		}(t)
	}

	wg.Wait()

	if len(errs) > 0 {
		return combineErrors(errs)
	}
	return nil
}

// getParallelWorkers returns the number of parallel workers to use.
// Invalid STRUCTYL_PARALLEL values (non-numeric, <1, >256) log a warning
// and fall back to runtime.NumCPU().
func getParallelWorkers() int {
	if env := os.Getenv("STRUCTYL_PARALLEL"); env != "" {
		n, err := strconv.Atoi(env)
		if err != nil {
			out.WarningSimple("invalid STRUCTYL_PARALLEL value %q (not a number), using default", env)
		} else if n < 1 || n > maxParallelWorkers {
			out.WarningSimple("STRUCTYL_PARALLEL=%d out of range [1-%d], using default", n, maxParallelWorkers)
		} else {
			return n
		}
	}
	return runtime.NumCPU()
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
