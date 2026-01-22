package testparser

import (
	"regexp"
	"strings"
)

// Static regexes for Go test output parsing.
// Compiled once at package init for performance.
var (
	goPassRegex = regexp.MustCompile(`(?m)^---\s+PASS:\s+`)
	goFailRegex = regexp.MustCompile(`(?m)^---\s+FAIL:\s+(\S+)`)
	goSkipRegex = regexp.MustCompile(`(?m)^---\s+SKIP:\s+`)
	goErrorLine = regexp.MustCompile(`^\s+\S+\.go:\d+:`)
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

	counts.Passed = len(goPassRegex.FindAllString(output, -1))
	counts.Skipped = len(goSkipRegex.FindAllString(output, -1))

	// Extract failed test names and their reasons
	failMatches := goFailRegex.FindAllStringSubmatch(output, -1)
	counts.Failed = len(failMatches)

	if counts.Failed > 0 {
		counts.FailedTests = p.extractFailedTests(output, failMatches)
	}

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

// extractFailedTests extracts detailed failure information for each failed test.
func (p *GoParser) extractFailedTests(output string, failMatches [][]string) []FailedTest {
	var failedTests []FailedTest
	lines := strings.Split(output, "\n")

	// Build a map of test name to failure reason
	// Go test output format:
	//   === RUN   TestFoo
	//       file_test.go:15: expected X, got Y
	//   --- FAIL: TestFoo (0.00s)
	for _, match := range failMatches {
		if len(match) < 2 {
			continue
		}
		testName := match[1]
		reason := p.findFailureReason(lines, testName)
		failedTests = append(failedTests, FailedTest{
			Name:   testName,
			Reason: reason,
		})
	}

	return failedTests
}

// Static regex for matching FAIL lines generically (captures test name).
var goFailLineRegex = regexp.MustCompile(`^---\s+FAIL:\s+(\S+)\s+`)

// isTestBoundary returns true if the line marks the start of a test run
// or the result of a different test (PASS/FAIL/SKIP).
func isTestBoundary(line string) bool {
	return strings.HasPrefix(line, "=== RUN") ||
		strings.HasPrefix(line, "--- PASS:") ||
		strings.HasPrefix(line, "--- FAIL:") ||
		strings.HasPrefix(line, "--- SKIP:")
}

// findFailureReason searches for the failure reason for a given test.
func (p *GoParser) findFailureReason(lines []string, testName string) string {
	// Find the FAIL line for this test
	failLineIdx := -1

	for i, line := range lines {
		match := goFailLineRegex.FindStringSubmatch(line)
		if match != nil && match[1] == testName {
			failLineIdx = i
			break
		}
	}

	if failLineIdx == -1 {
		return ""
	}

	// Look backwards for error messages (lines with file:line: pattern)
	// These are typically indented with spaces/tabs
	var reasons []string

	for i := failLineIdx - 1; i >= 0; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Stop at RUN line or another test result
		if isTestBoundary(line) {
			break
		}

		// Capture error lines (file:line: format)
		if goErrorLine.MatchString(line) && trimmed != "" {
			reasons = append([]string{trimmed}, reasons...)
		}
	}

	if len(reasons) == 0 {
		return ""
	}

	// Return the first (most relevant) error, truncated if too long
	reason := reasons[0]
	// Extract just the message part after file:line:
	if idx := strings.Index(reason, ".go:"); idx != -1 {
		// Find the colon after line number
		afterFile := reason[idx+4:]
		if colonIdx := strings.Index(afterFile, ": "); colonIdx != -1 {
			reason = strings.TrimSpace(afterFile[colonIdx+2:])
		}
	}

	// Truncate if too long. 80 chars is a common terminal width that keeps
	// failure reasons readable in summary output without excessive wrapping.
	const maxLen = 80
	if len(reason) > maxLen {
		reason = reason[:maxLen-3] + "..."
	}

	return reason
}
