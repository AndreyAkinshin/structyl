// Package tests provides the reference test system for Structyl.
package tests

// TestCase represents a single test case loaded from JSON.
type TestCase struct {
	Name   string                 // Test name (from filename)
	Suite  string                 // Test suite name (parent directory)
	Path   string                 // Full path to the test file
	Input  map[string]interface{} // Input data for the test
	Output interface{}            // Expected output
}

// ComparisonConfig configures how test outputs are compared.
type ComparisonConfig struct {
	FloatTolerance float64 `json:"float_tolerance"` // Tolerance for float comparison
	ToleranceMode  string  `json:"tolerance_mode"`  // "relative", "absolute", or "ulp"
	ArrayOrder     string  `json:"array_order"`     // "strict" or "unordered"
	NaNEqualsNaN   bool    `json:"nan_equals_nan"`  // Whether NaN == NaN
}

// DefaultComparisonConfig returns the default comparison settings.
func DefaultComparisonConfig() ComparisonConfig {
	return ComparisonConfig{
		FloatTolerance: 1e-9,
		ToleranceMode:  "relative",
		ArrayOrder:     "strict",
		NaNEqualsNaN:   false,
	}
}

// TestResult represents the result of running a test case.
type TestResult struct {
	TestCase   *TestCase
	Passed     bool
	Actual     interface{}
	Diff       string
	Error      error
	DurationMs int64
}

// TestSuiteResult represents results for an entire test suite.
type TestSuiteResult struct {
	Suite   string
	Results []TestResult
	Passed  int
	Failed  int
	Skipped int
}
