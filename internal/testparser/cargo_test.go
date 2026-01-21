package testparser

import "testing"

func TestCargoParser(t *testing.T) {
	t.Parallel()
	parser := &CargoParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name: "basic pass",
			output: `running 47 tests
test test_foo ... ok
test test_bar ... ok

test result: ok. 47 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.12s`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 0, Total: 47, Parsed: true},
		},
		{
			name: "with failures",
			output: `running 50 tests
test test_foo ... ok
test test_bar ... FAILED

test result: FAILED. 45 passed; 2 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.15s`,
			expected: TestCounts{Passed: 45, Failed: 2, Skipped: 3, Total: 50, Parsed: true},
		},
		{
			name: "multiple test binaries",
			output: `running 20 tests
test result: ok. 20 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.05s

running 30 tests
test result: ok. 27 passed; 0 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.08s`,
			expected: TestCounts{Passed: 47, Failed: 0, Skipped: 3, Total: 50, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "   Compiling example v0.1.0\n    Finished test [unoptimized + debuginfo] target(s)\n",
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

func TestCargoParserName(t *testing.T) {
	t.Parallel()
	parser := &CargoParser{}
	if parser.Name() != "cargo" {
		t.Errorf("Name: got %s, want cargo", parser.Name())
	}
}
