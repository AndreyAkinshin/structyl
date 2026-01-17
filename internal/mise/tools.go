// Package mise provides mise integration for structyl projects.
package mise

import (
	"sort"

	"github.com/AndreyAkinshin/structyl/internal/config"
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

// GetMiseTools returns the mise tool mapping for a given toolchain.
// Returns nil if the toolchain is not recognized or has no mise mapping.
func GetMiseTools(toolchain string) *ToolMapping {
	if mapping, ok := toolchainMappings[toolchain]; ok {
		if mapping.PrimaryTool != "" {
			return &mapping
		}
	}
	return nil
}

// GetAllToolsFromConfig aggregates all unique mise tools from a project config.
// Returns a map of tool names to versions.
func GetAllToolsFromConfig(cfg *config.Config) map[string]string {
	tools := make(map[string]string)

	for _, target := range cfg.Targets {
		mapping := GetMiseTools(target.Toolchain)
		if mapping == nil {
			continue
		}

		// Add primary tool if not already present or if version is more specific
		if mapping.PrimaryTool != "" {
			if _, exists := tools[mapping.PrimaryTool]; !exists {
				tools[mapping.PrimaryTool] = mapping.Version
			}
		}

		// Add extra tools
		for tool, version := range mapping.ExtraTools {
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

// IsToolchainSupported checks if a toolchain has mise support.
func IsToolchainSupported(toolchain string) bool {
	return GetMiseTools(toolchain) != nil
}

// SupportedToolchains returns a list of all toolchains with mise support.
func SupportedToolchains() []string {
	var supported []string
	for name, mapping := range toolchainMappings {
		if mapping.PrimaryTool != "" {
			supported = append(supported, name)
		}
	}
	sort.Strings(supported)
	return supported
}
