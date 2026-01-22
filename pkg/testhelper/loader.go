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
// # Error Type Patterns
//
// This package uses two error patterns:
//
//   - Struct error types with sentinel matching: [ProjectNotFoundError],
//     [SuiteNotFoundError], [TestCaseNotFoundError]. These carry context (path,
//     name) and implement Is() for matching via [errors.Is]. Use when the error
//     context is meaningful for debugging or recovery.
//
//   - Bare sentinel errors: [ErrEmptySuiteName], [ErrInvalidSuiteName],
//     [ErrFileReferenceNotSupported]. These are simple sentinel values without
//     additional context. Use [errors.Is] to match. Appropriate when the error
//     condition is self-explanatory and no additional context would help.
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
// # Existence Checks
//
// Two patterns are available for checking if suites or test cases exist:
//
//   - [SuiteExists], [TestCaseExists] return bool (false on any error)
//   - [SuiteExistsErr], [TestCaseExistsErr] distinguish "not found" from "access error"
//
// Use the *Err variants when you need to differentiate between a missing
// resource and a permission or I/O error.
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
// # Copy Semantics Warning
//
// [TestCase.Clone] and all With* builder methods perform SHALLOW copies.
// The [TestCase.Output] field is NOT copied - both original and copy share
// the same reference. Modifying Output on a clone affects the original:
//
//	tc := original.WithInput(newInput)  // uses Clone internally
//	tc.Output.(map[string]interface{})["key"] = "changed"
//	// Surprise: original.Output is also changed!
//
// Use [TestCase.DeepClone] when you need to modify Output independently.
//
// [TestCase] provides two copy methods with different guarantees:
//
//   - [TestCase.Clone] creates a shallow copy suitable for most use cases.
//     It deep-copies [TestCase.Input] (top-level keys only) and [TestCase.Tags],
//     but [TestCase.Output] remains a shared reference.
//   - [TestCase.DeepClone] creates a fully independent copy via JSON round-trip.
//     Use this when you need to mutate Output or nested Input values.
//
// All builder methods ([TestCase.WithSuite], [TestCase.WithInput], [TestCase.WithTags],
// [TestCase.WithSkip], [TestCase.WithOutput], [TestCase.WithDescription]) use Clone
// internally, so they inherit its shallow copy semantics.
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
	//
	// IMMUTABILITY: Input SHOULD be treated as immutable after loading.
	// Modifying Input values may affect other code sharing the same TestCase
	// (especially when using [Clone], which performs a shallow copy of top-level
	// keys but shares nested values). For safe mutation, use [DeepClone] to
	// create a fully independent copy.
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
	//
	// Clone behavior: [TestCase.Clone] performs a shallow copy of Output.
	// Both original and clone reference the same underlying value. This is
	// intentional since Output is typically consumed read-only in assertions.
	// If you need to modify Output independently, copy it manually.
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

// Clone returns a copy of TestCase with independent Input and Tags.
// WARNING: Output is NOT copied; both original and clone share the same reference.
//
// # Copy Semantics Summary
//
//	| Field       | Copy Type        | Modify Clone → Original?           |
//	|-------------|------------------|------------------------------------|
//	| Name        | value copy       | No                                 |
//	| Suite       | value copy       | No                                 |
//	| Input       | shallow map copy | Only at top level; nested shared   |
//	| Output      | NOT copied       | Yes (shared reference)             |
//	| Tags        | slice copy       | No                                 |
//	| Skip        | value copy       | No                                 |
//	| Description | value copy       | No                                 |
//
// # Deep-Copied Fields
//
// The following fields receive new allocations:
//   - Input: a new map with the same top-level keys and values (shallow copy of values)
//   - Tags: a new slice with the same elements
//
// # NOT Deep-Copied Fields (Shared References)
//
// Output is NOT deep-copied; both original and clone reference the same underlying
// value. Modifying this field on the clone also modifies the original:
//
//	original := TestCase{Output: map[string]interface{}{"key": "value"}}
//	clone := original.Clone()
//	clone.Output.(map[string]interface{})["key"] = "changed"
//	// original.Output["key"] is now "changed" too!
//
// # Design Rationale
//
// Output is not deep-copied because:
//  1. Output is typically consumed read-only in test assertions
//  2. Deep-copying arbitrary interface{} values safely is complex (cycles, non-clonable types)
//  3. Performance: deep-copying large expected outputs would be wasteful
//
// If you need to modify Output independently, copy it manually before modification.
//
// # Nil Handling
//
// Nil fields remain nil; empty slices/maps remain empty (not collapsed to nil).
func (tc TestCase) Clone() TestCase {
	clone := tc // shallow copy of struct

	if tc.Input != nil {
		clone.Input = make(map[string]interface{}, len(tc.Input))
		for k, v := range tc.Input {
			clone.Input[k] = v
		}
	}

	if tc.Tags != nil {
		clone.Tags = make([]string, len(tc.Tags))
		copy(clone.Tags, tc.Tags)
	}

	return clone
}

