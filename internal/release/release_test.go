package release

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/output"
)

// createTestGitRepo creates a git repo with initial commit for testing.
// Disables GPG signing to work in environments with strict git configs.
// Skips the test if git is not available in the environment.
func createTestGitRepo(t *testing.T) string {
	t.Helper()

	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user (required for commits) and disable signing
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	// Create initial commit (--no-gpg-sign for extra safety in strict envs)
	cmd = exec.Command("git", "commit", "--allow-empty", "--no-gpg-sign", "-m", "initial")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("initial commit failed: %v", err)
	}

	return dir
}

// captureStdout captures stdout during function execution.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestNewReleaser(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
	}

	r := NewReleaser("/test/path", cfg)

	if r == nil {
		t.Fatal("NewReleaser() returned nil")
	}
	if r.projectRoot != "/test/path" {
		t.Errorf("projectRoot = %q, want %q", r.projectRoot, "/test/path")
	}
	if r.config != cfg {
		t.Error("config not set correctly")
	}
}

func TestGetRemote_Default(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   string
	}{
		{
			name:   "nil release config",
			config: &config.Config{},
			want:   "origin",
		},
		{
			name: "empty remote",
			config: &config.Config{
				Release: &config.ReleaseConfig{},
			},
			want: "origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReleaser("/tmp", tt.config)
			if got := r.getRemote(); got != tt.want {
				t.Errorf("getRemote() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRemote_Custom(t *testing.T) {
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			Remote: "upstream",
		},
	}

	r := NewReleaser("/tmp", cfg)
	if got := r.getRemote(); got != "upstream" {
		t.Errorf("getRemote() = %q, want %q", got, "upstream")
	}
}

func TestGetBranch_Default(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   string
	}{
		{
			name:   "nil release config",
			config: &config.Config{},
			want:   "main",
		},
		{
			name: "empty branch",
			config: &config.Config{
				Release: &config.ReleaseConfig{},
			},
			want: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReleaser("/tmp", tt.config)
			if got := r.getBranch(); got != tt.want {
				t.Errorf("getBranch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBranch_Custom(t *testing.T) {
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			Branch: "master",
		},
	}

	r := NewReleaser("/tmp", cfg)
	if got := r.getBranch(); got != "master" {
		t.Errorf("getBranch() = %q, want %q", got, "master")
	}
}

func TestGetTags_DefaultFormat(t *testing.T) {
	cfg := &config.Config{}

	r := NewReleaser("/tmp", cfg)
	tags := r.getTags("1.2.3")

	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}
	if tags[0] != "v1.2.3" {
		t.Errorf("tags[0] = %q, want %q", tags[0], "v1.2.3")
	}
}

func TestGetTags_CustomFormat(t *testing.T) {
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			TagFormat: "release-{version}",
		},
	}

	r := NewReleaser("/tmp", cfg)
	tags := r.getTags("1.2.3")

	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}
	if tags[0] != "release-1.2.3" {
		t.Errorf("tags[0] = %q, want %q", tags[0], "release-1.2.3")
	}
}

func TestGetTags_ExtraTags(t *testing.T) {
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			TagFormat: "v{version}",
			ExtraTags: []string{"latest", "{version}-stable"},
		},
	}

	r := NewReleaser("/tmp", cfg)
	tags := r.getTags("1.2.3")

	if len(tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3", len(tags))
	}
	expected := []string{"v1.2.3", "latest", "1.2.3-stable"}
	for i, want := range expected {
		if tags[i] != want {
			t.Errorf("tags[%d] = %q, want %q", i, tags[i], want)
		}
	}
}

func TestSetVersion_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	r := NewReleaser(dir, cfg)
	err := r.setVersion("1.2.3")
	if err != nil {
		t.Fatalf("setVersion() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".structyl", "PROJECT_VERSION"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "1.2.3\n" {
		t.Errorf("content = %q, want %q", string(content), "1.2.3\n")
	}
}

