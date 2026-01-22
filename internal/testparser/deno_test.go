package testparser

import "testing"

func TestDenoParser(t *testing.T) {
	t.Parallel()
	parser := &DenoParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name:     "pipe format pass",
			output:   "ok | 47 passed | 0 failed (123ms)",
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "pipe format with failures",
			output:   "FAILED | 45 passed | 2 failed (123ms)",
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "semicolon format",
			output:   "47 passed; 0 failed",
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "semicolon format with ignored",
			output:   "45 passed; 2 failed; 3 ignored",
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 3, Total: 50, Parsed: true},
		},
		{
			name: "verbose output",
			output: `running 50 tests from ./tests/
test_foo ... ok (5ms)
test_bar ... ok (3ms)

ok | 47 passed | 0 failed (123ms)`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "Check file:///path/to/tests.ts\n",
			expected: TestCounts{Parsed: false},
		},
		{
			// Deno output without timing information (pipe format).
			// The parser uses fallback individual pattern matching.
			name:     "pipe format without timing",
			output:   "ok | 5 passed | 0 failed",
			expected: TestCounts{Passed: 5, Failed: 0, Skipped: 0, Total: 5, Parsed: true},
		},
		{
			// Pipe format with ignored: the pipe regex only captures passed|failed,
			// so ignored is lost when pipe format matches. This documents current behavior.
			// Note: Deno's actual pipe format doesn't include ignored in most versions.
			name:     "pipe format with ignored (limitation)",
			output:   "ok | 5 passed | 0 failed | 2 ignored (50ms)",
			expected: TestCounts{Passed: 5, Failed: 0, Skipped: 0, Total: 5, Parsed: true},
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
