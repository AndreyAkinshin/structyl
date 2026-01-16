// Package testhelper provides reusable test loading and comparison utilities
// for language implementations in structyl projects.
//
// This package is designed to be used by individual language implementations
// to load test cases and compare their outputs against expected values.
//
// Example usage in a Go test:
//
//	func TestCalculation(t *testing.T) {
//	    root, err := testhelper.FindProjectRoot()
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    cases, err := testhelper.LoadTestSuite(root, "calculation")
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    for _, tc := range cases {
//	        t.Run(tc.Name, func(t *testing.T) {
//	            actual := runCalculation(tc.Input)
//	            if !testhelper.CompareOutput(tc.Output, actual, testhelper.DefaultOptions()) {
//	                t.Errorf("mismatch for %s", tc.Name)
//	            }
//	        })
//	    }
//	}
package testhelper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TestCase represents a single test case loaded from a JSON file.
type TestCase struct {
	// Name is the test case name (derived from filename).
	Name string `json:"-"`

	// Suite is the test suite name (directory name).
	Suite string `json:"-"`

	// Input contains the input data for the test.
	Input map[string]interface{} `json:"input"`

	// Output contains the expected output.
	Output interface{} `json:"output"`

	// Description provides optional documentation.
	Description string `json:"description,omitempty"`

	// Skip marks the test as skipped if true.
	Skip bool `json:"skip,omitempty"`

	// Tags provides optional categorization.
	Tags []string `json:"tags,omitempty"`
}

// LoadTestSuite loads all test cases from a suite directory.
// It looks for JSON files in <projectRoot>/tests/<suite>/*.json
func LoadTestSuite(projectRoot, suite string) ([]TestCase, error) {
	pattern := filepath.Join(projectRoot, "tests", suite, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var cases []TestCase
	for _, f := range files {
		tc, err := LoadTestCase(f)
		if err != nil {
			return nil, err
		}
		tc.Suite = suite
		cases = append(cases, *tc)
	}

	return cases, nil
}

// LoadTestCase loads a single test case from a JSON file.
func LoadTestCase(path string) (*TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tc TestCase
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, err
	}

	tc.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	return &tc, nil
}

// LoadAllSuites loads test cases from all suites in the tests directory.
func LoadAllSuites(projectRoot string) (map[string][]TestCase, error) {
	testsDir := filepath.Join(projectRoot, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, err
	}

	suites := make(map[string][]TestCase)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		suiteName := entry.Name()
		cases, err := LoadTestSuite(projectRoot, suiteName)
		if err != nil {
			return nil, err
		}

		if len(cases) > 0 {
			suites[suiteName] = cases
		}
	}

	return suites, nil
}

// FindProjectRoot walks up the directory tree to find .structyl/config.json.
// It returns the directory containing .structyl/config.json.
func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindProjectRootFrom(cwd)
}

// FindProjectRootFrom finds the project root starting from a specific directory.
func FindProjectRootFrom(startDir string) (string, error) {
	dir := startDir

	for {
		configPath := filepath.Join(dir, ".structyl", "config.json")
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", &ProjectNotFoundError{StartDir: startDir}
}

// ProjectNotFoundError indicates .structyl/config.json was not found.
type ProjectNotFoundError struct {
	StartDir string
}

func (e *ProjectNotFoundError) Error() string {
	return ".structyl/config.json not found (searched from " + e.StartDir + ")"
}

// ListSuites returns the names of all available test suites.
func ListSuites(projectRoot string) ([]string, error) {
	testsDir := filepath.Join(projectRoot, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var suites []string
	for _, entry := range entries {
		if entry.IsDir() {
			suites = append(suites, entry.Name())
		}
	}

	return suites, nil
}

// SuiteExists checks if a test suite exists.
func SuiteExists(projectRoot, suite string) bool {
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	info, err := os.Stat(suiteDir)
	return err == nil && info.IsDir()
}

// TestCaseExists checks if a specific test case exists.
func TestCaseExists(projectRoot, suite, name string) bool {
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
	_, err := os.Stat(path)
	return err == nil
}
