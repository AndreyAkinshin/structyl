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

func TestInvalidToolchainReferenceError(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "invalid-toolchain")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("LoadProjectFrom: %v", err)
	}

	// Error occurs during registry creation, not during project loading
	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Fatal("expected error when creating registry with unknown toolchain reference")
	}

	// Verify error message mentions the unknown toolchain
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown toolchain") {
		t.Errorf("expected error to mention 'unknown toolchain', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "nonexistent-toolchain") {
		t.Errorf("expected error to mention toolchain name 'nonexistent-toolchain', got: %v", errMsg)
	}
}

func TestRegistryCreation_PartialTargetFailure(t *testing.T) {
	t.Parallel()
	// When one target has an invalid toolchain, registry creation fails entirely.
	// This tests that the error is reported clearly and doesn't cause panics.
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "invalid-toolchain")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("LoadProjectFrom: %v", err)
	}

	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Fatal("expected error for invalid toolchain")
	}

	// Verify error is actionable (contains target name or toolchain name)
	errMsg := err.Error()
	if !strings.Contains(errMsg, "toolchain") {
		t.Errorf("error should mention 'toolchain' for actionable diagnostics, got: %v", errMsg)
	}
}

func TestProjectLoad_ValidConfigWithMissingOptionalDirs(t *testing.T) {
	t.Parallel()
	// Test graceful handling when optional directories (like tests/) don't exist
	fixtureDir := filepath.Join(fixturesDir(), "minimal")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("LoadProjectFrom: %v", err)
	}

	// Project should load successfully even without tests/ directory
	if proj.Config == nil {
		t.Error("expected non-nil config")
	}
	if proj.Root == "" {
		t.Error("expected non-empty root path")
	}
}
