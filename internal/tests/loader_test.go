package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTestSuite_ValidSuite_ReturnsTestCases(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "suite1")
	if err := os.MkdirAll(suiteDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	test1 := `{"input": {"a": 1}, "output": 2}`
	test2 := `{"input": {"b": 2}, "output": 4}`
	if err := os.WriteFile(filepath.Join(suiteDir, "add.json"), []byte(test1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(suiteDir, "double.json"), []byte(test2), 0644); err != nil {
		t.Fatal(err)
	}

	cases, err := LoadTestSuite(tmpDir, "suite1", "*.json")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	if len(cases) != 2 {
		t.Errorf("len(cases) = %d, want 2", len(cases))
	}

	// Verify sorted order (add before double)
	if len(cases) >= 2 {
		if cases[0].Name != "add" {
			t.Errorf("cases[0].Name = %q, want %q", cases[0].Name, "add")
		}
		if cases[1].Name != "double" {
			t.Errorf("cases[1].Name = %q, want %q", cases[1].Name, "double")
		}
	}

	// Verify suite is set
	for _, tc := range cases {
		if tc.Suite != "suite1" {
			t.Errorf("Suite = %q, want %q", tc.Suite, "suite1")
		}
	}
}

func TestLoadTestSuite_NonExistentDir_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadTestSuite(tmpDir, "nonexistent", "*.json")
	if err == nil {
		t.Error("LoadTestSuite() expected error for non-existent suite")
	}
}

func TestLoadTestSuite_EmptySuite_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(suiteDir, 0755); err != nil {
		t.Fatal(err)
	}

	cases, err := LoadTestSuite(tmpDir, "empty", "*.json")
	if err != nil {
		t.Fatalf("LoadTestSuite() error = %v", err)
	}

	if len(cases) != 0 {
		t.Errorf("len(cases) = %d, want 0", len(cases))
	}
}

func TestLoadTestCase_ValidJSON_ParsesCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{
		"input": {"x": 1, "y": 2},
		"output": {"sum": 3}
	}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tc, err := LoadTestCase(testFile)
	if err != nil {
		t.Fatalf("LoadTestCase() error = %v", err)
	}

	if tc.Name != "test" {
		t.Errorf("Name = %q, want %q", tc.Name, "test")
	}
	if tc.Path != testFile {
		t.Errorf("Path = %q, want %q", tc.Path, testFile)
	}
	if tc.Input["x"] != float64(1) {
		t.Errorf("Input[x] = %v, want 1", tc.Input["x"])
	}
	if tc.Input["y"] != float64(2) {
		t.Errorf("Input[y] = %v, want 2", tc.Input["y"])
	}

	outputMap, ok := tc.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Output type = %T, want map[string]interface{}", tc.Output)
	}
	if outputMap["sum"] != float64(3) {
		t.Errorf("Output[sum] = %v, want 3", outputMap["sum"])
	}
}

func TestLoadTestCase_MissingInput_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{"output": 42}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for missing input")
	}
}

func TestLoadTestCase_MissingOutput_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{"input": {"x": 1}}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error for missing output")
	}
}

func TestLoadTestCase_InputNotObject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{"input": [1, 2, 3], "output": 6}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTestCase(testFile)
	if err == nil {
		t.Error("LoadTestCase() expected error when input is not an object")
	}
}

