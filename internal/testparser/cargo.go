package testparser

import (
	"regexp"
	"strconv"
)

// Static regex for Cargo test output parsing.
// Compiled once at package init for performance.
var cargoResultRegex = regexp.MustCompile(`test result: \w+\.\s*(\d+) passed;\s*(\d+) failed;\s*(\d+) ignored`)

// CargoParser parses Rust/Cargo test output.
type CargoParser struct{}

// Name returns the parser name.
func (p *CargoParser) Name() string {
	return "cargo"
}

// Parse extracts test counts from Cargo test output.
// Cargo test outputs a summary line like:
//
//	test result: ok. 47 passed; 0 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.12s
//	test result: FAILED. 45 passed; 2 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.12s
func (p *CargoParser) Parse(output string) TestCounts {
	counts := TestCounts{}

	// Match the test result summary line
	// Format: test result: (ok|FAILED). N passed; N failed; N ignored; ...
	matches := cargoResultRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return counts
	}

	// Aggregate all test result lines (there may be multiple test binaries)
	for _, match := range matches {
		if len(match) >= 4 {
			passed, _ := strconv.Atoi(match[1])
			failed, _ := strconv.Atoi(match[2])
			ignored, _ := strconv.Atoi(match[3])

			counts.Passed += passed
			counts.Failed += failed
			counts.Skipped += ignored
		}
	}

	counts.Total = counts.Passed + counts.Failed + counts.Skipped
	counts.Parsed = counts.Total > 0 || len(matches) > 0

	return counts
}