func TestSetVersion_CustomPath(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Version: &config.VersionConfig{
			Source: "custom_version.txt",
		},
	}

	r := NewReleaser(dir, cfg)
	err := r.setVersion("2.0.0")
	if err != nil {
		t.Fatalf("setVersion() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "custom_version.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "2.0.0\n" {
		t.Errorf("content = %q, want %q", string(content), "2.0.0\n")
	}

	// Ensure default VERSION was not created
	if _, err := os.Stat(filepath.Join(dir, ".structyl", "PROJECT_VERSION")); !os.IsNotExist(err) {
		t.Error(".structyl/VERSION file should not exist when custom source is set")
	}
}

func TestSetVersion_Overwrites(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	// setVersion creates directory, so pre-create file to test overwrite
	structylDir := filepath.Join(dir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(structylDir, "PROJECT_VERSION")
	if err := os.WriteFile(versionPath, []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewReleaser(dir, cfg)
	err := r.setVersion("2.0.0")
	if err != nil {
		t.Fatalf("setVersion() error = %v", err)
	}

	content, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "2.0.0\n" {
		t.Errorf("content = %q, want %q", string(content), "2.0.0\n")
	}
}

func TestDryRun_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	var buf bytes.Buffer
	r := NewReleaser(dir, cfg)
	r.SetOutput(output.NewWithWriters(&buf, &buf, false))
	opts := Options{Version: "1.2.3", DryRun: true}

	err := r.dryRun(context.Background(), "1.2.3", opts)
	if err != nil {
		t.Fatalf("dryRun() error = %v", err)
	}

	out := buf.String()

	// Check expected output
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %s", out)
	}
	if !strings.Contains(out, "Set version to: 1.2.3") {
		t.Errorf("output should contain version, got: %s", out)
	}
	if !strings.Contains(out, "Create commit") {
		t.Errorf("output should contain commit step, got: %s", out)
	}
	if !strings.Contains(out, "Move main branch to HEAD") {
		t.Errorf("output should contain branch step, got: %s", out)
	}
}

func TestDryRun_WithVersionFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Version: &config.VersionConfig{
			Files: []config.VersionFileConfig{
				{Path: "package.json", Pattern: "version", Replace: "version"},
				{Path: "setup.py", Pattern: "version", Replace: "version"},
			},
		},
	}

	var buf bytes.Buffer
	r := NewReleaser(dir, cfg)
	r.SetOutput(output.NewWithWriters(&buf, &buf, false))
	opts := Options{Version: "1.2.3", DryRun: true}

	err := r.dryRun(context.Background(), "1.2.3", opts)
	if err != nil {
		t.Fatalf("dryRun() error = %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Propagate version to:") {
		t.Errorf("output should mention version propagation, got: %s", out)
	}
	if !strings.Contains(out, "package.json") {
		t.Errorf("output should list package.json, got: %s", out)
	}
	if !strings.Contains(out, "setup.py") {
		t.Errorf("output should list setup.py, got: %s", out)
	}
}

func TestDryRun_WithPreCommands(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			PreCommands: []string{"make test", "make lint"},
		},
	}

	var buf bytes.Buffer
	r := NewReleaser(dir, cfg)
	r.SetOutput(output.NewWithWriters(&buf, &buf, false))
	opts := Options{Version: "1.2.3", DryRun: true}

	err := r.dryRun(context.Background(), "1.2.3", opts)
	if err != nil {
		t.Fatalf("dryRun() error = %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Run pre-commit commands:") {
		t.Errorf("output should mention pre-commit commands, got: %s", out)
	}
	if !strings.Contains(out, "make test") {
		t.Errorf("output should list 'make test', got: %s", out)
	}
	if !strings.Contains(out, "make lint") {
		t.Errorf("output should list 'make lint', got: %s", out)
	}
}

func TestDryRun_WithPush(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Release: &config.ReleaseConfig{
			Remote:    "upstream",
			Branch:    "master",
			TagFormat: "v{version}",
			ExtraTags: []string{"latest"},
		},
	}

	var buf bytes.Buffer
	r := NewReleaser(dir, cfg)
	r.SetOutput(output.NewWithWriters(&buf, &buf, false))
	opts := Options{Version: "1.2.3", DryRun: true, Push: true}

	err := r.dryRun(context.Background(), "1.2.3", opts)
	if err != nil {
		t.Fatalf("dryRun() error = %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Push to upstream:") {
		t.Errorf("output should mention push to upstream, got: %s", out)
	}
	if !strings.Contains(out, "Branch: master") {
		t.Errorf("output should show branch, got: %s", out)
	}
	if !strings.Contains(out, "Tag: v1.2.3") {
		t.Errorf("output should show tag v1.2.3, got: %s", out)
	}
	if !strings.Contains(out, "Tag: latest") {
		t.Errorf("output should show tag latest, got: %s", out)
	}
}

