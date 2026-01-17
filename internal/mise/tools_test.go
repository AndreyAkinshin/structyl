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
		if !IsToolchainSupported(tc) {
			t.Errorf("IsToolchainSupported(%q) = false, want true", tc)
		}
	}

	unsupported := []string{"make", "nonexistent"}
	for _, tc := range unsupported {
		if IsToolchainSupported(tc) {
			t.Errorf("IsToolchainSupported(%q) = true, want false", tc)
		}
	}
}

func TestSupportedToolchains(t *testing.T) {
	supported := SupportedToolchains()

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
