package testparser

import (
	"regexp"
	"strings"
)

// GoParser parses Go test output.
type GoParser struct{}

// Name returns the parser name.
func (p *GoParser) Name() string {
	return "go"
}

// Parse extracts test counts from Go test output.
// Go test outputs lines like:
//
//	--- PASS: TestFoo (0.00s)
//	--- FAIL: TestBar (0.01s)
//	--- SKIP: TestBaz (0.00s)
func (p *GoParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Match individual test results
	passRegex := regexp.MustCompile(`(?m)^---\s+PASS:\s+`)
	failRegex := regexp.MustCompile(`(?m)^---\s+FAIL:\s+`)
	skipRegex := regexp.MustCompile(`(?m)^---\s+SKIP:\s+`)

	counts.Passed = len(passRegex.FindAllString(output, -1))
	counts.Failed = len(failRegex.FindAllString(output, -1))
	counts.Skipped = len(skipRegex.FindAllString(output, -1))

	// Also check for the summary line at the end for verification
	// ok  	package	0.123s
	// FAIL	package	0.123s
	// This gives us an overall pass/fail but not detailed counts

	// If we found any results, mark as parsed
	if counts.Passed > 0 || counts.Failed > 0 || counts.Skipped > 0 {
		counts.Parsed = true
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
		return counts
	}

	// Try alternative: check for "PASS" or "FAIL" package summary
	// Sometimes tests don't have individual --- PASS lines
	if strings.Contains(output, "\nPASS\n") || strings.Contains(output, "\nok ") {
		// Package passed but we don't know individual test count
		// Return unparsed to fall back to task status
		return counts
	}

	return counts
}
