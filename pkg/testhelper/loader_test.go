package testhelper

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTestCase(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test1.json")
	content := `{
		"input": {"a": 1, "b": 2},
		"output": {"sum": 3},
		"description": "add two numbers"
	}`
	os.WriteFile(testFile, []byte(content), 0644)

	tc, err := LoadTestCase(testFile)
	if err != nil {
		t.Fatalf("LoadTestCase() error = %v", err)
	}

	if tc.Name != "test1" {
		t.Errorf("Name = %q, want %q", tc.Name, "test1")
	}
	if tc.Description != "add two numbers" {
		t.Errorf("Description = %q, want %q", tc.Description, "add two numbers")
	}
	if tc.Input["a"] != float64(1) {
		t.Errorf("Input[a] = %v, want 1", tc.Input["a"])
	}
	// LoadTestCase should not set Suite
	if tc.Suite != "" {
		t.Errorf("Suite = %q, want empty string", tc.Suite)
	}
}

func TestLoadTestCaseWithSuite(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test1.json")
	content := `{
		"input": {"a": 1, "b": 2},
		"output": {"sum": 3}
	}`
	os.WriteFile(testFile, []byte(content), 0644)

	tc, err := LoadTestCaseWithSuite(testFile, "math")
	if err != nil {
		t.Fatalf("LoadTestCaseWithSuite() error = %v", err)
	}

	if tc.Name != "test1" {
		t.Errorf("Name = %q, want %q", tc.Name, "test1")
	}
	if tc.Suite != "math" {
		t.Errorf("Suite = %q, want %q", tc.Suite, "math")
	}
}

func TestLoadTestSuite(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "math")
	os.MkdirAll(suiteDir, 0755)

	// Create test files
	os.WriteFile(filepath.Join(suiteDir, "add.json"), []byte(`{"input": {}, "output": 1}`), 0644)
	os.WriteFile(filepath.Join(suiteDir, "sub.json"), []byte(`{"input": {}, "output": 2}`), 0644)

	cases, err := LoadTestSuite(tmpDir, "math")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	if len(cases) != 2 {
		t.Errorf("len(cases) = %d, want 2", len(cases))
	}

	for _, tc := range cases {
		if tc.Suite != "math" {
			t.Errorf("Suite = %q, want %q", tc.Suite, "math")
		}
	}
}

func TestLoadTestSuite_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "empty")
	os.MkdirAll(suiteDir, 0755)

	cases, err := LoadTestSuite(tmpDir, "empty")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	if len(cases) != 0 {
		t.Errorf("len(cases) = %d, want 0", len(cases))
	}
}

func TestLoadTestSuite_EmptyDirectory_ReturnsEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "empty")
	os.MkdirAll(suiteDir, 0755)

	cases, err := LoadTestSuite(tmpDir, "empty")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	// Verify empty slice (not nil) for consistency with ListSuites/LoadAllSuites
	if cases == nil {
		t.Error("LoadTestSuite() should return empty slice, not nil")
	}
}

func TestLoadTestSuite_NonexistentSuite_ReturnsSuiteNotFoundError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create tests directory but not the suite
	os.MkdirAll(filepath.Join(tmpDir, "tests"), 0755)

	_, err := LoadTestSuite(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("LoadTestSuite() expected error for nonexistent suite")
	}

	// Check error type
	var snfErr *SuiteNotFoundError
	if !errors.As(err, &snfErr) {
		t.Errorf("error type = %T, want *SuiteNotFoundError", err)
	}

	// Check errors.Is with sentinel
	if !errors.Is(err, ErrSuiteNotFound) {
		t.Error("errors.Is(err, ErrSuiteNotFound) should return true")
	}

	// Check suite name in error
	if snfErr != nil && snfErr.Suite != "nonexistent" {
		t.Errorf("SuiteNotFoundError.Suite = %q, want %q", snfErr.Suite, "nonexistent")
	}
}

func TestSuiteNotFoundError(t *testing.T) {
	err := &SuiteNotFoundError{Suite: "mysuite"}

	if err.Error() == "" {
		t.Error("Error() should return message")
	}

	if !strings.Contains(err.Error(), "mysuite") {
		t.Error("Error() should contain suite name")
	}

	if !errors.Is(err, ErrSuiteNotFound) {
		t.Error("errors.Is(SuiteNotFoundError, ErrSuiteNotFound) should return true")
	}
}

