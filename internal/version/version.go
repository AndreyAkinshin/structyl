// Package version provides version reading, parsing, and manipulation.
package version

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// SemverRegex validates semantic version strings.
var SemverRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-([a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*))?(\+([a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*))?$`)

// Semver represents a parsed semantic version.
type Semver struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

// Read reads a version from a file and validates it.
func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("version source file not found: %s", path)
	}

	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("version source file is empty: %s", path)
	}

	if err := Validate(version); err != nil {
		return "", fmt.Errorf("invalid version in %s: %w", path, err)
	}

	return version, nil
}

// Write writes a version to a file.
func Write(path, version string) error {
	if err := Validate(version); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(version+"\n"), 0644)
}

// Validate checks if a version string is valid semver.
func Validate(version string) error {
	if !SemverRegex.MatchString(version) {
		return fmt.Errorf("invalid semver format: %q", version)
	}
	return nil
}

// Parse parses a semantic version string.
func Parse(version string) (*Semver, error) {
	match := SemverRegex.FindStringSubmatch(version)
	if match == nil {
		return nil, fmt.Errorf("invalid semver format: %q", version)
	}

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])

	return &Semver{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: match[5], // Group 5 is prerelease without the dash
		Build:      match[8], // Group 8 is build without the plus
	}, nil
}

// String returns the semver string representation.
func (s *Semver) String() string {
	result := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Prerelease != "" {
		result += "-" + s.Prerelease
	}
	if s.Build != "" {
		result += "+" + s.Build
	}
	return result
}

// Bump increments the specified part of the version.
func Bump(current, part string) (string, error) {
	v, err := Parse(current)
	if err != nil {
		return "", err
	}

	switch part {
	case "major":
		v.Major++
		v.Minor = 0
		v.Patch = 0
		v.Prerelease = ""
	case "minor":
		v.Minor++
		v.Patch = 0
		v.Prerelease = ""
	case "patch":
		v.Patch++
		v.Prerelease = ""
	case "prerelease":
		if v.Prerelease == "" {
			// If no prerelease, bump patch and add prerelease
			v.Patch++
			v.Prerelease = "alpha.1"
		} else {
			// Increment prerelease number if present
			v.Prerelease = bumpPrerelease(v.Prerelease)
		}
	case "release":
		// Remove prerelease designation
		v.Prerelease = ""
	default:
		return "", fmt.Errorf("unknown version part: %q (use major, minor, patch, prerelease, or release)", part)
	}

	// Clear build metadata on bump
	v.Build = ""

	return v.String(), nil
}

// bumpPrerelease increments a prerelease version.
func bumpPrerelease(prerelease string) string {
	// Try to find and increment a numeric suffix
	parts := strings.Split(prerelease, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if n, err := strconv.Atoi(parts[i]); err == nil {
			parts[i] = strconv.Itoa(n + 1)
			return strings.Join(parts, ".")
		}
	}

	// No numeric part found, append .1
	return prerelease + ".1"
}

// Compare compares two semver strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b string) (int, error) {
	va, err := Parse(a)
	if err != nil {
		return 0, err
	}
	vb, err := Parse(b)
	if err != nil {
		return 0, err
	}

	if va.Major != vb.Major {
		return compareInt(va.Major, vb.Major), nil
	}
	if va.Minor != vb.Minor {
		return compareInt(va.Minor, vb.Minor), nil
	}
	if va.Patch != vb.Patch {
		return compareInt(va.Patch, vb.Patch), nil
	}

	// Prerelease comparison: version without prerelease is greater
	if va.Prerelease == "" && vb.Prerelease != "" {
		return 1, nil
	}
	if va.Prerelease != "" && vb.Prerelease == "" {
		return -1, nil
	}
	if va.Prerelease != vb.Prerelease {
		return strings.Compare(va.Prerelease, vb.Prerelease), nil
	}

	return 0, nil
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
