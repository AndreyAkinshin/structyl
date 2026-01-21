package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

func TestConfigValidateMissingName(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "missing-name")
	configPath := filepath.Join(fixtureDir, ".structyl", "config.json")

	_, _, err := config.LoadAndValidate(configPath)
	if err == nil {
		t.Error("expected error for config with missing project name")
	}
}

func TestConfigValidateCircularDeps(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "circular-deps")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	// Registry creation should fail due to circular dependencies
	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Error("expected error for circular dependencies")
	}
	// Verify error message mentions circular dependency
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "circular") && !strings.Contains(errStr, "cycle") {
		t.Errorf("error = %q, want to mention 'circular' or 'cycle'", err.Error())
	}
}

func TestConfigValidateInvalidToolchain(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "invalid", "invalid-toolchain")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	// Registry creation should fail due to invalid toolchain
	_, err = target.NewRegistry(proj.Config, proj.Root)
	if err == nil {
		t.Error("expected error for invalid toolchain")
	}
	// Verify error message mentions toolchain
	if !strings.Contains(strings.ToLower(err.Error()), "toolchain") {
		t.Errorf("error = %q, want to mention 'toolchain'", err.Error())
	}
}

func TestConfigTargetTypes(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")
	configPath := filepath.Join(fixtureDir, ".structyl", "config.json")

	cfg, _, err := config.LoadAndValidate(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// All targets should be language type
	for name, targetCfg := range cfg.Targets {
		if targetCfg.Type != "language" {
			t.Errorf("expected target %q to have type 'language', got %q", name, targetCfg.Type)
		}
	}
}

func TestConfigWithAllFields(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "with-docker")
	configPath := filepath.Join(fixtureDir, ".structyl", "config.json")

	cfg, _, err := config.LoadAndValidate(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify project
	if cfg.Project.Name != "docker-project" {
		t.Errorf("expected project name %q, got %q", "docker-project", cfg.Project.Name)
	}

	// Verify docker config
	if cfg.Docker == nil {
		t.Fatal("expected docker config to be set")
	}

	// Verify targets
	goTarget, ok := cfg.Targets["go"]
	if !ok {
		t.Error("expected 'go' target to exist")
	} else {
		if goTarget.Directory != "." {
			t.Errorf("expected go directory %q, got %q", ".", goTarget.Directory)
		}
	}
}

func TestRegistryTargetCommands(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	pyTarget, ok := registry.Get("py")
	if !ok {
		t.Fatal("expected 'py' target to exist")
	}

	// Python targets should have standard commands from toolchain
	commands := pyTarget.Commands()
	if len(commands) == 0 {
		t.Error("expected python target to have commands")
	}

	// Check for common commands
	hasTest := false
	hasBuild := false
	for _, cmd := range commands {
		if cmd == "test" {
			hasTest = true
		}
		if cmd == "build" {
			hasBuild = true
		}
	}

	if !hasTest {
		t.Error("expected python target to have 'test' command")
	}
	if !hasBuild {
		t.Error("expected python target to have 'build' command")
	}
}
