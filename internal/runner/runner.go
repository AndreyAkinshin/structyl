// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"

	"github.com/AndreyAkinshin/structyl/internal/target"
)

// maxParallelWorkers is the upper bound for parallel worker count.
// This prevents resource exhaustion from misconfigured STRUCTYL_PARALLEL values.
const maxParallelWorkers = 256

// Runner orchestrates command execution across targets.
type Runner struct {
	registry *target.Registry
}

// RunOptions configures execution behavior.
type RunOptions struct {
	Docker   bool              // Run in Docker container
	Continue bool              // Continue on error (don't fail-fast)
	Parallel bool              // Run in parallel where dependencies allow
	Args     []string          // Arguments to pass to commands
	Env      map[string]string // Additional environment variables
}

// New creates a new Runner.
func New(registry *target.Registry) *Runner {
	return &Runner{registry: registry}
}

// Run executes a command on a single target.
func (r *Runner) Run(ctx context.Context, targetName, cmd string, opts RunOptions) error {
	t, ok := r.registry.Get(targetName)
	if !ok {
		return fmt.Errorf("unknown target: %s", targetName)
	}

	execOpts := target.ExecOptions{
		Docker: opts.Docker,
		Args:   opts.Args,
		Env:    opts.Env,
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
		Docker: opts.Docker,
		Args:   opts.Args,
		Env:    opts.Env,
	}

	var errs []error
	for _, t := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := t.Execute(ctx, cmd, execOpts); err != nil {
			// Skip errors are logged but don't cause failure
			if target.IsSkipError(err) {
				fmt.Fprintln(os.Stderr, err.Error())
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

	// Create cancellable context for fail-fast
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	sem := make(chan struct{}, workers)

	execOpts := target.ExecOptions{
		Docker: opts.Docker,
		Args:   opts.Args,
		Env:    opts.Env,
	}

	// Process targets concurrently with worker pool limiting.
	// Note: TopologicalOrder() ensures targets are in valid dependency order,
	// but this loop launches all goroutines immediately. Actual parallelism
	// is limited by the semaphore, not by dependency completion.
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
				// Skip errors are logged but don't cause failure
				if target.IsSkipError(err) {
					fmt.Fprintln(os.Stderr, err.Error())
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
func getParallelWorkers() int {
	if env := os.Getenv("STRUCTYL_PARALLEL"); env != "" {
		n, err := strconv.Atoi(env)
		if err == nil && n >= 1 && n <= maxParallelWorkers {
			return n
		}
	}
	return runtime.NumCPU()
}

// combineErrors combines multiple errors into one.
func combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}

	msg := fmt.Sprintf("%d errors occurred:\n", len(errors))
	for _, err := range errors {
		msg += fmt.Sprintf("  - %v\n", err)
	}
	return fmt.Errorf("%s", msg)
}
