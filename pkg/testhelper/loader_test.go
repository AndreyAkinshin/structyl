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

func TestSuiteExistsErr(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "tests", "exists"), 0755)

	// Existing suite returns (true, nil)
	exists, err := SuiteExistsErr(tmpDir, "exists")
	if err != nil {
		t.Errorf("SuiteExistsErr() unexpected error: %v", err)
	}
	if !exists {
		t.Error("SuiteExistsErr should return true for existing suite")
	}

	// Non-existing suite returns (false, nil)
	exists, err = SuiteExistsErr(tmpDir, "notexists")
	if err != nil {
		t.Errorf("SuiteExistsErr() unexpected error: %v", err)
	}
	if exists {
		t.Error("SuiteExistsErr should return false for non-existing suite")
	}
}

func TestSuiteExistsErr_FileNotDir(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	os.MkdirAll(testsDir, 0755)
	// Create a file instead of directory
	os.WriteFile(filepath.Join(testsDir, "notadir"), []byte{}, 0644)

	// File (not dir) returns (false, nil)
	exists, err := SuiteExistsErr(tmpDir, "notadir")
	if err != nil {
		t.Errorf("SuiteExistsErr() unexpected error: %v", err)
	}
	if exists {
		t.Error("SuiteExistsErr should return false when path exists but is not a directory")
	}
}

