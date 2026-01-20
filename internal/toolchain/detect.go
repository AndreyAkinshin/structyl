package toolchain

import (
	"os"
	"path/filepath"
	"strings"
)

// MarkerFile defines a file pattern and its associated toolchain.
type MarkerFile struct {
	Pattern   string
	Toolchain string
}

// markerFiles defines the auto-detection order for toolchains.
// First match wins.
var markerFiles = []MarkerFile{
	// Rust
	{"Cargo.toml", "cargo"},
	// Go
	{"go.mod", "go"},
	// JavaScript/TypeScript - specific lock files first
	{"deno.jsonc", "deno"},
	{"deno.json", "deno"},
	{"pnpm-lock.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"bun.lockb", "bun"},
	{"package.json", "npm"},
	// Python - specific lock files first
	{"uv.lock", "uv"},
	{"poetry.lock", "poetry"},
	{"pyproject.toml", "python"},
	{"setup.py", "python"},
	// JVM
	{"build.gradle.kts", "gradle"},
	{"build.gradle", "gradle"},
	{"pom.xml", "maven"},
	{"build.sbt", "sbt"},
	// Apple
	{"Package.swift", "swift"},
	// C/C++
	{"CMakeLists.txt", "cmake"},
	// Generic
	{"Makefile", "make"},
	// .NET - solution/props files often at root, csproj in subdirs
	{"*.sln", "dotnet"},
	{"Directory.Build.props", "dotnet"},
	{"global.json", "dotnet"},
	{"*.csproj", "dotnet"},
	{"*.fsproj", "dotnet"},
	// Ruby
	{"Gemfile", "bundler"},
	// PHP
	{"composer.json", "composer"},
	// Elixir
	{"mix.exs", "mix"},
	// Haskell - stack before cabal (more specific)
	{"stack.yaml", "stack"},
	{"*.cabal", "cabal"},
	// OCaml
	{"dune-project", "dune"},
	// Clojure
	{"project.clj", "lein"},
	// Zig
	{"build.zig", "zig"},
	// Erlang
	{"rebar.config", "rebar3"},
	// R
	{"DESCRIPTION", "r"},
}

// Detect attempts to auto-detect the toolchain for a directory.
// Returns the toolchain name and true if detected, empty string and false otherwise.
func Detect(dir string) (string, bool) {
	for _, marker := range markerFiles {
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

// GetMarkerFiles returns the list of marker file patterns and their toolchains.
func GetMarkerFiles() []MarkerFile {
	return markerFiles
}
