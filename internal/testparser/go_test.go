package testparser

import "testing"

func TestGoParser(t *testing.T) {
	t.Parallel()
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

func TestGoParserName(t *testing.T) {
	t.Parallel()
	parser := &GoParser{}
	if parser.Name() != "go" {
		t.Errorf("Name: got %s, want go", parser.Name())
	}
}

func TestGoParserFailedTestDetails(t *testing.T) {
	t.Parallel()
	parser := &GoParser{}

	tests := []struct {
		name           string
		output         string
		expectedFailed int
		expectedTests  []FailedTest
	}{
		{
			name: "single failure with reason",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
    bar_test.go:15: expected 42, got 0
--- FAIL: TestBar (0.01s)
FAIL`,
			expectedFailed: 1,
			expectedTests: []FailedTest{
				{Name: "TestBar", Reason: "expected 42, got 0"},
			},
		},
		{
			name: "multiple failures",
			output: `=== RUN   TestFoo
    foo_test.go:10: assertion failed: wrong value
--- FAIL: TestFoo (0.00s)
=== RUN   TestBar
    bar_test.go:20: unexpected error: connection refused
--- FAIL: TestBar (0.01s)
FAIL`,
			expectedFailed: 2,
			expectedTests: []FailedTest{
				{Name: "TestFoo", Reason: "assertion failed: wrong value"},
				{Name: "TestBar", Reason: "unexpected error: connection refused"},
			},
		},
		{
			name: "failure without explicit reason line",
			output: `=== RUN   TestFoo
--- FAIL: TestFoo (0.00s)
FAIL`,
			expectedFailed: 1,
			expectedTests: []FailedTest{
				{Name: "TestFoo", Reason: ""},
			},
		},
		{
			name: "subtest failure",
			output: `=== RUN   TestFoo
=== RUN   TestFoo/subcase
    foo_test.go:25: subtest failed
--- FAIL: TestFoo/subcase (0.00s)
--- FAIL: TestFoo (0.01s)
FAIL`,
			expectedFailed: 2,
			expectedTests: []FailedTest{
				{Name: "TestFoo/subcase", Reason: "subtest failed"},
				{Name: "TestFoo", Reason: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.Parse(tt.output)
			if result.Failed != tt.expectedFailed {
				t.Errorf("Failed count: got %d, want %d", result.Failed, tt.expectedFailed)
			}
			if len(result.FailedTests) != len(tt.expectedTests) {
				t.Errorf("FailedTests length: got %d, want %d", len(result.FailedTests), len(tt.expectedTests))
				return
			}
			for i, expected := range tt.expectedTests {
				got := result.FailedTests[i]
				if got.Name != expected.Name {
					t.Errorf("FailedTests[%d].Name: got %q, want %q", i, got.Name, expected.Name)
				}
				if got.Reason != expected.Reason {
					t.Errorf("FailedTests[%d].Reason: got %q, want %q", i, got.Reason, expected.Reason)
				}
			}
		})
	}
}

func TestGoParserReasonTruncation(t *testing.T) {
	t.Parallel()
	parser := &GoParser{}

	// maxLen is 80 in go.go:152
	// Generate strings of specific lengths to test boundary conditions
	chars79 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"                       // 79 chars
	chars80 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"                      // 80 chars
	chars81 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"                     // 81 chars
	chars100 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 100 chars

	// Truncated result: 77 chars + "..." = 80 chars
	chars77 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 77 chars

	tests := []struct {
		name           string
		output         string
		expectedReason string
	}{
		{
			name: "reason_79_chars_no_truncation",
			output: `=== RUN   TestFoo
    foo_test.go:10: ` + chars79 + `
--- FAIL: TestFoo (0.00s)
FAIL`,
			expectedReason: chars79,
		},
		{
			name: "reason_80_chars_no_truncation",
			output: `=== RUN   TestFoo
    foo_test.go:10: ` + chars80 + `
--- FAIL: TestFoo (0.00s)
FAIL`,
			expectedReason: chars80,
		},
		{
			name: "reason_81_chars_truncated",
			output: `=== RUN   TestFoo
    foo_test.go:10: ` + chars81 + `
--- FAIL: TestFoo (0.00s)
FAIL`,
			expectedReason: chars77 + "...",
		},
		{
			name: "reason_100_chars_truncated",
			output: `=== RUN   TestFoo
    foo_test.go:10: ` + chars100 + `
--- FAIL: TestFoo (0.00s)
FAIL`,
			expectedReason: chars77 + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.Parse(tt.output)
			if result.Failed != 1 {
				t.Errorf("Failed count: got %d, want 1", result.Failed)
				return
			}
			if len(result.FailedTests) != 1 {
				t.Errorf("FailedTests length: got %d, want 1", len(result.FailedTests))
				return
			}
			got := result.FailedTests[0].Reason
			if got != tt.expectedReason {
				t.Errorf("Reason: got %q (len=%d), want %q (len=%d)",
					got, len(got), tt.expectedReason, len(tt.expectedReason))
			}
		})
	}
}