// DeepClone returns a deep copy of TestCase including Output.
// Unlike [Clone], modifying DeepClone's Output does not affect the original.
//
// # Implementation
//
// DeepClone uses JSON marshal/unmarshal to create a deep copy of Output.
// This approach:
//   - Handles arbitrarily nested structures (maps, slices)
//   - Preserves JSON-compatible types correctly
//   - Returns an error if Output cannot be marshaled/unmarshaled
//
// # Limitations
//
// Because this uses JSON serialization:
//   - Output must be JSON-serializable (no channels, functions, or cycles)
//   - Type information may be normalized (e.g., int becomes float64)
//   - This is consistent with how test cases are loaded from JSON files
//
// # When to Use
//
// Use DeepClone when you need to modify Output independently:
//
//	original := loadedTestCase
//	modified := original.DeepClone()
//	modified.Output.(map[string]interface{})["key"] = "new value"
//	// original.Output is unchanged
//
// For most test assertions where Output is read-only, [Clone] is sufficient
// and more efficient.
//
// # Error Handling
//
// Returns an error if Output cannot be deep-copied (e.g., contains non-JSON types).
// The error wraps the underlying JSON error for debugging.
func (tc TestCase) DeepClone() (TestCase, error) {
	// Start with shallow clone for all fields
	clone := tc.Clone()

	// Deep copy Output via JSON round-trip
	if tc.Output != nil {
		data, err := json.Marshal(tc.Output)
		if err != nil {
			return TestCase{}, fmt.Errorf("DeepClone: failed to marshal Output: %w", err)
		}

		var deepCopy interface{}
		if err := json.Unmarshal(data, &deepCopy); err != nil {
			return TestCase{}, fmt.Errorf("DeepClone: failed to unmarshal Output: %w", err)
		}
		clone.Output = deepCopy
	}

	return clone, nil
}

// WithName returns a copy of the TestCase with the Name field set to the given value.
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithName(name string) TestCase {
	clone := tc.Clone()
	clone.Name = name
	return clone
}

// WithSuite returns a copy of the TestCase with the Suite field set to the given value.
// This is useful when loading test cases with [LoadTestCase], which does not populate
// the Suite field, and you want to associate the test case with a suite name.
//
// Example:
//
//	tc, err := testhelper.LoadTestCase("/path/to/test.json")
//	if err != nil {
//	    return err
//	}
//	tc = tc.WithSuite("my-suite")
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithSuite(suite string) TestCase {
	clone := tc.Clone()
	clone.Suite = suite
	return clone
}

// WithDescription returns a copy of the TestCase with the Description field set.
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithDescription(description string) TestCase {
	clone := tc.Clone()
	clone.Description = description
	return clone
}

// WithInput returns a copy of the TestCase with the Input field replaced.
// The provided input map is shallow-copied to prevent external modifications.
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithInput(input map[string]interface{}) TestCase {
	clone := tc.Clone()
	if input == nil {
		clone.Input = nil
	} else {
		clone.Input = make(map[string]interface{}, len(input))
		for k, v := range input {
			clone.Input[k] = v
		}
	}
	return clone
}

// WithTags returns a copy of the TestCase with the Tags field replaced.
// The provided tags slice is copied to prevent external modifications.
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithTags(tags []string) TestCase {
	clone := tc.Clone()
	if tags == nil {
		clone.Tags = nil
	} else {
		clone.Tags = make([]string, len(tags))
		copy(clone.Tags, tags)
	}
	return clone
}

// WithSkip returns a copy of the TestCase with the Skip field set.
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithSkip(skip bool) TestCase {
	clone := tc.Clone()
	clone.Skip = skip
	return clone
}

