// Package mise provides mise integration for structyl projects.
package mise

import (
	"sort"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// ToolMapping maps a structyl toolchain to mise tools.
type ToolMapping struct {
	// PrimaryTool is the main tool name in mise (e.g., "rust", "node")
	PrimaryTool string
	// Version is the version constraint for the primary tool
	Version string
	// ExtraTools contains additional tools needed (e.g., golangci-lint for go)
	ExtraTools map[string]string
}

// toolchainMappings maps structyl toolchain names to mise tool configurations.
var toolchainMappings = map[string]ToolMapping{
	"cargo": {
		PrimaryTool: "rust",
		Version:     "stable",
	},
	"dotnet": {
		PrimaryTool: "dotnet",
		Version:     "8.0",
	},
	"go": {
		PrimaryTool: "go",
		Version:     "1.22",
		ExtraTools: map[string]string{
			"golangci-lint": "latest",
		},
	},
	"npm": {
		PrimaryTool: "node",
		Version:     "20",
	},
	"pnpm": {
		PrimaryTool: "node",
		Version:     "20",
		ExtraTools: map[string]string{
			"pnpm": "9",
		},
	},
	"yarn": {
		PrimaryTool: "node",
		Version:     "20",
	},
	"bun": {
		PrimaryTool: "bun",
		Version:     "latest",
	},
	"python": {
		PrimaryTool: "python",
		Version:     "3.12",
	},
	"uv": {
		PrimaryTool: "python",
		Version:     "3.12",
		ExtraTools: map[string]string{
			"uv":   "0.5",
			"ruff": "latest",
		},
	},
	"poetry": {
		PrimaryTool: "python",
		Version:     "3.12",
	},
	"gradle": {
		PrimaryTool: "java",
		Version:     "temurin-21",
	},
	"maven": {
		PrimaryTool: "java",
		Version:     "temurin-21",
	},
	"deno": {
		PrimaryTool: "deno",
		Version:     "latest",
	},
	"swift": {
		PrimaryTool: "swift",
		Version:     "latest",
	},
	"bundler": {
		PrimaryTool: "ruby",
		Version:     "3.3",
	},
	"composer": {
		PrimaryTool: "php",
		Version:     "8.3",
	},
	"mix": {
		PrimaryTool: "elixir",
		Version:     "1.16",
	},
	"sbt": {
		PrimaryTool: "java",
		Version:     "temurin-21",
		ExtraTools: map[string]string{
			"sbt": "latest",
		},
	},
	"cabal": {
		PrimaryTool: "ghc",
		Version:     "9.8",
	},
	"stack": {
		PrimaryTool: "ghc",
		Version:     "9.8",
	},
	"dune": {
		PrimaryTool: "ocaml",
		Version:     "5.1",
	},
	"lein": {
		PrimaryTool: "java",
		Version:     "temurin-21",
		ExtraTools: map[string]string{
			"leiningen": "latest",
		},
	},
	"zig": {
		PrimaryTool: "zig",
		Version:     "latest",
	},
	"rebar3": {
		PrimaryTool: "erlang",
		Version:     "26",
	},
	"cmake": {
		PrimaryTool: "cmake",
		Version:     "latest",
	},
	"make": {
		PrimaryTool: "",
		Version:     "",
	},
	"r": {
		PrimaryTool: "r",
		Version:     "4.4",
	},
}

// GetMiseTools returns the mise tool mapping for a given toolchain from hardcoded defaults.
// Returns nil if the toolchain is not recognized or has no mise mapping.
// Deprecated: Use GetMiseToolsFromConfig for loaded toolchains configuration.
func GetMiseTools(tcName string) *ToolMapping {
	if mapping, ok := toolchainMappings[tcName]; ok {
		if mapping.PrimaryTool != "" {
			return &mapping
		}
	}
	return nil
}

// GetMiseToolsFromConfig returns the mise tool mapping for a given toolchain
// using the loaded toolchains configuration.
// Returns nil if the toolchain has no mise mapping.
func GetMiseToolsFromConfig(tcName string, loaded *toolchain.ToolchainsFile) *ToolMapping {
	if loaded == nil {
		return GetMiseTools(tcName)
	}

	miseConfig := toolchain.GetMiseConfigFromToolchains(tcName, loaded)
	if miseConfig == nil || miseConfig.PrimaryTool == "" {
		return nil
	}

	return &ToolMapping{
		PrimaryTool: miseConfig.PrimaryTool,
		Version:     miseConfig.Version,
		ExtraTools:  miseConfig.ExtraTools,
	}
}

// GetAllToolsFromConfig aggregates all unique mise tools from a project config.
// Returns a map of tool names to versions.
// Deprecated: Use GetAllToolsWithToolchains for loaded toolchains configuration.
func GetAllToolsFromConfig(cfg *config.Config) map[string]string {
	return GetAllToolsWithToolchains(cfg, nil)
}

// GetAllToolsWithToolchains aggregates all unique mise tools from a project config
// using the loaded toolchains configuration.
// Returns a map of tool names to versions.
// Priority for version resolution:
//  1. Target-level toolchain_version
//  2. Custom toolchain definition version (in config.json)
//  3. Loaded toolchains.json configuration
//  4. Default from built-in MiseToolMapping
func GetAllToolsWithToolchains(cfg *config.Config, loaded *toolchain.ToolchainsFile) map[string]string {
	tools := make(map[string]string)

	for _, target := range cfg.Targets {
		mapping := GetMiseToolsFromConfig(target.Toolchain, loaded)
		if mapping == nil {
			continue
		}

		// Determine version with priority: target > toolchain config > loaded/default
		version := mapping.Version
		if tcCfg, ok := cfg.Toolchains[target.Toolchain]; ok && tcCfg.Version != "" {
			version = tcCfg.Version
		}
		if target.ToolchainVersion != "" {
			version = target.ToolchainVersion
		}

		// Add primary tool if not already present
		if mapping.PrimaryTool != "" {
			if _, exists := tools[mapping.PrimaryTool]; !exists {
				tools[mapping.PrimaryTool] = version
			}
		}

		// Add extra tools from mapping
		for tool, ver := range mapping.ExtraTools {
			if _, exists := tools[tool]; !exists {
				tools[tool] = ver
			}
		}
	}

	// Add extra tools from mise config
	if cfg.Mise != nil {
		for tool, version := range cfg.Mise.ExtraTools {
			if _, exists := tools[tool]; !exists {
				tools[tool] = version
			}
		}
	}

	return tools
}

// GetToolsSorted returns tools as sorted key-value pairs for deterministic output.
func GetToolsSorted(tools map[string]string) [][2]string {
	// Collect keys
	keys := make([]string, 0, len(tools))
	for k := range tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted pairs
	result := make([][2]string, 0, len(tools))
	for _, k := range keys {
		result = append(result, [2]string{k, tools[k]})
	}
	return result
}

// IsToolchainSupported checks if a toolchain has mise support using loaded config.
func IsToolchainSupported(tcName string, loaded *toolchain.ToolchainsFile) bool {
	return GetMiseToolsFromConfig(tcName, loaded) != nil
}

// SupportedToolchains returns a list of all toolchains with mise support.
func SupportedToolchains(loaded *toolchain.ToolchainsFile) []string {
	var supported []string

	if loaded != nil {
		for name, entry := range loaded.Toolchains {
			if entry.Mise != nil && entry.Mise.PrimaryTool != "" {
				supported = append(supported, name)
			}
		}
	} else {
		for name, mapping := range toolchainMappings {
			if mapping.PrimaryTool != "" {
				supported = append(supported, name)
			}
		}
	}

	sort.Strings(supported)
	return supported
}
