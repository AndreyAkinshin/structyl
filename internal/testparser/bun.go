package testparser

import (
	"regexp"
	"strconv"
)

// BunParser parses Bun test output.
type BunParser struct{}

// Name returns the parser name.
func (p *BunParser) Name() string {
	return "bun"
}

// Parse extracts test counts from Bun test output.
// Bun test outputs summary lines like:
//
//	47 pass
//	2 fail
//	3 skip
//
// Or combined:
//
//	47 pass, 2 fail, 3 skip
func (p *BunParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Match patterns like "N pass", "N fail", "N skip"
	passRegex := regexp.MustCompile(`(\d+)\s+pass`)
	failRegex := regexp.MustCompile(`(\d+)\s+fail`)
	skipRegex := regexp.MustCompile(`(\d+)\s+skip`)

	if match := passRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Passed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := failRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Failed, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if match := skipRegex.FindStringSubmatch(output); len(match) >= 2 {
		counts.Skipped, _ = strconv.Atoi(match[1])
		counts.Parsed = true
	}

	if counts.Parsed {
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
	}

	return counts
}
