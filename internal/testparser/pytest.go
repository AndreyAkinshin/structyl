package testparser

import (
	"regexp"
	"strconv"
)

// Static regexes for pytest output parsing.
// Compiled once at package init for performance.
var (
	pytestPassedRegex  = regexp.MustCompile(`(\d+) passed`)
	pytestFailedRegex  = regexp.MustCompile(`(\d+) failed`)
	pytestSkippedRegex = regexp.MustCompile(`(\d+) skipped`)
)

// PytestParser parses Python pytest output.
type PytestParser struct{}

// Name returns the parser name.
func (p *PytestParser) Name() string {
	return "pytest"
}

// Parse extracts test counts from pytest output.
// pytest outputs summary lines like:
//
//	======= 47 passed in 0.12s =======
//	======= 45 passed, 2 failed in 0.12s =======
//	======= 30 passed, 0 failed, 3 skipped in 0.12s =======
//	======= 1 passed, 2 failed, 3 skipped, 4 warnings in 0.12s =======
func (p *PytestParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Find passed count
	if match := pytestPassedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	// Find failed count
	if match := pytestFailedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Failed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	// Find skipped count
	if match := pytestSkippedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Skipped, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if counts.Parsed {
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
	}

	return counts
}
