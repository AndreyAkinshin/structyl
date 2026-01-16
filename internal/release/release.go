// Package release provides release workflow functionality.
package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/akinshin/structyl/internal/config"
	"github.com/akinshin/structyl/internal/version"
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
}

// NewReleaser creates a new Releaser.
func NewReleaser(projectRoot string, cfg *config.Config) *Releaser {
	return &Releaser{
		projectRoot: projectRoot,
		config:      cfg,
	}
}

// Release performs the release workflow.
func (r *Releaser) Release(ctx context.Context, opts Options) error {
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

	// 1. Update VERSION file
	fmt.Printf("Setting version to %s\n", verStr)
	if err := r.setVersion(verStr); err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	// 2. Propagate version to configured files
	if r.config.Version != nil && len(r.config.Version.Files) > 0 {
		fmt.Println("Propagating version to configured files...")
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
		fmt.Println("Running pre-commit commands...")
		for _, cmdStr := range r.config.Release.PreCommands {
			fmt.Printf("  Running: %s\n", cmdStr)
			if err := r.runCommand(ctx, cmdStr); err != nil {
				return fmt.Errorf("pre-commit command %q failed: %w", cmdStr, err)
			}
		}
	}

	// 4. Git add and commit
	fmt.Println("Creating commit...")
	if err := r.gitAddAll(ctx); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	commitMsg := fmt.Sprintf("set version %s", verStr)
	if err := r.gitCommit(ctx, commitMsg); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// 5. Move main branch to HEAD (if configured)
	branch := r.getBranch()
	currentBranch, err := r.getCurrentBranch(ctx)
	if err == nil && currentBranch != branch {
		fmt.Printf("Moving %s branch to HEAD...\n", branch)
		if err := r.gitBranchForce(ctx, branch); err != nil {
			return fmt.Errorf("failed to move branch: %w", err)
		}
	}

	// 6. Push if requested
	if opts.Push {
		remote := r.getRemote()
		fmt.Printf("Pushing to %s...\n", remote)

		// Create tags
		tags := r.getTags(verStr)
		for _, tag := range tags {
			fmt.Printf("Creating tag: %s\n", tag)
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

	fmt.Printf("\nRelease %s completed successfully!\n", verStr)
	if !opts.Push {
		fmt.Println("Run with --push to push to remote.")
	}

	return nil
}

// dryRun prints what would be done without doing it.
func (r *Releaser) dryRun(ctx context.Context, verStr string, opts Options) error {
	fmt.Println("=== DRY RUN ===")
	fmt.Println()

	fmt.Printf("1. Set version to: %s\n", verStr)

	if r.config.Version != nil && len(r.config.Version.Files) > 0 {
		fmt.Println("2. Propagate version to:")
		for _, f := range r.config.Version.Files {
			fmt.Printf("   - %s\n", f.Path)
		}
	}

	if r.config.Release != nil && len(r.config.Release.PreCommands) > 0 {
		fmt.Println("3. Run pre-commit commands:")
		for _, cmd := range r.config.Release.PreCommands {
			fmt.Printf("   - %s\n", cmd)
		}
	}

	fmt.Printf("4. Create commit: \"set version %s\"\n", verStr)

	branch := r.getBranch()
	fmt.Printf("5. Move %s branch to HEAD\n", branch)

	if opts.Push {
		remote := r.getRemote()
		tags := r.getTags(verStr)
		fmt.Printf("6. Push to %s:\n", remote)
		fmt.Printf("   - Branch: %s\n", branch)
		for _, tag := range tags {
			fmt.Printf("   - Tag: %s\n", tag)
		}
	}

	fmt.Println()
	fmt.Println("=== END DRY RUN ===")
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
	versionFile := "VERSION"
	if r.config.Version != nil && r.config.Version.Source != "" {
		versionFile = r.config.Version.Source
	}

	path := filepath.Join(r.projectRoot, versionFile)
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
