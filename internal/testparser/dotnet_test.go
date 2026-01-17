package testparser

import "testing"

func TestDotnetParser(t *testing.T) {
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

func TestDotnetParserName(t *testing.T) {
	parser := &DotnetParser{}
	if parser.Name() != "dotnet" {
		t.Errorf("Name: got %s, want dotnet", parser.Name())
	}
}