func TestTestCaseExistsErr(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "tests", "suite"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "tests", "suite", "test.json"), []byte(`{}`), 0644)

	// Existing test returns (true, nil)
	exists, err := TestCaseExistsErr(tmpDir, "suite", "test")
	if err != nil {
		t.Errorf("TestCaseExistsErr() unexpected error: %v", err)
	}
	if !exists {
		t.Error("TestCaseExistsErr should return true for existing test")
	}

	// Non-existing test returns (false, nil)
	exists, err = TestCaseExistsErr(tmpDir, "suite", "notexists")
	if err != nil {
		t.Errorf("TestCaseExistsErr() unexpected error: %v", err)
	}
	if exists {
		t.Error("TestCaseExistsErr should return false for non-existing test")
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

func TestTestCase_HasSuite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tc   TestCase
		want bool
	}{
		{
			name: "empty_suite",
			tc:   TestCase{Name: "test1", Suite: ""},
			want: false,
		},
		{
			name: "suite_set",
			tc:   TestCase{Name: "test1", Suite: "math"},
			want: true,
		},
		{
			name: "whitespace_suite",
			tc:   TestCase{Name: "test1", Suite: " "},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.tc.HasSuite(); got != tt.want {
				t.Errorf("HasSuite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestCase_TagsContain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tc   TestCase
		tag  string
		want bool
	}{
		{
			name: "nil_tags",
			tc:   TestCase{Name: "test1", Tags: nil},
			tag:  "slow",
			want: false,
		},
		{
			name: "empty_tags",
			tc:   TestCase{Name: "test1", Tags: []string{}},
			tag:  "slow",
			want: false,
		},
		{
			name: "tag_found",
			tc:   TestCase{Name: "test1", Tags: []string{"slow", "integration"}},
			tag:  "slow",
			want: true,
		},
		{
			name: "tag_not_found",
			tc:   TestCase{Name: "test1", Tags: []string{"slow", "integration"}},
			tag:  "fast",
			want: false,
		},
		{
			name: "case_sensitive",
			tc:   TestCase{Name: "test1", Tags: []string{"Slow", "integration"}},
			tag:  "slow",
			want: false,
		},
		{
			name: "empty_tag_search",
			tc:   TestCase{Name: "test1", Tags: []string{"slow", ""}},
			tag:  "",
			want: true,
		},
		{
			name: "empty_tag_search_not_present",
			tc:   TestCase{Name: "test1", Tags: []string{"slow"}},
			tag:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.tc.TagsContain(tt.tag); got != tt.want {
				t.Errorf("TagsContain(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestTestCase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tc   TestCase
		want string
	}{
		{
			name: "name only",
			tc: TestCase{
				Name: "basic",
			},
			want: "TestCase{basic}",
		},
		{
			name: "with suite",
			tc: TestCase{
				Name:  "addition",
				Suite: "math",
			},
			want: "TestCase{math/addition}",
		},
		{
			name: "with skip flag",
			tc: TestCase{
				Name: "skipped_test",
				Skip: true,
			},
			want: "TestCase{skipped_test [SKIP]}",
		},
		{
			name: "with suite and skip",
			tc: TestCase{
				Name:  "broken",
				Suite: "integration",
				Skip:  true,
			},
			want: "TestCase{integration/broken [SKIP]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.tc.String()
			if result != tt.want {
				t.Errorf("String() = %q, want %q", result, tt.want)
			}
		})
	}
}

// withWorkingDir changes to the specified directory for the duration of the
// function call, then restores the original working directory.
//
// NOTE: Tests using this helper MUST NOT call t.Parallel() because os.Chdir
// affects the entire process. Parallel tests would race on the working directory.
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

	// Verify error includes filename for debugging context
	if err != nil && !strings.Contains(err.Error(), "invalid.json") {
		t.Errorf("error should contain filename, got: %v", err)
	}
}

func TestLoadTestCase_EmptyFilename_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file named ".json" which results in empty name after suffix trim
	testFile := filepath.Join(tmpDir, ".json")

	validJSON := `{"input": {}, "output": {}}`
	if err := os.WriteFile(testFile, []byte(validJSON), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for empty filename")
	}

	if err != nil && !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("error should mention empty name, got: %v", err)
	}
}

func TestLoadTestCase_FileNotFound_ReturnsTestCaseNotFoundError(t *testing.T) {
	_, err := LoadTestCase("/nonexistent/path/test.json")
	if err == nil {
		t.Fatal("LoadTestCase() expected error for missing file")
	}

	// Check error type
	var tcnfErr *TestCaseNotFoundError
	if !errors.As(err, &tcnfErr) {
		t.Errorf("error type = %T, want *TestCaseNotFoundError", err)
	}

	// Check errors.Is with sentinel
	if !errors.Is(err, ErrTestCaseNotFound) {
		t.Error("errors.Is(err, ErrTestCaseNotFound) should return true")
	}

	// Check path in error
	if tcnfErr != nil && tcnfErr.Path != "/nonexistent/path/test.json" {
		t.Errorf("TestCaseNotFoundError.Path = %q, want %q", tcnfErr.Path, "/nonexistent/path/test.json")
	}
}

func TestTestCaseNotFoundError(t *testing.T) {
	err := &TestCaseNotFoundError{Path: "/some/path/test.json"}

	if err.Error() == "" {
		t.Error("Error() should return message")
	}

	if !strings.Contains(err.Error(), "/some/path/test.json") {
		t.Error("Error() should contain path")
	}

	if !errors.Is(err, ErrTestCaseNotFound) {
		t.Error("errors.Is(TestCaseNotFoundError, ErrTestCaseNotFound) should return true")
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
	// Error message should specifically indicate the field is missing (not null)
	if err != nil && !strings.Contains(err.Error(), "missing required field \"output\"") {
		t.Errorf("error should mention missing output field, got: %v", err)
	}
}

func TestLoadTestCase_NullOutput_ReturnsError(t *testing.T) {
	// JSON null is NOT a valid output value. The spec requires an explicit
	// value (empty string, empty object, etc.) for expected empty output.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "null_output.json")

	content := `{"input": {"x": 1}, "output": null}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for null output")
	}
	// Error message should specifically indicate null is not valid (distinct from missing)
	if err != nil && !strings.Contains(err.Error(), "\"output\" field is null") {
		t.Errorf("error should mention output field is null, got: %v", err)
	}
}

func TestLoadTestCase_ArrayInput_ReturnsError(t *testing.T) {
	// Input MUST be a JSON object, not an array. Arrays silently unmarshal
	// to nil when target is map[string]interface{}, so validation must check
	// the raw JSON to provide a clear error message.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "array_input.json")

	content := `{"input": [1, 2, 3], "output": 6}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for array input")
	}
	if err != nil && !strings.Contains(err.Error(), "must be an object, not an array") {
		t.Errorf("error should mention input must be an object, got: %v", err)
	}
}

func TestLoadTestCase_ScalarInput_ReturnsError(t *testing.T) {
	// Input MUST be a JSON object, not a scalar value.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "scalar_input.json")

	content := `{"input": 42, "output": 42}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for scalar input")
	}
	if err != nil && !strings.Contains(err.Error(), "must be an object, not a scalar") {
		t.Errorf("error should mention input must be an object, got: %v", err)
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

func TestLoadTestSuite_DeterministicOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "tests", "ordering")
	os.MkdirAll(suiteDir, 0755)

	// Create files in non-alphabetical order to test sorting
	os.WriteFile(filepath.Join(suiteDir, "zebra.json"), []byte(`{"input": {}, "output": "z"}`), 0644)
	os.WriteFile(filepath.Join(suiteDir, "alpha.json"), []byte(`{"input": {}, "output": "a"}`), 0644)
	os.WriteFile(filepath.Join(suiteDir, "middle.json"), []byte(`{"input": {}, "output": "m"}`), 0644)
	os.WriteFile(filepath.Join(suiteDir, "beta.json"), []byte(`{"input": {}, "output": "b"}`), 0644)

	cases, err := LoadTestSuite(tmpDir, "ordering")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	if len(cases) != 4 {
		t.Fatalf("expected 4 test cases, got %d", len(cases))
	}

	// Verify alphabetical ordering by name
	expectedOrder := []string{"alpha", "beta", "middle", "zebra"}
	for i, tc := range cases {
		if tc.Name != expectedOrder[i] {
			t.Errorf("cases[%d].Name = %q, want %q", i, tc.Name, expectedOrder[i])
		}
	}
}

func TestLoadTestCase_FileReference_ReturnsError(t *testing.T) {
	// $file references are not supported in pkg/testhelper (internal feature only).
	// This test verifies the error is returned correctly.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file_ref.json")

	content := `{"input": {"data": {"$file": "binary.bin"}}, "output": 1}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Fatal("LoadTestCase() expected error for $file reference")
	}

	// Verify it's the correct error type
	if !errors.Is(err, ErrFileReferenceNotSupported) {
		t.Errorf("error should be ErrFileReferenceNotSupported, got: %v", err)
	}

	// Verify error message contains the filename for context
	if !strings.Contains(err.Error(), "file_ref.json") {
		t.Errorf("error message should contain filename, got: %v", err)
	}
}

