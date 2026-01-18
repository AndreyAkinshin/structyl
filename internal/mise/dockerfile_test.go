package mise

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

func TestGenerateDockerfile_Basic(t *testing.T) {
	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		Directory: "rs",
	}

	content, err := GenerateDockerfile("rs", targetCfg)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error = %v", err)
	}

	checks := []string{
		"FROM ubuntu:22.04",
		"curl -fsSL https://mise.run",
		"mise trust && mise install",
		"WORKDIR /workspace/rs",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("missing %q in Dockerfile", check)
		}
	}
}

func TestGenerateDockerfile_DefaultDirectory(t *testing.T) {
	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		// No Directory set - should use target name
	}

	content, err := GenerateDockerfile("rust", targetCfg)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error = %v", err)
	}

	if !strings.Contains(content, "WORKDIR /workspace/rust") {
		t.Error("should use target name as working directory when Directory not set")
	}
}

func TestGenerateDockerfile_WindowsPath(t *testing.T) {
	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		Directory: "src\\rs", // Windows-style path
	}

	content, err := GenerateDockerfile("rs", targetCfg)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error = %v", err)
	}

	// Should convert to forward slashes for Docker
	if !strings.Contains(content, "WORKDIR /workspace/src/rs") {
		t.Error("should convert backslashes to forward slashes")
	}
}

func TestWriteDockerfile(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}

	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		Directory: "rs",
	}

	// First write should create file
	created, err := WriteDockerfile(tmpDir, "rs", targetCfg, false)
	if err != nil {
		t.Fatalf("WriteDockerfile() error = %v", err)
	}
	if !created {
		t.Error("WriteDockerfile() = false, want true")
	}

	// Second write without force should not overwrite
	created, err = WriteDockerfile(tmpDir, "rs", targetCfg, false)
	if err != nil {
		t.Fatalf("WriteDockerfile() error = %v", err)
	}
	if created {
		t.Error("WriteDockerfile() = true, want false (file exists)")
	}

	// Third write with force should overwrite
	created, err = WriteDockerfile(tmpDir, "rs", targetCfg, true)
	if err != nil {
		t.Fatalf("WriteDockerfile() error = %v", err)
	}
	if !created {
		t.Error("WriteDockerfile(force=true) = false, want true")
	}

	// Verify file exists
	dockerfilePath := filepath.Join(tmpDir, "rs", "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err != nil {
		t.Errorf("Dockerfile not created at %s", dockerfilePath)
	}
}

func TestWriteDockerfile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create target directory - WriteDockerfile should create it

	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		Directory: "rs",
	}

	created, err := WriteDockerfile(tmpDir, "rs", targetCfg, false)
	if err != nil {
		t.Fatalf("WriteDockerfile() error = %v", err)
	}
	if !created {
		t.Error("WriteDockerfile() = false, want true")
	}

	// Verify file exists
	dockerfilePath := filepath.Join(tmpDir, "rs", "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err != nil {
		t.Errorf("Dockerfile not created at %s", dockerfilePath)
	}
}

func TestWriteAllDockerfiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs":     {Toolchain: "cargo", Directory: "rs"},
			"go":     {Toolchain: "go", Directory: "go"},
			"custom": {Toolchain: "make"}, // Unsupported - should be skipped
		},
	}

	results, err := WriteAllDockerfiles(tmpDir, cfg, false)
	if err != nil {
		t.Fatalf("WriteAllDockerfiles() error = %v", err)
	}

	// Should have results for rs and go, but not custom
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	if !results["rs"] {
		t.Error("results[rs] = false, want true")
	}
	if !results["go"] {
		t.Error("results[go] = false, want true")
	}
	if _, ok := results["custom"]; ok {
		t.Error("results should not include unsupported toolchain 'custom'")
	}
}

func TestDockerfileExists(t *testing.T) {
	tmpDir := t.TempDir()

	targetCfg := config.TargetConfig{
		Toolchain: "cargo",
		Directory: "rs",
	}

	// Should not exist initially
	if DockerfileExists(tmpDir, "rs", targetCfg) {
		t.Error("DockerfileExists() = true, want false")
	}

	// Create Dockerfile
	targetDir := filepath.Join(tmpDir, "rs")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "Dockerfile"), []byte("FROM ubuntu"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Should exist now
	if !DockerfileExists(tmpDir, "rs", targetCfg) {
		t.Error("DockerfileExists() = false, want true")
	}
}

func TestGetDockerfilePath(t *testing.T) {
	tests := []struct {
		name       string
		targetName string
		targetCfg  config.TargetConfig
		wantSuffix string
	}{
		{
			name:       "with directory",
			targetName: "rs",
			targetCfg:  config.TargetConfig{Directory: "rust"},
			wantSuffix: "rust/Dockerfile",
		},
		{
			name:       "without directory",
			targetName: "rs",
			targetCfg:  config.TargetConfig{},
			wantSuffix: "rs/Dockerfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := GetDockerfilePath("/project", tt.targetName, tt.targetCfg)
			// Normalize path separators for cross-platform comparison
			normalizedPath := filepath.ToSlash(path)
			if !strings.HasSuffix(normalizedPath, tt.wantSuffix) {
				t.Errorf("GetDockerfilePath() = %q, want suffix %q", path, tt.wantSuffix)
			}
		})
	}
}