func TestRelease_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	tests := []string{
		"",
		"invalid",
		"1.2",
		"v1.2.3",
		"1.2.3.4",
	}

	for _, ver := range tests {
		t.Run(ver, func(t *testing.T) {
			r := NewReleaser(dir, cfg)
			err := r.Release(context.Background(), Options{Version: ver})
			if err == nil {
				t.Error("Release() expected error for invalid version")
			}
			if !strings.Contains(err.Error(), "invalid version") {
				t.Errorf("error = %q, want to contain 'invalid version'", err.Error())
			}
		})
	}
}

func TestRelease_DryRunMode_NoFileChanges(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	r := NewReleaser(dir, cfg)

	// Capture stdout to suppress output
	captureStdout(t, func() {
		err := r.Release(context.Background(), Options{
			Version: "1.2.3",
			DryRun:  true,
			Force:   true, // Skip git clean check
		})
		if err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	})

	// VERSION file should NOT be created in dry-run mode
	if _, err := os.Stat(filepath.Join(dir, ".structyl", "PROJECT_VERSION")); !os.IsNotExist(err) {
		t.Error("VERSION file should not exist in dry-run mode")
	}
}

func TestCheckGitClean_CleanRepo(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}

	r := NewReleaser(dir, cfg)
	err := r.checkGitClean(context.Background())
	if err != nil {
		t.Errorf("checkGitClean() error = %v, want nil", err)
	}
}

