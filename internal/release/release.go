// Package release provides release workflow functionality.
package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

// Options configures release behavior.
type Options struct {
	Version string // Version to release
	Push    bool   // Push to remote after commit
	DryRun  bool   // Print what would be done without doing it
	Force   bool   // Force release even with uncommitted changes
}

// Releaser handles the release workflow.
type Releaser struct {
	projectRoot string
	config      *config.Config
	out         *output.Writer
}

// NewReleaser creates a new Releaser.
func NewReleaser(projectRoot string, cfg *config.Config) *Releaser {
	return &Releaser{
		projectRoot: projectRoot,
		config:      cfg,
		out:         output.New(),
	}
}

// SetOutput sets a custom output writer (for testing).
func (r *Releaser) SetOutput(out *output.Writer) {
	r.out = out
}

// Release performs the release workflow.
func (r *Releaser) Release(ctx context.Context, opts Options) error {
	// Check for cancellation at start
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate version format
	ver, err := version.Parse(opts.Version)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}
	verStr := ver.String()

	// Check git state
	if !opts.Force {
		if err := r.checkGitClean(ctx); err != nil {
			return err
		}
	}

	if opts.DryRun {
		return r.dryRun(ctx, verStr, opts)
	}

	stepNum := 1

	// 1. Update VERSION file
	if err := ctx.Err(); err != nil {
		return err
	}
	r.out.Step(stepNum, "Setting version to %s", verStr)
	stepNum++
	if err := r.setVersion(verStr); err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	// 2. Propagate version to configured files
	if r.config.Version != nil && len(r.config.Version.Files) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.out.Step(stepNum, "Propagating version to configured files...")
		stepNum++
		// Resolve paths relative to project root
		resolvedFiles := make([]config.VersionFileConfig, len(r.config.Version.Files))
		for i, f := range r.config.Version.Files {
			resolvedFiles[i] = config.VersionFileConfig{
				Path:    filepath.Join(r.projectRoot, f.Path),
				Pattern: f.Pattern,
				Replace: f.Replace,
			}
		}
		if err := version.Propagate(verStr, resolvedFiles); err != nil {
			return fmt.Errorf("failed to propagate version: %w", err)
		}
	}

	// 3. Run pre-commit commands
	if r.config.Release != nil && len(r.config.Release.PreCommands) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.out.Step(stepNum, "Running pre-commit commands...")
		stepNum++
		for _, cmdStr := range r.config.Release.PreCommands {
			if err := ctx.Err(); err != nil {
				return err
			}
			r.out.StepDetail("Running: %s", cmdStr)
			if err := r.runCommand(ctx, cmdStr); err != nil {
				return fmt.Errorf("pre-commit command %q failed: %w", cmdStr, err)
			}
		}
	}

	// 4. Git add and commit
	if err := ctx.Err(); err != nil {
		return err
	}
	r.out.Step(stepNum, "Creating commit...")
	stepNum++
	if err := r.gitAddAll(ctx); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	commitMsg := fmt.Sprintf("set version %s", verStr)
	if err := r.gitCommit(ctx, commitMsg); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// 5. Move main branch to HEAD (if configured)
	if err := ctx.Err(); err != nil {
		return err
	}
	branch := r.getBranch()
	currentBranch, err := r.getCurrentBranch(ctx)
	if err == nil && currentBranch != branch {
		r.out.Step(stepNum, "Moving %s branch to HEAD...", branch)
		stepNum++
		if err := r.gitBranchForce(ctx, branch); err != nil {
			return fmt.Errorf("failed to move branch: %w", err)
		}
	}

	// 6. Push if requested
	if opts.Push {
		if err := ctx.Err(); err != nil {
			return err
		}
		remote := r.getRemote()
		r.out.Step(stepNum, "Pushing to %s...", remote)

		// Create tags
		tags := r.getTags(verStr)
		for _, tag := range tags {
			r.out.StepDetail("Creating tag: %s", tag)
			if err := r.gitTag(ctx, tag); err != nil {
				return fmt.Errorf("failed to create tag %s: %w", tag, err)
			}
		}

		// Push branch
		if err := r.gitPush(ctx, remote, branch); err != nil {
			return fmt.Errorf("failed to push branch: %w", err)
		}

		// Push tags
		for _, tag := range tags {
			if err := r.gitPushTag(ctx, remote, tag); err != nil {
				return fmt.Errorf("failed to push tag %s: %w", tag, err)
			}
		}
	}

	r.out.FinalSuccess("Release %s completed successfully!", verStr)
	if !opts.Push {
		r.out.Hint("Run with --push to push to remote.")
	}

	return nil
}

