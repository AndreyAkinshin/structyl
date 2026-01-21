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

func TestDenoParserName(t *testing.T) {
	t.Parallel()
	parser := &DenoParser{}
	if parser.Name() != "deno" {
		t.Errorf("Name: got %s, want deno", parser.Name())
	}
}
