package project

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectToolchain is removed - toolchain detection is tested comprehensively
// in internal/toolchain/detect_test.go. DetectToolchain is a thin wrapper around
// toolchain.Detect().

func TestDiscoverTargets(t *testing.T) {
	root := t.TempDir()

	// Create some target directories with marker files
	dirs := map[string]string{
		"rs":  "Cargo.toml",
		"py":  "pyproject.toml",
		"cs":  "MyProject.csproj",
		"go":  "go.mod",
		"lib": "", // No marker - should not be discovered
	}

	for name, marker := range dirs {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if marker != "" {
			path := filepath.Join(dir, marker)
			if err := os.WriteFile(path, []byte{}, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Add excluded directories
	excludedDirs := []string{"node_modules", "vendor", ".git"}
	for _, name := range excludedDirs {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// Add a marker file that would normally match
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	targets, err := DiscoverTargets(root)
	if err != nil {
		t.Fatalf("DiscoverTargets() error = %v", err)
	}

	// Check expected targets
	expected := map[string]string{
		"rs": "cargo",
		"py": "python",
		"cs": "dotnet",
		"go": "go",
	}

	for name, toolchain := range expected {
		if got, ok := targets[name]; !ok {
			t.Errorf("DiscoverTargets() missing target %q", name)
		} else if got != toolchain {
			t.Errorf("DiscoverTargets()[%q] = %q, want %q", name, got, toolchain)
		}
	}

	// Check lib is not discovered (no marker)
	if _, ok := targets["lib"]; ok {
		t.Error("DiscoverTargets() should not discover 'lib' without marker file")
	}

	// Check excluded dirs are not discovered
	for _, name := range excludedDirs {
		if _, ok := targets[name]; ok {
			t.Errorf("DiscoverTargets() should not discover excluded dir %q", name)
		}
	}
}

// Note: TestDetectToolchain_DotNetSolution and TestDetectToolchain_DotNetDirectoryBuildProps
// have been removed. These cases are now covered in internal/toolchain/detect_test.go
// which tests all dotnet marker files including .sln, Directory.Build.props, and global.json.

func TestValidateTargetDirectory(t *testing.T) {
	root := t.TempDir()

	// Create a valid directory
	validDir := filepath.Join(root, "valid")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file (not a directory)
	filePath := filepath.Join(root, "file")
	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		dir       string
		expectErr bool
	}{
		{"valid directory", validDir, false},
		{"nonexistent", filepath.Join(root, "nonexistent"), true},
		{"file not dir", filePath, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetDirectory(tt.dir, "test-target")
			if (err != nil) != tt.expectErr {
				t.Errorf("validateTargetDirectory() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}
