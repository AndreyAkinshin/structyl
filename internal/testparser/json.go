package testparser

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// TestEvent represents a single event from go test -json output.
type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

// JSONParser parses go test -json output.
type JSONParser struct{}

// ParseJSON parses go test -json output from a reader and returns test counts.
func (p *JSONParser) ParseJSON(r io.Reader) TestCounts {
	counts := TestCounts{}
	scanner := bufio.NewScanner(r)

	// Track failed tests and their output
	failedTestOutput := make(map[string][]string) // test name -> output lines
	currentOutput := make(map[string][]string)    // accumulate output per test

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Skip package-level events (no test name)
		if event.Test == "" {
			continue
		}

		switch event.Action {
		case "output":
			// Accumulate output for potential failure message
			if event.Output != "" {
				currentOutput[event.Test] = append(currentOutput[event.Test], event.Output)
			}

		case "pass":
			counts.Passed++
			delete(currentOutput, event.Test)

		case "fail":
			counts.Failed++
			// Save the output for this failed test
			failedTestOutput[event.Test] = currentOutput[event.Test]
			delete(currentOutput, event.Test)

		case "skip":
			counts.Skipped++
			delete(currentOutput, event.Test)
		}
	}

	// Extract failure reasons from captured output
	for testName, outputLines := range failedTestOutput {
		reason := extractFailureReason(outputLines)
		counts.FailedTests = append(counts.FailedTests, FailedTest{
			Name:   testName,
			Reason: reason,
		})
	}

	if counts.Passed > 0 || counts.Failed > 0 || counts.Skipped > 0 {
		counts.Parsed = true
		counts.Total = counts.Passed + counts.Failed + counts.Skipped
	}

	return counts
}

// maxReasonLength is the maximum length for failure reason strings.
const maxReasonLength = 100

// extractFailureReason extracts the most relevant failure message from test output.
// Uses a two-pass strategy:
// 1. Look for Go's standard error format (file.go:line: message)
// 2. Fall back to the first non-boilerplate line
func extractFailureReason(outputLines []string) string {
	// First pass: look for structured error format
	if reason := extractGoErrorMessage(outputLines); reason != "" {
		return reason
	}

	// Second pass: first meaningful line
	return extractFirstMeaningfulLine(outputLines)
}

// extractGoErrorMessage looks for Go's standard error format: file.go:line: message
func extractGoErrorMessage(outputLines []string) string {
	for _, line := range outputLines {
		trimmed := strings.TrimSpace(line)
		if isBoilerplateLine(trimmed) {
			continue
		}

		// Look for error lines (file.go:123: message)
		if !strings.Contains(trimmed, ".go:") || !strings.Contains(trimmed, ": ") {
			continue
		}

		idx := strings.Index(trimmed, ".go:")
		afterFile := trimmed[idx+4:]
		colonIdx := strings.Index(afterFile, ": ")
		if colonIdx == -1 {
			continue
		}

		reason := strings.TrimSpace(afterFile[colonIdx+2:])
		return truncate(reason, maxReasonLength)
	}
	return ""
}

// extractFirstMeaningfulLine returns the first non-empty, non-boilerplate line.
func extractFirstMeaningfulLine(outputLines []string) string {
	for _, line := range outputLines {
		trimmed := strings.TrimSpace(line)
		if !isBoilerplateLine(trimmed) && !strings.HasPrefix(trimmed, "--- PASS") {
			return truncate(trimmed, maxReasonLength)
		}
	}
	return ""
}

// isBoilerplateLine returns true for lines that are test framework noise.
func isBoilerplateLine(line string) bool {
	return line == "" ||
		strings.HasPrefix(line, "=== RUN") ||
		strings.HasPrefix(line, "--- FAIL")
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
