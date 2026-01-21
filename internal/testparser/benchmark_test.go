package testparser

import (
	"strings"
	"testing"
)

// Sample test outputs for benchmarking

var goTestOutput = `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- PASS: TestBar (0.01s)
=== RUN   TestBaz
--- PASS: TestBaz (0.02s)
=== RUN   TestQux
--- FAIL: TestQux (0.01s)
    qux_test.go:15: expected 42, got 0
=== RUN   TestQuux
--- SKIP: TestQuux (0.00s)
FAIL
exit status 1`

var cargoTestOutput = `running 50 tests
test test_foo ... ok
test test_bar ... ok
test test_baz ... FAILED
test test_qux ... ok

failures:

---- test_baz stdout ----
thread 'test_baz' panicked at 'assertion failed'

failures:
    test_baz

test result: FAILED. 47 passed; 1 failed; 2 ignored; 0 measured; 0 filtered out; finished in 0.15s`

var dotnetTestOutput = `Build started...
Build succeeded.

Test run for /path/to/tests.dll (.NETCoreApp,Version=v8.0)
Passed!  - Failed:     2, Passed:   147, Skipped:     5, Total:   154`

var pytestTestOutput = `============================= test session starts ==============================
platform linux -- Python 3.12.0, pytest-8.0.0
collected 50 items

test_foo.py .................................................. [100%]

===== 45 passed, 3 failed, 2 skipped in 0.50s =====`

var bunTestOutput = `bun test v1.0.0

test/foo.test.ts:
✓ test foo (0.01ms)
✓ test bar (0.02ms)
✗ test baz (0.01ms)

45 pass
3 fail
2 skip`

var denoTestOutput = `running 50 tests from ./test/
test foo ... ok (1ms)
test bar ... ok (2ms)
test baz ... FAILED (1ms)

ok | 45 passed | 3 failed | 2 ignored (500ms)`

var jsonTestOutput = `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:01Z","Action":"output","Package":"example","Test":"TestFoo","Output":"testing...\n"}
{"Time":"2024-01-01T00:00:02Z","Action":"pass","Package":"example","Test":"TestFoo","Elapsed":0.01}
{"Time":"2024-01-01T00:00:03Z","Action":"run","Package":"example","Test":"TestBar"}
{"Time":"2024-01-01T00:00:04Z","Action":"fail","Package":"example","Test":"TestBar","Elapsed":0.02}
{"Time":"2024-01-01T00:00:05Z","Action":"run","Package":"example","Test":"TestBaz"}
{"Time":"2024-01-01T00:00:06Z","Action":"skip","Package":"example","Test":"TestBaz","Elapsed":0.00}`

// BenchmarkGoParser benchmarks Go test output parsing.
// Run: go test -bench=BenchmarkGoParser -benchmem ./internal/testparser
func BenchmarkGoParser(b *testing.B) {
	parser := &GoParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(goTestOutput)
	}
}

// BenchmarkCargoParser benchmarks Cargo test output parsing.
func BenchmarkCargoParser(b *testing.B) {
	parser := &CargoParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(cargoTestOutput)
	}
}

// BenchmarkDotnetParser benchmarks .NET test output parsing.
func BenchmarkDotnetParser(b *testing.B) {
	parser := &DotnetParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(dotnetTestOutput)
	}
}

// BenchmarkPytestParser benchmarks pytest output parsing.
func BenchmarkPytestParser(b *testing.B) {
	parser := &PytestParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(pytestTestOutput)
	}
}

// BenchmarkBunParser benchmarks Bun test output parsing.
func BenchmarkBunParser(b *testing.B) {
	parser := &BunParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(bunTestOutput)
	}
}

// BenchmarkDenoParser benchmarks Deno test output parsing.
func BenchmarkDenoParser(b *testing.B) {
	parser := &DenoParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(denoTestOutput)
	}
}

// BenchmarkJSONParser benchmarks JSON (go test -json) output parsing.
func BenchmarkJSONParser(b *testing.B) {
	parser := &JSONParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.ParseJSON(strings.NewReader(jsonTestOutput))
	}
}

// BenchmarkGoParser_Large benchmarks Go parser with larger output.
func BenchmarkGoParser_Large(b *testing.B) {
	// Generate a large test output with 1000 tests
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("=== RUN   Test")
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString("\n--- PASS: Test")
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString(" (0.0")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString("s)\n")
	}
	sb.WriteString("PASS\nok\texample.com/pkg\t1.234s")
	largeOutput := sb.String()

	parser := &GoParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(largeOutput)
	}
}

// BenchmarkCargoParser_Multiple benchmarks Cargo parser with multiple test binaries.
func BenchmarkCargoParser_Multiple(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("running 100 tests\n")
		sb.WriteString("test result: ok. 95 passed; 3 failed; 2 ignored; 0 measured; 0 filtered out; finished in 0.5s\n\n")
	}
	multiOutput := sb.String()

	parser := &CargoParser{}
	b.ResetTimer()
	for b.Loop() {
		parser.Parse(multiOutput)
	}
}

// BenchmarkTestCounts_Add benchmarks the Add method for aggregating test counts.
func BenchmarkTestCounts_Add(b *testing.B) {
	base := &TestCounts{Passed: 10, Failed: 2, Skipped: 1, Total: 13, Parsed: true}
	other := &TestCounts{Passed: 5, Failed: 1, Skipped: 0, Total: 6, Parsed: true}

	b.ResetTimer()
	for b.Loop() {
		tc := *base // copy to avoid accumulation
		tc.Add(other)
	}
}

// BenchmarkRegistry_GetParser benchmarks registry lookup.
func BenchmarkRegistry_GetParser(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for b.Loop() {
		registry.GetParser("go")
	}
}

// BenchmarkRegistry_GetParserForTask benchmarks getting parser by task name.
func BenchmarkRegistry_GetParserForTask(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for b.Loop() {
		registry.GetParserForTask("test:go")
	}
}

// BenchmarkNewRegistry benchmarks registry creation.
func BenchmarkNewRegistry(b *testing.B) {
	for b.Loop() {
		NewRegistry()
	}
}
