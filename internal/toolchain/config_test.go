package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// LoadToolchains Tests
// =============================================================================

func TestLoadToolchains_NoFile_ReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := LoadToolchains(tmpDir)
	if err != nil {
		t.Fatalf("LoadToolchains() error = %v", err)
	}

	if result == nil {
		t.Fatal("LoadToolchains() returned nil")
	}

	// Should return defaults
	if len(result.Toolchains) == 0 {
		t.Error("LoadToolchains() returned empty toolchains, want defaults")
	}

	// Verify known toolchain exists
	if _, ok := result.Toolchains["cargo"]; !ok {
		t.Error("LoadToolchains() missing 'cargo' toolchain from defaults")
	}
}

func TestLoadToolchains_ValidFile_MergesWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create toolchains.json with custom config
	toolchainsJSON := `{
		"version": "1.0",
		"toolchains": {
			"cargo": {
				"mise": {
					"version": "nightly"
				}
			},
			"custom": {
				"commands": {
					"build": "custom build"
				}
			}
		}
	}`
	toolchainsPath := filepath.Join(structylDir, "toolchains.json")
	if err := os.WriteFile(toolchainsPath, []byte(toolchainsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadToolchains(tmpDir)
	if err != nil {
		t.Fatalf("LoadToolchains() error = %v", err)
	}

	// Cargo should have merged config
	cargo, ok := result.Toolchains["cargo"]
	if !ok {
		t.Fatal("LoadToolchains() missing 'cargo' toolchain")
	}
	if cargo.Mise == nil {
		t.Fatal("cargo.Mise is nil")
	}
	if cargo.Mise.Version != "nightly" {
		t.Errorf("cargo.Mise.Version = %q, want %q", cargo.Mise.Version, "nightly")
	}
	// Should still have PrimaryTool from defaults
	if cargo.Mise.PrimaryTool != "rust" {
		t.Errorf("cargo.Mise.PrimaryTool = %q, want %q (from defaults)", cargo.Mise.PrimaryTool, "rust")
	}

	// Custom toolchain should exist
	custom, ok := result.Toolchains["custom"]
	if !ok {
		t.Fatal("LoadToolchains() missing 'custom' toolchain")
	}
	if custom.Commands == nil {
		t.Fatal("custom.Commands is nil")
	}
	if custom.Commands["build"] != "custom build" {
		t.Errorf("custom.Commands[build] = %v, want %q", custom.Commands["build"], "custom build")
	}
}

func TestLoadToolchains_InvalidJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create invalid JSON
	toolchainsPath := filepath.Join(structylDir, "toolchains.json")
	if err := os.WriteFile(toolchainsPath, []byte("not valid json {"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToolchains(tmpDir)
	if err == nil {
		t.Error("LoadToolchains() expected error for invalid JSON")
	}
}

// =============================================================================
// MergeToolchains Tests
// =============================================================================

func TestMergeToolchains_OverridesDefaults(t *testing.T) {
	defaults := &ToolchainsFile{
		Version: "1.0",
		Toolchains: map[string]ToolchainFileEntry{
			"test": {
				Mise: &MiseConfig{
					PrimaryTool: "original",
					Version:     "1.0",
				},
				Commands: map[string]interface{}{
					"build": "original build",
				},
			},
		},
	}

	loaded := &ToolchainsFile{
		Version: "2.0",
		Toolchains: map[string]ToolchainFileEntry{
			"test": {
				Mise: &MiseConfig{
					Version: "2.0",
				},
				Commands: map[string]interface{}{
					"build": "new build",
				},
			},
		},
	}

	result := MergeToolchains(defaults, loaded)

	if result.Version != "2.0" {
		t.Errorf("Version = %q, want %q", result.Version, "2.0")
	}

	test := result.Toolchains["test"]
	if test.Mise.Version != "2.0" {
		t.Errorf("test.Mise.Version = %q, want %q", test.Mise.Version, "2.0")
	}
	// PrimaryTool should be preserved from defaults
	if test.Mise.PrimaryTool != "original" {
		t.Errorf("test.Mise.PrimaryTool = %q, want %q (preserved)", test.Mise.PrimaryTool, "original")
	}
	if test.Commands["build"] != "new build" {
		t.Errorf("test.Commands[build] = %v, want %q", test.Commands["build"], "new build")
	}
}

func TestMergeToolchains_AddsNewToolchains(t *testing.T) {
	defaults := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"existing": {
				Commands: map[string]interface{}{"build": "existing"},
			},
		},
	}

	loaded := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"new": {
				Commands: map[string]interface{}{"build": "new build"},
			},
		},
	}

	result := MergeToolchains(defaults, loaded)

	// Should have both toolchains
	if _, ok := result.Toolchains["existing"]; !ok {
		t.Error("result missing 'existing' toolchain")
	}
	if _, ok := result.Toolchains["new"]; !ok {
		t.Error("result missing 'new' toolchain")
	}
}

