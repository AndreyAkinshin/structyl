package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/runner"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

func TestProjectNotFoundError(t *testing.T) {
	// Try to load from non-existent directory
	_, err := project.LoadProjectFrom("/nonexistent/path")
	if err == nil {
		t.Error("expected error when loading from nonexistent path")
	}
}

func TestConfigFileMissingError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".structyl", "config.json")

	_, err := config.Load(configPath)
	if err == nil {
		t.Error("expected error when loading missing config file")
	}
}

func TestConfigInvalidJSONError(t *testing.T) {
	tmpDir := t.TempDir()
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := mkdir(structylDir); err != nil {
		t.Fatalf("failed to create .structyl dir: %v", err)
	}
	configPath := filepath.Join(structylDir, "config.json")

	// Write invalid JSON
	err := writeFile(configPath, "{ invalid json }")
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = config.Load(configPath)
	if err == nil {
		t.Error("expected error when loading invalid JSON config")
	}
}

func TestTargetNotFoundError(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "minimal")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("expected Get to return false for nonexistent target")
	}
}

func TestCircularDependencyError(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "circular-deps")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Error("expected error for circular dependencies")
	}

	// Error message should mention circular dependency
	if err != nil && !containsAny(err.Error(), "circular") {
		t.Errorf("expected error to mention 'circular', got: %v", err)
	}
}

func TestUndefinedDependencyError(t *testing.T) {
	tmpDir := t.TempDir()
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := mkdir(structylDir); err != nil {
		t.Fatalf("failed to create .structyl dir: %v", err)
	}
	configPath := filepath.Join(structylDir, "config.json")
	targetDir := filepath.Join(tmpDir, "target")

	// Create target directory
	if err := mkdir(targetDir); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Write config with undefined dependency
	configContent := `{
		"project": {"name": "test"},
		"targets": {
			"a": {
				"type": "language",
				"title": "Target A",
				"toolchain": "go",
				"directory": "target",
				"depends_on": ["undefined"]
			}
		}
	}`

	if err := writeFile(configPath, configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	proj, err := project.LoadProjectFrom(tmpDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Error("expected error for undefined dependency")
	}
}

func TestInvalidToolchainError(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "invalid-toolchain")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Error("expected error for invalid toolchain")
	}
}

func TestDockerUnavailableError(t *testing.T) {
	// This tests the error type, not actual Docker availability
	err := &runner.DockerUnavailableError{}

	if err.ExitCode() != 3 {
		t.Errorf("expected DockerUnavailableError exit code 3, got %d", err.ExitCode())
	}

	if err.Error() == "" {
		t.Error("expected DockerUnavailableError to have error message")
	}
}

func TestVersionReadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	versionPath := filepath.Join(tmpDir, "VERSION")

	// Write empty version file
	if err := writeFile(versionPath, ""); err != nil {
		t.Fatalf("failed to write empty version file: %v", err)
	}

	_, err := version.Read(versionPath)
	if err == nil {
		t.Error("expected error when reading empty version file")
	}
}

func TestVersionReadInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	versionPath := filepath.Join(tmpDir, "VERSION")

	// Write invalid version
	if err := writeFile(versionPath, "not-a-version"); err != nil {
		t.Fatalf("failed to write invalid version: %v", err)
	}

	_, err := version.Read(versionPath)
	if err == nil {
		t.Error("expected error when reading invalid version")
	}
}

// Helper functions

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

func containsAny(s string, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
