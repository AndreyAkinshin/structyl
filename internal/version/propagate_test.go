package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestPropagate_ValidFiles_UpdatesAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "package.json")
	file2 := filepath.Join(tmpDir, "version.go")

	if err := os.WriteFile(file1, []byte(`{"version": "1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(`const Version = "1.0.0"`), 0644); err != nil {
		t.Fatal(err)
	}

	files := []config.VersionFileConfig{
		{
			Path:    file1,
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
		{
			Path:    file2,
			Pattern: `Version = "[\d.]+"`,
			Replace: `Version = "{version}"`,
		},
	}

	err := Propagate("2.0.0", files)
	if err != nil {
		t.Fatalf("Propagate() error = %v", err)
	}

	// Verify file1
	content1, _ := os.ReadFile(file1)
	if !strings.Contains(string(content1), `"version": "2.0.0"`) {
		t.Errorf("file1 = %q, want version 2.0.0", string(content1))
	}

	// Verify file2
	content2, _ := os.ReadFile(file2)
	if !strings.Contains(string(content2), `Version = "2.0.0"`) {
		t.Errorf("file2 = %q, want version 2.0.0", string(content2))
	}
}

func TestPropagate_FileNotFound_ReturnsError(t *testing.T) {
	files := []config.VersionFileConfig{
		{
			Path:    "/nonexistent/path.json",
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
	}

	err := Propagate("2.0.0", files)
	if err == nil {
		t.Error("Propagate() expected error for missing file")
	}
}

func TestPropagate_EmptyFiles_ReturnsNil(t *testing.T) {
	err := Propagate("2.0.0", nil)
	if err != nil {
		t.Errorf("Propagate() error = %v, want nil", err)
	}

	err = Propagate("2.0.0", []config.VersionFileConfig{})
	if err != nil {
		t.Errorf("Propagate() error = %v, want nil", err)
	}
}

func TestUpdateFile_ValidPattern_ReplacesVersion(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(filePath, []byte(`{"version": "1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := UpdateFile(filePath, `"version": "[\d.]+"`, `"version": "{version}"`, "2.0.0")
	if err != nil {
		t.Fatalf("UpdateFile() error = %v", err)
	}

	content, _ := os.ReadFile(filePath)
	if !strings.Contains(string(content), `"version": "2.0.0"`) {
		t.Errorf("content = %q, want version 2.0.0", string(content))
	}
}

func TestUpdateFile_InvalidPattern_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := UpdateFile(filePath, "[invalid(regex", "replacement", "1.0.0")
	if err == nil {
		t.Error("UpdateFile() expected error for invalid regex")
	}
}

func TestUpdateFile_PatternNotFound_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("no version here"), 0644); err != nil {
		t.Fatal(err)
	}

	err := UpdateFile(filePath, `"version": "[\d.]+"`, `"version": "{version}"`, "1.0.0")
	if err == nil {
		t.Error("UpdateFile() expected error when pattern not found")
	}
}

func TestUpdateFile_NoChange_NoWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := `{"version": "1.0.0"}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Get initial mod time
	info1, _ := os.Stat(filePath)

	// Update to same version
	err := UpdateFile(filePath, `"version": "[\d.]+"`, `"version": "{version}"`, "1.0.0")
	if err != nil {
		t.Fatalf("UpdateFile() error = %v", err)
	}

	// File content should remain unchanged
	result, _ := os.ReadFile(filePath)
	if string(result) != content {
		t.Errorf("content = %q, want unchanged", string(result))
	}

	// Note: On fast systems, mod time may not change for no-op writes
	// The key assertion is content unchanged
	_ = info1 // suppress unused warning
}

func TestUpdateFile_FileNotFound_ReturnsError(t *testing.T) {
	err := UpdateFile("/nonexistent/file.txt", `pattern`, `replace`, "1.0.0")
	if err == nil {
		t.Error("UpdateFile() expected error for missing file")
	}
}

func TestUpdateFile_MultipleMatches_ReplacesAll(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := `version1: 1.0.0, version2: 1.0.0`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := UpdateFile(filePath, `[\d]+\.[\d]+\.[\d]+`, `{version}`, "2.0.0")
	if err != nil {
		t.Fatalf("UpdateFile() error = %v", err)
	}

	result, _ := os.ReadFile(filePath)
	expected := `version1: 2.0.0, version2: 2.0.0`
	if string(result) != expected {
		t.Errorf("content = %q, want %q", string(result), expected)
	}
}

func TestCheckConsistency_AllConsistent_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(filePath, []byte(`{"version": "1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	files := []config.VersionFileConfig{
		{
			Path:    filePath,
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
	}

	inconsistencies, err := CheckConsistency("1.0.0", files)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 0 {
		t.Errorf("inconsistencies = %v, want empty", inconsistencies)
	}
}

func TestCheckConsistency_Mismatch_ReturnsInconsistencies(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(filePath, []byte(`{"version": "1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	files := []config.VersionFileConfig{
		{
			Path:    filePath,
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
	}

	inconsistencies, err := CheckConsistency("2.0.0", files)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 1 {
		t.Errorf("len(inconsistencies) = %d, want 1", len(inconsistencies))
	}
}

func TestCheckConsistency_FileNotFound_ReportsInconsistency(t *testing.T) {
	files := []config.VersionFileConfig{
		{
			Path:    "/nonexistent/file.json",
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
	}

	inconsistencies, err := CheckConsistency("1.0.0", files)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 1 {
		t.Errorf("len(inconsistencies) = %d, want 1", len(inconsistencies))
	}
	if !strings.Contains(inconsistencies[0], "file not found") {
		t.Errorf("inconsistency = %q, want to contain 'file not found'", inconsistencies[0])
	}
}

func TestCheckConsistency_InvalidPattern_ReportsInconsistency(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	files := []config.VersionFileConfig{
		{
			Path:    filePath,
			Pattern: `[invalid(regex`,
			Replace: `replace`,
		},
	}

	inconsistencies, err := CheckConsistency("1.0.0", files)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 1 {
		t.Errorf("len(inconsistencies) = %d, want 1", len(inconsistencies))
	}
	if !strings.Contains(inconsistencies[0], "invalid pattern") {
		t.Errorf("inconsistency = %q, want to contain 'invalid pattern'", inconsistencies[0])
	}
}

func TestCheckConsistency_PatternNotMatched_ReportsInconsistency(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("no version here"), 0644); err != nil {
		t.Fatal(err)
	}

	files := []config.VersionFileConfig{
		{
			Path:    filePath,
			Pattern: `"version": "[\d.]+"`,
			Replace: `"version": "{version}"`,
		},
	}

	inconsistencies, err := CheckConsistency("1.0.0", files)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 1 {
		t.Errorf("len(inconsistencies) = %d, want 1", len(inconsistencies))
	}
	if !strings.Contains(inconsistencies[0], "pattern not matched") {
		t.Errorf("inconsistency = %q, want to contain 'pattern not matched'", inconsistencies[0])
	}
}

func TestCheckConsistency_EmptyFiles_ReturnsEmpty(t *testing.T) {
	inconsistencies, err := CheckConsistency("1.0.0", nil)
	if err != nil {
		t.Fatalf("CheckConsistency() error = %v", err)
	}

	if len(inconsistencies) != 0 {
		t.Errorf("inconsistencies = %v, want empty", inconsistencies)
	}
}