func TestLoadTestCase_FileReferenceNested(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "file_ref_in_array",
			content: `{"input": {"items": [{"$file": "data.bin"}]}, "output": 1}`,
		},
		{
			name:    "file_ref_deeply_nested",
			content: `{"input": {"level1": {"level2": {"level3": {"$file": "deep.bin"}}}}, "output": 1}`,
		},
		{
			name:    "file_ref_in_output_array",
			content: `{"input": {}, "output": [{"$file": "result.bin"}]}`,
		},
		{
			name:    "file_ref_in_output_nested",
			content: `{"input": {}, "output": {"data": {"nested": {"$file": "out.bin"}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, tt.name+".json")

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := LoadTestCase(testFile)
			if err == nil {
				t.Fatal("LoadTestCase() expected error for nested $file reference")
			}

			if !errors.Is(err, ErrFileReferenceNotSupported) {
				t.Errorf("error should be ErrFileReferenceNotSupported, got: %v", err)
			}
		})
	}
}

func TestValidateSuiteName(t *testing.T) {
	tests := []struct {
		name    string
		suite   string
		wantErr error
	}{
		{"valid_name", "math", nil},
		{"valid_with_hyphen", "math-advanced", nil},
		{"valid_with_underscore", "math_basic", nil},
		{"empty_string", "", ErrEmptySuiteName},
		// Path traversal cases
		{"path_traversal_parent", "..", ErrInvalidSuiteName},
		{"path_traversal_prefix", "../foo", ErrInvalidSuiteName},
		{"path_traversal_suffix", "foo/..", ErrInvalidSuiteName},
		{"path_traversal_middle", "foo/../bar", ErrInvalidSuiteName},
		// Path separator cases
		{"forward_slash", "foo/bar", ErrInvalidSuiteName},
		{"backslash", "foo\\bar", ErrInvalidSuiteName},
		{"forward_slash_only", "/", ErrInvalidSuiteName},
		{"backslash_only", "\\", ErrInvalidSuiteName},
		// Null byte case
		{"null_byte", "foo\x00bar", ErrInvalidSuiteName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSuiteName(tt.suite)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateSuiteName(%q) = %v, want nil", tt.suite, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateSuiteName(%q) = %v, want %v", tt.suite, err, tt.wantErr)
				}
			}
		})
	}
}

func TestLoadTestCaseWithSuite_EmptySuite_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test1.json")
	content := `{"input": {"a": 1}, "output": 1}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCaseWithSuite(testFile, "")
	if err == nil {
		t.Fatal("LoadTestCaseWithSuite() expected error for empty suite")
	}

	if !errors.Is(err, ErrEmptySuiteName) {
		t.Errorf("error should be ErrEmptySuiteName, got: %v", err)
	}
}

func TestTestCase_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tc      TestCase
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			tc: TestCase{
				Name:   "test1",
				Input:  map[string]interface{}{"a": 1},
				Output: 42,
			},
			wantErr: false,
		},
		{
			name: "valid_empty_input",
			tc: TestCase{
				Name:   "test2",
				Input:  map[string]interface{}{},
				Output: "result",
			},
			wantErr: false,
		},
		{
			name: "empty_name",
			tc: TestCase{
				Name:   "",
				Input:  map[string]interface{}{"a": 1},
				Output: 42,
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "nil_input",
			tc: TestCase{
				Name:   "test3",
				Input:  nil,
				Output: 42,
			},
			wantErr: true,
			errMsg:  "input",
		},
		{
			name: "nil_output",
			tc: TestCase{
				Name:   "test4",
				Input:  map[string]interface{}{"a": 1},
				Output: nil,
			},
			wantErr: true,
			errMsg:  "output",
		},
		{
			name: "both_nil",
			tc: TestCase{
				Name:   "test5",
				Input:  nil,
				Output: nil,
			},
			wantErr: true,
			errMsg:  "input",
		},
		{
			name: "empty_name_takes_priority",
			tc: TestCase{
				Name:   "",
				Input:  nil,
				Output: nil,
			},
			wantErr: true,
			errMsg:  "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tc.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() = nil, want error containing %q", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
			}
		})
	}
}
