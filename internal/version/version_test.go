package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	t.Parallel()
	tests := []string{
		"0.0.0",
		"1.0.0",
		"1.2.3",
		"10.20.30",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-0.3.7",
		"1.0.0-x.7.z.92",
		"1.0.0+build",
		"1.0.0+build.123",
		"1.0.0-alpha+build",
		"1.0.0-beta.1+build.456",
	}

	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if err := Validate(v); err != nil {
				t.Errorf("Validate(%q) = %v, want nil", v, err)
			}
		})
	}
}

func TestValidate_Invalid(t *testing.T) {
	t.Parallel()
	tests := []string{
		"",
		"1",
		"1.2",
		"1.2.3.4",
		"v1.2.3",
		"1.2.3-",
		"1.2.3+",
		"1.2.3-@",
		"a.b.c",
	}

	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if err := Validate(v); err == nil {
				t.Errorf("Validate(%q) = nil, want error", v)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input      string
		major      int
		minor      int
		patch      int
		prerelease string
		build      string
	}{
		{"1.2.3", 1, 2, 3, "", ""},
		{"0.0.0", 0, 0, 0, "", ""},
		{"1.0.0-alpha", 1, 0, 0, "alpha", ""},
		{"1.0.0-alpha.1", 1, 0, 0, "alpha.1", ""},
		{"1.0.0+build", 1, 0, 0, "", "build"},
		{"1.0.0-rc.1+build.123", 1, 0, 0, "rc.1", "build.123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if v.Major != tt.major {
				t.Errorf("Major = %d, want %d", v.Major, tt.major)
			}
			if v.Minor != tt.minor {
				t.Errorf("Minor = %d, want %d", v.Minor, tt.minor)
			}
			if v.Patch != tt.patch {
				t.Errorf("Patch = %d, want %d", v.Patch, tt.patch)
			}
			if v.Prerelease != tt.prerelease {
				t.Errorf("Prerelease = %q, want %q", v.Prerelease, tt.prerelease)
			}
			if v.Build != tt.build {
				t.Errorf("Build = %q, want %q", v.Build, tt.build)
			}
		})
	}
}

func TestSemver_String(t *testing.T) {
	tests := []struct {
		v    Semver
		want string
	}{
		{Semver{1, 2, 3, "", ""}, "1.2.3"},
		{Semver{1, 0, 0, "alpha", ""}, "1.0.0-alpha"},
		{Semver{1, 0, 0, "", "build"}, "1.0.0+build"},
		{Semver{1, 0, 0, "rc.1", "build"}, "1.0.0-rc.1+build"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBump(t *testing.T) {
	tests := []struct {
		current string
		part    string
		want    string
	}{
		{"1.2.3", "major", "2.0.0"},
		{"1.2.3", "minor", "1.3.0"},
		{"1.2.3", "patch", "1.2.4"},
		{"1.0.0-alpha", "major", "2.0.0"},
		{"1.0.0-alpha", "release", "1.0.0"},
		{"1.2.3", "prerelease", "1.2.4-alpha.1"},
		{"1.0.0-alpha.1", "prerelease", "1.0.0-alpha.2"},
		{"1.0.0-rc", "prerelease", "1.0.0-rc.1"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"/"+tt.part, func(t *testing.T) {
			got, err := Bump(tt.current, tt.part)
			if err != nil {
				t.Fatalf("Bump() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Bump() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBump_Invalid(t *testing.T) {
	t.Parallel()
	invalidParts := []string{
		"invalid", // unknown keyword
		"",        // empty string
		"1",       // numeric string
		"Major",   // wrong case
		"PATCH",   // all caps
		"unknown", // another unknown
	}
	for _, part := range invalidParts {
		t.Run(part, func(t *testing.T) {
			t.Parallel()
			_, err := Bump("1.2.3", part)
			if err == nil {
				t.Errorf("Bump(1.2.3, %q) = nil, want error", part)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Basic version comparison
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},

		// Prerelease vs release
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-alpha", 1},

		// Alphanumeric prerelease identifiers (string comparison)
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-alpha", 1},

		// Numeric prerelease identifiers (SemVer §11: numeric comparison)
		{"1.0.0-alpha.2", "1.0.0-alpha.10", -1},
		{"1.0.0-alpha.10", "1.0.0-alpha.2", 1},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1},
		{"1.0.0-1", "1.0.0-2", -1},
		{"1.0.0-10", "1.0.0-2", 1},

		// Mixed identifiers (SemVer §11: numeric has lower precedence than alphanumeric)
		{"1.0.0-1", "1.0.0-alpha", -1},
		{"1.0.0-alpha", "1.0.0-1", 1},

		// Equal prereleases
		{"1.0.0-alpha.1", "1.0.0-alpha.1", 0},

		// Build metadata MUST be ignored per SemVer §10
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"1.0.0+abc", "1.0.0", 0},
		{"1.0.0-alpha+build", "1.0.0-alpha+other", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := Compare(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")

	// Test valid version
	if err := os.WriteFile(path, []byte("1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if v != "1.2.3" {
		t.Errorf("Read() = %q, want %q", v, "1.2.3")
	}

	// Test missing file
	_, err = Read(filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Error("Read() expected error for missing file")
	}

	// Test invalid version
	if err := os.WriteFile(path, []byte("invalid\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = Read(path)
	if err == nil {
		t.Error("Read() expected error for invalid version")
	}
}

func TestRead_WhitespaceEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"trailing_newline", "1.2.3\n", "1.2.3"},
		{"no_trailing_newline", "1.2.3", "1.2.3"},
		{"crlf", "1.2.3\r\n", "1.2.3"},
		{"trailing_spaces", "1.2.3   \n", "1.2.3"},
		{"multiple_newlines", "1.2.3\n\n", "1.2.3"},
		{"leading_trailing_spaces", "  1.2.3  \n", "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "VERSION")

			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Read() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")

	err := Write(path, "1.2.3")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "1.2.3\n" {
		t.Errorf("file content = %q, want %q", string(data), "1.2.3\n")
	}
}

func TestWrite_InvalidVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")

	// Write with invalid version should fail validation before writing
	err := Write(path, "invalid-version")
	if err == nil {
		t.Error("Write() expected error for invalid version")
	}

	// File should not have been created
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("file should not exist after failed Write")
	}
}

func TestWrite_InvalidPath(t *testing.T) {
	t.Parallel()
	// Try to write to a path where the parent directory doesn't exist
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deeply", "VERSION")

	err := Write(path, "1.2.3")
	if err == nil {
		t.Error("Write() expected error for invalid path")
	}
}

func TestRead_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")

	// Create empty file
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path)
	if err == nil {
		t.Error("Read() expected error for empty file")
	}
}

func TestRead_WhitespaceOnlyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")

	// Create file with only whitespace
	if err := os.WriteFile(path, []byte("   \n\t\n  "), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path)
	if err == nil {
		t.Error("Read() expected error for whitespace-only file")
	}
}

func TestCompare_IdenticalVersions(t *testing.T) {
	t.Parallel()
	tests := []string{
		"1.0.0",
		"1.2.3",
		"0.0.0",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0+build",
		"1.0.0-rc.1+build.123",
	}

	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			got, err := Compare(v, v)
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if got != 0 {
				t.Errorf("Compare(%q, %q) = %d, want 0", v, v, got)
			}
		})
	}
}
