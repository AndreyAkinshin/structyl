package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadTestSuite loads all test cases from a suite directory.
func LoadTestSuite(testsDir, suite, pattern string) ([]TestCase, error) {
	suiteDir := filepath.Join(testsDir, suite)

	if _, err := os.Stat(suiteDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("test suite directory not found: %s", suiteDir)
	}

	// Find matching files
	matches, err := findMatches(suiteDir, pattern)
	if err != nil {
		return nil, err
	}

	var cases []TestCase
	for _, path := range matches {
		tc, err := LoadTestCase(path)
		if err != nil {
			return nil, fmt.Errorf("test suite %q: %w (file: %s)", suite, err, path)
		}
		tc.Suite = suite
		cases = append(cases, *tc)
	}

	// Sort by name for deterministic order
	sort.Slice(cases, func(i, j int) bool {
		return cases[i].Name < cases[j].Name
	})

	return cases, nil
}

// LoadAllSuites loads test cases from all suite directories.
func LoadAllSuites(testsDir, pattern string) (map[string][]TestCase, error) {
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tests directory: %w", err)
	}

	suites := make(map[string][]TestCase)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		suite := entry.Name()
		cases, err := LoadTestSuite(testsDir, suite, pattern)
		if err != nil {
			return nil, err
		}

		if len(cases) > 0 {
			suites[suite] = cases
		}
	}

	return suites, nil
}

// LoadTestCase loads a single test case from a JSON file.
func LoadTestCase(path string) (*TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	input, ok := raw["input"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"input\"")
	}
	output, ok := raw["output"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"output\"")
	}

	// Resolve $file references
	baseDir := filepath.Dir(path)
	input, err = resolveFileRefs(input, baseDir)
	if err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	output, err = resolveFileRefs(output, baseDir)
	if err != nil {
		return nil, fmt.Errorf("output: %w", err)
	}

	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("\"input\" must be an object")
	}

	return &TestCase{
		Name:   strings.TrimSuffix(filepath.Base(path), ".json"),
		Path:   path,
		Input:  inputMap,
		Output: output,
	}, nil
}

// findMatches finds files matching the glob pattern.
//
// Pattern support is intentionally limited to common use cases:
//   - "*.json" matches all .json files in the directory tree
//   - "**/*.json" matches all .json files recursively (simplified double-star)
//   - Standard filepath.Match patterns on the filename portion
//
// Note: This is NOT a full glob implementation. The double-star ("**") support
// is simplified—it matches any .json file recursively rather than providing
// true globstar semantics. For complex patterns, consider using the doublestar
// library or restructuring test directories.
func findMatches(dir, pattern string) ([]string, error) {
	var matches []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get path relative to dir
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Check if it matches the pattern
		matched, err := filepath.Match(pattern, filepath.Base(rel))
		if err != nil {
			return err
		}

		// For simple patterns like "*.json", also check full pattern
		if !matched && strings.Contains(pattern, "**") {
			// Simple double-star support: match any .json file
			if strings.HasSuffix(pattern, "*.json") && strings.HasSuffix(path, ".json") {
				matched = true
			}
		} else if !matched && strings.HasSuffix(pattern, ".json") && strings.HasSuffix(path, ".json") {
			matched = true
		}

		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(matches)
	return matches, nil
}

// resolveFileRefs recursively resolves $file references in test data.
func resolveFileRefs(value interface{}, baseDir string) (interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		// Check if this is a $file reference
		if fileRef, ok := v["$file"].(string); ok {
			return loadFileRef(fileRef, baseDir)
		}

		// Recursively resolve nested values
		result := make(map[string]interface{})
		for key, val := range v {
			resolved, err := resolveFileRefs(val, baseDir)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			resolved, err := resolveFileRefs(val, baseDir)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return value, nil
	}
}

// loadFileRef loads a file referenced by $file.
func loadFileRef(ref, baseDir string) (interface{}, error) {
	// Security: prevent path traversal
	if strings.Contains(ref, "..") {
		return nil, fmt.Errorf("$file path contains \"..\": %s", ref)
	}

	path := filepath.Join(baseDir, ref)

	// Verify the resolved path is still within baseDir
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(absPath, absBase) {
		return nil, fmt.Errorf("$file path escapes test directory: %s", ref)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("$file %q: %w", ref, err)
	}

	// Return as base64 string for binary data, or try to parse as JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue, nil
	}

	// Return raw bytes as string (could be base64 encoded if needed)
	return string(data), nil
}
