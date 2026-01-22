package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRootFrom_Found(t *testing.T) {
	// Create temp project structure
	root := t.TempDir()
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"name":"test"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Test from root
	found, err := FindRootFrom(root)
	if err != nil {
		t.Fatalf("FindRootFrom() error = %v", err)
	}
	if found != root {
		t.Errorf("FindRootFrom() = %q, want %q", found, root)
	}
}

func TestFindRootFrom_FoundFromSubdir(t *testing.T) {
	root := t.TempDir()
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"name":"test"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested subdirectory
	subdir := filepath.Join(root, "src", "module", "deep")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test from subdirectory
	found, err := FindRootFrom(subdir)
	if err != nil {
		t.Fatalf("FindRootFrom() error = %v", err)
	}
	if found != root {
		t.Errorf("FindRootFrom() = %q, want %q", found, root)
	}
}

func TestFindRootFrom_NotFound(t *testing.T) {
	// Create temp dir without config.json
	dir := t.TempDir()

	_, err := FindRootFrom(dir)
	if err != ErrNoProjectRoot {
		t.Errorf("FindRootFrom() error = %v, want ErrNoProjectRoot", err)
	}
}

func TestLoadProjectFrom_Minimal(t *testing.T) {
	root := t.TempDir()
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(structylDir, "config.json")
	config := `{"project":{"name":"myproject"}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := LoadProjectFrom(root)
	if err != nil {
		t.Fatalf("LoadProjectFrom() error = %v", err)
	}
	if proj.Root != root {
		t.Errorf("Project.Root = %q, want %q", proj.Root, root)
	}
	if proj.Config.Project.Name != "myproject" {
		t.Errorf("Project.Config.Project.Name = %q, want %q", proj.Config.Project.Name, "myproject")
	}
}

func TestLoadProjectFrom_WithTargets(t *testing.T) {
	root := t.TempDir()

	// Create target directory
	csDir := filepath.Join(root, "cs")
	if err := os.MkdirAll(csDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#"}
		}
	}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := LoadProjectFrom(root)
	if err != nil {
		t.Fatalf("LoadProjectFrom() error = %v", err)
	}
	if len(proj.Config.Targets) != 1 {
		t.Errorf("len(Config.Targets) = %d, want 1", len(proj.Config.Targets))
	}
}

func TestLoadProjectFrom_MissingTargetDir(t *testing.T) {
	root := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Target directory does NOT exist
	config := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#"}
		}
	}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectFrom(root)
	if err == nil {
		t.Fatal("LoadProjectFrom() expected error for missing target directory")
	}
}

func TestProject_ConfigPath(t *testing.T) {
	root := "/project/root"
	proj := &Project{Root: root}
	expected := filepath.Join(root, ".structyl", "config.json")
	if got := proj.ConfigPath(); got != expected {
		t.Errorf("ConfigPath() = %q, want %q", got, expected)
	}
}

func TestProject_TargetDirectory_Found(t *testing.T) {
	root := t.TempDir()

	// Create target directory
	csDir := filepath.Join(root, "cs")
	if err := os.MkdirAll(csDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(structylDir, "config.json")
	config := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#", "directory": "cs"}
		}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := LoadProjectFrom(root)
	if err != nil {
		t.Fatalf("LoadProjectFrom() error = %v", err)
	}

	targetDir, err := proj.TargetDirectory("cs")
	if err != nil {
		t.Fatalf("TargetDirectory() error = %v", err)
	}

	expected := filepath.Join(root, "cs")
	if targetDir != expected {
		t.Errorf("TargetDirectory() = %q, want %q", targetDir, expected)
	}
}

func TestProject_TargetDirectory_NotFound(t *testing.T) {
	root := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(structylDir, "config.json")
	config := `{"project": {"name": "myproject"}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := LoadProjectFrom(root)
	if err != nil {
		t.Fatalf("LoadProjectFrom() error = %v", err)
	}

	_, err = proj.TargetDirectory("nonexistent")
	if err == nil {
		t.Error("TargetDirectory() expected error for nonexistent target")
	}
}

func TestProject_TargetDirectory_CustomDirectory(t *testing.T) {
	root := t.TempDir()

	// Create custom directory
	customDir := filepath.Join(root, "src", "csharp")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(structylDir, "config.json")
	config := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#", "directory": "src/csharp"}
		}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := LoadProjectFrom(root)
	if err != nil {
		t.Fatalf("LoadProjectFrom() error = %v", err)
	}

	targetDir, err := proj.TargetDirectory("cs")
	if err != nil {
		t.Fatalf("TargetDirectory() error = %v", err)
	}

	expected := filepath.Join(root, "src", "csharp")
	if targetDir != expected {
		t.Errorf("TargetDirectory() = %q, want %q", targetDir, expected)
	}
}

func TestFindRoot_FromProjectRoot(t *testing.T) {
	// Create temp project structure
	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"name":"test"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Change to project root
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	// Restore working directory after test
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	found, err := FindRoot()
	if err != nil {
		t.Fatalf("FindRoot() error = %v", err)
	}
	if found != root {
		t.Errorf("FindRoot() = %q, want %q", found, root)
	}
}

func TestLoadProjectFrom_MalformedToolchains(t *testing.T) {
	root := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create valid config.json
	configPath := filepath.Join(structylDir, "config.json")
	config := `{"project": {"name": "myproject"}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	// Create malformed toolchains.json (invalid JSON)
	toolchainsPath := filepath.Join(structylDir, "toolchains.json")
	if err := os.WriteFile(toolchainsPath, []byte(`{invalid json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectFrom(root)
	if err == nil {
		t.Fatal("LoadProjectFrom() expected error for malformed toolchains.json")
	}
	if !strings.Contains(err.Error(), "failed to load toolchains") {
		t.Errorf("error = %q, want error containing 'failed to load toolchains'", err)
	}
}

func TestLoadProjectFrom_InvalidVersionFile(t *testing.T) {
	root := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create valid config.json
	configPath := filepath.Join(structylDir, "config.json")
	config := `{"project": {"name": "myproject"}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	// Create VERSION file with invalid semver
	versionPath := filepath.Join(structylDir, "PROJECT_VERSION")
	if err := os.WriteFile(versionPath, []byte("not-a-valid-semver"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectFrom(root)
	if err == nil {
		t.Fatal("LoadProjectFrom() expected error for invalid VERSION file")
	}
	if !strings.Contains(err.Error(), "version validation failed") {
		t.Errorf("error = %q, want error containing 'version validation failed'", err)
	}
}

// Note: TestFindRoot_FromSubdirectory, TestFindRoot_NotFound, TestLoadProject_Success,
// and TestLoadProject_NotFound have been removed because they duplicate coverage from
// TestFindRootFrom_* and TestLoadProjectFrom_* tests. These wrapper functions (FindRoot,
// LoadProject) are thin wrappers that add only os.Getwd() behavior.
//
// TestFindRoot_FromProjectRoot is retained to verify the os.Getwd() integration works.
