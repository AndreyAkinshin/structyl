package testparser

import "testing"

func TestDotnetParser(t *testing.T) {
	t.Parallel()
	parser := &DotnetParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name:     "summary line format",
			output:   "Passed!  - Failed:     0, Passed:    47, Skipped:     3, Total:    50",
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 3, Total: 50, Parsed: true},
		},
		{
			name:     "with failures",
			output:   "Failed!  - Failed:     2, Passed:    45, Skipped:     3, Total:    50",
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 3, Total: 50, Parsed: true},
		},
		{
			name: "multi-line format",
			output: `Total tests: 50
     Passed: 47
     Failed: 2
    Skipped: 1`,
			expected: TestCounts{Passed: 47, Failed: 2, Skipped: 1, Total: 50, Parsed: true},
		},
		{
			name: "verbose output",
			output: `Build started...
Build succeeded.

Test run for /path/to/tests.dll (.NETCoreApp,Version=v8.0)
Passed!  - Failed:     0, Passed:    47, Skipped:     0, Total:    47`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "Build started...\nBuild succeeded.\n",
			expected: TestCounts{Parsed: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.Parse(tt.output)
			assertTestCountsEqual(t, result, tt.expected)
		})
	}
}

// Note: Parser name verification is covered by TestRegistry in registry_test.go,
// which validates all parser names through the registration system.
