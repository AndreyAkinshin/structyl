package integration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

func TestProjectNotFoundError(t *testing.T) {
	t.Parallel()
	// Try to load from non-existent directory
	_, err := project.LoadProjectFrom("/nonexistent/path")
	if err == nil {
		t.Error("expected error when loading from nonexistent path")
	}
}

func TestConfigInvalidJSONError(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestMalformedJSONFixtureError(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "malformed-json")

	_, err := project.LoadProjectFrom(fixtureDir)
	if err == nil {
		t.Fatal("expected JSON parse error when loading malformed config")
	}

	// Verify it's a JSON syntax error using errors.As for proper error chain traversal
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected error chain to contain *json.SyntaxError, got: %v (type: %T)", err, err)
	}
}

// Helper functions

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}
