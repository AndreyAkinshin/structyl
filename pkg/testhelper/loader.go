// Package testhelper provides reusable test loading and comparison utilities
// for language implementations in structyl projects.
//
// This package is designed to be used by individual language implementations
// to load test cases and compare their outputs against expected values.
//
// # Thread Safety
//
// All functions in this package are safe for concurrent use:
//
//   - Loader functions ([LoadTestSuite], [LoadTestCase], etc.) perform read-only
//     filesystem operations and can be called concurrently.
//   - Comparison functions ([Equal], [Compare], [FormatComparisonResult]) are
//     pure functions with no shared state.
//   - The [TestCase] type is safe to read concurrently, but callers must not
//     modify a TestCase while other goroutines are reading it.
//
// # Filesystem Conventions
//
// All path parameters use the host operating system's path separator.
// Functions use [filepath.Join] internally, so callers should:
//   - Use [filepath.Join] to construct paths
//   - Not assume forward slashes work on Windows
//
// Returned paths are always absolute and use the OS path separator.
// Symlinks are followed during path resolution (e.g., in [FindProjectRoot]).
//
// # Limitations
//
//   - $file references are not supported. File reference resolution is only
//     available in the internal test runner. Test cases using $file syntax
//     should use the internal tests package or embed data directly in JSON.
//
// # Panic Behavior
//
// Comparison functions ([Equal], [Compare], [FormatComparisonResult]) panic if
// [CompareOptions] contains invalid values. This design treats invalid options
// as programmer errors (options are typically constants or static config) rather
// than runtime conditions. To validate options before comparison:
//
//   - Use [NewCompareOptions] for validated construction
//   - Use [ValidateOptions] to check options explicitly
//   - Use [CompareE] for an error-returning variant that does not panic
//
// # String() Output Stability
//
// The String() methods on [TestCase] and [CompareOptions] return human-readable
// representations for debugging. These formats are NOT stable and may change
// between versions without notice. Do not parse, compare, or rely on this output
// in tests or production code.
//
// # JSON Schema
//
// The JSON format for test case files is defined in the structyl JSON schema.
// See docs/specs/test-system.md for the complete test file specification,
// including field definitions, comparison options, and special value handling.
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
//	            if !testhelper.Equal(tc.Output, actual, testhelper.DefaultOptions()) {
//	                t.Errorf("mismatch for %s", tc.Name)
//	            }
//	        })
//	    }
//	}
package testhelper