func TestCheckGitClean_DirtyRepo(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}

	// Create uncommitted file
	testFile := filepath.Join(dir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Stage the file
	cmd := exec.Command("git", "add", testFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	r := NewReleaser(dir, cfg)
	err := r.checkGitClean(context.Background())
	if err == nil {
		t.Error("checkGitClean() expected error for dirty repo")
	}
	if !strings.Contains(err.Error(), "not clean") {
		t.Errorf("error = %q, want to contain 'not clean'", err.Error())
	}
}

func TestCheckGitClean_NotARepo(t *testing.T) {
	dir := t.TempDir() // Not a git repo
	cfg := &config.Config{}

	r := NewReleaser(dir, cfg)
	err := r.checkGitClean(context.Background())
	if err == nil {
		t.Error("checkGitClean() expected error for non-repo")
	}
}

func TestRelease_DirtyRepo_ReturnsError(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}

	// Create uncommitted file
	testFile := filepath.Join(dir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Stage the file
	cmd := exec.Command("git", "add", testFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	r := NewReleaser(dir, cfg)
	err := r.Release(context.Background(), Options{Version: "1.2.3"})
	if err == nil {
		t.Error("Release() expected error for dirty repo")
	}
	if !strings.Contains(err.Error(), "not clean") {
		t.Errorf("error = %q, want to contain 'not clean'", err.Error())
	}
}

func TestRelease_CleanRepo_Success(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}

	r := NewReleaser(dir, cfg)

	// Capture stdout to suppress output
	captureStdout(t, func() {
		err := r.Release(context.Background(), Options{
			Version: "1.2.3",
		})
		if err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	})

	// Verify VERSION file was created
	content, err := os.ReadFile(filepath.Join(dir, ".structyl", "PROJECT_VERSION"))
	if err != nil {
		t.Fatalf("VERSION file not created: %v", err)
	}
	if string(content) != "1.2.3\n" {
		t.Errorf("VERSION content = %q, want %q", string(content), "1.2.3\n")
	}

	// Verify commit was created
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	if !strings.Contains(string(out), "set version 1.2.3") {
		t.Errorf("commit message = %q, want to contain 'set version 1.2.3'", string(out))
	}
}

func TestRelease_DirtyRepoWithForce_Success(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}

	// Create uncommitted file
	testFile := filepath.Join(dir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewReleaser(dir, cfg)

	// Capture stdout to suppress output
	captureStdout(t, func() {
		err := r.Release(context.Background(), Options{
			Version: "1.2.3",
			Force:   true,
		})
		if err != nil {
			t.Fatalf("Release() with Force error = %v", err)
		}
	})

	// Verify VERSION file was created
	if _, err := os.Stat(filepath.Join(dir, ".structyl", "PROJECT_VERSION")); os.IsNotExist(err) {
		t.Error("VERSION file should exist")
	}
}

func TestConfigDefaults_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		config     *config.Config
		wantRemote string
		wantBranch string
		wantTag    string
	}{
		{
			name:       "empty config",
			config:     &config.Config{},
			wantRemote: "origin",
			wantBranch: "main",
			wantTag:    "v1.0.0",
		},
		{
			name: "custom remote only",
			config: &config.Config{
				Release: &config.ReleaseConfig{
					Remote: "upstream",
				},
			},
			wantRemote: "upstream",
			wantBranch: "main",
			wantTag:    "v1.0.0",
		},
		{
			name: "custom branch only",
			config: &config.Config{
				Release: &config.ReleaseConfig{
					Branch: "master",
				},
			},
			wantRemote: "origin",
			wantBranch: "master",
			wantTag:    "v1.0.0",
		},
		{
			name: "custom tag format",
			config: &config.Config{
				Release: &config.ReleaseConfig{
					TagFormat: "release-{version}",
				},
			},
			wantRemote: "origin",
			wantBranch: "main",
			wantTag:    "release-1.0.0",
		},
		{
			name: "all custom",
			config: &config.Config{
				Release: &config.ReleaseConfig{
					Remote:    "github",
					Branch:    "develop",
					TagFormat: "{version}",
				},
			},
			wantRemote: "github",
			wantBranch: "develop",
			wantTag:    "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReleaser("/tmp", tt.config)

			if got := r.getRemote(); got != tt.wantRemote {
				t.Errorf("getRemote() = %q, want %q", got, tt.wantRemote)
			}
			if got := r.getBranch(); got != tt.wantBranch {
				t.Errorf("getBranch() = %q, want %q", got, tt.wantBranch)
			}
			tags := r.getTags("1.0.0")
			if len(tags) < 1 || tags[0] != tt.wantTag {
				t.Errorf("getTags() = %v, want first tag %q", tags, tt.wantTag)
			}
		})
	}
}

// =============================================================================
// Work Item 2: Git Operations Tests
// =============================================================================

// createTestGitRepoWithRemote creates a git repo with a local bare remote.
// Disables GPG signing to work in environments with strict git configs.
func createTestGitRepoWithRemote(t *testing.T) (repoDir, remoteDir string) {
	t.Helper()

	// Create bare remote repo
	remoteDir = t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init --bare failed: %v", err)
	}

	// Create working repo
	repoDir = t.TempDir()
	cmd = exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user and disable signing
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git config failed: %v", err)
		}
	}

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	// Create initial commit (--no-gpg-sign for extra safety in strict envs)
	cmd = exec.Command("git", "commit", "--allow-empty", "--no-gpg-sign", "-m", "initial")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("initial commit failed: %v", err)
	}

	// Push to remote to establish tracking
	cmd = exec.Command("git", "push", "-u", "origin", "master")
	cmd.Dir = repoDir
	// Ignore error - might be main instead of master
	cmd.Run()

	// Try main branch
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = repoDir
	cmd.Run()

	return repoDir, remoteDir
}

func TestGitTag_CreatesTag(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}
	r := NewReleaser(dir, cfg)

	ctx := context.Background()
	err := r.gitTag(ctx, "v1.0.0")
	if err != nil {
		t.Fatalf("gitTag() error = %v", err)
	}

	// Verify tag was created
	cmd := exec.Command("git", "tag", "-l", "v1.0.0")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git tag -l failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "v1.0.0" {
		t.Errorf("tag not created: got %q", string(out))
	}
}