// WithOutput returns a copy of the TestCase with the Output field replaced.
//
// WARNING: Unlike WithInput, this method stores output by REFERENCE, not by copy.
// If output contains nested maps or slices, modifications to the passed value
// will affect this TestCase:
//
//	output := map[string]interface{}{"key": "value"}
//	tc := tc.WithOutput(output)
//	output["key"] = "modified"  // tc.Output["key"] is now "modified"
//
// For complete isolation, either:
//   - Pass an immutable value (string, float64, bool)
//   - Copy the value before passing it
//   - Call [DeepClone] on the resulting TestCase
//
// Note: This performs a shallow copy like [Clone]; see Clone documentation for
// details on which fields are deep-copied.
func (tc TestCase) WithOutput(output interface{}) TestCase {
	clone := tc.Clone()
	clone.Output = output
	return clone
}

// Validate checks that TestCase fields satisfy basic structural requirements.
// Returns nil if valid, or an error describing the first validation failure.
//
// This method is useful for callers who create TestCase programmatically
// (e.g., test case generators) rather than loading from JSON files.
// Loader functions already validate these requirements, so calling Validate
// after LoadTestCase or LoadTestSuite is unnecessary.
//
// For stricter validation, see [ValidateStrict] (adds Output type checking) and
// [ValidateDeep] (adds recursive type validation for nested values).
//
// Validation rules:
//   - Name must not be empty
//   - Input must not be nil (empty map {} is valid)
//   - Output must not be nil (use explicit value like "", {}, or [] instead)
//
// Important: This method performs structural validation only. It does NOT verify
// that Output is one of the five JSON-compatible Go types (float64, string,
// bool, []interface{}, map[string]interface{}). Loader functions provide
// stronger type guarantees because they unmarshal from JSON; use Validate()
// for programmatically-constructed test cases where structural checks suffice.
//
// WARNING: Comparison functions ([Equal], [Compare]) may produce undefined results
// or panic if Output contains types other than the five JSON-compatible types.
// If creating TestCase programmatically, ensure Output types match JSON semantics.
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

// ValidateStrict performs all checks from [Validate] plus type validation for Output.
// Use this method when creating TestCase instances programmatically to ensure
// Output contains only JSON-compatible Go types.
//
// # Validation Hierarchy
//
// The validation methods form a hierarchy of increasing strictness:
//
//		[Validate] < [ValidateStrict] < [ValidateDeep]
//
//	  - Validate: structural checks only (Name, Input, Output non-nil)
//	  - ValidateStrict: adds top-level Output type check
//	  - ValidateDeep: adds recursive type validation for all nested values
//
// In addition to Validate() checks, ValidateStrict verifies that Output is one of:
//   - float64 (JSON numbers)
//   - string (JSON strings)
//   - bool (JSON booleans)
//   - []interface{} (JSON arrays)
//   - map[string]interface{} (JSON objects)
//
// If Output is a different type (e.g., int, custom struct), ValidateStrict returns
// an error. This catches type mismatches that would cause undefined behavior or
// panics in comparison functions.
//
// Note: ValidateStrict only checks the top-level type of Output. It does NOT
// recursively validate that array elements or map values are also JSON-compatible.
// For deeply nested structures, ensure all values follow JSON type conventions.
func (tc TestCase) ValidateStrict() error {
	if err := tc.Validate(); err != nil {
		return err
	}
	return validateOutputType(tc.Output)
}

// validateOutputType checks that v is a JSON-compatible Go type.
func validateOutputType(v interface{}) error {
	switch v.(type) {
	case float64, string, bool, []interface{}, map[string]interface{}:
		return nil
	default:
		return fmt.Errorf("output has unsupported type %T; must be float64, string, bool, []interface{}, or map[string]interface{}", v)
	}
}

// ValidateDeep performs all Validate() checks plus recursive type validation.
// Verifies that all nested values in Input and Output are JSON-compatible types.
//
// This is the most comprehensive validation method. See [ValidateStrict] for
// the validation hierarchy: [Validate] < [ValidateStrict] < [ValidateDeep].
//
// In addition to Validate() and ValidateStrict() checks, ValidateDeep recursively
// verifies that every nested value in arrays and maps is also a valid JSON type:
//   - nil
//   - float64 (JSON numbers)
//   - string (JSON strings)
//   - bool (JSON booleans)
//   - []interface{} (JSON arrays)
//   - map[string]interface{} (JSON objects)
//
// Use this method when creating TestCase instances programmatically with complex
// nested structures to ensure all values follow JSON type conventions.
//
// Returns the first validation error encountered during traversal, with a path
// indicating the location of the invalid value (e.g., "input.users[0].age").
func (tc TestCase) ValidateDeep() error {
	if err := tc.Validate(); err != nil {
		return err
	}
	if err := validateDeepType("input", tc.Input); err != nil {
		return err
	}
	return validateDeepType("output", tc.Output)
}