import (
	"bytes"
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
	// Empty string is never a valid suite name; suite directories must have
	// non-empty names, so a zero value always means "not set" rather than
	// "suite named empty string".
	Suite string `json:"-"`

	// Input contains the input data for the test as a JSON object.
	// Input MUST be a JSON object (not an array or scalar). The object may be
	// empty ({}). A nil Input (missing field) causes a validation error.
	//
	// Post-validation guarantee: After LoadTestCase, LoadTestCaseWithSuite, or
	// LoadTestSuite returns successfully, Input is guaranteed to be non-nil.
	//
	// Why object-only? Test inputs typically represent named parameters or
	// configuration. Objects provide named access to individual values and
	// align with how most test frameworks expect structured input.
	// Arrays and scalar values as top-level input are not supported.
	Input map[string]interface{} `json:"input"`

	// Output contains the expected output.
	// Output must not be nil; a nil Output causes a validation error.
	// Use an explicit value (e.g., empty string "", empty object {}, or empty
	// array []) for expected empty output.
	//
	// Important: JSON null is NOT a valid output value. The loader validates
	// this and returns distinct errors:
	//   - Missing "output" field: "missing required field \"output\""
	//   - Explicit "output": null: "\"output\" field is null"
	//
	// If your test expects a null/nil result, use one of these patterns:
	//
	//   {"output": {"value": null}}  // wrap in object with nullable field
	//   {"output": "__NULL__"}       // use sentinel string + custom handling
	//
	// Why interface{}? Unlike Input, Output may be any non-null JSON value:
	// a scalar (number, string, boolean), an array, or an object.
	// This flexibility accommodates functions that return simple values,
	// collections, or complex structures.
	//
	// Post-load type guarantee: After successful loading, Output will be
	// exactly one of these Go types (matching the JSON Type Mapping above):
	//   - float64 (JSON numbers, including integers)
	//   - string (JSON strings)
	//   - bool (JSON booleans)
	//   - []interface{} (JSON arrays)
	//   - map[string]interface{} (JSON objects)
	Output interface{} `json:"output"`

	// Description provides optional documentation.
	Description string `json:"description,omitempty"`

	// Skip marks the test as skipped if true.
	Skip bool `json:"skip,omitempty"`

	// Tags provides optional categorization for filtering or grouping tests.
	// Unlike other TestCase fields, Tags has no built-in semantics in structyl.
	// Language implementations MAY use tags to:
	//   - Filter test execution (e.g., run only "slow" or "integration" tests)
	//   - Group tests in output
	//   - Skip tests based on environment capabilities
	//
	// Tag values are free-form strings with no validation: empty strings,
	// duplicates, and any characters are permitted. Establish conventions
	// per-project. This permissive design is intentional to avoid constraining
	// downstream tooling.
	//
	// Recommended conventions (not enforced):
	//   - Use lowercase, hyphen-separated names (e.g., "slow", "integration", "skip-ci")
	//   - Avoid whitespace-only or empty string tags
	//   - Prefix environment-specific tags (e.g., "env-linux", "env-docker")
	Tags []string `json:"tags,omitempty"`
}

// HasSuite reports whether the Suite field was explicitly set.
// Returns false for the zero value (""), true otherwise.
// This is useful when distinguishing between "Suite not set" (from LoadTestCase)
// and "Suite explicitly set" (from LoadTestSuite or LoadTestCaseWithSuite).
func (tc TestCase) HasSuite() bool {
	return tc.Suite != ""
}

