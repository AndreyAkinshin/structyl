package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdTestSummary_Help(t *testing.T) {
	tests := []struct {
		args []string
	}{
		{[]string{"-h"}},
		{[]string{"--help"}},
	}

	for _, tc := range tests {
		code := cmdTestSummary(tc.args)
		if code != 0 {
			t.Errorf("cmdTestSummary(%v) = %d, want 0", tc.args, code)
		}
	}
}

func TestCmdTestSummary_FileNotFound(t *testing.T) {
	code := cmdTestSummary([]string{"/nonexistent/path/test.json"})
	if code != 1 {
		t.Errorf("cmdTestSummary(nonexistent file) = %d, want 1", code)
	}
}

func TestCmdTestSummary_EmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.json")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	code := cmdTestSummary([]string{testFile})
	if code != 1 {
		t.Errorf("cmdTestSummary(empty file) = %d, want 1 (no test results)", code)
	}
}

func TestCmdTestSummary_ValidJSON_AllPassing(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "passing.json")

	// Simulate go test -json output with passing tests
	jsonContent := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestFoo","Elapsed":0.1}
{"Time":"2024-01-01T00:00:01Z","Action":"run","Package":"example","Test":"TestBar"}
{"Time":"2024-01-01T00:00:02Z","Action":"pass","Package":"example","Test":"TestBar","Elapsed":0.2}
`

	if err := os.WriteFile(testFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	code := cmdTestSummary([]string{testFile})
	if code != 0 {
		t.Errorf("cmdTestSummary(all passing) = %d, want 0", code)
	}
}

func TestCmdTestSummary_ValidJSON_WithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "failing.json")

	// Simulate go test -json output with a failing test
	jsonContent := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestFoo","Elapsed":0.1}
{"Time":"2024-01-01T00:00:01Z","Action":"run","Package":"example","Test":"TestBar"}
{"Time":"2024-01-01T00:00:02Z","Action":"output","Package":"example","Test":"TestBar","Output":"    foo_test.go:42: expected 1, got 2\n"}
{"Time":"2024-01-01T00:00:02Z","Action":"fail","Package":"example","Test":"TestBar","Elapsed":0.2}
`

	if err := os.WriteFile(testFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	code := cmdTestSummary([]string{testFile})
	if code != 1 {
		t.Errorf("cmdTestSummary(with failures) = %d, want 1", code)
	}
}

func TestCmdTestSummary_ValidJSON_WithSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "skipped.json")

	// Simulate go test -json output with skipped tests
	jsonContent := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestFoo","Elapsed":0.1}
{"Time":"2024-01-01T00:00:01Z","Action":"run","Package":"example","Test":"TestBar"}
{"Time":"2024-01-01T00:00:02Z","Action":"skip","Package":"example","Test":"TestBar","Elapsed":0.0}
`

	if err := os.WriteFile(testFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	code := cmdTestSummary([]string{testFile})
	if code != 0 {
		t.Errorf("cmdTestSummary(with skipped) = %d, want 0 (skipped tests don't fail)", code)
	}
}

func TestCmdTestSummary_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.json")

	// Non-JSON content
	if err := os.WriteFile(testFile, []byte("not json at all"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	code := cmdTestSummary([]string{testFile})
	if code != 1 {
		t.Errorf("cmdTestSummary(invalid JSON) = %d, want 1 (no test results)", code)
	}
}
