package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
		expectOK bool
	}{
		{"cargo", []string{"Cargo.toml"}, "cargo", true},
		{"go", []string{"go.mod"}, "go", true},
		{"npm", []string{"package.json"}, "npm", true},
		{"pnpm", []string{"pnpm-lock.yaml", "package.json"}, "pnpm", true}, // pnpm takes precedence
		{"yarn", []string{"yarn.lock", "package.json"}, "yarn", true},
		{"bun", []string{"bun.lockb", "package.json"}, "bun", true},
		{"python pyproject", []string{"pyproject.toml"}, "python", true},
		{"uv", []string{"uv.lock", "pyproject.toml"}, "uv", true}, // uv takes precedence
		{"poetry", []string{"poetry.lock", "pyproject.toml"}, "poetry", true},
		{"dotnet csproj", []string{"MyProject.csproj"}, "dotnet", true},
		{"dotnet fsproj", []string{"MyProject.fsproj"}, "dotnet", true},
		{"gradle kotlin", []string{"build.gradle.kts"}, "gradle", true},
		{"gradle groovy", []string{"build.gradle"}, "gradle", true},
		{"maven", []string{"pom.xml"}, "maven", true},
		{"cmake", []string{"CMakeLists.txt"}, "cmake", true},
		{"make", []string{"Makefile"}, "make", true},
		{"swift", []string{"Package.swift"}, "swift", true},
		{"deno json", []string{"deno.json"}, "deno", true},
		{"deno jsonc", []string{"deno.jsonc"}, "deno", true},
		{"bundler", []string{"Gemfile"}, "bundler", true},
		{"composer", []string{"composer.json"}, "composer", true},
		{"mix", []string{"mix.exs"}, "mix", true},
		{"sbt", []string{"build.sbt"}, "sbt", true},
		{"stack", []string{"stack.yaml"}, "stack", true},
		{"cabal", []string{"example.cabal"}, "cabal", true},
		{"dune", []string{"dune-project"}, "dune", true},
		{"lein", []string{"project.clj"}, "lein", true},
		{"zig", []string{"build.zig"}, "zig", true},
		{"rebar3", []string{"rebar.config"}, "rebar3", true},
		{"r", []string{"DESCRIPTION"}, "r", true},
		{"no match", []string{"README.md"}, "", false},
		{"empty", []string{}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
			}

			toolchain, ok := Detect(dir)
			if ok != tt.expectOK {
				t.Errorf("Detect() ok = %v, want %v", ok, tt.expectOK)
			}
			if toolchain != tt.expected {
				t.Errorf("Detect() = %q, want %q", toolchain, tt.expected)
			}
		})
	}
}

func TestGetMarkerFiles(t *testing.T) {
	markers := GetMarkerFiles()
	if len(markers) == 0 {
		t.Error("GetMarkerFiles() returned empty")
	}

	// Check some expected markers exist
	found := make(map[string]bool)
	for _, m := range markers {
		found[m.Toolchain] = true
	}

	expected := []string{"cargo", "go", "npm", "dotnet"}
	for _, tc := range expected {
		if !found[tc] {
			t.Errorf("GetMarkerFiles() missing toolchain %q", tc)
		}
	}
}
