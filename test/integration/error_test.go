package integration

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

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

func TestCircularDependencyError(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "circular-deps")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}

	// Verify error mentions circular dependency
	errMsg := err.Error()
	if !containsAny(errMsg, "circular", "cycle") {
		t.Errorf("error = %q, want to mention 'circular' or 'cycle'", errMsg)
	}
}

func TestInvalidToolchainError(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "invalid-toolchain")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Fatal("expected error for unknown toolchain, got nil")
	}

	// Verify error mentions unknown toolchain
	errMsg := err.Error()
	if !containsAny(errMsg, "unknown toolchain", "nonexistent-toolchain") {
		t.Errorf("error = %q, want to mention 'unknown toolchain' or the invalid name", errMsg)
	}
}

func TestMissingProjectNameError(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "missing-name")

	_, err := project.LoadProjectFrom(fixtureDir)
	if err == nil {
		t.Fatal("expected validation error for missing project name, got nil")
	}

	// Verify error mentions project name
	errMsg := err.Error()
	if !containsAny(errMsg, "name", "required") {
		t.Errorf("error = %q, want to mention 'name' or 'required'", errMsg)
	}
}

// Helper functions

func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
