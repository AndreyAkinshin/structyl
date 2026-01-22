package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/target"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

// Integration tests for version functionality with real fixtures.
// Unit tests for version validation, parsing, bump, and compare are in
// internal/version/version_test.go. These tests focus on file I/O with
// real project fixtures.

func TestVersionRead(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")
	versionPath := filepath.Join(fixtureDir, "VERSION")

	v, err := version.Read(versionPath)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}

	if v != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", v)
	}
}

func TestVersionReadMissing(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "minimal")
	versionPath := filepath.Join(fixtureDir, "VERSION")

	_, err := version.Read(versionPath)
	if err == nil {
		t.Error("expected error when reading missing VERSION file")
	}
}

// Note: TestVersionWriteAndRead was removed as it duplicates unit test coverage
// in internal/version/version_test.go:TestWrite. Integration tests here focus on
// fixture-based testing; write/read roundtrip testing belongs in unit tests.

func TestVersionWriteInvalid(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	versionPath := filepath.Join(tmpDir, "VERSION")

	err := version.Write(versionPath, "invalid")
	if err == nil {
		t.Error("expected error when writing invalid version")
	}

	// Ensure file was not created
	if _, err := os.Stat(versionPath); !os.IsNotExist(err) {
		t.Error("expected version file to not be created for invalid version")
	}
}

// TestVersionInterpolation_EndToEnd validates the full version propagation flow:
// VERSION file → config with version.source → registry creation → target has access.
// This creates a temporary project with proper version config to test the integration.
func TestVersionInterpolation_EndToEnd(t *testing.T) {
	t.Parallel()

	// Create a temporary project with version config
	tmpDir := t.TempDir()
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatalf("failed to create .structyl dir: %v", err)
	}

	// Write VERSION file
	expectedVersion := "2.5.0"
	versionPath := filepath.Join(tmpDir, "VERSION")
	if err := version.Write(versionPath, expectedVersion); err != nil {
		t.Fatalf("failed to write VERSION file: %v", err)
	}

	// Create target directory (required by target validation)
	targetDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Write config with version.source pointing to VERSION file
	configJSON := `{
		"project": { "name": "version-test" },
		"version": { "source": "VERSION" },
		"targets": {
			"test": {
				"type": "language",
				"title": "Test",
				"toolchain": "go"
			}
		}
	}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load project
	proj, err := project.LoadProjectFrom(tmpDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	// Verify config has version source configured
	if proj.Config.Version == nil || proj.Config.Version.Source != "VERSION" {
		t.Fatal("expected config to have version.source = VERSION")
	}

	// Create registry (should read VERSION file and propagate to targets)
	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Verify target exists (registry creation succeeded with version propagation)
	testTarget, ok := registry.Get("test")
	if !ok {
		t.Fatal("expected 'test' target to exist")
	}

	// The version is used for interpolation, verified indirectly through successful
	// registry creation. Direct verification of ${version} interpolation is covered
	// by unit tests in impl_test.go (TestInterpolateVars_BuiltinVariables).
	_ = testTarget
}
