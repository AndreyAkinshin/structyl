package testparser

import (
	"strings"
	"testing"
)

// FuzzGoParser tests the Go test output parser with arbitrary input.
// Run: go test -fuzz=FuzzGoParser -fuzztime=30s ./internal/testparser
func FuzzGoParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid Go test output
		"=== RUN   TestFoo\n--- PASS: TestFoo (0.00s)\nPASS\nok\texample.com/pkg\t0.012s",
		"=== RUN   TestBar\n--- FAIL: TestBar (0.01s)\nFAIL\nexit status 1",
		"=== RUN   TestBaz\n--- SKIP: TestBaz (0.00s)\nPASS\nok\texample.com/pkg\t0.012s",
		// Subtests
		"=== RUN   TestFoo\n=== RUN   TestFoo/subtest1\n--- PASS: TestFoo/subtest1 (0.00s)\n--- PASS: TestFoo (0.01s)\nPASS",
		// Failure with reason
		"=== RUN   TestFoo\n    foo_test.go:15: expected 42, got 0\n--- FAIL: TestFoo (0.01s)\nFAIL",
		// Empty and edge cases
		"",
		"\n",
		"building...\ncompiling...\n",
		"PASS",
		"FAIL",
		// Partial/malformed
		"=== RUN   Test",
		"--- PASS:",
		"--- FAIL: (0.00s)",
		"ok  \t",
		// Edge cases: very long test names
		"=== RUN   " + strings.Repeat("x", 10000) + "\n--- PASS: " + strings.Repeat("x", 10000) + " (0.00s)\nPASS",
		// Edge cases: binary-like prefix before valid output
		"\x00\x01\x02=== RUN   TestWithBinaryPrefix\n--- PASS: TestWithBinaryPrefix (0.00s)\nPASS",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &GoParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants that must hold for any input
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// FailedTests length should match Failed count
		if len(result.FailedTests) > result.Failed {
			t.Errorf("FailedTests length %d exceeds Failed count %d",
				len(result.FailedTests), result.Failed)
		}

		// FailedTests elements should have non-empty names
		for i, ft := range result.FailedTests {
			if ft.Name == "" {
				t.Errorf("FailedTests[%d].Name is empty", i)
			}
		}
	})
}

// FuzzCargoParser tests the Cargo test output parser with arbitrary input.
// Run: go test -fuzz=FuzzCargoParser -fuzztime=30s ./internal/testparser
func FuzzCargoParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid Cargo test output
		"running 47 tests\ntest test_foo ... ok\ntest result: ok. 47 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.12s",
		"test result: FAILED. 45 passed; 2 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.15s",
		// Multiple test binaries
		"running 20 tests\ntest result: ok. 20 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.05s\n\nrunning 30 tests\ntest result: ok. 27 passed; 0 failed; 3 ignored; 0 measured; 0 filtered out; finished in 0.08s",
		// Empty and edge cases
		"",
		"\n",
		"   Compiling example v0.1.0\n    Finished test [unoptimized + debuginfo] target(s)\n",
		// Partial/malformed
		"test result:",
		"test result: ok.",
		"test result: ok. passed",
		"test result: ok. 0 passed;",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &CargoParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed.
		// Note: Cargo parser may have Parsed=true with Total=0 if the regex matches
		// a line like "test result: ok. 0 passed; 0 failed; 0 ignored;". The invariant
		// Total == sum still holds (0 == 0+0+0).
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// Note: CargoParser does not populate FailedTests (see cargo_test.go).
		// Only GoParser and JSONParser extract failure details.
		if len(result.FailedTests) != 0 {
			t.Errorf("CargoParser should not populate FailedTests, got %d entries",
				len(result.FailedTests))
		}
	})
}

// FuzzDotnetParser tests the .NET test output parser with arbitrary input.
// Run: go test -fuzz=FuzzDotnetParser -fuzztime=30s ./internal/testparser
func FuzzDotnetParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid dotnet test output
		"Passed!  - Failed:     0, Passed:    47, Skipped:     3, Total:    50",
		"Failed!  - Failed:     2, Passed:    45, Skipped:     3, Total:    50",
		// Multi-line format
		"Total tests: 50\n     Passed: 47\n     Failed: 2\n    Skipped: 1",
		// With build output
		"Build started...\nBuild succeeded.\n\nTest run for /path/to/tests.dll (.NETCoreApp,Version=v8.0)\nPassed!  - Failed:     0, Passed:    47, Skipped:     0, Total:    47",
		// Empty and edge cases
		"",
		"\n",
		"Build started...\nBuild succeeded.\n",
		// Partial/malformed
		"Failed!  -",
		"Passed:    47",
		"Total tests:",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &DotnetParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// Note: DotnetParser does not populate FailedTests.
		// Only GoParser and JSONParser extract failure details.
		if len(result.FailedTests) != 0 {
			t.Errorf("DotnetParser should not populate FailedTests, got %d entries",
				len(result.FailedTests))
		}
	})
}

