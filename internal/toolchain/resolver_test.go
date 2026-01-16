package toolchain

import (
	"testing"

	"github.com/akinshin/structyl/internal/config"
)

func TestResolver_ResolveBuiltin(t *testing.T) {
	r, err := NewResolver(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}

	tc, err := r.Resolve("cargo")
	if err != nil {
		t.Fatalf("Resolve(cargo) error = %v", err)
	}
	if tc.Name != "cargo" {
		t.Errorf("tc.Name = %q, want %q", tc.Name, "cargo")
	}
}

func TestResolver_ResolveUnknown(t *testing.T) {
	r, err := NewResolver(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("nonexistent")
	if err == nil {
		t.Error("Resolve(nonexistent) expected error")
	}
}

func TestResolver_CustomToolchain(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Toolchains: map[string]config.ToolchainConfig{
			"my-toolchain": {
				Commands: map[string]interface{}{
					"build": "custom-build",
					"test":  "custom-test",
				},
			},
		},
	}

	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}

	tc, err := r.Resolve("my-toolchain")
	if err != nil {
		t.Fatalf("Resolve(my-toolchain) error = %v", err)
	}

	cmd, _ := tc.GetCommand("build")
	if cmd != "custom-build" {
		t.Errorf("GetCommand(build) = %v, want custom-build", cmd)
	}
}

func TestResolver_ExtendedToolchain(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Toolchains: map[string]config.ToolchainConfig{
			"cargo-workspace": {
				Extends: "cargo",
				Commands: map[string]interface{}{
					"build": "cargo build --workspace",
					"test":  "cargo test --workspace",
				},
			},
		},
	}

	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}

	tc, err := r.Resolve("cargo-workspace")
	if err != nil {
		t.Fatalf("Resolve(cargo-workspace) error = %v", err)
	}

	// Overridden commands
	cmd, _ := tc.GetCommand("build")
	if cmd != "cargo build --workspace" {
		t.Errorf("GetCommand(build) = %v, want 'cargo build --workspace'", cmd)
	}

	// Inherited commands
	cmd, _ = tc.GetCommand("clean")
	if cmd != "cargo clean" {
		t.Errorf("GetCommand(clean) = %v, want 'cargo clean' (inherited)", cmd)
	}

	// Inherited format command
	cmd, _ = tc.GetCommand("format")
	if cmd != "cargo fmt" {
		t.Errorf("GetCommand(format) = %v, want 'cargo fmt' (inherited)", cmd)
	}
}

func TestResolver_ExtendUnknownToolchain(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Toolchains: map[string]config.ToolchainConfig{
			"bad-toolchain": {
				Extends: "nonexistent",
			},
		},
	}

	_, err := NewResolver(cfg)
	if err == nil {
		t.Error("NewResolver() expected error for extending unknown toolchain")
	}
}

func TestResolver_ValidateTargetToolchains(t *testing.T) {
	r, _ := NewResolver(&config.Config{})

	// Valid toolchain
	err := r.ValidateTargetToolchains(map[string]config.TargetConfig{
		"rs": {Toolchain: "cargo"},
	})
	if err != nil {
		t.Errorf("ValidateTargetToolchains() error = %v", err)
	}

	// Invalid toolchain
	err = r.ValidateTargetToolchains(map[string]config.TargetConfig{
		"bad": {Toolchain: "nonexistent"},
	})
	if err == nil {
		t.Error("ValidateTargetToolchains() expected error for unknown toolchain")
	}
}

func TestResolver_GetResolvedCommands(t *testing.T) {
	r, _ := NewResolver(&config.Config{})

	// Target with toolchain and overrides
	target := config.TargetConfig{
		Toolchain: "cargo",
		Commands: map[string]interface{}{
			"demo": "cargo run --example demo",
		},
	}

	cmds, err := r.GetResolvedCommands(target)
	if err != nil {
		t.Fatalf("GetResolvedCommands() error = %v", err)
	}

	// Should have toolchain commands
	if cmds["build"] != "cargo build" {
		t.Errorf("cmds[build] = %v, want 'cargo build'", cmds["build"])
	}

	// Should have target-specific commands
	if cmds["demo"] != "cargo run --example demo" {
		t.Errorf("cmds[demo] = %v, want 'cargo run --example demo'", cmds["demo"])
	}
}

func TestResolver_GetResolvedCommands_NoToolchain(t *testing.T) {
	r, _ := NewResolver(&config.Config{})

	target := config.TargetConfig{
		Commands: map[string]interface{}{
			"build": "make build",
			"test":  "make test",
		},
	}

	cmds, err := r.GetResolvedCommands(target)
	if err != nil {
		t.Fatalf("GetResolvedCommands() error = %v", err)
	}

	if cmds["build"] != "make build" {
		t.Errorf("cmds[build] = %v, want 'make build'", cmds["build"])
	}
}