func TestMergeToolchains_PreservesUnmodifiedDefaults(t *testing.T) {
	defaults := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"unchanged": {
				Mise: &MiseConfig{PrimaryTool: "tool1"},
				Commands: map[string]interface{}{
					"build": "default build",
					"test":  "default test",
				},
			},
		},
	}

	loaded := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{}, // Empty - no overrides
	}

	result := MergeToolchains(defaults, loaded)

	unchanged := result.Toolchains["unchanged"]
	if unchanged.Mise.PrimaryTool != "tool1" {
		t.Errorf("unchanged.Mise.PrimaryTool = %q, want %q", unchanged.Mise.PrimaryTool, "tool1")
	}
	if unchanged.Commands["build"] != "default build" {
		t.Errorf("unchanged.Commands[build] = %v, want %q", unchanged.Commands["build"], "default build")
	}
}

// =============================================================================
// mergeToolchainEntry Tests
// =============================================================================

func TestMergeToolchainEntry_MergesMiseConfig(t *testing.T) {
	defaultEntry := ToolchainFileEntry{
		Mise: &MiseConfig{
			PrimaryTool: "original",
			Version:     "1.0",
			ExtraTools:  map[string]string{"tool1": "v1"},
		},
	}

	loadedEntry := ToolchainFileEntry{
		Mise: &MiseConfig{
			Version:    "2.0",
			ExtraTools: map[string]string{"tool2": "v2"},
		},
	}

	result := mergeToolchainEntry(defaultEntry, loadedEntry)

	if result.Mise.PrimaryTool != "original" {
		t.Errorf("PrimaryTool = %q, want %q (preserved)", result.Mise.PrimaryTool, "original")
	}
	if result.Mise.Version != "2.0" {
		t.Errorf("Version = %q, want %q (overridden)", result.Mise.Version, "2.0")
	}
	if result.Mise.ExtraTools["tool1"] != "v1" {
		t.Error("ExtraTools[tool1] should be preserved")
	}
	if result.Mise.ExtraTools["tool2"] != "v2" {
		t.Error("ExtraTools[tool2] should be added")
	}
}

func TestMergeToolchainEntry_MergesCommands(t *testing.T) {
	defaultEntry := ToolchainFileEntry{
		Commands: map[string]interface{}{
			"build": "default build",
			"test":  "default test",
		},
	}

	loadedEntry := ToolchainFileEntry{
		Commands: map[string]interface{}{
			"build": "new build",
			"clean": "new clean",
		},
	}

	result := mergeToolchainEntry(defaultEntry, loadedEntry)

	if result.Commands["build"] != "new build" {
		t.Errorf("Commands[build] = %v, want %q (overridden)", result.Commands["build"], "new build")
	}
	if result.Commands["test"] != "default test" {
		t.Errorf("Commands[test] = %v, want %q (preserved)", result.Commands["test"], "default test")
	}
	if result.Commands["clean"] != "new clean" {
		t.Errorf("Commands[clean] = %v, want %q (added)", result.Commands["clean"], "new clean")
	}
}

func TestMergeToolchainEntry_NilMiseInLoaded(t *testing.T) {
	defaultEntry := ToolchainFileEntry{
		Mise: &MiseConfig{PrimaryTool: "tool"},
	}

	loadedEntry := ToolchainFileEntry{
		Mise: nil, // No mise config in loaded
	}

	result := mergeToolchainEntry(defaultEntry, loadedEntry)

	if result.Mise == nil || result.Mise.PrimaryTool != "tool" {
		t.Error("Mise config should be preserved when loaded has nil")
	}
}

func TestMergeToolchainEntry_NilMiseInDefault(t *testing.T) {
	defaultEntry := ToolchainFileEntry{
		Mise: nil, // No mise config in default
	}

	loadedEntry := ToolchainFileEntry{
		Mise: &MiseConfig{PrimaryTool: "new-tool"},
	}

	result := mergeToolchainEntry(defaultEntry, loadedEntry)

	if result.Mise == nil || result.Mise.PrimaryTool != "new-tool" {
		t.Error("Mise config should be set from loaded when default has nil")
	}
}

