package version

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// Propagate updates version in all configured files.
func Propagate(version string, files []config.VersionFileConfig) error {
	for _, f := range files {
		if err := UpdateFile(f.Path, f.Pattern, f.Replace, version); err != nil {
			return fmt.Errorf("failed to update %s: %w", f.Path, err)
		}
	}
	return nil
}

// UpdateFile updates version in a single file using regex pattern.
func UpdateFile(path, pattern, replace, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	matches := re.FindAllIndex(data, -1)
	if len(matches) == 0 {
		return fmt.Errorf("pattern not found in file")
	}

	// Substitute {version} placeholder in replace string
	replacement := strings.ReplaceAll(replace, "{version}", version)

	result := re.ReplaceAllString(string(data), replacement)

	// Check if anything actually changed
	if result == string(data) {
		return nil // Already up to date
	}

	return os.WriteFile(path, []byte(result), 0644)
}

// CheckConsistency verifies version is consistent across all files.
func CheckConsistency(sourceVersion string, files []config.VersionFileConfig) ([]string, error) {
	var inconsistencies []string

	for _, f := range files {
		data, err := os.ReadFile(f.Path)
		if err != nil {
			inconsistencies = append(inconsistencies, fmt.Sprintf("%s: file not found", f.Path))
			continue
		}

		re, err := regexp.Compile(f.Pattern)
		if err != nil {
			inconsistencies = append(inconsistencies, fmt.Sprintf("%s: invalid pattern: %v", f.Path, err))
			continue
		}

		// Extract version from file
		match := re.FindSubmatch(data)
		if match == nil {
			inconsistencies = append(inconsistencies, fmt.Sprintf("%s: pattern not matched", f.Path))
			continue
		}

		// Check if the file would need updating
		replacement := strings.ReplaceAll(f.Replace, "{version}", sourceVersion)
		result := re.ReplaceAllString(string(data), replacement)

		if result != string(data) {
			inconsistencies = append(inconsistencies, fmt.Sprintf("%s: version mismatch", f.Path))
		}
	}

	return inconsistencies, nil
}
