package toolchain

import (
	"testing"
)

func TestGet_BuiltinToolchains(t *testing.T) {
	tests := []string{
		"cargo", "dotnet", "go", "npm", "pnpm", "yarn", "bun",
		"python", "uv", "poetry", "gradle", "maven", "make", "cmake", "swift",
		"r", "deno", "bundler", "composer", "mix", "sbt",
		"cabal", "stack", "dune", "lein", "zig", "rebar3",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			tc, ok := Get(name)
			if !ok {
				t.Errorf("Get(%q) = not found, want found", name)
				return
			}
			if tc.Name != name {
				t.Errorf("tc.Name = %q, want %q", tc.Name, name)
			}
		})
	}
}

func TestGet_UnknownToolchain(t *testing.T) {
	_, ok := Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) = found, want not found")
	}
}

func TestToolchain_GetCommand(t *testing.T) {
	tc, _ := Get("cargo")

	// Test string command
	cmd, ok := tc.GetCommand("build")
	if !ok {
		t.Error("GetCommand(build) = not found")
	}
	if cmdStr, isStr := cmd.(string); !isStr || cmdStr != "cargo build" {
		t.Errorf("GetCommand(build) = %v, want 'cargo build'", cmd)
	}

	// Test composite command ([]string)
	cmd, ok = tc.GetCommand("check")
	if !ok {
		t.Error("GetCommand(check) = not found")
	}
	if _, isSlice := cmd.([]string); !isSlice {
		t.Errorf("GetCommand(check) = %T, want []string", cmd)
	}

	// Test nil command
	cmd, ok = tc.GetCommand("restore")
	if !ok {
		t.Error("GetCommand(restore) = not found")
	}
	if cmd != nil {
		t.Errorf("GetCommand(restore) = %v, want nil", cmd)
	}
}

func TestToolchain_HasCommand(t *testing.T) {
	tc, _ := Get("cargo")

	if !tc.HasCommand("build") {
		t.Error("HasCommand(build) = false, want true")
	}
	if !tc.HasCommand("restore") { // nil command still exists
		t.Error("HasCommand(restore) = false, want true")
	}
	if tc.HasCommand("nonexistent") {
		t.Error("HasCommand(nonexistent) = true, want false")
	}
}

func TestIsBuiltin(t *testing.T) {
	if !IsBuiltin("cargo") {
		t.Error("IsBuiltin(cargo) = false, want true")
	}
	if IsBuiltin("custom") {
		t.Error("IsBuiltin(custom) = true, want false")
	}
}

func TestList(t *testing.T) {
	names := List()
	if len(names) == 0 {
		t.Error("List() returned empty")
	}

	// Check known toolchains are in the list
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	expected := []string{"cargo", "dotnet", "go", "npm"}
	for _, e := range expected {
		if !nameSet[e] {
			t.Errorf("List() missing %q", e)
		}
	}
}

func TestBuiltinToolchains_HaveStandardCommands(t *testing.T) {
	standardCommands := []string{"build", "test", "clean"}

	// Use List() for deterministic iteration order
	names := List()
	for _, name := range names {
		tc, ok := Get(name)
		if !ok {
			t.Errorf("toolchain %q from List() not found via Get()", name)
			continue
		}
		for _, cmd := range standardCommands {
			if !tc.HasCommand(cmd) {
				t.Errorf("toolchain %q missing standard command %q", name, cmd)
			}
		}
	}
}

func TestGetFromConfig_WithLoadedConfig(t *testing.T) {
	t.Parallel()
	loaded := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"custom": {
				Commands: map[string]interface{}{
					"build": "custom build cmd",
					"test":  "custom test cmd",
				},
			},
		},
	}

	// Test loaded config returns custom toolchain
	tc, ok := GetFromConfig("custom", loaded)
	if !ok {
		t.Fatal("GetFromConfig() should find custom toolchain in loaded config")
	}
	if tc.Name != "custom" {
		t.Errorf("tc.Name = %q, want %q", tc.Name, "custom")
	}
	if tc.Commands["build"] != "custom build cmd" {
		t.Errorf("tc.Commands[build] = %v, want %q", tc.Commands["build"], "custom build cmd")
	}

	// Test fallback to builtin when not in loaded config
	tc, ok = GetFromConfig("cargo", loaded)
	if !ok {
		t.Fatal("GetFromConfig() should fall back to builtin for cargo")
	}
	if tc.Name != "cargo" {
		t.Errorf("tc.Name = %q, want %q", tc.Name, "cargo")
	}
}

func TestGetFromConfig_NilLoaded(t *testing.T) {
	t.Parallel()
	// When loaded is nil, should fall back to builtin
	tc, ok := GetFromConfig("cargo", nil)
	if !ok {
		t.Fatal("GetFromConfig(nil) should fall back to builtin")
	}
	if tc.Name != "cargo" {
		t.Errorf("tc.Name = %q, want %q", tc.Name, "cargo")
	}
}

func TestGetFromConfig_NotFound(t *testing.T) {
	t.Parallel()
	loaded := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{},
	}

	// Non-existent in both loaded and builtin
	_, ok := GetFromConfig("nonexistent", loaded)
	if ok {
		t.Error("GetFromConfig() should return false for nonexistent toolchain")
	}
}