func TestLoadTestCase_InvalidJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	content := `{invalid json}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
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

func TestResolveFileRefs_NestedMap_ResolvesRecursively(t *testing.T) {
	tmpDir := t.TempDir()

	// Create referenced file
	refFile := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(refFile, []byte(`{"value": 42}`), 0644); err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": map[string]interface{}{
				"$file": "data.json",
			},
		},
	}

	result, err := resolveFileRefs(input, tmpDir)
	if err != nil {
		t.Fatalf("resolveFileRefs() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	outer := resultMap["outer"].(map[string]interface{})
	inner := outer["inner"].(map[string]interface{})
	if inner["value"] != float64(42) {
		t.Errorf("inner[value] = %v, want 42", inner["value"])
	}
}

func TestResolveFileRefs_Array_ResolvesElements(t *testing.T) {
	tmpDir := t.TempDir()

	// Create referenced file
	refFile := filepath.Join(tmpDir, "item.json")
	if err := os.WriteFile(refFile, []byte(`"resolved"`), 0644); err != nil {
		t.Fatal(err)
	}

	input := []interface{}{
		"static",
		map[string]interface{}{"$file": "item.json"},
	}

	result, err := resolveFileRefs(input, tmpDir)
	if err != nil {
		t.Fatalf("resolveFileRefs() error = %v", err)
	}

	arr := result.([]interface{})
	if arr[0] != "static" {
		t.Errorf("arr[0] = %v, want static", arr[0])
	}
	if arr[1] != "resolved" {
		t.Errorf("arr[1] = %v, want resolved", arr[1])
	}
}

func TestResolveFileRefs_Primitive_ReturnsUnchanged(t *testing.T) {
	tests := []interface{}{
		"string",
		float64(42),
		true,
		nil,
	}

	for _, input := range tests {
		result, err := resolveFileRefs(input, "/tmp")
		if err != nil {
			t.Errorf("resolveFileRefs(%v) error = %v", input, err)
		}
		if result != input {
			t.Errorf("resolveFileRefs(%v) = %v, want unchanged", input, result)
		}
	}
}

func TestLoadFileRef_PathTraversal_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := loadFileRef("../escape.json", tmpDir)
	if err == nil {
		t.Error("loadFileRef() expected error for path traversal")
	}
}

func TestLoadFileRef_EscapesBaseDir_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Even without "..", try to escape via symlink-like path
	// This test verifies the absolute path check
	_, err := loadFileRef("../escape.txt", subDir)
	if err == nil {
		t.Error("loadFileRef() expected error for path escaping base directory")
	}
}

func TestLoadFileRef_ValidJSONFile_ParsesAsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	refFile := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(refFile, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := loadFileRef("data.json", tmpDir)
	if err != nil {
		t.Fatalf("loadFileRef() error = %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result)
	}
	if resultMap["key"] != "value" {
		t.Errorf("result[key] = %v, want value", resultMap["key"])
	}
}

func TestLoadFileRef_NonJSONFile_ReturnsString(t *testing.T) {
	tmpDir := t.TempDir()
	refFile := filepath.Join(tmpDir, "data.txt")
	if err := os.WriteFile(refFile, []byte("plain text content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := loadFileRef("data.txt", tmpDir)
	if err != nil {
		t.Fatalf("loadFileRef() error = %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("result type = %T, want string", result)
	}
	if str != "plain text content" {
		t.Errorf("result = %q, want %q", str, "plain text content")
	}
}

func TestLoadFileRef_FileNotFound_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := loadFileRef("nonexistent.json", tmpDir)
	if err == nil {
		t.Error("loadFileRef() expected error for missing file")
	}
}

func TestFindMatches_GlobPattern_FindsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "test1.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test2.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	matches, err := findMatches(tmpDir, "*.json")
	if err != nil {
		t.Fatalf("findMatches() error = %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("len(matches) = %d, want 2", len(matches))
	}

	// Should be sorted
	for i, m := range matches {
		if !filepath.IsAbs(m) {
			t.Errorf("matches[%d] = %q, expected absolute path", i, m)
		}
	}
}

func TestFindMatches_NoMatches_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-matching file
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	matches, err := findMatches(tmpDir, "*.json")
	if err != nil {
		t.Fatalf("findMatches() error = %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("len(matches) = %d, want 0", len(matches))
	}
}

func TestLoadAllSuites_MultipleSuites_ReturnsAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create suite1
	suite1Dir := filepath.Join(tmpDir, "suite1")
	if err := os.MkdirAll(suite1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(suite1Dir, "test.json"), []byte(`{"input": {}, "output": 1}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create suite2
	suite2Dir := filepath.Join(tmpDir, "suite2")
	if err := os.MkdirAll(suite2Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(suite2Dir, "test.json"), []byte(`{"input": {}, "output": 2}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file (not dir) that should be skipped
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	suites, err := LoadAllSuites(tmpDir, "*.json")
	if err != nil {
		t.Fatalf("LoadAllSuites() error = %v", err)
	}

	if len(suites) != 2 {
		t.Errorf("len(suites) = %d, want 2", len(suites))
	}

	if _, ok := suites["suite1"]; !ok {
		t.Error("suites missing 'suite1'")
	}
	if _, ok := suites["suite2"]; !ok {
		t.Error("suites missing 'suite2'")
	}
}

func TestLoadAllSuites_EmptyDir_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	suites, err := LoadAllSuites(tmpDir, "*.json")
	if err != nil {
		t.Fatalf("LoadAllSuites() error = %v", err)
	}

	if len(suites) != 0 {
		t.Errorf("len(suites) = %d, want 0", len(suites))
	}
}

func TestLoadAllSuites_EmptySuitesSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create suite with no JSON files
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create suite with JSON files
	withFilesDir := filepath.Join(tmpDir, "withfiles")
	if err := os.MkdirAll(withFilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withFilesDir, "test.json"), []byte(`{"input": {}, "output": 1}`), 0644); err != nil {
		t.Fatal(err)
	}

	suites, err := LoadAllSuites(tmpDir, "*.json")
	if err != nil {
		t.Fatalf("LoadAllSuites() error = %v", err)
	}

	// Only "withfiles" should be included since "empty" has no tests
	if len(suites) != 1 {
		t.Errorf("len(suites) = %d, want 1", len(suites))
	}
	if _, ok := suites["withfiles"]; !ok {
		t.Error("suites missing 'withfiles'")
	}
}

func TestFindMatches_DoubleStarPattern_FindsNestedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	subDir := filepath.Join(tmpDir, "nested", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files at different levels
	if err := os.WriteFile(filepath.Join(tmpDir, "root.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "nested", "mid.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "deep.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "other.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with ** pattern
	matches, err := findMatches(tmpDir, "**/*.json")
	if err != nil {
		t.Fatalf("findMatches() error = %v", err)
	}

	// Should find all 3 JSON files
	if len(matches) != 3 {
		t.Errorf("len(matches) = %d, want 3", len(matches))
	}

	// Should not include .txt files
	for _, m := range matches {
		if filepath.Ext(m) != ".json" {
			t.Errorf("matches contains non-json file: %q", m)
		}
	}
}