// TagsContain reports whether tc.Tags contains the given tag.
// Comparison is exact and case-sensitive.
// Returns false if tc.Tags is nil or empty.
func (tc TestCase) TagsContain(tag string) bool {
	for _, t := range tc.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of TestCase for debugging.
// The format is for debugging only and may change without notice.
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

// Validate checks that TestCase fields satisfy the spec requirements.
// Returns nil if valid, or an error describing the first validation failure.
//
// This method is useful for callers who create TestCase programmatically
// (e.g., test case generators) rather than loading from JSON files.
// Loader functions already validate these requirements, so calling Validate
// after LoadTestCase or LoadTestSuite is unnecessary.
//
// Validation rules:
//   - Name must not be empty
//   - Input must not be nil (empty map {} is valid)
//   - Output must not be nil (use explicit value like "", {}, or [] instead)
//
// Note: This method does NOT check for $file references. Programmatically
// constructed TestCase instances may contain $file syntax in Input or Output,
// which will cause errors when used with the internal test runner. The loader
// functions (LoadTestCase, LoadTestSuite) reject $file references; this method
// does not, to avoid coupling Validate() to internal implementation details.
func (tc TestCase) Validate() error {
	if tc.Name == "" {
		return errors.New("name must not be empty")
	}
	if tc.Input == nil {
		return errors.New("input must not be nil")
	}
	if tc.Output == nil {
		return errors.New("output must not be nil")
	}
	return nil
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
// WARNING: This function does NOT set TestCase.Suite. The Suite field will be
// empty ("") after loading. If your code requires suite information, use
// [LoadTestCaseWithSuite] to explicitly set the suite, or [LoadTestSuite] to
// load all cases from a suite directory (which sets Suite automatically).
func LoadTestCase(path string) (*TestCase, error) {
	return loadTestCaseInternal(path, "")
}

// LoadTestCaseWithSuite loads a single test case from a JSON file and sets the suite name.
// This is a convenience function that combines LoadTestCase with setting the Suite field.
// Returns ErrEmptySuiteName if suite is empty.
// Returns an error if the file cannot be read, contains invalid JSON,
// or is missing required fields (input and output).
func LoadTestCaseWithSuite(path, suite string) (*TestCase, error) {
	if err := ValidateSuiteName(suite); err != nil {
		return nil, err
	}
	return loadTestCaseInternal(path, suite)
}

// outputPresenceChecker is used to distinguish between missing and null output fields.
// Go's json.Unmarshal sets interface{} to nil for both cases, so we check if the
// "output" key exists in the raw JSON object.
type outputPresenceChecker struct {
	m map[string]json.RawMessage
}

func (c *outputPresenceChecker) hasOutputField(data []byte) bool {
	if err := json.Unmarshal(data, &c.m); err != nil {
		return false
	}
	_, exists := c.m["output"]
	return exists
}

// validateInputFieldType checks that the "input" field, if present, is a JSON object.
// JSON arrays silently unmarshal to nil when the target is map[string]interface{},
// so we must inspect the raw JSON to catch this case and provide a clear error message.
func validateInputFieldType(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// Let the main unmarshal handle JSON syntax errors
		return nil
	}

	inputRaw, exists := raw["input"]
	if !exists {
		// Missing field will be caught by the nil check after unmarshal
		return nil
	}

	// Check the first non-whitespace character to determine JSON type
	trimmed := bytes.TrimSpace(inputRaw)
	if len(trimmed) == 0 {
		return nil
	}

	switch trimmed[0] {
	case '{':
		// Object - this is the expected type
		return nil
	case '[':
		return errors.New("\"input\" must be an object, not an array")
	case 'n':
		// null - will be caught as missing field
		return nil
	default:
		// Scalar value (string, number, boolean)
		return errors.New("\"input\" must be an object, not a scalar value")
	}
}

// loadTestCaseInternal is the shared implementation for LoadTestCase and LoadTestCaseWithSuite.
func loadTestCaseInternal(path, suite string) (*TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &TestCaseNotFoundError{Path: path}
		}
		return nil, err
	}

	// Pre-validate input field type before unmarshaling into TestCase.
	// JSON arrays silently unmarshal to nil when the target is map[string]interface{},
	// so we must check the raw JSON to distinguish missing from wrong type.
	if err := validateInputFieldType(data); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	var tc TestCase
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", filepath.Base(path), err)
	}

	// Detect $file references which are not supported in this package
	if containsFileReference(tc.Input) || containsFileReference(tc.Output) {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), ErrFileReferenceNotSupported)
	}

	// Validate required fields per spec
	if tc.Input == nil {
		return nil, fmt.Errorf("%s: missing required field \"input\"", filepath.Base(path))
	}
	if tc.Output == nil {
		// Distinguish between missing output field and explicit null.
		var checker outputPresenceChecker
		if checker.hasOutputField(data) {
			return nil, fmt.Errorf("%s: \"output\" field is null (use empty string, object, or array instead)", filepath.Base(path))
		}
		return nil, fmt.Errorf("%s: missing required field \"output\"", filepath.Base(path))
	}

	tc.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	if tc.Name == "" {
		return nil, fmt.Errorf("%s: invalid filename (name cannot be empty)", filepath.Base(path))
	}
	tc.Suite = suite
	return &tc, nil
}

