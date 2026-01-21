// Package version provides version reading, parsing, and manipulation.
package version

import (
	"cmp"
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
// Returns the underlying os error (wrapped) if the file cannot be read,
// allowing callers to use errors.Is(err, os.ErrNotExist) to distinguish
// missing files from other errors.
func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("version source file not found: %w", err)
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

	// Errors ignored: regex guarantees these capture groups contain only digits
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
//
// Supported parts:
//   - "major": increments major, resets minor/patch/prerelease (1.2.3 → 2.0.0)
//   - "minor": increments minor, resets patch/prerelease (1.2.3 → 1.3.0)
//   - "patch": increments patch, clears prerelease (1.2.3 → 1.2.4)
//   - "prerelease": if no prerelease exists, bumps patch and adds "alpha.1" (1.2.3 → 1.2.4-alpha.1);
//     if prerelease exists, increments the numeric suffix (1.2.4-alpha.1 → 1.2.4-alpha.2)
//   - "release": removes prerelease designation (1.2.4-alpha.1 → 1.2.4)
//
// Build metadata is always cleared on bump.
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
		return cmp.Compare(va.Major, vb.Major), nil
	}
	if va.Minor != vb.Minor {
		return cmp.Compare(va.Minor, vb.Minor), nil
	}
	if va.Patch != vb.Patch {
		return cmp.Compare(va.Patch, vb.Patch), nil
	}

	// Prerelease comparison per SemVer §9:
	// - Version without prerelease is greater than version with prerelease
	// - If both have prereleases, compare them per §11
	// - If both empty (or equal), fall through to return 0
	if va.Prerelease == "" && vb.Prerelease != "" {
		return 1, nil
	}
	if va.Prerelease != "" && vb.Prerelease == "" {
		return -1, nil
	}
	if va.Prerelease != vb.Prerelease {
		return comparePrerelease(va.Prerelease, vb.Prerelease), nil
	}

	return 0, nil
}

// comparePrerelease compares prerelease strings per SemVer §11:
// - Split by dots into identifiers
// - Numeric identifiers compare as integers
// - Alphanumeric identifiers compare as strings
// - Numeric identifiers have lower precedence than alphanumeric
// - Fewer identifiers has lower precedence if all preceding are equal
func comparePrerelease(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	minLen := len(partsA)
	if len(partsB) < minLen {
		minLen = len(partsB)
	}

	for i := 0; i < minLen; i++ {
		cmp := compareIdentifier(partsA[i], partsB[i])
		if cmp != 0 {
			return cmp
		}
	}

	// Longer prerelease has higher precedence if all shared identifiers are equal
	return cmp.Compare(len(partsA), len(partsB))
}

// compareIdentifier compares two prerelease identifiers per SemVer §11.
func compareIdentifier(a, b string) int {
	aNum, aIsNum := parseNumeric(a)
	bNum, bIsNum := parseNumeric(b)

	// Both numeric: compare as integers
	if aIsNum && bIsNum {
		return cmp.Compare(aNum, bNum)
	}
	// Numeric has lower precedence than alphanumeric
	if aIsNum {
		return -1
	}
	if bIsNum {
		return 1
	}
	// Both alphanumeric: string comparison
	return strings.Compare(a, b)
}

// parseNumeric attempts to parse a string as a non-negative integer.
// Returns (value, true) if successful, (0, false) otherwise.
func parseNumeric(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	// Reject leading zeros (except "0" itself) per SemVer spec
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
