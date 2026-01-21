package testparser

import "testing"

func TestTestCountsAdd_NilReceiver(t *testing.T) {
	t.Parallel()
	// Document that nil receiver panics (standard Go behavior)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil receiver, got none")
		}
	}()

	var tc *TestCounts
	tc.Add(&TestCounts{Passed: 1})
}

func TestTestCountsAdd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		base     TestCounts
		add      *TestCounts
		expected TestCounts
	}{
		{
			name:     "add to zero",
			base:     TestCounts{},
			add:      &TestCounts{Passed: 10, Failed: 2, Skipped: 3, Total: 15, Parsed: true},
			expected: TestCounts{Passed: 10, Failed: 2, Skipped: 3, Total: 15, Parsed: true},
		},
		{
			name:     "add to existing",
			base:     TestCounts{Passed: 5, Failed: 1, Skipped: 2, Total: 8, Parsed: true},
			add:      &TestCounts{Passed: 10, Failed: 2, Skipped: 3, Total: 15, Parsed: true},
			expected: TestCounts{Passed: 15, Failed: 3, Skipped: 5, Total: 23, Parsed: true},
		},
		{
			name:     "add nil",
			base:     TestCounts{Passed: 5, Failed: 1, Skipped: 2, Total: 8, Parsed: true},
			add:      nil,
			expected: TestCounts{Passed: 5, Failed: 1, Skipped: 2, Total: 8, Parsed: true},
		},
		{
			name:     "add unparsed to parsed",
			base:     TestCounts{Passed: 5, Parsed: true},
			add:      &TestCounts{Passed: 10, Parsed: false},
			expected: TestCounts{Passed: 15, Parsed: true}, // Stays parsed
		},
		{
			name:     "add parsed to unparsed",
			base:     TestCounts{Passed: 5, Parsed: false},
			add:      &TestCounts{Passed: 10, Parsed: true},
			expected: TestCounts{Passed: 15, Parsed: true}, // Becomes parsed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := tt.base
			base.Add(tt.add)

			if base.Passed != tt.expected.Passed {
				t.Errorf("Passed: got %d, want %d", base.Passed, tt.expected.Passed)
			}
			if base.Failed != tt.expected.Failed {
				t.Errorf("Failed: got %d, want %d", base.Failed, tt.expected.Failed)
			}
			if base.Skipped != tt.expected.Skipped {
				t.Errorf("Skipped: got %d, want %d", base.Skipped, tt.expected.Skipped)
			}
			if base.Total != tt.expected.Total {
				t.Errorf("Total: got %d, want %d", base.Total, tt.expected.Total)
			}
			if base.Parsed != tt.expected.Parsed {
				t.Errorf("Parsed: got %v, want %v", base.Parsed, tt.expected.Parsed)
			}
		})
	}
}