func TestGitTag_DuplicateTag_ReturnsError(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}
	r := NewReleaser(dir, cfg)

	ctx := context.Background()
	// Create tag first time
	if err := r.gitTag(ctx, "v1.0.0"); err != nil {
		t.Fatalf("first gitTag() error = %v", err)
	}

	// Second time should fail
	err := r.gitTag(ctx, "v1.0.0")
	if err == nil {
		t.Error("gitTag() expected error for duplicate tag")
	}
}

func TestGitBranchForce_MovesBranch(t *testing.T) {
	dir := createTestGitRepo(t)
	cfg := &config.Config{}
	r := NewReleaser(dir, cfg)

	ctx := context.Background()

	// Create a new branch that we'll move
	cmd := exec.Command("git", "branch", "release")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git branch failed: %v", err)
	}

	// Create a new commit so HEAD differs from release branch
	cmd = exec.Command("git", "commit", "--allow-empty", "--no-gpg-sign", "-m", "second")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	// Move branch to HEAD
	err := r.gitBranchForce(ctx, "release")
	if err != nil {
		t.Fatalf("gitBranchForce() error = %v", err)
	}

	// Verify branch points to HEAD
	cmd = exec.Command("git", "rev-parse", "release")
	cmd.Dir = dir
	releaseRef, _ := cmd.Output()

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	headRef, _ := cmd.Output()

	if strings.TrimSpace(string(releaseRef)) != strings.TrimSpace(string(headRef)) {
		t.Errorf("branch not moved to HEAD: release=%q HEAD=%q", releaseRef, headRef)
	}
}

func TestGitPush_LocalRemote_Success(t *testing.T) {
	repoDir, _ := createTestGitRepoWithRemote(t)
	cfg := &config.Config{}
	r := NewReleaser(repoDir, cfg)

	ctx := context.Background()

	// Get current branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	branchOut, _ := cmd.Output()
	branch := strings.TrimSpace(string(branchOut))

	// Create a new commit
	cmd = exec.Command("git", "commit", "--allow-empty", "--no-gpg-sign", "-m", "test commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Push to remote
	err := r.gitPush(ctx, "origin", branch)
	if err != nil {
		t.Fatalf("gitPush() error = %v", err)
	}
}

func TestGitPushTag_LocalRemote_Success(t *testing.T) {
	repoDir, _ := createTestGitRepoWithRemote(t)
	cfg := &config.Config{}
	r := NewReleaser(repoDir, cfg)

	ctx := context.Background()

	// Create a tag
	if err := r.gitTag(ctx, "v1.0.0"); err != nil {
		t.Fatalf("gitTag() error = %v", err)
	}

	// Push tag to remote
	err := r.gitPushTag(ctx, "origin", "v1.0.0")
	if err != nil {
		t.Fatalf("gitPushTag() error = %v", err)
	}
}

func TestRunCommand_Success(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	r := NewReleaser(dir, cfg)

	ctx := context.Background()
	err := r.runCommand(ctx, "echo hello")
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}
}

func TestRunCommand_Failure_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	r := NewReleaser(dir, cfg)

	ctx := context.Background()
	err := r.runCommand(ctx, "exit 1")
	if err == nil {
		t.Error("runCommand() expected error for failing command")
	}
}

func TestRelease_WithPush_ExecutesPushOperations(t *testing.T) {
	repoDir, _ := createTestGitRepoWithRemote(t)
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
	}
	r := NewReleaser(repoDir, cfg)

	ctx := context.Background()
	opts := Options{
		Version: "1.0.0",
		Push:    true,
		Force:   true, // Skip git clean check since temp dir may have untracked files
	}

	err := r.Release(ctx, opts)
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// Verify VERSION file was created
	content, err := os.ReadFile(filepath.Join(repoDir, ".structyl", "PROJECT_VERSION"))
	if err != nil {
		t.Fatalf("failed to read VERSION: %v", err)
	}
	if strings.TrimSpace(string(content)) != "1.0.0" {
		t.Errorf("VERSION = %q, want %q", string(content), "1.0.0")
	}

	// Verify tag was created
	cmd := exec.Command("git", "tag", "-l", "v1.0.0")
	cmd.Dir = repoDir
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "v1.0.0" {
		t.Errorf("tag v1.0.0 not created")
	}
}
