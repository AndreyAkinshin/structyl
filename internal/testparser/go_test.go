package testparser

import "testing"

func TestGoParser(t *testing.T) {
	parser := &GoParser{}

	tests := []struct {
		name     string
		output   string
		expected TestCounts
	}{
		{
			name: "basic pass",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- PASS: TestBar (0.01s)
PASS
ok  	example.com/pkg	0.012s`,
			expected: TestCounts{Passed: 2, Failed: 0, Skipped: 0, Total: 2, Parsed: true},
		},
		{
			name: "mixed results",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- FAIL: TestBar (0.01s)
=== RUN   TestBaz
--- SKIP: TestBaz (0.00s)
FAIL
exit status 1`,
			expected: TestCounts{Passed: 1, Failed: 1, Skipped: 1, Total: 3, Parsed: true},
		},
		{
			name: "all skip",
			output: `=== RUN   TestFoo
--- SKIP: TestFoo (0.00s)
=== RUN   TestBar
--- SKIP: TestBar (0.00s)
PASS
ok  	example.com/pkg	0.012s`,
			expected: TestCounts{Passed: 0, Failed: 0, Skipped: 2, Total: 2, Parsed: true},
		},
		{
			name: "subtests",
			output: `=== RUN   TestFoo
=== RUN   TestFoo/subtest1
--- PASS: TestFoo/subtest1 (0.00s)
=== RUN   TestFoo/subtest2
--- PASS: TestFoo/subtest2 (0.00s)
--- PASS: TestFoo (0.01s)
PASS
ok  	example.com/pkg	0.012s`,
			expected: TestCounts{Passed: 3, Failed: 0, Skipped: 0, Total: 3, Parsed: true},
		},
		{
			name:     "empty output",
			output:   "",
			expected: TestCounts{Parsed: false},
		},
		{
			name:     "no test results",
			output:   "building...\ncompiling...\n",
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

func TestGoParserName(t *testing.T) {
	parser := &GoParser{}
	if parser.Name() != "go" {
		t.Errorf("Name: got %s, want go", parser.Name())
	}
}