func TestLoadAllSuites(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple suites
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite1"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite2"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "tests", "suite1", "test.json"), []byte(`{"input": {}, "output": 1}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "tests", "suite2", "test.json"), []byte(`{"input": {}, "output": 2}`), 0644)

	suites, err := LoadAllSuites(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllSuites() error = %v", err)
	}

	if len(suites) != 2 {
		t.Errorf("len(suites) = %d, want 2", len(suites))
	}
}

func TestFindProjectRootFrom(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "src", "deep")
	os.MkdirAll(subDir, 0755)

	// Create .structyl/config.json at root
	structylDir := filepath.Join(tmpDir, ".structyl")
	os.MkdirAll(structylDir, 0755)
	os.WriteFile(filepath.Join(structylDir, "config.json"), []byte(`{}`), 0644)

	// Find from subdir
	root, err := FindProjectRootFrom(subDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != tmpDir {
		t.Errorf("root = %q, want %q", root, tmpDir)
	}
}

func TestFindProjectRootFrom_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindProjectRootFrom(tmpDir)
	if err == nil {
		t.Error("expected error when config.json not found")
	}

	if _, ok := err.(*ProjectNotFoundError); !ok {
		t.Errorf("error type = %T, want *ProjectNotFoundError", err)
	}
}

func TestProjectNotFoundError(t *testing.T) {
	err := &ProjectNotFoundError{StartDir: "/some/path"}

	if err.Error() == "" {
		t.Error("Error() should return message")
	}
}

func TestProjectNotFoundError_Is(t *testing.T) {
	err := &ProjectNotFoundError{StartDir: "/some/path"}

	// Test errors.Is with sentinel
	if !errors.Is(err, ErrProjectNotFound) {
		t.Error("errors.Is(ProjectNotFoundError, ErrProjectNotFound) should return true")
	}

	// Test errors.Is with unrelated error
	if errors.Is(err, errors.New("unrelated")) {
		t.Error("errors.Is should return false for unrelated errors")
	}
}

func TestErrProjectNotFound_FromFindProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindProjectRootFrom(tmpDir)
	if err == nil {
		t.Fatal("expected error")
	}

	// Should match sentinel via errors.Is
	if !errors.Is(err, ErrProjectNotFound) {
		t.Error("FindProjectRootFrom error should match ErrProjectNotFound via errors.Is")
	}
}

func TestListSuites(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite1"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "tests", "file.txt"), []byte{}, 0644) // Non-directory

	suites, err := ListSuites(tmpDir)
	if err != nil {
		t.Fatalf("ListSuites() error = %v", err)
	}

	if len(suites) != 2 {
		t.Errorf("len(suites) = %d, want 2", len(suites))
	}
}

func TestListSuites_NoTestsDir(t *testing.T) {
	tmpDir := t.TempDir()

	suites, err := ListSuites(tmpDir)
	if err != nil {
		t.Fatalf("ListSuites() error = %v", err)
	}

	if suites == nil {
		t.Error("ListSuites() should return empty slice, not nil")
	}
	if len(suites) != 0 {
		t.Errorf("expected empty slice, got %v", suites)
	}
}

func TestSuiteExists(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "tests", "exists"), 0755)

	if !SuiteExists(tmpDir, "exists") {
		t.Error("SuiteExists should return true for existing suite")
	}

	if SuiteExists(tmpDir, "notexists") {
		t.Error("SuiteExists should return false for non-existing suite")
	}
}

func TestTestCaseExists(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "tests", "suite", "test.json"), []byte(`{}`), 0644)

	if !TestCaseExists(tmpDir, "suite", "test") {
		t.Error("TestCaseExists should return true for existing test")
	}

	if TestCaseExists(tmpDir, "suite", "notexists") {
		t.Error("TestCaseExists should return false for non-existing test")
	}
}

func TestTestCase_Fields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{
		"input": {"x": 1},
		"output": 42,
		"description": "desc",
		"skip": true,
		"tags": ["math", "basic"]
	}`
	os.WriteFile(testFile, []byte(content), 0644)

	tc, err := LoadTestCase(testFile)
	if err != nil {
		t.Fatalf("LoadTestCase() error = %v", err)
	}

	if tc.Description != "desc" {
		t.Errorf("Description = %q, want %q", tc.Description, "desc")
	}
	if !tc.Skip {
		t.Error("Skip should be true")
	}
	if len(tc.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(tc.Tags))
	}
}

// =============================================================================
// Work Item 9: FindProjectRoot Tests
// =============================================================================

// withWorkingDir changes to the specified directory for the duration of the
// function call, then restores the original working directory.
func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to %s: %v", dir, err)
	}

	defer func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	fn()
}

func TestFindProjectRoot_FromProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Handle macOS symlinks
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create .structyl/config.json at root
	structylDir := filepath.Join(root, ".structyl")
	os.MkdirAll(structylDir, 0755)
	os.WriteFile(filepath.Join(structylDir, "config.json"), []byte(`{}`), 0644)

	withWorkingDir(t, root, func() {
		foundRoot, err := FindProjectRoot()
		if err != nil {
			t.Errorf("FindProjectRoot() error = %v", err)
		}
		if foundRoot != root {
			t.Errorf("FindProjectRoot() = %q, want %q", foundRoot, root)
		}
	})
}

