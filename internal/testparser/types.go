// Package testparser provides test output parsing for various test frameworks.
package testparser

// FailedTest holds information about a single failed test.
type FailedTest struct {
	Name   string // Test name (e.g., "TestFoo/subtest")
	Reason string // Failure reason/error message
}

// TestCounts holds parsed test result counts.
type TestCounts struct {
	Passed      int
	Failed      int
	Skipped     int
	Total       int
	Parsed      bool         // true if counts were successfully extracted
	FailedTests []FailedTest // details of failed tests
}

// Add adds another TestCounts to this one, aggregating the counts.
// The Parsed flag uses "sticky true" semantics: if any added TestCounts
// has Parsed=true, the aggregate will have Parsed=true. This means
// Parsed indicates "at least one result was successfully parsed",
// not "all results were parsed".
func (tc *TestCounts) Add(other *TestCounts) {
	if other == nil {
		return
	}
	tc.Passed += other.Passed
	tc.Failed += other.Failed
	tc.Skipped += other.Skipped
	tc.Total += other.Total
	tc.FailedTests = append(tc.FailedTests, other.FailedTests...)
	if other.Parsed {
		tc.Parsed = true
	}
}

// Parser defines the interface for test output parsers.
type Parser interface {
	// Parse extracts test counts from the test framework output.
	Parse(output string) TestCounts
	// Name returns the name of the parser.
	Name() string
}