// validateDeepType recursively checks that v and all nested values are JSON-compatible Go types.
func validateDeepType(path string, v interface{}) error {
	switch val := v.(type) {
	case nil, float64, string, bool:
		return nil
	case []interface{}:
		for i, elem := range val {
			if err := validateDeepType(fmt.Sprintf("%s[%d]", path, i), elem); err != nil {
				return err
			}
		}
		return nil
	case map[string]interface{}:
		for k, elem := range val {
			if err := validateDeepType(path+"."+k, elem); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: unsupported type %T; must be nil, float64, string, bool, []interface{}, or map[string]interface{}", path, v)
	}
}

// LoadTestSuite loads all test cases from a suite directory.
// It looks for JSON files in <projectRoot>/tests/<suite>/*.json.
//
// Note: This function uses *.json pattern which matches JSON files in the
// immediate suite directory only. Recursive patterns (**/*.json) are NOT
// supported by this public package. For recursive loading, use Structyl's
// internal test runner or iterate subdirectories manually.
// See docs/specs/test-system.md for pattern support details.
//
// Returns ErrEmptySuiteName or ErrInvalidSuiteName if the suite name is invalid.
// Returns SuiteNotFoundError if the suite directory does not exist.
// Returns an empty slice (not nil) if the suite exists but contains no JSON files.
func LoadTestSuite(projectRoot, suite string) ([]TestCase, error) {
	if err := ValidateSuiteName(suite); err != nil {
		return nil, err
	}
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	if _, err := os.Stat(suiteDir); os.IsNotExist(err) {
		return nil, &SuiteNotFoundError{Root: projectRoot, Suite: suite}
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
			return nil, fmt.Errorf("suite %q: %w", suite, err)
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
// Note: This function returns *TestCase (pointer), while [LoadTestSuite] returns
// []TestCase (slice of values). The pointer return allows nil on error. When
// combining results from both functions, dereference the pointer:
//
//	tc, _ := LoadTestCase(path)
//	suite, _ := LoadTestSuite(root, name)
//	all := append(suite, *tc)  // dereference tc
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

// LoadTestCaseByName loads a single test case by suite and name from a project root.
// This is a convenience function that constructs the correct path and sets the Suite field.
//
// The test case is loaded from: {projectRoot}/tests/{suite}/{name}.json
//
// Returns:
//   - [ErrEmptySuiteName] or [ErrInvalidSuiteName] if suite validation fails
//   - [ErrEmptyTestCaseName] or [ErrInvalidTestCaseName] if name validation fails
//   - [ErrTestCaseNotFound] if the test case file does not exist
//   - Other errors for invalid JSON or missing required fields
func LoadTestCaseByName(projectRoot, suite, name string) (*TestCase, error) {
	if err := ValidateSuiteName(suite); err != nil {
		return nil, err
	}
	if err := ValidateTestCaseName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
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
		// Return nil to let the caller's json.Unmarshal produce the syntax error.
		// This function only validates input field TYPE; syntax errors are handled
		// by loadTestCaseInternal's main unmarshal call with a clearer error path.
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
	Root  string // Project root that was searched
	Suite string // Suite name that wasn't found
}

func (e *SuiteNotFoundError) Error() string {
	if e.Root != "" {
		return fmt.Sprintf("test suite not found: %s (searched in %s)", e.Suite, e.Root)
	}
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
// File references are only supported in the internal test runner (internal/tests package).
// Use embedded data instead, or use Structyl's internal test runner for $file support.
var ErrFileReferenceNotSupported = errors.New("$file references not supported in pkg/testhelper; use internal/tests package or embed data directly in JSON")

// ErrEmptySuiteName is returned when an empty suite name is provided.
// Suite names must be non-empty strings corresponding to directory names.
//
// Note: Unlike [ErrProjectNotFound] or [ErrSuiteNotFound], this sentinel has no
// companion struct type. An empty suite name provides no useful context beyond
// the error message itself—there's no path or attempted name to include.
var ErrEmptySuiteName = errors.New("suite name cannot be empty")

// ErrInvalidSuiteName is returned when a suite name contains invalid characters.
// Invalid characters include path separators (/, \), path traversal sequences (..),
// and null bytes. These restrictions prevent path injection attacks and ensure
// suite names map safely to filesystem directories.
//
// Use [errors.Is] to check for this error type:
//
//	if errors.Is(err, testhelper.ErrInvalidSuiteName) {
//	    // handle invalid suite name
//	}
//
// The actual returned error may be an [InvalidSuiteNameError] with additional
// context (the suite name and rejection reason).
var ErrInvalidSuiteName = errors.New("suite name contains invalid characters")

// InvalidSuiteNameReason constants for [InvalidSuiteNameError.Reason].
// These constants ensure type safety and enable compile-time checking
// when handling invalid suite name errors.
const (
	// ReasonPathTraversal indicates the suite name contains ".." sequences.
	ReasonPathTraversal = "path_traversal"

	// ReasonPathSeparator indicates the suite name contains "/" or "\" characters.
	ReasonPathSeparator = "path_separator"

	// ReasonNullByte indicates the suite name contains null byte characters.
	ReasonNullByte = "null_byte"
)

// InvalidSuiteNameError indicates a suite name contains invalid characters.
// It carries the original name and the reason for rejection.
type InvalidSuiteNameError struct {
	Name   string // The invalid suite name
	Reason string // Why it was rejected: ReasonPathTraversal, ReasonPathSeparator, or ReasonNullByte
}

func (e *InvalidSuiteNameError) Error() string {
	return fmt.Sprintf("invalid suite name %q: %s", e.Name, e.Reason)
}

// Is implements error matching for [errors.Is].
// Returns true when target is [ErrInvalidSuiteName].
func (e *InvalidSuiteNameError) Is(target error) bool {
	return target == ErrInvalidSuiteName
}

// ErrEmptyTestCaseName is returned when an empty test case name is provided.
// Test case names must be non-empty strings corresponding to JSON file names.
//
// Note: Unlike [ErrProjectNotFound] or [ErrSuiteNotFound], this sentinel has no
// companion struct type. An empty test case name provides no useful context beyond
// the error message itself—there's no path or attempted name to include.
var ErrEmptyTestCaseName = errors.New("test case name cannot be empty")

// ErrInvalidTestCaseName is returned when a test case name contains invalid characters.
// Invalid characters include path separators (/, \), path traversal sequences (..),
// and null bytes. These restrictions prevent path injection attacks and ensure
// test case names map safely to filesystem filenames.
//
// Use [errors.Is] to check for this error type:
//
//	if errors.Is(err, testhelper.ErrInvalidTestCaseName) {
//	    // handle invalid test case name
//	}
//
// The actual returned error may be an [InvalidTestCaseNameError] with additional
// context (the test case name and rejection reason).
var ErrInvalidTestCaseName = errors.New("test case name contains invalid characters")

// InvalidTestCaseNameError indicates a test case name contains invalid characters.
// It carries the original name and the reason for rejection.
// This mirrors [InvalidSuiteNameError] for API symmetry.
type InvalidTestCaseNameError struct {
	Name   string // The invalid test case name
	Reason string // Why it was rejected: ReasonPathTraversal, ReasonPathSeparator, or ReasonNullByte
}

func (e *InvalidTestCaseNameError) Error() string {
	return fmt.Sprintf("invalid test case name %q: %s", e.Name, e.Reason)
}

// Is implements error matching for [errors.Is].
// Returns true when target is [ErrInvalidTestCaseName].
func (e *InvalidTestCaseNameError) Is(target error) bool {
	return target == ErrInvalidTestCaseName
}

// ValidateSuiteName checks if a suite name is valid.
// Returns ErrEmptySuiteName if the name is empty.
// Returns ErrInvalidSuiteName if the name contains path traversal sequences (..),
// path separators (/ or \), or null bytes.
//
// Valid suite names consist of any characters except the above restrictions.
// This includes Unicode characters, leading dots, hyphens, and underscores.
// Suite names should follow directory naming conventions for your target
// filesystem to ensure portability.
func ValidateSuiteName(name string) error {
	if name == "" {
		return ErrEmptySuiteName
	}
	// Check for path traversal sequences
	if strings.Contains(name, "..") {
		return &InvalidSuiteNameError{Name: name, Reason: ReasonPathTraversal}
	}
	// Check for path separators (both Unix and Windows)
	if strings.ContainsAny(name, "/\\") {
		return &InvalidSuiteNameError{Name: name, Reason: ReasonPathSeparator}
	}
	// Check for null bytes
	if strings.ContainsRune(name, '\x00') {
		return &InvalidSuiteNameError{Name: name, Reason: ReasonNullByte}
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
//
// This function validates the suite name and returns false for invalid names
// (containing path separators or traversal sequences like ".."). This prevents
// path injection when the suite name comes from untrusted input.
func SuiteExists(projectRoot, suite string) bool {
	if err := ValidateSuiteName(suite); err != nil {
		return false
	}
	suiteDir := filepath.Join(projectRoot, "tests", suite)
	info, err := os.Stat(suiteDir)
	return err == nil && info.IsDir()
}

// TestCaseExists checks if a specific test case exists.
// Returns false for any error (including permission errors), not just "not found".
// Use [LoadTestCase] for detailed error information, or [TestCaseExistsErr] to
// distinguish "not found" from "permission denied" or other errors.
//
// This function validates both suite and name parameters and returns false for
// invalid names (containing path separators or traversal sequences like "..").
// This prevents path injection when names come from untrusted input.
func TestCaseExists(projectRoot, suite, name string) bool {
	if err := ValidateSuiteName(suite); err != nil {
		return false
	}
	if err := validatePathComponent(name); err != nil {
		return false
	}
	path := filepath.Join(projectRoot, "tests", suite, name+".json")
	_, err := os.Stat(path)
	return err == nil
}

// validatePathComponent checks if a path component is safe.
// Returns an error if the name is empty, contains path traversal sequences (..),
// path separators (/ or \), or null bytes.
// This is used internally for test case name validation.
func validatePathComponent(name string) error {
	if name == "" {
		return ErrEmptyTestCaseName
	}
	if strings.Contains(name, "..") {
		return &InvalidTestCaseNameError{Name: name, Reason: ReasonPathTraversal}
	}
	if strings.ContainsAny(name, "/\\") {
		return &InvalidTestCaseNameError{Name: name, Reason: ReasonPathSeparator}
	}
	if strings.ContainsRune(name, '\x00') {
		return &InvalidTestCaseNameError{Name: name, Reason: ReasonNullByte}
	}
	return nil
}

// ValidateTestCaseName checks if a test case name is valid.
// Returns [ErrEmptyTestCaseName] if the name is empty.
// Returns [ErrInvalidTestCaseName] if the name contains path traversal sequences (..),
// path separators (/ or \), or null bytes.
//
// Valid test case names consist of any characters except the above restrictions.
// This includes Unicode characters, leading dots, hyphens, and underscores.
// Test case names should follow file naming conventions for your target
// filesystem to ensure portability.
//
// This function provides symmetry with [ValidateSuiteName] for callers who
// construct test paths programmatically.
func ValidateTestCaseName(name string) error {
	return validatePathComponent(name)
}

// SuiteExistsErr checks if a test suite exists, returning detailed error information.
// Returns (true, nil) if the suite exists, (false, nil) if it doesn't exist,
// or (false, error) for other errors like permission denied or invalid suite name.
// This variant is useful when callers need to distinguish "not found" from "access error"
// or "invalid input".
//
// This function validates the suite name and returns (false, error) for invalid names
// (containing path separators or traversal sequences like ".."). Use [errors.Is] with
// [ErrInvalidSuiteName] or [ErrEmptySuiteName] to detect validation failures.
//
// Note: This differs from [SuiteExists], which returns false for validation errors
// without distinguishing them from "not found".
func SuiteExistsErr(projectRoot, suite string) (bool, error) {
	if err := ValidateSuiteName(suite); err != nil {
		return false, err
	}
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
// or (false, error) for other errors like permission denied or invalid input.
// This variant is useful when callers need to distinguish "not found" from "access error"
// or "invalid input".
//
// This function validates both suite and name parameters and returns (false, error) for
// invalid names (containing path separators or traversal sequences like ".."). Use [errors.Is]
// with [ErrInvalidSuiteName] or [ErrEmptySuiteName] to detect suite validation failures.
//
// Note: This differs from [TestCaseExists], which returns false for validation errors
// without distinguishing them from "not found".
func TestCaseExistsErr(projectRoot, suite, name string) (bool, error) {
	if err := ValidateSuiteName(suite); err != nil {
		return false, err
	}
	if err := validatePathComponent(name); err != nil {
		return false, err
	}
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