func TestFindProjectRoot_FromSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	// Handle macOS symlinks
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(root, "src", "deep", "nested")
	os.MkdirAll(subDir, 0755)

	// Create .structyl/config.json at root
	structylDir := filepath.Join(root, ".structyl")
	os.MkdirAll(structylDir, 0755)
	os.WriteFile(filepath.Join(structylDir, "config.json"), []byte(`{}`), 0644)

	withWorkingDir(t, subDir, func() {
		foundRoot, err := FindProjectRoot()
		if err != nil {
			t.Errorf("FindProjectRoot() error = %v", err)
		}
		if foundRoot != root {
			t.Errorf("FindProjectRoot() = %q, want %q", foundRoot, root)
		}
	})
}

func TestFindProjectRoot_NotFound_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	// Handle macOS symlinks
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// No config.json in this directory tree
	withWorkingDir(t, root, func() {
		_, err := FindProjectRoot()
		if err == nil {
			t.Error("FindProjectRoot() expected error when config.json not found")
		}

		// Verify it's the correct error type
		if _, ok := err.(*ProjectNotFoundError); !ok {
			t.Errorf("error type = %T, want *ProjectNotFoundError", err)
		}
	})
}

// =============================================================================
// Work Item 5: Additional Coverage Tests
// =============================================================================

func TestLoadTestCase_InvalidJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(testFile, []byte("{invalid json}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for invalid JSON")
	}
}

func TestLoadTestCase_FileNotFound_ReturnsError(t *testing.T) {
	_, err := LoadTestCase("/nonexistent/path/test.json")
	if err == nil {
		t.Error("LoadTestCase() expected error for missing file")
	}
}

func TestLoadTestCase_MissingInput_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "missing_input.json")

	// Test case with output but no input
	content := `{"output": 42}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for missing input field")
	}
	if err != nil && !strings.Contains(err.Error(), "missing required field \"input\"") {
		t.Errorf("error should mention missing input field, got: %v", err)
	}
}

func TestLoadTestCase_MissingOutput_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "missing_output.json")

	// Test case with input but no output
	content := `{"input": {"a": 1}}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for missing output field")
	}
	if err != nil && !strings.Contains(err.Error(), "missing required field \"output\"") {
		t.Errorf("error should mention missing output field, got: %v", err)
	}
}

func TestLoadTestSuite_InvalidJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "broken")
	os.MkdirAll(suiteDir, 0755)

	// Create valid test file
	os.WriteFile(filepath.Join(suiteDir, "valid.json"), []byte(`{"input": {}, "output": 1}`), 0644)
	// Create invalid test file
	os.WriteFile(filepath.Join(suiteDir, "invalid.json"), []byte("{broken json}"), 0644)

	_, err := LoadTestSuite(tmpDir, "broken")
	if err == nil {
		t.Error("LoadTestSuite() expected error when a test case has invalid JSON")
	}
}

func TestLoadAllSuites_MissingTestsDir_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create the tests directory

	suites, err := LoadAllSuites(tmpDir)
	if err != nil {
		t.Errorf("LoadAllSuites() unexpected error: %v", err)
	}
	if suites == nil {
		t.Error("LoadAllSuites() should return empty map, not nil")
	}
	if len(suites) != 0 {
		t.Errorf("LoadAllSuites() should return empty map, got %d suites", len(suites))
	}
}

func TestLoadAllSuites_InvalidTestCase_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "broken")
	os.MkdirAll(suiteDir, 0755)

	// Create an invalid JSON file
	os.WriteFile(filepath.Join(suiteDir, "bad.json"), []byte("{invalid}"), 0644)

	_, err := LoadAllSuites(tmpDir)
	if err == nil {
		t.Error("LoadAllSuites() expected error when a test case has invalid JSON")
	}
}

func TestLoadAllSuites_EmptySuite_Skipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty suite directory
	os.MkdirAll(filepath.Join(tmpDir, "tests", "empty"), 0755)

	// Create suite with tests
	suiteDir := filepath.Join(tmpDir, "tests", "hasTests")
	os.MkdirAll(suiteDir, 0755)
	os.WriteFile(filepath.Join(suiteDir, "test.json"), []byte(`{"input": {}, "output": 1}`), 0644)

	suites, err := LoadAllSuites(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllSuites() error = %v", err)
	}

	// Empty suite should not be in the result
	if _, exists := suites["empty"]; exists {
		t.Error("empty suite should not be included in results")
	}

	// Suite with tests should be included
	if _, exists := suites["hasTests"]; !exists {
		t.Error("suite with tests should be included in results")
	}
}
