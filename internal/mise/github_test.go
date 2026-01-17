package mise

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestGenerateGitHubWorkflow_Basic(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo", Title: "Rust"},
			"go": {Toolchain: "go", Title: "Go"},
		},
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	checks := []string{
		"name: CI",
		"on:",
		"push:",
		"branches: [main]",
		"pull_request:",
		"jobs:",
		"uses: actions/checkout@v4",
		"uses: jdx/mise-action@v2",
		"mise run ci:rs",
		"mise run ci:go",
		"name: Rust",
		"name: Go",
		"runs-on: ubuntu-latest",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("missing %q in workflow", check)
		}
	}
}

func TestGenerateGitHubWorkflow_Empty(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{},
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	// Should still have header
	if !strings.Contains(content, "name: CI") {
		t.Error("missing workflow name")
	}
	// But no jobs
	if strings.Contains(content, "mise run ci:") {
		t.Error("should not have job steps for empty targets")
	}
}

func TestGenerateGitHubWorkflow_UnsupportedToolchain(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"custom": {Toolchain: "make", Title: "Custom"}, // Unsupported
		},
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	// Should not have a job for the unsupported toolchain
	if strings.Contains(content, "mise run ci:custom") {
		t.Error("should not have job for unsupported toolchain")
	}
}

func TestGenerateGitHubWorkflow_UsesTargetTitle(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo", Title: "Rust Library"},
		},
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	if !strings.Contains(content, "name: Rust Library") {
		t.Error("should use target title in job name")
	}
}

func TestGenerateGitHubWorkflow_FallbackToTargetName(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo"}, // No Title set
		},
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	if !strings.Contains(content, "name: rs") {
		t.Error("should fallback to target name when title not set")
	}
}

func TestWriteGitHubWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo"},
		},
	}

	// First write should create file and directory
	created, err := WriteGitHubWorkflow(tmpDir, cfg, false)
	if err != nil {
		t.Fatalf("WriteGitHubWorkflow() error = %v", err)
	}
	if !created {
		t.Error("WriteGitHubWorkflow() = false, want true")
	}

	// Verify file exists
	workflowPath := filepath.Join(tmpDir, ".github", "workflows", "ci.yml")
	if _, err := os.Stat(workflowPath); err != nil {
		t.Errorf("workflow file not created at %s", workflowPath)
	}

	// Second write without force should not overwrite
	created, err = WriteGitHubWorkflow(tmpDir, cfg, false)
	if err != nil {
		t.Fatalf("WriteGitHubWorkflow() error = %v", err)
	}
	if created {
		t.Error("WriteGitHubWorkflow() = true, want false (file exists)")
	}

	// Third write with force should overwrite
	created, err = WriteGitHubWorkflow(tmpDir, cfg, true)
	if err != nil {
		t.Fatalf("WriteGitHubWorkflow() error = %v", err)
	}
	if !created {
		t.Error("WriteGitHubWorkflow(force=true) = false, want true")
	}
}

func TestGitHubWorkflowExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if GitHubWorkflowExists(tmpDir) {
		t.Error("GitHubWorkflowExists() = true, want false")
	}

	// Create workflow file
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("name: CI"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Should exist now
	if !GitHubWorkflowExists(tmpDir) {
		t.Error("GitHubWorkflowExists() = false, want true")
	}
}

func TestGetGitHubWorkflowPath(t *testing.T) {
	path := GetGitHubWorkflowPath("/project")

	expected := filepath.Join("/project", ".github", "workflows", "ci.yml")
	if path != expected {
		t.Errorf("GetGitHubWorkflowPath() = %q, want %q", path, expected)
	}
}

func TestGenerateGitHubWorkflow_Deterministic(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"zz": {Toolchain: "cargo"},
			"aa": {Toolchain: "go"},
			"mm": {Toolchain: "npm"},
		},
	}

	// Generate twice and compare
	content1, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	content2, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		t.Fatalf("GenerateGitHubWorkflow() error = %v", err)
	}

	if content1 != content2 {
		t.Error("GenerateGitHubWorkflow() should be deterministic")
	}

	// Jobs should appear in sorted order
	aaIdx := strings.Index(content1, "mise run ci:aa")
	mmIdx := strings.Index(content1, "mise run ci:mm")
	zzIdx := strings.Index(content1, "mise run ci:zz")

	if aaIdx > mmIdx || mmIdx > zzIdx {
		t.Error("jobs should be in sorted order by target name")
	}
}