// =============================================================================
// deepCopyToolchainEntry Tests
// =============================================================================

func TestDeepCopyToolchainEntry_CopiesAllFields(t *testing.T) {
	original := ToolchainFileEntry{
		Mise: &MiseConfig{
			PrimaryTool: "tool",
			Version:     "1.0",
			ExtraTools:  map[string]string{"extra": "v1"},
		},
		Commands: map[string]interface{}{
			"build": "cmd",
			"multi": []string{"a", "b"},
		},
	}

	copied := deepCopyToolchainEntry(original)

	// Verify values are equal
	if copied.Mise.PrimaryTool != original.Mise.PrimaryTool {
		t.Error("PrimaryTool not copied")
	}
	if copied.Commands["build"] != original.Commands["build"] {
		t.Error("Commands[build] not copied")
	}

	// Verify deep copy (modifications don't affect original)
	copied.Mise.PrimaryTool = "modified"
	if original.Mise.PrimaryTool == "modified" {
		t.Error("Modifying copy should not affect original")
	}

	copied.Mise.ExtraTools["extra"] = "modified"
	if original.Mise.ExtraTools["extra"] == "modified" {
		t.Error("Modifying copy ExtraTools should not affect original")
	}
}

func TestDeepCopyToolchainEntry_NilFields(t *testing.T) {
	original := ToolchainFileEntry{
		Mise:     nil,
		Commands: nil,
	}

	copied := deepCopyToolchainEntry(original)

	if copied.Mise != nil {
		t.Error("nil Mise should remain nil")
	}
	if copied.Commands != nil {
		t.Error("nil Commands should remain nil")
	}
}

// =============================================================================
// deepCopyCommand Tests
// =============================================================================

func TestDeepCopyCommand_String(t *testing.T) {
	original := "command string"
	copied := deepCopyCommand(original)

	if copied != original {
		t.Errorf("deepCopyCommand(string) = %v, want %v", copied, original)
	}
}

func TestDeepCopyCommand_SliceInterface(t *testing.T) {
	original := []interface{}{"a", "b", "c"}
	copied := deepCopyCommand(original)

	copiedSlice, ok := copied.([]interface{})
	if !ok {
		t.Fatalf("deepCopyCommand([]interface{}) returned %T", copied)
	}

	if len(copiedSlice) != len(original) {
		t.Errorf("copied length = %d, want %d", len(copiedSlice), len(original))
	}

	// Verify it's a copy, not same slice
	copiedSlice[0] = "modified"
	if original[0] == "modified" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDeepCopyCommand_SliceString(t *testing.T) {
	original := []string{"a", "b", "c"}
	copied := deepCopyCommand(original)

	copiedSlice, ok := copied.([]string)
	if !ok {
		t.Fatalf("deepCopyCommand([]string) returned %T", copied)
	}

	if len(copiedSlice) != len(original) {
		t.Errorf("copied length = %d, want %d", len(copiedSlice), len(original))
	}

	// Verify it's a copy
	copiedSlice[0] = "modified"
	if original[0] == "modified" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDeepCopyCommand_Nil(t *testing.T) {
	copied := deepCopyCommand(nil)
	if copied != nil {
		t.Errorf("deepCopyCommand(nil) = %v, want nil", copied)
	}
}

// =============================================================================
// GetToolchainFromConfig Tests
// =============================================================================

func TestGetToolchainFromConfig_Found(t *testing.T) {
	config := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"test": {
				Commands: map[string]interface{}{
					"build": "test build",
				},
			},
		},
	}

	tc, ok := GetToolchainFromConfig("test", config)
	if !ok {
		t.Fatal("GetToolchainFromConfig() = not found, want found")
	}
	if tc.Name != "test" {
		t.Errorf("tc.Name = %q, want %q", tc.Name, "test")
	}
	if tc.Commands["build"] != "test build" {
		t.Errorf("tc.Commands[build] = %v, want %q", tc.Commands["build"], "test build")
	}
}

func TestGetToolchainFromConfig_NotFound(t *testing.T) {
	config := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"other": {},
		},
	}

	_, ok := GetToolchainFromConfig("nonexistent", config)
	if ok {
		t.Error("GetToolchainFromConfig() = found, want not found")
	}
}

