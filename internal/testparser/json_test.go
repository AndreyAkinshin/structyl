package testparser

import (
	"strings"
	"testing"
)

func TestJSONParser(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCounts TestCounts
	}{
		{
			name: "all passing",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Test":"TestFoo","Output":"=== RUN   TestFoo\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Test":"TestBar","Elapsed":0.02}`,
			expectedCounts: TestCounts{Passed: 2, Failed: 0, Skipped: 0, Total: 2, Parsed: true},
		},
		{
			name: "one failure",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Test":"TestBar","Output":"    bar_test.go:15: expected 42, got 0\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestBar","Elapsed":0.02}`,
			expectedCounts: TestCounts{Passed: 1, Failed: 1, Skipped: 0, Total: 2, Parsed: true},
		},
		{
			name: "mixed results",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestPass"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Test":"TestPass","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFail"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestFail","Elapsed":0.02}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestSkip"}
{"Time":"2024-01-01T00:00:00Z","Action":"skip","Package":"example.com/pkg","Test":"TestSkip","Elapsed":0.0}`,
			expectedCounts: TestCounts{Passed: 1, Failed: 1, Skipped: 1, Total: 3, Parsed: true},
		},
		{
			name:           "empty input",
			input:          "",
			expectedCounts: TestCounts{Parsed: false},
		},
		{
			name:           "no test events",
			input:          `{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Output":"building...\n"}`,
			expectedCounts: TestCounts{Parsed: false},
		},
		{
			name: "package level events ignored",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"example.com/pkg","Elapsed":0.5}`,
			expectedCounts: TestCounts{Passed: 1, Failed: 0, Skipped: 0, Total: 1, Parsed: true},
		},
	}

	parser := &JSONParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseJSON(strings.NewReader(tt.input))
			if result.Passed != tt.expectedCounts.Passed {
				t.Errorf("Passed: got %d, want %d", result.Passed, tt.expectedCounts.Passed)
			}
			if result.Failed != tt.expectedCounts.Failed {
				t.Errorf("Failed: got %d, want %d", result.Failed, tt.expectedCounts.Failed)
			}
			if result.Skipped != tt.expectedCounts.Skipped {
				t.Errorf("Skipped: got %d, want %d", result.Skipped, tt.expectedCounts.Skipped)
			}
			if result.Total != tt.expectedCounts.Total {
				t.Errorf("Total: got %d, want %d", result.Total, tt.expectedCounts.Total)
			}
			if result.Parsed != tt.expectedCounts.Parsed {
				t.Errorf("Parsed: got %v, want %v", result.Parsed, tt.expectedCounts.Parsed)
			}
		})
	}
}

func TestJSONParserFailedTestDetails(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTests []FailedTest
	}{
		{
			name: "failure with reason",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Test":"TestBar","Output":"    bar_test.go:15: expected 42, got 0\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestBar","Elapsed":0.02}`,
			expectedTests: []FailedTest{
				{Name: "TestBar", Reason: "expected 42, got 0"},
			},
		},
		{
			name: "failure without explicit reason",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestBar","Elapsed":0.02}`,
			expectedTests: []FailedTest{
				{Name: "TestBar", Reason: ""},
			},
		},
		{
			name: "multiple failures",
			input: `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Test":"TestFoo","Output":"    foo_test.go:10: wrong value\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestFoo","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"example.com/pkg","Test":"TestBar","Output":"    bar_test.go:20: connection refused\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"example.com/pkg","Test":"TestBar","Elapsed":0.02}`,
			expectedTests: []FailedTest{
				{Name: "TestFoo", Reason: "wrong value"},
				{Name: "TestBar", Reason: "connection refused"},
			},
		},
	}

	parser := &JSONParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseJSON(strings.NewReader(tt.input))
			if len(result.FailedTests) != len(tt.expectedTests) {
				t.Errorf("FailedTests length: got %d, want %d", len(result.FailedTests), len(tt.expectedTests))
				return
			}
			// Check that all expected failures are present (order may vary due to map iteration)
			for _, expected := range tt.expectedTests {
				found := false
				for _, got := range result.FailedTests {
					if got.Name == expected.Name && got.Reason == expected.Reason {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected failed test %q with reason %q not found", expected.Name, expected.Reason)
				}
			}
		})
	}
}