// FuzzPytestParser tests the pytest output parser with arbitrary input.
// Run: go test -fuzz=FuzzPytestParser -fuzztime=30s ./internal/testparser
func FuzzPytestParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid pytest output
		"===== 5 passed in 0.12s =====",
		"===== 3 passed, 2 failed in 0.15s =====",
		"===== 1 passed, 1 failed, 1 skipped in 0.10s =====",
		"===== 10 passed, 2 failed, 3 skipped, 1 error in 0.50s =====",
		// Short format
		"5 passed",
		"3 passed, 2 failed",
		// Empty and edge cases
		"",
		"\n",
		"collecting ...",
		// Partial/malformed
		"===== passed =====",
		"===== 0 =====",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &PytestParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// Note: PytestParser does not populate FailedTests.
		// Only GoParser and JSONParser extract failure details.
		if len(result.FailedTests) != 0 {
			t.Errorf("PytestParser should not populate FailedTests, got %d entries",
				len(result.FailedTests))
		}
	})
}

// FuzzBunParser tests the Bun test output parser with arbitrary input.
// Run: go test -fuzz=FuzzBunParser -fuzztime=30s ./internal/testparser
func FuzzBunParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid Bun test output
		"5 pass\n0 fail",
		"3 pass\n2 fail\n1 skip",
		"10 pass\n0 fail\n0 skip",
		// Empty and edge cases
		"",
		"\n",
		"bun test v1.0.0",
		// Partial/malformed
		"pass",
		"0 pass",
		"fail",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &BunParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// Note: BunParser does not populate FailedTests.
		// Only GoParser and JSONParser extract failure details.
		if len(result.FailedTests) != 0 {
			t.Errorf("BunParser should not populate FailedTests, got %d entries",
				len(result.FailedTests))
		}
	})
}

// FuzzDenoParser tests the Deno test output parser with arbitrary input.
// Run: go test -fuzz=FuzzDenoParser -fuzztime=30s ./internal/testparser
func FuzzDenoParser(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid Deno test output
		"ok | 5 passed | 0 failed (50ms)",
		"ok | 3 passed | 2 failed (100ms)",
		"ok | 10 passed (1s)",
		"ok | 5 passed | 1 failed | 2 ignored (200ms)",
		// Empty and edge cases
		"",
		"\n",
		"running 5 tests from",
		// Partial/malformed
		"ok |",
		"passed",
		"| 0 failed",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &DenoParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.Parse(input)

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// Note: DenoParser does not populate FailedTests.
		// Only GoParser and JSONParser extract failure details.
		if len(result.FailedTests) != 0 {
			t.Errorf("DenoParser should not populate FailedTests, got %d entries",
				len(result.FailedTests))
		}
	})
}

// FuzzJSONParser tests the JSON test output parser with arbitrary input.
// Run: go test -fuzz=FuzzJSONParser -fuzztime=30s ./internal/testparser
func FuzzJSONParser(f *testing.F) {
	// Seed corpus with representative inputs (go test -json format - newline-delimited JSON)
	seeds := []string{
		// Valid go test -json output
		`{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example","Test":"TestFoo"}`,
		`{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example","Test":"TestBar"}`,
		`{"Time":"2024-01-01T00:00:00Z","Action":"skip","Package":"example","Test":"TestBaz"}`,
		// Multiple events
		`{"Action":"run","Test":"TestFoo"}
{"Action":"output","Test":"TestFoo","Output":"testing...\n"}
{"Action":"pass","Test":"TestFoo"}`,
		// Empty and edge cases
		"",
		"{}",
		"\n",
		// Partial/malformed JSON
		`{"Action":`,
		`{"Action": "pass"}`,
		`{Action: pass}`,
		// Invalid actions
		`{"Action":"unknown","Test":"Test"}`,
		// Edge case: deeply nested JSON (not typical but should not panic)
		`{"Action":"pass","Test":"Test","Extra":{"nested":{"deep":{"value":"test"}}}}`,
		// Edge case: very large output field
		`{"Action":"output","Test":"Test","Output":"` + strings.Repeat("x", 10000) + `"}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parser := &JSONParser{}
	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic on any input
		result := parser.ParseJSON(strings.NewReader(input))

		// Basic invariants
		if result.Passed < 0 || result.Failed < 0 || result.Skipped < 0 || result.Total < 0 {
			t.Errorf("negative count: passed=%d failed=%d skipped=%d total=%d",
				result.Passed, result.Failed, result.Skipped, result.Total)
		}

		// Total should equal sum of components when parsed
		if result.Parsed {
			sum := result.Passed + result.Failed + result.Skipped
			if result.Total != sum {
				t.Errorf("total mismatch: total=%d, sum=%d", result.Total, sum)
			}
		}

		// When not parsed, all counts must be zero
		if !result.Parsed {
			if result.Passed != 0 || result.Failed != 0 || result.Skipped != 0 || result.Total != 0 {
				t.Errorf("unparsed result has non-zero counts: passed=%d failed=%d skipped=%d total=%d",
					result.Passed, result.Failed, result.Skipped, result.Total)
			}
		}

		// FailedTests length should not exceed Failed count
		if len(result.FailedTests) > result.Failed {
			t.Errorf("FailedTests length %d exceeds Failed count %d",
				len(result.FailedTests), result.Failed)
		}

		// FailedTests elements should have non-empty names
		for i, ft := range result.FailedTests {
			if ft.Name == "" {
				t.Errorf("FailedTests[%d].Name is empty", i)
			}
		}
	})
}