func TestGetToolchainFromConfig_NilInput(t *testing.T) {
	_, ok := GetToolchainFromConfig("test", nil)
	if ok {
		t.Error("GetToolchainFromConfig(nil) = found, want not found")
	}
}

// =============================================================================
// GetMiseConfigFromToolchains Tests
// =============================================================================

func TestGetMiseConfigFromToolchains_Found(t *testing.T) {
	config := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"test": {
				Mise: &MiseConfig{
					PrimaryTool: "tool",
					Version:     "1.0",
				},
			},
		},
	}

	mise := GetMiseConfigFromToolchains("test", config)
	if mise == nil {
		t.Fatal("GetMiseConfigFromToolchains() = nil, want config")
	}
	if mise.PrimaryTool != "tool" {
		t.Errorf("mise.PrimaryTool = %q, want %q", mise.PrimaryTool, "tool")
	}
}

func TestGetMiseConfigFromToolchains_NotFound(t *testing.T) {
	config := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{},
	}

	mise := GetMiseConfigFromToolchains("nonexistent", config)
	if mise != nil {
		t.Errorf("GetMiseConfigFromToolchains() = %v, want nil", mise)
	}
}

func TestGetMiseConfigFromToolchains_NilInput(t *testing.T) {
	mise := GetMiseConfigFromToolchains("test", nil)
	if mise != nil {
		t.Errorf("GetMiseConfigFromToolchains(nil) = %v, want nil", mise)
	}
}

func TestGetMiseConfigFromToolchains_ToolchainHasNoMise(t *testing.T) {
	config := &ToolchainsFile{
		Toolchains: map[string]ToolchainFileEntry{
			"test": {
				Mise:     nil, // No mise config
				Commands: map[string]interface{}{"build": "cmd"},
			},
		},
	}

	mise := GetMiseConfigFromToolchains("test", config)
	if mise != nil {
		t.Errorf("GetMiseConfigFromToolchains() = %v, want nil (toolchain has no mise)", mise)
	}
}

// =============================================================================
// GetDefaultToolchains Tests
// =============================================================================

func TestGetDefaultToolchains_ReturnsNonEmpty(t *testing.T) {
	defaults := GetDefaultToolchains()

	if defaults == nil {
		t.Fatal("GetDefaultToolchains() = nil")
	}
	if len(defaults.Toolchains) == 0 {
		t.Error("GetDefaultToolchains() returned empty toolchains")
	}
}

func TestGetDefaultToolchains_ContainsKnownToolchains(t *testing.T) {
	defaults := GetDefaultToolchains()

	knownToolchains := []string{
		"cargo", "dotnet", "go", "npm", "pnpm", "yarn", "bun",
		"python", "uv", "poetry", "gradle", "maven", "make", "cmake",
		"swift", "r", "deno", "bundler", "composer", "mix", "sbt",
		"cabal", "stack", "dune", "lein", "zig", "rebar3",
	}

	for _, name := range knownToolchains {
		if _, ok := defaults.Toolchains[name]; !ok {
			t.Errorf("GetDefaultToolchains() missing %q toolchain", name)
		}
	}
}

func TestGetDefaultToolchains_HasValidStructure(t *testing.T) {
	defaults := GetDefaultToolchains()

	if defaults.Schema == "" {
		t.Error("defaults.Schema should not be empty")
	}
	if defaults.Version == "" {
		t.Error("defaults.Version should not be empty")
	}

	// Verify each toolchain has basic structure
	for name, entry := range defaults.Toolchains {
		if entry.Commands == nil {
			t.Errorf("toolchain %q has nil Commands", name)
		}
		// Verify standard commands exist
		standardCmds := []string{"build", "test", "clean"}
		for _, cmd := range standardCmds {
			if _, ok := entry.Commands[cmd]; !ok {
				t.Errorf("toolchain %q missing standard command %q", name, cmd)
			}
		}
	}
}

func TestGetDefaultToolchains_MiseConfigValid(t *testing.T) {
	defaults := GetDefaultToolchains()

	// Most toolchains should have mise config
	toolchainsWithMise := []string{
		"cargo", "dotnet", "go", "npm", "python", "deno",
	}

	for _, name := range toolchainsWithMise {
		entry := defaults.Toolchains[name]
		if entry.Mise == nil {
			t.Errorf("toolchain %q should have Mise config", name)
			continue
		}
		if entry.Mise.PrimaryTool == "" {
			t.Errorf("toolchain %q Mise.PrimaryTool should not be empty", name)
		}
	}
}
