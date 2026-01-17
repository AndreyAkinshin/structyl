package testparser

import (
	"regexp"
	"strconv"
)

// DenoParser parses Deno test output.
type DenoParser struct{}

// Name returns the parser name.
func (p *DenoParser) Name() string {
	return "deno"
}

// Parse extracts test counts from Deno test output.
// Deno test outputs summary lines like:
//
//	ok | 47 passed | 0 failed (123ms)
//	FAILED | 45 passed | 2 failed (123ms)
//
// Or in newer versions:
//
//	47 passed; 2 failed; 3 ignored
func (p *DenoParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Try the pipe-separated format first
	// Format: ok | N passed | N failed (duration)
	pipeRegex := regexp.MustCompile(`(\d+) passed\s*\|\s*(\d+) failed`)
	if match := pipeRegex.FindStringSubmatch(output); len(match) >= 3 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Failed, _ = strconv.Atoi(match[2])
		counts.Total = counts.Passed + counts.Failed
		counts.Parsed = true
		return counts
	}

	// Try semicolon-separated format
	// Format: N passed; N failed; N ignored
	semiRegex := regexp.MustCompile(`(\d+) passed;\s*(\d+) failed(?:;\s*(\d+) ignored)?`)
	if match := semiRegex.FindStringSubmatch(output); len(match) >= 3 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Failed, _ = strconv.Atoi(match[2])
		if len(match) >= 4 && match[3] != "" {
			counts.Skipped, _ = strconv.Atoi(match[3])
		}
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
		counts.Parsed = true
		return counts
	}

	// Fallback: look for individual patterns
	passedRegex := regexp.MustCompile(`(\d+) passed`)
	failedRegex := regexp.MustCompile(`(\d+) failed`)
	ignoredRegex := regexp.MustCompile(`(\d+) ignored`)

	if match := passedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := failedRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Failed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := ignoredRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Skipped, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if counts.Parsed {
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
	}

	return counts
}
