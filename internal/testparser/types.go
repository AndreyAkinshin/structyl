// Package testparser provides test output parsing for various test frameworks.
package testparser

// TestCounts holds parsed test result counts.
type TestCounts struct {
	Passed  int
	Failed  int
	Skipped int
	Total   int
	Parsed  bool // true if counts were successfully extracted
}

// Add adds another TestCounts to this one, aggregating the counts.
func (tc *TestCounts) Add(other *TestCounts) {
	if other == nil {
		return
	}
	tc.Passed += other.Passed
	tc.Failed += other.Failed
	tc.Skipped += other.Skipped
	tc.Total += other.Total
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