// LoadAllSuites loads test cases from all suites in the tests directory.
// Returns an empty map (not nil) if the tests directory doesn't exist.
//
// Note: Empty suites (directories with no .json test files) are excluded from
// the returned map. Use [ListSuites] to enumerate all suite directories
// regardless of whether they contain test cases.
//
// Iteration order: The returned map has no guaranteed iteration order (Go maps
// are unordered). Within each suite's []TestCase slice, test cases are sorted
// alphabetically by filename for deterministic ordering. For deterministic
// iteration over suites, sort the map keys:
//
//	suites, _ := LoadAllSuites(root)
//	keys := make([]string, 0, len(suites))
//	for k := range suites {
//	    keys = append(keys, k)
//	}
//	sort.Strings(keys)
//	for _, suite := range keys {
//	    // process suites[suite] in alphabetical order
//	}
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
		return "", fmt.Errorf("FindProjectRoot: cannot determine working directory: %w", err)
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

// ErrTestCaseNotFound is returned when a test case file does not exist.
// Use errors.Is(err, ErrTestCaseNotFound) to check for this condition.
var ErrTestCaseNotFound = errors.New("test case not found")

// TestCaseNotFoundError indicates a test case file was not found.
type TestCaseNotFoundError struct {
	Path string
}

func (e *TestCaseNotFoundError) Error() string {
	return "test case not found: " + e.Path
}

// Is implements error matching for errors.Is().
func (e *TestCaseNotFoundError) Is(target error) bool {
	return target == ErrTestCaseNotFound
}

// ErrFileReferenceNotSupported is returned when a test case contains $file references.
// File references are only supported in the internal test runner. Use embedded data instead.
var ErrFileReferenceNotSupported = errors.New("$file references not supported; embed data directly in JSON")

// ErrEmptySuiteName is returned when an empty suite name is provided.
// Suite names must be non-empty strings corresponding to directory names.
//
// Note: Unlike [ErrProjectNotFound] or [ErrSuiteNotFound], this sentinel has no
// companion struct type. An empty suite name provides no useful context beyond
// the error message itself—there's no path or attempted name to include.
var ErrEmptySuiteName = errors.New("suite name cannot be empty")

// ValidateSuiteName checks if a suite name is valid.
// Returns ErrEmptySuiteName if the name is empty.
// Suite names must be non-empty strings corresponding to directory names.
func ValidateSuiteName(name string) error {
	if name == "" {
		return ErrEmptySuiteName
	}
	return nil
}

// containsFileReference recursively checks if a value contains a $file reference object.
func containsFileReference(v interface{}) bool {
	switch val := v.(type) {
	case map[string]interface{}:
		if _, ok := val["$file"]; ok {
			return true
		}
		for _, child := range val {
			if containsFileReference(child) {
				return true
			}
		}
	case []interface{}:
		for _, elem := range val {
			if containsFileReference(elem) {
				return true
			}
		}
	}
	return false
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
// Use [LoadTestSuite] for detailed error information, or [SuiteExistsErr] to
// distinguish "not found" from "permission denied" or other errors.
func SuiteExists(projectRoot, suite string) bool {
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	info, err := os.Stat(suiteDir)
	return err == nil && info.IsDir()
}

// TestCaseExists checks if a specific test case exists.
// Returns false for any error (including permission errors), not just "not found".
// Use [LoadTestCase] for detailed error information, or [TestCaseExistsErr] to
// distinguish "not found" from "permission denied" or other errors.
func TestCaseExists(projectRoot, suite, name string) bool {
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
	_, err := os.Stat(path)
	return err == nil
}

// SuiteExistsErr checks if a test suite exists, returning detailed error information.
// Returns (true, nil) if the suite exists, (false, nil) if it doesn't exist,
// or (false, error) for other errors like permission denied.
// This variant is useful when callers need to distinguish "not found" from "access error".
func SuiteExistsErr(projectRoot, suite string) (bool, error) {
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	info, err := os.Stat(suiteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	return true, nil
}

// TestCaseExistsErr checks if a specific test case exists, returning detailed error information.
// Returns (true, nil) if the test case exists, (false, nil) if it doesn't exist,
// or (false, error) for other errors like permission denied.
// This variant is useful when callers need to distinguish "not found" from "access error".
func TestCaseExistsErr(projectRoot, suite, name string) (bool, error) {
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
