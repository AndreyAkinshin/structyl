package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/runner" //nolint:staticcheck // SA1019: intentionally using deprecated package for backwards compatibility
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

	// Verify it's a JSON syntax error (wrapped in the error chain)
	var syntaxErr *json.SyntaxError
	if !containsJSONSyntaxError(err) {
		// Fallback: check error message for JSON-related keywords
		errStr := err.Error()
		hasJSONKeyword := containsIgnoreCase(errStr, "invalid") ||
			containsIgnoreCase(errStr, "unexpected") ||
			containsIgnoreCase(errStr, "syntax") ||
			containsIgnoreCase(errStr, "parse")
		if !hasJSONKeyword {
			t.Errorf("expected JSON parse error, got: %v (type: %T)", err, err)
		}
	}
	_ = syntaxErr // silence unused variable warning
}

func TestDockerUnavailableError(t *testing.T) {
	t.Parallel()
	// This tests the error type, not actual Docker availability
	err := &runner.DockerUnavailableError{}

	if err.ExitCode() != 3 {
		t.Errorf("expected DockerUnavailableError exit code 3, got %d", err.ExitCode())
	}

	if err.Error() == "" {
		t.Error("expected DockerUnavailableError to have error message")
	}
}

// Helper functions

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// containsJSONSyntaxError checks if the error chain contains a json.SyntaxError.
func containsJSONSyntaxError(err error) bool {
	for err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			return true
		}
		// Unwrap the error if possible
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