// dryRun prints what would be done without doing it.
func (r *Releaser) dryRun(ctx context.Context, verStr string, opts Options) error {
	r.out.DryRunStart()

	stepNum := 1
	r.out.Step(stepNum, "Set version to: %s", verStr)
	stepNum++

	if r.config.Version != nil && len(r.config.Version.Files) > 0 {
		r.out.Step(stepNum, "Propagate version to:")
		stepNum++
		for _, f := range r.config.Version.Files {
			r.out.StepDetail("%s", f.Path)
		}
	}

	if r.config.Release != nil && len(r.config.Release.PreCommands) > 0 {
		r.out.Step(stepNum, "Run pre-commit commands:")
		stepNum++
		for _, cmd := range r.config.Release.PreCommands {
			r.out.StepDetail("%s", cmd)
		}
	}

	r.out.Step(stepNum, "Create commit: \"set version %s\"", verStr)
	stepNum++

	branch := r.getBranch()
	r.out.Step(stepNum, "Move %s branch to HEAD", branch)
	stepNum++

	if opts.Push {
		remote := r.getRemote()
		tags := r.getTags(verStr)
		r.out.Step(stepNum, "Push to %s:", remote)
		r.out.StepDetail("Branch: %s", branch)
		for _, tag := range tags {
			r.out.StepDetail("Tag: %s", tag)
		}
	}

	r.out.DryRunEnd()
	return nil
}

// checkGitClean verifies the git working directory is clean.
func (r *Releaser) checkGitClean(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "diff-index", "--quiet", "HEAD", "--")
	cmd.Dir = r.projectRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git working directory is not clean; commit or stash changes first (use --force to override)")
	}
	return nil
}

// setVersion writes the version to the VERSION file.
func (r *Releaser) setVersion(verStr string) error {
	versionFile := ".structyl/PROJECT_VERSION"
	if r.config.Version != nil && r.config.Version.Source != "" {
		versionFile = r.config.Version.Source
	}

	path := filepath.Join(r.projectRoot, versionFile)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(verStr+"\n"), 0644)
}

// runCommand runs a shell command.
func (r *Releaser) runCommand(ctx context.Context, cmdStr string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = r.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitAddAll stages all changes.
func (r *Releaser) gitAddAll(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = r.projectRoot
	return cmd.Run()
}

// gitCommit creates a commit with the given message.
func (r *Releaser) gitCommit(ctx context.Context, message string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = r.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitTag creates a tag.
func (r *Releaser) gitTag(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", tag)
	cmd.Dir = r.projectRoot
	return cmd.Run()
}

// gitBranchForce moves a branch to HEAD.
func (r *Releaser) gitBranchForce(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "branch", "-f", branch, "HEAD")
	cmd.Dir = r.projectRoot
	return cmd.Run()
}

// gitPush pushes a branch to remote.
func (r *Releaser) gitPush(ctx context.Context, remote, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "push", remote, branch)
	cmd.Dir = r.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitPushTag pushes a tag to remote.
func (r *Releaser) gitPushTag(ctx context.Context, remote, tag string) error {
	cmd := exec.CommandContext(ctx, "git", "push", remote, tag)
	cmd.Dir = r.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getCurrentBranch returns the current git branch.
func (r *Releaser) getCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// getRemote returns the remote name from config or default.
func (r *Releaser) getRemote() string {
	if r.config.Release != nil && r.config.Release.Remote != "" {
		return r.config.Release.Remote
	}
	return "origin"
}

// getBranch returns the branch name from config or default.
func (r *Releaser) getBranch() string {
	if r.config.Release != nil && r.config.Release.Branch != "" {
		return r.config.Release.Branch
	}
	return "main"
}

// getTags returns the list of tags to create for the version.
func (r *Releaser) getTags(verStr string) []string {
	tagFormat := "v{version}"
	if r.config.Release != nil && r.config.Release.TagFormat != "" {
		tagFormat = r.config.Release.TagFormat
	}

	tags := []string{
		strings.ReplaceAll(tagFormat, "{version}", verStr),
	}

	// Add extra tags
	if r.config.Release != nil {
		for _, extraTag := range r.config.Release.ExtraTags {
			tags = append(tags, strings.ReplaceAll(extraTag, "{version}", verStr))
		}
	}

	return tags
}
