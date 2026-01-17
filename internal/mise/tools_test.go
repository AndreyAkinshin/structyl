package mise

import (
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestGetMiseTools_KnownToolchains(t *testing.T) {
	tests := []struct {
		toolchain   string
		wantTool    string
		wantVersion string
	}{
		{"cargo", "rust", "stable"},
		{"dotnet", "dotnet", "8.0"},
		{"go", "go", "1.22"},
		{"npm", "node", "20"},
		{"pnpm", "node", "20"},
		{"yarn", "node", "20"},
		{"bun", "bun", "latest"},
		{"python", "python", "3.12"},
		{"uv", "python", "3.12"},
		{"poetry", "python", "3.12"},
		{"gradle", "java", "temurin-21"},
		{"maven", "java", "temurin-21"},
		{"deno", "deno", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.toolchain, func(t *testing.T) {
			mapping := GetMiseTools(tt.toolchain)
			if mapping == nil {
				t.Errorf("GetMiseTools(%q) = nil, want mapping", tt.toolchain)
				return
			}
			if mapping.PrimaryTool != tt.wantTool {
				t.Errorf("PrimaryTool = %q, want %q", mapping.PrimaryTool, tt.wantTool)
			}
			if mapping.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", mapping.Version, tt.wantVersion)
			}
		})
	}
}

func TestGetMiseTools_UnknownToolchain(t *testing.T) {
	mapping := GetMiseTools("nonexistent")
	if mapping != nil {
		t.Errorf("GetMiseTools(nonexistent) = %v, want nil", mapping)
	}
}

func TestGetMiseTools_Make_NoMiseMapping(t *testing.T) {
	// make toolchain doesn't have a mise mapping (empty PrimaryTool)
	mapping := GetMiseTools("make")
	if mapping != nil {
		t.Errorf("GetMiseTools(make) = %v, want nil", mapping)
	}
}

func TestGetMiseTools_ExtraTools(t *testing.T) {
	tests := []struct {
		toolchain      string
		wantExtraTools map[string]string
	}{
		{"go", map[string]string{"golangci-lint": "latest"}},
		{"pnpm", map[string]string{"pnpm": "9"}},
		{"uv", map[string]string{"uv": "0.5", "ruff": "latest"}},
	}

	for _, tt := range tests {
		t.Run(tt.toolchain, func(t *testing.T) {
			mapping := GetMiseTools(tt.toolchain)
			if mapping == nil {
				t.Errorf("GetMiseTools(%q) = nil", tt.toolchain)
				return
			}
			for tool, version := range tt.wantExtraTools {
				if got := mapping.ExtraTools[tool]; got != version {
					t.Errorf("ExtraTools[%q] = %q, want %q", tool, got, version)
				}
			}
		})
	}
}

func TestGetAllToolsFromConfig(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {Toolchain: "cargo"},
			"go": {Toolchain: "go"},
			"ts": {Toolchain: "npm"},
		},
	}

	tools := GetAllToolsFromConfig(cfg)

	expected := map[string]string{
		"rust":          "stable",
		"go":            "1.22",
		"golangci-lint": "latest",
		"node":          "20",
	}

	for tool, version := range expected {
		if got := tools[tool]; got != version {
			t.Errorf("tools[%q] = %q, want %q", tool, got, version)
		}
	}
}

func TestGetAllToolsFromConfig_Empty(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{},
	}

	tools := GetAllToolsFromConfig(cfg)

	if len(tools) != 0 {
		t.Errorf("GetAllToolsFromConfig(empty) = %v, want empty", tools)
	}
}

func TestGetAllToolsFromConfig_UnsupportedToolchain(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"custom": {Toolchain: "make"}, // make has no mise mapping
		},
	}

	tools := GetAllToolsFromConfig(cfg)

	if len(tools) != 0 {
		t.Errorf("GetAllToolsFromConfig(make) = %v, want empty", tools)
	}
}

func TestGetToolsSorted(t *testing.T) {
	tools := map[string]string{
		"node":          "20",
		"go":            "1.22",
		"rust":          "stable",
		"golangci-lint": "latest",
	}

	sorted := GetToolsSorted(tools)

	// Should be sorted alphabetically
	expected := [][2]string{
		{"go", "1.22"},
		{"golangci-lint", "latest"},
		{"node", "20"},
		{"rust", "stable"},
	}

	if len(sorted) != len(expected) {
		t.Errorf("len(sorted) = %d, want %d", len(sorted), len(expected))
		return
	}

	for i, pair := range sorted {
		if pair[0] != expected[i][0] || pair[1] != expected[i][1] {
			t.Errorf("sorted[%d] = %v, want %v", i, pair, expected[i])
		}
	}
}

func TestIsToolchainSupported(t *testing.T) {
	supported := []string{"cargo", "go", "npm", "python", "uv"}
	for _, tc := range supported {
		if !IsToolchainSupported(tc, nil) {
			t.Errorf("IsToolchainSupported(%q) = false, want true", tc)
		}
	}

	unsupported := []string{"make", "nonexistent"}
	for _, tc := range unsupported {
		if IsToolchainSupported(tc, nil) {
			t.Errorf("IsToolchainSupported(%q) = true, want false", tc)
		}
	}
}

