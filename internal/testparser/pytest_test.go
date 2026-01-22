package testparser

import "testing"

func TestPytestParser(t *testing.T) {
	t.Parallel()
	parser := &PytestParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name:     "basic pass",
			output:   "======= 47 passed in 0.12s =======",
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "with failures",
			output:   "======= 45 passed, 2 failed in 0.12s =======",
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "with skipped",
			output:   "======= 30 passed, 0 failed, 3 skipped in 0.12s =======",
			expected: TestCounts{Passed: 30, Failed: 0, Skipped: 3, Total: 33, Parsed: true},
		},
		{
			name:     "full summary",
			output:   "======= 30 passed, 2 failed, 3 skipped, 4 warnings in 0.12s =======",
			expected: TestCounts{Passed: 30, Failed: 2, Skipped: 3, Total: 35, Parsed: true},
		},
		{
			name: "verbose output",
			output: `tests/test_foo.py::test_bar PASSED
tests/test_foo.py::test_baz PASSED
======= 47 passed in 0.12s =======`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "collecting ...\ncollected 0 items\n",
			expected: TestCounts{Parsed: false},
		},
		{
			// pytest reports "errors" separately from "failed" (e.g., collection errors).
			// Current parser does not extract errors; they are not counted.
			name:     "with errors",
			output:   "======= 10 passed, 2 errors in 0.12s =======",
			expected: TestCounts{Passed: 10, Failed: 0, Skipped: 0, Total: 10, Parsed: true},
		},
		{
			// pytest reports "deselected" for tests excluded via -k or markers.
			// Current parser does not extract deselected; they are not counted.
			name:     "with deselected",
			output:   "======= 10 passed, 5 deselected in 0.12s =======",
			expected: TestCounts{Passed: 10, Failed: 0, Skipped: 0, Total: 10, Parsed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.Parse(tt.output)
			if result.Passed != tt.expected.Passed {
				t.Errorf("Passed: got %d, want %d", result.Passed, tt.expected.Passed)
			}
			if result.Failed != tt.expected.Failed {
				t.Errorf("Failed: got %d, want %d", result.Failed, tt.expected.Failed)
			}
			if result.Skipped != tt.expected.Skipped {
				t.Errorf("Skipped: got %d, want %d", result.Skipped, tt.expected.Skipped)
			}
			if result.Total != tt.expected.Total {
				t.Errorf("Total: got %d, want %d", result.Total, tt.expected.Total)
			}
			if result.Parsed != tt.expected.Parsed {
				t.Errorf("Parsed: got %v, want %v", result.Parsed, tt.expected.Parsed)
			}
		})
	}
}

// Note: Parser name verification is covered by TestRegistry in registry_test.go,
// which validates all parser names through the registration system.
