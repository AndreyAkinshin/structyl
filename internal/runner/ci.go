// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

// CIOptions configures CI pipeline execution.
type CIOptions struct {
	Docker      bool   // Run in Docker containers
	Continue    bool   // Continue on error
	Release     bool   // Use release build variants
	Parallel    bool   // Run language targets in parallel
	ArtifactDir string // Directory to collect artifacts
}

// CIResult contains the results of a CI pipeline run.
type CIResult struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	PhaseResults  []PhaseResult
	TargetResults map[string]TargetResult
	Success       bool
	ArtifactCount int
}

// PhaseResult contains results for a CI phase.
type PhaseResult struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
	Error     error
}

// TargetResult contains results for a specific target.
type TargetResult struct {
	Name     string
	Success  bool
	Errors   []error
	Duration time.Duration
}

// RunCI executes the full CI pipeline.
func (r *Runner) RunCI(ctx context.Context, opts CIOptions) (*CIResult, error) {
	result := &CIResult{
		StartTime:     time.Now(),
		TargetResults: make(map[string]TargetResult),
		Success:       true,
	}

	// Determine pipeline phases based on release mode
	pipeline := []string{"clean", "restore", "check", "build", "test"}
	if opts.Release {
		pipeline = []string{"clean", "restore", "check", "build:release", "test"}
	}

	// Get targets
	targets, err := r.registry.TopologicalOrder()
	if err != nil {
		return nil, err
	}

	// Separate by type
	var auxTargets, langTargets []target.Target
	for _, t := range targets {
		if t.Type() == target.TypeAuxiliary {
			auxTargets = append(auxTargets, t)
		} else {
			langTargets = append(langTargets, t)
		}
	}

	// Execute pipeline for auxiliary targets first (in order)
	for _, phase := range pipeline {
		phaseResult := r.runPhase(ctx, phase, auxTargets, opts, false)
		result.PhaseResults = append(result.PhaseResults, phaseResult)
		if !phaseResult.Success {
			result.Success = false
			if !opts.Continue {
				break
			}
		}
	}

	// Execute pipeline for language targets (can be parallel)
	if result.Success || opts.Continue {
		for _, phase := range pipeline {
			phaseResult := r.runPhase(ctx, phase, langTargets, opts, opts.Parallel)
			result.PhaseResults = append(result.PhaseResults, phaseResult)
			if !phaseResult.Success {
				result.Success = false
				if !opts.Continue {
					break
				}
			}
		}
	}

	// Collect artifacts
	if opts.ArtifactDir != "" && (result.Success || opts.Continue) {
		artifactCount, err := r.collectArtifacts(ctx, targets, opts.ArtifactDir, nil)
		if err != nil {
			result.Success = false
		}
		result.ArtifactCount = artifactCount
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// runPhase executes a single phase of the CI pipeline.
func (r *Runner) runPhase(ctx context.Context, phase string, targets []target.Target, opts CIOptions, parallel bool) PhaseResult {
	result := PhaseResult{
		Name:      phase,
		StartTime: time.Now(),
		Success:   true,
	}

	runOpts := RunOptions{
		Docker:   opts.Docker,
		Continue: opts.Continue,
		Parallel: parallel,
	}

	// Filter to targets that have this command
	var filtered []target.Target
	for _, t := range targets {
		if _, ok := t.GetCommand(phase); ok {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	var err error
	if parallel {
		err = r.runParallel(ctx, filtered, phase, runOpts)
	} else {
		err = r.runSequential(ctx, filtered, phase, runOpts)
	}

	if err != nil {
		result.Success = false
		result.Error = err
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// collectArtifacts collects build artifacts to the output directory.
func (r *Runner) collectArtifacts(ctx context.Context, targets []target.Target, outputDir string, out *output.Writer) (int, error) {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	count := 0

	for _, t := range targets {
		// Look for common artifact patterns
		artifacts := findArtifacts(t)
		for _, artifact := range artifacts {
			destPath := filepath.Join(outputDir, filepath.Base(artifact))
			if err := copyFile(artifact, destPath); err != nil {
				// Log but don't fail
				if out != nil {
					out.Warning("failed to copy artifact %s: %v", artifact, err)
				}
				continue
			}
			count++
		}
	}

	return count, nil
}

// findArtifacts finds artifact files for a target.
func findArtifacts(t target.Target) []string {
	var artifacts []string

	// Common artifact patterns by toolchain/target name
	patterns := []string{
		// Rust
		"target/release/*.exe",
		"target/release/*.dll",
		"target/release/*.so",
		"target/release/*.dylib",
		// .NET
		"bin/Release/**/*.nupkg",
		"bin/Release/**/*.dll",
		// Go
		"bin/*",
		// Node
		"dist/*.tgz",
		// Python
		"dist/*.whl",
		"dist/*.tar.gz",
	}

	dir := t.Directory()
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		artifacts = append(artifacts, matches...)
	}

	return artifacts
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	_, err = io.Copy(destination, source)
	return err
}

// PrintCISummary prints a summary of CI results.
func PrintCISummary(result *CIResult, out *output.Writer) {
	out.SummaryHeader("CI Summary")

	// Print detailed phase listing
	out.SummarySectionLabel("Phases:")
	for _, p := range result.PhaseResults {
		var errMsg string
		if p.Error != nil {
			errMsg = p.Error.Error()
		}
		out.SummaryAction(p.Name, p.Success, FormatDuration(p.Duration), errMsg)
	}
	out.Println("")

	// Phase summary
	var successPhases, failedPhases []string
	for _, p := range result.PhaseResults {
		if p.Success {
			successPhases = append(successPhases, p.Name)
		} else {
			failedPhases = append(failedPhases, p.Name)
		}
	}

	if len(successPhases) > 0 {
		out.SummaryPassed("Passed", strings.Join(successPhases, ", "))
	}
	if len(failedPhases) > 0 {
		out.SummaryFailed("Failed", strings.Join(failedPhases, ", "))
	}

	// Timing
	out.SummaryItem("Duration", FormatDuration(result.Duration))

	// Artifacts
	if result.ArtifactCount > 0 {
		out.SummaryItem("Artifacts", fmt.Sprintf("%d", result.ArtifactCount))
	}

	// Overall status
	if result.Success {
		out.FinalSuccess("CI pipeline completed successfully.")
	} else {
		out.FinalFailure("CI pipeline failed.")
	}
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

// PhaseOrder returns the standard CI phase order.
func PhaseOrder(release bool) []string {
	if release {
		return []string{"clean", "restore", "check", "build:release", "test"}
	}
	return []string{"clean", "restore", "check", "build", "test"}
}