func TestSupportedToolchains(t *testing.T) {
	supported := SupportedToolchains(nil)

	if len(supported) == 0 {
		t.Error("SupportedToolchains() returned empty")
	}

	// Check it's sorted
	for i := 1; i < len(supported); i++ {
		if supported[i] < supported[i-1] {
			t.Errorf("SupportedToolchains() not sorted: %v", supported)
			break
		}
	}

	// Check known toolchains are present
	known := []string{"cargo", "go", "npm", "python"}
	supportedSet := make(map[string]bool)
	for _, s := range supported {
		supportedSet[s] = true
	}

	for _, k := range known {
		if !supportedSet[k] {
			t.Errorf("SupportedToolchains() missing %q", k)
		}
	}
}

// =============================================================================
// GetMiseToolsFromConfig Tests
// =============================================================================

func TestGetMiseToolsFromConfig_NilLoaded_FallsBackToDefault(t *testing.T) {
	// When loaded is nil, should fall back to GetMiseTools
	mapping := GetMiseToolsFromConfig("cargo", nil)
	if mapping == nil {
		t.Fatal("GetMiseToolsFromConfig(cargo, nil) = nil, want mapping")
	}
	if mapping.PrimaryTool != "rust" {
		t.Errorf("PrimaryTool = %q, want %q", mapping.PrimaryTool, "rust")
	}
}

func TestGetMiseToolsFromConfig_UnknownToolchain(t *testing.T) {
	mapping := GetMiseToolsFromConfig("nonexistent", nil)
	if mapping != nil {
		t.Errorf("GetMiseToolsFromConfig(nonexistent, nil) = %v, want nil", mapping)
	}
}

// =============================================================================
// GetAllToolsWithToolchains Tests
// =============================================================================

func TestGetAllToolsWithToolchains_ToolchainVersionOverride(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Toolchain: "go", ToolchainVersion: "1.21"}, // Override default 1.22
		},
	}

	tools := GetAllToolsWithToolchains(cfg, nil)

	if tools["go"] != "1.21" {
		t.Errorf("tools[go] = %q, want %q (target override)", tools["go"], "1.21")
	}
}

func TestGetAllToolsWithToolchains_ToolchainConfigVersionOverride(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Toolchain: "go"}, // No target-level override
		},
		Toolchains: map[string]config.ToolchainConfig{
			"go": {Version: "1.20"}, // Config-level override
		},
	}

	tools := GetAllToolsWithToolchains(cfg, nil)

	if tools["go"] != "1.20" {
		t.Errorf("tools[go] = %q, want %q (toolchain config override)", tools["go"], "1.20")
	}
}

func TestGetAllToolsWithToolchains_TargetOverridesThanToolchainConfig(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Toolchain: "go", ToolchainVersion: "1.21"}, // Target-level override
		},
		Toolchains: map[string]config.ToolchainConfig{
			"go": {Version: "1.20"}, // Config-level override (lower priority)
		},
	}

	tools := GetAllToolsWithToolchains(cfg, nil)

	// Target override should win over toolchain config
	if tools["go"] != "1.21" {
		t.Errorf("tools[go] = %q, want %q (target override should win)", tools["go"], "1.21")
	}
}

func TestGetAllToolsWithToolchains_MiseExtraTools(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Toolchain: "go"},
		},
		Mise: &config.MiseConfig{
			ExtraTools: map[string]string{
				"custom-tool": "1.0.0",
			},
		},
	}

	tools := GetAllToolsWithToolchains(cfg, nil)

	if tools["custom-tool"] != "1.0.0" {
		t.Errorf("tools[custom-tool] = %q, want %q", tools["custom-tool"], "1.0.0")
	}
}

func TestGetAllToolsWithToolchains_DoesNotOverrideExisting(t *testing.T) {
	// When a tool already exists, extra tools should not override it
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"go": {Toolchain: "go"}, // Has golangci-lint as extra tool
		},
		Mise: &config.MiseConfig{
			ExtraTools: map[string]string{
				"golangci-lint": "v1.0.0", // Try to override
			},
		},
	}

	tools := GetAllToolsWithToolchains(cfg, nil)

	// Original extra tool version should be preserved
	if tools["golangci-lint"] != "latest" {
		t.Errorf("tools[golangci-lint] = %q, want %q (original should be preserved)", tools["golangci-lint"], "latest")
	}
}

func TestGetToolsSorted_Empty(t *testing.T) {
	sorted := GetToolsSorted(map[string]string{})
	if len(sorted) != 0 {
		t.Errorf("GetToolsSorted(empty) = %v, want empty", sorted)
	}
}

func TestGetToolsSorted_SingleItem(t *testing.T) {
	sorted := GetToolsSorted(map[string]string{"go": "1.22"})
	if len(sorted) != 1 {
		t.Errorf("len(sorted) = %d, want 1", len(sorted))
	}
	if sorted[0][0] != "go" || sorted[0][1] != "1.22" {
		t.Errorf("sorted[0] = %v, want [go 1.22]", sorted[0])
	}
}
