package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// DetectToolchain attempts to auto-detect the toolchain for a directory.
// This delegates to toolchain.Detect to maintain a single source of truth
// for marker file patterns.
func DetectToolchain(dir string) (string, bool) {
	return toolchain.Detect(dir)
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
// These directories fall into three categories:
//
// 1. Dependency directories (contain third-party code, not project targets):
//   - node_modules: npm/pnpm/yarn packages
//   - vendor: Go modules, PHP composer, Ruby bundler
//
// 2. Build output directories (generated artifacts, not source targets):
//   - build, dist, out: common output directories
//   - target: Rust/Cargo build directory
//   - artifacts: structyl output directory
//
// 3. Support directories (auxiliary content, not buildable targets):
//   - tests: cross-language test data (JSON fixtures)
//   - templates: project templates, scaffolding
//   - docs: documentation (VitePress, etc.)
//   - scripts: build/CI scripts
func isExcludedDir(name string) bool {
	excluded := map[string]bool{
		// Dependency directories
		"node_modules": true,
		"vendor":       true,
		// Build output directories
		"build":     true,
		"dist":      true,
		"out":       true,
		"target":    true,
		"artifacts": true,
		// Support directories
		"tests":     true,
		"templates": true,
		"docs":      true,
		"scripts":   true,
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
