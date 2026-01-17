package testparser

import "testing"

func TestBunParser(t *testing.T) {
	parser := &BunParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name:     "basic pass",
			output:   "47 pass",
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "with failures",
			output:   "45 pass\n2 fail",
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "with skip",
			output:   "30 pass\n0 fail\n3 skip",
			expected: TestCounts{Passed: 30, Failed: 0, Skipped: 3, Total: 33, Parsed: true},
		},
		{
			name:     "combined format",
			output:   "47 pass, 2 fail, 3 skip",
			expected: TestCounts{Passed: 47, Failed: 2, Skipped: 3, Total: 52, Parsed: true},
		},
		{
			name: "verbose output",
			output: `bun test v1.0.0
test_foo.ts:
  ✓ should work
  ✓ should also work

47 pass`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "Starting tests...\n",
			expected: TestCounts{Parsed: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestBunParserName(t *testing.T) {
	parser := &BunParser{}
	if parser.Name() != "bun" {
		t.Errorf("Name: got %s, want bun", parser.Name())
	}
}
