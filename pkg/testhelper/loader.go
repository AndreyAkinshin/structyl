// Package testhelper provides reusable test loading and comparison utilities
// for language implementations in structyl projects.
//
// This package is designed to be used by individual language implementations
// to load test cases and compare their outputs against expected values.
//
// Limitations:
//   - $file references are not supported. File reference resolution is only
//     available in the internal test runner. Test cases using $file syntax
//     should use the internal tests package or embed data directly in JSON.
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TestCase represents a single test case loaded from a JSON file.
//
// JSON Type Mapping:
//   - JSON number → Go float64 (all numeric values, including integers)
//   - JSON string → Go string
//   - JSON boolean → Go bool
//   - JSON array → Go []interface{}
//   - JSON object → Go map[string]interface{}
//   - JSON null → Go nil
//
// Note: JSON does not distinguish integers from floats. All numbers in Input
// and Output are unmarshaled as float64. Callers should convert as needed
// (e.g., int(tc.Input["count"].(float64))).
type TestCase struct {
	// Name is the test case name (derived from filename).
	Name string `json:"-"`

	// Suite is the test suite name (directory name).
	// Zero value ("") indicates suite was not set. Use LoadTestSuite for
	// automatic population, or LoadTestCaseWithSuite to set explicitly.
	// Note: LoadTestCase does NOT populate this field.
	Suite string `json:"-"`

	// Input contains the input data for the test as a JSON object.
	// Input MUST be a JSON object (not an array or scalar). The object may be
	// empty ({}). A nil Input (missing field) causes a validation error.
	//
	// Why object-only? Test inputs typically represent named parameters or
	// configuration. Objects provide named access to individual values and
	// align with how most test frameworks expect structured input.
	// Arrays and scalar values as top-level input are not supported.
	Input map[string]interface{} `json:"input"`

	// Output contains the expected output.
	// Output must not be nil; a nil Output causes a validation error.
	// Use an explicit value (e.g., empty string, empty object) for expected empty output.
	//
	// Why interface{}? Unlike Input, Output may be any JSON-serializable value:
	// a scalar (number, string, boolean, null), an array, or an object.
	// This flexibility accommodates functions that return simple values,
	// collections, or complex structures.
	Output interface{} `json:"output"`

	// Description provides optional documentation.
	Description string `json:"description,omitempty"`

	// Skip marks the test as skipped if true.
	Skip bool `json:"skip,omitempty"`

	// Tags provides optional categorization.
	Tags []string `json:"tags,omitempty"`
}

// String returns a human-readable representation of TestCase for debugging.
func (tc TestCase) String() string {
	skip := ""
	if tc.Skip {
		skip = " [SKIP]"
	}
	if tc.Suite != "" {
		return fmt.Sprintf("TestCase{%s/%s%s}", tc.Suite, tc.Name, skip)
	}
	return fmt.Sprintf("TestCase{%s%s}", tc.Name, skip)
}

// LoadTestSuite loads all test cases from a suite directory.
// It looks for JSON files in <projectRoot>/tests/<suite>/*.json.
// Returns SuiteNotFoundError if the suite directory does not exist.
// Returns an empty slice (not nil) if the suite exists but contains no JSON files.
func LoadTestSuite(projectRoot, suite string) ([]TestCase, error) {
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	if _, err := os.Stat(suiteDir); os.IsNotExist(err) {
		return nil, &SuiteNotFoundError{Suite: suite}
	}

	pattern := filepath.Join(suiteDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Sort files for deterministic ordering across platforms.
	// filepath.Glob returns files in filesystem-dependent order.
	sort.Strings(files)

	cases := make([]TestCase, 0, len(files))
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
// Returns an error if the file cannot be read, contains invalid JSON,
// or is missing required fields (input and output).
//
// Note: This function sets TestCase.Name from the filename but does NOT set
// TestCase.Suite. Use LoadTestCaseWithSuite or LoadTestSuite to load test cases
// with suite information, or set the Suite field manually after loading.
func LoadTestCase(path string) (*TestCase, error) {
	return loadTestCaseInternal(path, "")
}

// LoadTestCaseWithSuite loads a single test case from a JSON file and sets the suite name.
// This is a convenience function that combines LoadTestCase with setting the Suite field.
// Returns an error if the file cannot be read, contains invalid JSON,
// or is missing required fields (input and output).
func LoadTestCaseWithSuite(path, suite string) (*TestCase, error) {
	return loadTestCaseInternal(path, suite)
}

// loadTestCaseInternal is the shared implementation for LoadTestCase and LoadTestCaseWithSuite.
func loadTestCaseInternal(path, suite string) (*TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tc TestCase
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, err
	}

	// Validate required fields per spec
	if tc.Input == nil {
		return nil, fmt.Errorf("%s: missing required field \"input\"", filepath.Base(path))
	}
	if tc.Output == nil {
		return nil, fmt.Errorf("%s: missing required field \"output\"", filepath.Base(path))
	}

	tc.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	tc.Suite = suite
	return &tc, nil
}

// LoadAllSuites loads test cases from all suites in the tests directory.
// Returns an empty map (not nil) if the tests directory doesn't exist.
func LoadAllSuites(projectRoot string) (map[string][]TestCase, error) {
	testsDir := filepath.Join(projectRoot, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]TestCase{}, nil
		}
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

// ErrProjectNotFound is returned when .structyl/config.json cannot be found.
// Use errors.Is(err, ErrProjectNotFound) to check for this condition.
var ErrProjectNotFound = errors.New("project not found")

// ProjectNotFoundError indicates .structyl/config.json was not found.
type ProjectNotFoundError struct {
	StartDir string
}

func (e *ProjectNotFoundError) Error() string {
	return ".structyl/config.json not found (searched from " + e.StartDir + ")"
}

// Is implements error matching for errors.Is().
func (e *ProjectNotFoundError) Is(target error) bool {
	return target == ErrProjectNotFound
}

// ErrSuiteNotFound is returned when a test suite directory does not exist.
// Use errors.Is(err, ErrSuiteNotFound) to check for this condition.
var ErrSuiteNotFound = errors.New("suite not found")

// SuiteNotFoundError indicates a test suite directory does not exist.
type SuiteNotFoundError struct {
	Suite string
}

func (e *SuiteNotFoundError) Error() string {
	return "test suite not found: " + e.Suite
}

// Is implements error matching for errors.Is().
func (e *SuiteNotFoundError) Is(target error) bool {
	return target == ErrSuiteNotFound
}

// ListSuites returns the names of all available test suites.
// Returns an empty slice (not nil) if the tests directory doesn't exist.
func ListSuites(projectRoot string) ([]string, error) {
	testsDir := filepath.Join(projectRoot, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
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
// Returns false for any error (including permission errors), not just "not found".
// Use LoadTestSuite for detailed error information.
func SuiteExists(projectRoot, suite string) bool {
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	info, err := os.Stat(suiteDir)
	return err == nil && info.IsDir()
}

// TestCaseExists checks if a specific test case exists.
// Returns false for any error (including permission errors), not just "not found".
// Use LoadTestCase for detailed error information.
func TestCaseExists(projectRoot, suite, name string) bool {
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
	_, err := os.Stat(path)
	return err == nil
}
