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

// extractFailureReason extracts the most relevant failure message from test output.
func extractFailureReason(outputLines []string) string {
	// Look for lines with file:line: pattern (typical Go test error format)
	for _, line := range outputLines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and common noise
		if trimmed == "" || strings.HasPrefix(trimmed, "=== RUN") ||
			strings.HasPrefix(trimmed, "--- FAIL") {
			continue
		}
		// Look for error lines (file.go:123: message)
		if strings.Contains(trimmed, ".go:") && strings.Contains(trimmed, ": ") {
			// Extract just the message part
			idx := strings.Index(trimmed, ".go:")
			if idx >= 0 {
				afterFile := trimmed[idx+4:]
				if colonIdx := strings.Index(afterFile, ": "); colonIdx != -1 {
					reason := strings.TrimSpace(afterFile[colonIdx+2:])
					// Truncate if too long
					const maxLen = 100
					if len(reason) > maxLen {
						reason = reason[:maxLen-3] + "..."
					}
					return reason
				}
			}
		}
	}

	// Fallback: return the first non-empty, non-boilerplate line
	for _, line := range outputLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "=== RUN") &&
			!strings.HasPrefix(trimmed, "--- FAIL") && !strings.HasPrefix(trimmed, "--- PASS") {
			const maxLen = 100
			if len(trimmed) > maxLen {
				trimmed = trimmed[:maxLen-3] + "..."
			}
			return trimmed
		}
	}

	return ""
}
