package docs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akinshin/structyl/internal/config"
	"github.com/akinshin/structyl/internal/target"
)

func TestNewGenerator_ValidInputs_CreatesGenerator(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
	}

	g := NewGenerator(tmpDir, cfg, nil, "1.0.0")

	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}
	if g.projectRoot != tmpDir {
		t.Errorf("projectRoot = %q, want %q", g.projectRoot, tmpDir)
	}
	if g.version != "1.0.0" {
		t.Errorf("version = %q, want %q", g.version, "1.0.0")
	}
}

func TestGenerate_NoDocConfig_ReturnsNil(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Project:       config.ProjectConfig{Name: "test"},
		Documentation: nil,
	}

	g := NewGenerator(tmpDir, cfg, nil, "1.0.0")
	err := g.Generate()

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
}

func TestGenerate_EmptyReadmeTemplate_ReturnsNil(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Documentation: &config.DocsConfig{
			ReadmeTemplate: "",
		},
	}

	g := NewGenerator(tmpDir, cfg, nil, "1.0.0")
	err := g.Generate()

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
}

func TestGenerate_MissingTemplate_ReturnsError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Documentation: &config.DocsConfig{
			ReadmeTemplate: "nonexistent_template.md",
		},
	}

	g := NewGenerator(tmpDir, cfg, nil, "1.0.0")
	err := g.Generate()

	if err == nil {
		t.Error("Generate() expected error for missing template")
	}

	// Should be MissingFileError
	if _, ok := err.(*MissingFileError); !ok {
		t.Errorf("error type = %T, want *MissingFileError", err)
	}
}

func TestGenerate_ValidConfig_GeneratesReadmes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create template
	templatePath := filepath.Join(tmpDir, "README.template.md")
	template := "# $LANG_TITLE$\n\nVersion: $VERSION$"
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Documentation: &config.DocsConfig{
			ReadmeTemplate: "README.template.md",
		},
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Directory: "rs",
			},
		},
	}

	registry, err := target.NewRegistry(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	g := NewGenerator(tmpDir, cfg, registry, "2.0.0")
	err = g.Generate()

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check README was created
	readmePath := filepath.Join(targetDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("README not created: %v", err)
	}

	if string(content) != "# Rust\n\nVersion: 2.0.0" {
		t.Errorf("README content = %q", string(content))
	}
}

func TestGenerateForTarget_UnknownTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create template
	templatePath := filepath.Join(tmpDir, "README.template.md")
	if err := os.WriteFile(templatePath, []byte("template"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Documentation: &config.DocsConfig{
			ReadmeTemplate: "README.template.md",
		},
	}

	// Empty registry
	registry, _ := target.NewRegistry(cfg, tmpDir)

	g := NewGenerator(tmpDir, cfg, registry, "1.0.0")
	_, err := g.GenerateForTarget("nonexistent")

	if err == nil {
		t.Error("GenerateForTarget() expected error for unknown target")
	}
}

func TestGenerateForTarget_NoDocConfig_ReturnsError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Project:       config.ProjectConfig{Name: "test"},
		Documentation: nil,
	}

	g := NewGenerator(tmpDir, cfg, nil, "1.0.0")
	_, err := g.GenerateForTarget("rs")

	if err == nil {
		t.Error("GenerateForTarget() expected error when documentation not configured")
	}
}

func TestGenerateForTarget_ValidTarget_ReturnsReadme(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create template using valid placeholders
	templatePath := filepath.Join(tmpDir, "README.template.md")
	template := "# $LANG_TITLE$\n\nVersion: $VERSION$\nSlug: $LANG_SLUG$"
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "testproject"},
		Documentation: &config.DocsConfig{
			ReadmeTemplate: "README.template.md",
		},
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Directory: "rs",
			},
		},
	}

	registry, err := target.NewRegistry(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	g := NewGenerator(tmpDir, cfg, registry, "3.0.0")
	readme, err := g.GenerateForTarget("rs")

	if err != nil {
		t.Fatalf("GenerateForTarget() error = %v", err)
	}

	// Verify the placeholders were replaced
	expected := "# Rust\n\nVersion: 3.0.0\nSlug: rs"
	if readme != expected {
		t.Errorf("GenerateForTarget() = %q, want %q", readme, expected)
	}
}

func TestWriteReadme_ValidPath_WritesFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	mock := &mockTarget{
		name:      "rs",
		title:     "Rust",
		directory: "rs",
	}

	err := WriteReadme(tmpDir, mock, "# Test README")
	if err != nil {
		t.Fatalf("WriteReadme() error = %v", err)
	}

	// Verify file exists and has correct content
	readmePath := filepath.Join(targetDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if string(content) != "# Test README" {
		t.Errorf("content = %q, want %q", string(content), "# Test README")
	}
}

func TestWriteReadme_InvalidPath_ReturnsError(t *testing.T) {
	t.Parallel()
	mock := &mockTarget{
		name:      "rs",
		title:     "Rust",
		directory: "rs",
	}

	err := WriteReadme("/nonexistent/path", mock, "content")
	if err == nil {
		t.Error("WriteReadme() expected error for invalid path")
	}
}

func TestReadmeExists_Exists_ReturnsTrue(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory and README
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockTarget{
		name:      "rs",
		title:     "Rust",
		directory: "rs",
	}

	if !ReadmeExists(tmpDir, mock) {
		t.Error("ReadmeExists() = false, want true")
	}
}

func TestReadmeExists_NotExists_ReturnsFalse(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create target directory but no README
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	mock := &mockTarget{
		name:      "rs",
		title:     "Rust",
		directory: "rs",
	}

	if ReadmeExists(tmpDir, mock) {
		t.Error("ReadmeExists() = true, want false")
	}
}

func TestMissingFileError_Error(t *testing.T) {
	t.Parallel()
	err := &MissingFileError{
		Path:    "/path/to/file",
		Message: "file not found",
	}

	msg := err.Error()
	if msg != "file not found: /path/to/file" {
		t.Errorf("Error() = %q", msg)
	}
}

func TestMissingFileError_ExitCode(t *testing.T) {
	t.Parallel()
	err := &MissingFileError{}

	if err.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, want 2", err.ExitCode())
	}
}
