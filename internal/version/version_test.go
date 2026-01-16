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
	_, err := Bump("1.2.3", "invalid")
	if err == nil {
		t.Error("Bump() expected error for invalid part")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
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
