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
		{
			// Edge case: "PASS" summary without individual test results.
			// This happens when tests pass but no --- PASS lines appear in output
			// (e.g., truncated output or special test configurations).
			// Parser returns Parsed=false because it cannot determine counts.
			name:     "pass_summary_only",
			output:   "PASS\nok\texample.com/pkg\t0.001s",
			expected: TestCounts{Parsed: false},
		},
		{
			// Edge case: "FAIL" summary without individual test results.
			// Similar to pass_summary_only but for failures.
			name:     "fail_summary_only",
			output:   "FAIL\nexit status 1",
			expected: TestCounts{Parsed: false},
		},
		{
			// Edge case: newline-delimited PASS (as seen in fuzz seeds).
			name:     "pass_newline",
			output:   "\nPASS\n",
			expected: TestCounts{Parsed: false},
		},
		{
			// Edge case: newline-delimited FAIL (as seen in fuzz seeds).
			name:     "fail_newline",
			output:   "\nFAIL\n",
			expected: TestCounts{Parsed: false},
		},
		{
			name: "interleaved parallel output",
			output: `=== RUN   TestFoo
=== RUN   TestBar
    foo_test.go:10: foo failed
--- FAIL: TestFoo (0.00s)
    bar_test.go:20: bar assertion
--- PASS: TestBar (0.01s)
=== RUN   TestBaz
--- PASS: TestBaz (0.00s)
FAIL
exit status 1`,
			expected: TestCounts{Passed: 2, Failed: 1, Skipped: 0, Total: 3, Parsed: true},
		},
		{
			name: "panic_in_test",
			output: `=== RUN   TestPanic
--- FAIL: TestPanic (0.00s)
panic: runtime error: index out of range
FAIL	example.com/pkg	0.005s`,
			expected: TestCounts{Passed: 0, Failed: 1, Skipped: 0, Total: 1, Parsed: true},
		},
		{
			name: "test_name_with_special_chars",
			output: `=== RUN   TestFoo_Bar/case-1_[special]
--- PASS: TestFoo_Bar/case-1_[special] (0.00s)
PASS
ok  	example.com/pkg	0.001s`,
			expected: TestCounts{Passed: 1, Failed: 0, Skipped: 0, Total: 1, Parsed: true},
		},
		{
			// Edge case: Unicode test names (e.g., internationalization tests)
			name: "unicode_test_name",
			output: `=== RUN   Test日本語
--- PASS: Test日本語 (0.00s)
=== RUN   TestÜnicode_名前
--- PASS: TestÜnicode_名前 (0.01s)
PASS
ok  	example.com/pkg	0.012s`,
			expected: TestCounts{Passed: 2, Failed: 0, Skipped: 0, Total: 2, Parsed: true},
		},
		{
			// Edge case: ANSI color codes at line start break parsing.
			// The parser regex expects "=== RUN" at line start, so ANSI
			// prefix codes prevent matching. Callers should strip ANSI codes
			// before parsing if needed.
			name:     "ansi_prefix_breaks_parsing",
			output:   "\x1b[32m=== RUN   TestFoo\x1b[0m\n\x1b[32m--- PASS: TestFoo (0.00s)\x1b[0m\nPASS",
			expected: TestCounts{Parsed: false},
		},
		{
			// Edge case: ANSI codes after keywords still parse.
			// When ANSI codes appear after "--- FAIL:" they don't break regex.
			name:     "ansi_suffix_parses",
			output:   "=== RUN   TestFail\n--- FAIL: TestFail (0.01s)\x1b[0m\n    test.go:10: assertion failed\nFAIL",
			expected: TestCounts{Passed: 0, Failed: 1, Skipped: 0, Total: 1, Parsed: true},
		},
		{
			// Edge case: Extremely long test name (stress test for parsing)
			name: "long_test_name",
			output: `=== RUN   TestVeryLongTestNameThatExceedsNormalLengthLimitsAndMightCauseBufferIssuesInSomeImplementations_WithSubtest/AnotherLongSubtestNameHere
--- PASS: TestVeryLongTestNameThatExceedsNormalLengthLimitsAndMightCauseBufferIssuesInSomeImplementations_WithSubtest/AnotherLongSubtestNameHere (0.00s)
PASS
ok  	example.com/pkg	0.001s`,
			expected: TestCounts{Passed: 1, Failed: 0, Skipped: 0, Total: 1, Parsed: true},
		},
		{
			// Edge case: ANSI codes in the middle of test name.
			// Since the regex anchors to line start with "---\s+PASS:", ANSI codes
			// within the test name do not affect parsing.
			name:     "ansi_midstream_parses",
			output:   "=== RUN   Test\x1b[32mFoo\x1b[0m\n--- PASS: Test\x1b[32mFoo\x1b[0m (0.00s)\nPASS",
			expected: TestCounts{Passed: 1, Failed: 0, Skipped: 0, Total: 1, Parsed: true},
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
