package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ToolchainMarker defines a file pattern and its associated toolchain.
type ToolchainMarker struct {
	Pattern   string
	Toolchain string
}

// toolchainMarkers defines the auto-detection order for toolchains.
// First match wins.
var toolchainMarkers = []ToolchainMarker{
	{"Cargo.toml", "cargo"},
	{"go.mod", "go"},
	{"pnpm-lock.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"bun.lockb", "bun"},
	{"package.json", "npm"},
	{"uv.lock", "uv"},
	{"poetry.lock", "poetry"},
	{"pyproject.toml", "python"},
	{"setup.py", "python"},
	{"build.gradle.kts", "gradle"},
	{"build.gradle", "gradle"},
	{"pom.xml", "maven"},
	{"Package.swift", "swift"},
	{"CMakeLists.txt", "cmake"},
	{"Makefile", "make"},
	// Glob patterns for .NET
	{"*.csproj", "dotnet"},
	{"*.fsproj", "dotnet"},
}

// DetectToolchain attempts to auto-detect the toolchain for a directory.
func DetectToolchain(dir string) (string, bool) {
	for _, marker := range toolchainMarkers {
		if strings.Contains(marker.Pattern, "*") {
			// Glob pattern
			matches, err := filepath.Glob(filepath.Join(dir, marker.Pattern))
			if err == nil && len(matches) > 0 {
				return marker.Toolchain, true
			}
		} else {
			// Exact file match
			path := filepath.Join(dir, marker.Pattern)
			if _, err := os.Stat(path); err == nil {
				return marker.Toolchain, true
			}
		}
	}
	return "", false
}

// DiscoverTargets finds all potential targets in a project root.
// This is used when the targets section is empty or absent.
func DiscoverTargets(root string) (map[string]string, error) {
	targets := make(map[string]string)

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip hidden directories and common non-target directories
		if strings.HasPrefix(name, ".") || isExcludedDir(name) {
			continue
		}

		dir := filepath.Join(root, name)
		if toolchain, found := DetectToolchain(dir); found {
			targets[name] = toolchain
		}
	}

	return targets, nil
}

// isExcludedDir returns true for directories that should never be auto-discovered as targets.
func isExcludedDir(name string) bool {
	excluded := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		"tests":        true,
		"templates":    true,
		"artifacts":    true,
		"docs":         true,
		"scripts":      true,
		"build":        true,
		"dist":         true,
		"out":          true,
		"target":       true, // Rust build dir
	}
	return excluded[name]
}

// validateTargetDirectory checks if a target directory exists.
func validateTargetDirectory(dir string, targetName string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("target %q: directory %q does not exist", targetName, dir)
	}
	if err != nil {
		return fmt.Errorf("target %q: cannot access directory %q: %w", targetName, dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target %q: %q is not a directory", targetName, dir)
	}
	return nil
}
