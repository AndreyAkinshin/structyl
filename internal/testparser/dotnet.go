package testparser

import (
	"regexp"
	"strconv"
)

// DotnetParser parses .NET test output.
type DotnetParser struct{}

// Name returns the parser name.
func (p *DotnetParser) Name() string {
	return "dotnet"
}

// Parse extracts test counts from dotnet test output.
// dotnet test outputs summary lines like:
//
//	Passed!  - Failed:     0, Passed:    47, Skipped:     3, Total:    50
//	Failed!  - Failed:     2, Passed:    45, Skipped:     3, Total:    50
//
// Or in newer versions:
//
//	Total tests: 50
//	     Passed: 47
//	     Failed: 2
//	    Skipped: 3
func (p *DotnetParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Try the summary line format first
	// Format: Failed: N, Passed: N, Skipped: N, Total: N
	summaryRegex := regexp.MustCompile(`Failed:\s*(\d+),\s*Passed:\s*(\d+),\s*Skipped:\s*(\d+)`)
	if match := summaryRegex.FindStringSubmatch(output); len(match) >= 4 {
		counts.Failed, _ = strconv.Atoi(match[1])
		counts.Passed, _ = strconv.Atoi(match[2])
		counts.Skipped, _ = strconv.Atoi(match[3])
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
		counts.Parsed = true
		return counts
	}

	// Try newer multi-line format
	passedRegex := regexp.MustCompile(`(?m)^\s*Passed:\s*(\d+)`)
	failedRegex := regexp.MustCompile(`(?m)^\s*Failed:\s*(\d+)`)
	skippedRegex := regexp.MustCompile(`(?m)^\s*Skipped:\s*(\d+)`)

	if match := passedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := failedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Failed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := skippedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Skipped, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if counts.Parsed {
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
	}

	return counts
}
