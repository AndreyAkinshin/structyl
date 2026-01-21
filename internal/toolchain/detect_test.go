package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	// Note: Lockfile precedence tests (pnpm/yarn/bun over npm, uv/poetry over python)
	// are covered in TestDetect_Priority with explicit descriptions.
	tests := []struct {
		name     string
		files    []string
		expected string
		expectOK bool
	}{
		{"cargo", []string{"Cargo.toml"}, "cargo", true},
		{"go", []string{"go.mod"}, "go", true},
		{"npm", []string{"package.json"}, "npm", true},
		{"python pyproject", []string{"pyproject.toml"}, "python", true},
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

func TestDetect_UnknownMarkerFile(t *testing.T) {
	t.Parallel()
	// Verify that files resembling marker files but not in the detection list
	// do not cause false positives
	tests := []struct {
		name  string
		files []string
	}{
		{"unknown config file", []string{"unknown.config"}},
		{"custom build file", []string{"BUILD.custom"}},
		{"similar to cargo", []string{"Cargo.lock"}},             // lock file without .toml
		{"similar to go", []string{"go.sum"}},                    // sum without mod
		{"json file", []string{"config.json"}},                   // generic json
		{"toml file", []string{"config.toml"}},                   // generic toml
		{"makefile variant", []string{"makefile.inc"}},           // not exactly Makefile
		{"requirements.txt alone", []string{"requirements.txt"}}, // no pyproject.toml
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
			}

			toolchain, ok := Detect(dir)
			if ok {
				t.Errorf("Detect() = (%q, true), want (\"\", false) for files %v", toolchain, tt.files)
			}
		})
	}
}

func TestDetect_Priority(t *testing.T) {
	// Verify that more specific toolchains are detected before generic ones
	// when multiple marker files exist
	tests := []struct {
		name     string
		files    []string
		expected string
		desc     string
	}{
		{
			name:     "pnpm over npm",
			files:    []string{"pnpm-lock.yaml", "package.json"},
			expected: "pnpm",
			desc:     "pnpm lockfile should take precedence over bare package.json",
		},
		{
			name:     "yarn over npm",
			files:    []string{"yarn.lock", "package.json"},
			expected: "yarn",
			desc:     "yarn lockfile should take precedence over bare package.json",
		},
		{
			name:     "bun over npm",
			files:    []string{"bun.lockb", "package.json"},
			expected: "bun",
			desc:     "bun lockfile should take precedence over bare package.json",
		},
		{
			name:     "uv over python",
			files:    []string{"uv.lock", "pyproject.toml"},
			expected: "uv",
			desc:     "uv lockfile should take precedence over bare pyproject.toml",
		},
		{
			name:     "poetry over python",
			files:    []string{"poetry.lock", "pyproject.toml"},
			expected: "poetry",
			desc:     "poetry lockfile should take precedence over bare pyproject.toml",
		},
		{
			name:     "npm fallback",
			files:    []string{"package.json"},
			expected: "npm",
			desc:     "bare package.json should detect npm when no lockfile exists",
		},
		{
			name:     "python fallback",
			files:    []string{"pyproject.toml"},
			expected: "python",
			desc:     "bare pyproject.toml should detect python when no lockfile exists",
		},
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
			if !ok {
				t.Fatalf("Detect() returned ok=false for %v", tt.files)
			}
			if toolchain != tt.expected {
				t.Errorf("%s: Detect() = %q, want %q", tt.desc, toolchain, tt.expected)
			}
		})
	}
}
