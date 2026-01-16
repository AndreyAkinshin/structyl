package runner

import (
	"os"
	"strings"
	"testing"

	"github.com/akinshin/structyl/internal/config"
)

func TestGenerateComposeFile(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Toolchain: "cargo",
			},
			"go": {
				Type:      "language",
				Title:     "Go",
				Toolchain: "go",
			},
		},
	}

	content, err := GenerateComposeFile("/project", cfg)
	if err != nil {
		t.Fatalf("GenerateComposeFile() error = %v", err)
	}

	// Should contain services
	if !strings.Contains(content, "services:") {
		t.Error("compose file should contain 'services:'")
	}

	// Should have both targets as services
	if !strings.Contains(content, "rs:") {
		t.Error("compose file should contain 'rs' service")
	}
	if !strings.Contains(content, "go:") {
		t.Error("compose file should contain 'go' service")
	}
}

func TestGenerateComposeFile_CustomImage(t *testing.T) {
	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"custom": {
				Type:      "language",
				Title:     "Custom",
				Toolchain: "npm",
			},
		},
		Docker: &config.DockerConfig{
			Services: map[string]config.ServiceConfig{
				"custom": {
					BaseImage: "my-custom-image:latest",
				},
			},
		},
	}

	content, err := GenerateComposeFile("/project", cfg)
	if err != nil {
		t.Fatalf("GenerateComposeFile() error = %v", err)
	}

	if !strings.Contains(content, "my-custom-image:latest") {
		t.Error("compose file should use custom image")
	}
}

func TestGetDefaultImage(t *testing.T) {
	tests := []struct {
		toolchain string
		expected  string
	}{
		{"cargo", "rust:latest"},
		{"dotnet", "mcr.microsoft.com/dotnet/sdk:8.0"},
		{"go", "golang:latest"},
		{"npm", "node:lts"},
		{"python", "python:3.12"},
		{"unknown", "alpine:latest"},
		{"", "alpine:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.toolchain, func(t *testing.T) {
			result := getDefaultImage(tt.toolchain)
			if result != tt.expected {
				t.Errorf("getDefaultImage(%q) = %q, want %q", tt.toolchain, result, tt.expected)
			}
		})
	}
}

func TestGenerateServiceForTarget(t *testing.T) {
	targetCfg := config.TargetConfig{
		Type:      "language",
		Title:     "Python",
		Toolchain: "python",
		Directory: "src/python",
		Env: map[string]string{
			"PYTHONPATH": "src",
		},
	}

	service := generateServiceForTarget("py", targetCfg, &config.DockerConfig{})

	if service.Image != "python:3.12" {
		t.Errorf("Image = %q, want %q", service.Image, "python:3.12")
	}
	if service.WorkingDir != "/workspace/src/python" {
		t.Errorf("WorkingDir = %q, want %q", service.WorkingDir, "/workspace/src/python")
	}
	if service.Environment["PYTHONPATH"] != "src" {
		t.Error("Environment should include target env vars")
	}
}

func TestFormatVolumePath(t *testing.T) {
	tests := []struct {
		host      string
		container string
		readonly  bool
		expected  string
	}{
		{"./src", "/app/src", false, "./src:/app/src"},
		{"./config", "/config", true, "./config:/config:ro"},
		{"/host/path", "/container/path", false, "/host/path:/container/path"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatVolumePath(tt.host, tt.container, tt.readonly)
			if result != tt.expected {
				t.Errorf("FormatVolumePath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSplitVolumePath(t *testing.T) {
	tests := []struct {
		volume        string
		wantHost      string
		wantContainer string
		wantReadonly  bool
	}{
		{"./src:/app/src", "./src", "/app/src", false},
		{"./config:/config:ro", "./config", "/config", true},
		{"/single/path", "/single/path", "/single/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.volume, func(t *testing.T) {
			host, container, readonly := SplitVolumePath(tt.volume)
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if container != tt.wantContainer {
				t.Errorf("container = %q, want %q", container, tt.wantContainer)
			}
			if readonly != tt.wantReadonly {
				t.Errorf("readonly = %v, want %v", readonly, tt.wantReadonly)
			}
		})
	}
}

func TestMergeComposeFiles(t *testing.T) {
	base := &ComposeConfig{
		Version: "3.8",
		Services: map[string]ComposeService{
			"web": {
				Image:      "nginx:latest",
				WorkingDir: "/app",
			},
		},
	}

	override := &ComposeConfig{
		Services: map[string]ComposeService{
			"web": {
				Image: "nginx:alpine",
			},
			"db": {
				Image: "postgres:15",
			},
		},
	}

	result := MergeComposeFiles(base, override)

	// Should have both services
	if len(result.Services) != 2 {
		t.Errorf("merged should have 2 services, got %d", len(result.Services))
	}

	// Web should be overridden
	if result.Services["web"].Image != "nginx:alpine" {
		t.Error("web image should be overridden")
	}
	// WorkingDir should be preserved from base
	if result.Services["web"].WorkingDir != "/app" {
		t.Error("web working_dir should be preserved")
	}

	// DB should be added
	if result.Services["db"].Image != "postgres:15" {
		t.Error("db service should be added")
	}
}

func TestMergeServices(t *testing.T) {
	base := ComposeService{
		Image:      "node:lts",
		WorkingDir: "/app",
		Volumes:    []string{".:/app"},
		Environment: map[string]string{
			"NODE_ENV": "development",
		},
	}

	override := ComposeService{
		Image: "node:18",
		Environment: map[string]string{
			"DEBUG": "true",
		},
		Volumes: []string{"./data:/data"},
	}

	result := mergeServices(base, override)

	// Image should be overridden
	if result.Image != "node:18" {
		t.Errorf("Image = %q, want %q", result.Image, "node:18")
	}

	// WorkingDir should be preserved
	if result.WorkingDir != "/app" {
		t.Errorf("WorkingDir = %q, want %q", result.WorkingDir, "/app")
	}

	// Environment should be merged
	if result.Environment["NODE_ENV"] != "development" {
		t.Error("NODE_ENV should be preserved")
	}
	if result.Environment["DEBUG"] != "true" {
		t.Error("DEBUG should be added")
	}

	// Volumes should be merged
	if len(result.Volumes) != 2 {
		t.Errorf("len(Volumes) = %d, want 2", len(result.Volumes))
	}
}

func TestGetPlatform(t *testing.T) {
	platform := getPlatform()
	if !strings.HasPrefix(platform, "linux/") {
		t.Errorf("getPlatform() = %q, want linux/...", platform)
	}
}

func TestWriteComposeFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Toolchain: "cargo",
			},
		},
	}

	err := WriteComposeFile(tmpDir, cfg)
	if err != nil {
		t.Fatalf("WriteComposeFile() error = %v", err)
	}

	// Verify file exists
	composePath := tmpDir + "/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("docker-compose.yml was not created")
	}
}

func TestWriteComposeFile_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Targets: map[string]config.TargetConfig{
			"rs": {
				Type:      "language",
				Title:     "Rust",
				Toolchain: "cargo",
			},
		},
		Docker: &config.DockerConfig{
			ComposeFile: "custom-compose.yml",
		},
	}

	err := WriteComposeFile(tmpDir, cfg)
	if err != nil {
		t.Fatalf("WriteComposeFile() error = %v", err)
	}

	// Verify custom file exists
	composePath := tmpDir + "/custom-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("custom-compose.yml was not created")
	}
}

func TestComposeFileExists_True(t *testing.T) {
	tmpDir := t.TempDir()

	// Create compose file
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte("services: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	if !ComposeFileExists(tmpDir, nil) {
		t.Error("ComposeFileExists() = false, want true")
	}
}

func TestComposeFileExists_False(t *testing.T) {
	tmpDir := t.TempDir()

	if ComposeFileExists(tmpDir, nil) {
		t.Error("ComposeFileExists() = true, want false")
	}
}

func TestComposeFileExists_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create custom compose file
	composePath := tmpDir + "/custom.yml"
	if err := os.WriteFile(composePath, []byte("services: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.DockerConfig{ComposeFile: "custom.yml"}
	if !ComposeFileExists(tmpDir, cfg) {
		t.Error("ComposeFileExists() = false, want true for custom path")
	}
}

func TestValidateComposeFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx:latest
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidateComposeFile(tmpDir, nil)
	if err != nil {
		t.Errorf("ValidateComposeFile() error = %v", err)
	}
}

func TestValidateComposeFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	content := `not: valid: yaml: {{`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidateComposeFile(tmpDir, nil)
	if err == nil {
		t.Error("ValidateComposeFile() expected error for invalid YAML")
	}
}

func TestValidateComposeFile_NoServices(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: "3.8"
services: {}
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidateComposeFile(tmpDir, nil)
	if err == nil {
		t.Error("ValidateComposeFile() expected error for no services")
	}
}

func TestValidateComposeFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	err := ValidateComposeFile(tmpDir, nil)
	if err == nil {
		t.Error("ValidateComposeFile() expected error for missing file")
	}
}

func TestParseComposeFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx:latest
  db:
    image: postgres:15
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	compose, err := ParseComposeFile(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseComposeFile() error = %v", err)
	}

	if len(compose.Services) != 2 {
		t.Errorf("len(Services) = %d, want 2", len(compose.Services))
	}
	if compose.Services["web"].Image != "nginx:latest" {
		t.Errorf("web.Image = %q, want %q", compose.Services["web"].Image, "nginx:latest")
	}
}

func TestParseComposeFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ParseComposeFile(tmpDir, nil)
	if err == nil {
		t.Error("ParseComposeFile() expected error for missing file")
	}
}

func TestGetServiceNames_ReturnsNames(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx
  api:
    image: node
  db:
    image: postgres
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	names, err := GetServiceNames(tmpDir, nil)
	if err != nil {
		t.Fatalf("GetServiceNames() error = %v", err)
	}

	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"web", "api", "db"} {
		if !nameSet[expected] {
			t.Errorf("missing service name %q", expected)
		}
	}
}

func TestServiceExists_Found(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if !ServiceExists(tmpDir, nil, "web") {
		t.Error("ServiceExists(web) = false, want true")
	}
}

func TestServiceExists_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if ServiceExists(tmpDir, nil, "nonexistent") {
		t.Error("ServiceExists(nonexistent) = true, want false")
	}
}

func TestServiceExists_NoComposeFile(t *testing.T) {
	tmpDir := t.TempDir()

	if ServiceExists(tmpDir, nil, "web") {
		t.Error("ServiceExists() = true when no compose file, want false")
	}
}

func TestGetServiceImage_Found(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx:alpine
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	image, err := GetServiceImage(tmpDir, nil, "web")
	if err != nil {
		t.Fatalf("GetServiceImage() error = %v", err)
	}
	if image != "nginx:alpine" {
		t.Errorf("GetServiceImage() = %q, want %q", image, "nginx:alpine")
	}
}

func TestGetServiceImage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	content := `services:
  web:
    image: nginx
`
	composePath := tmpDir + "/docker-compose.yml"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := GetServiceImage(tmpDir, nil, "nonexistent")
	if err == nil {
		t.Error("GetServiceImage() expected error for nonexistent service")
	}
}

func TestMergeComposeFiles_VersionOverride(t *testing.T) {
	base := &ComposeConfig{
		Version:  "3.7",
		Services: map[string]ComposeService{},
	}
	override := &ComposeConfig{
		Version:  "3.9",
		Services: map[string]ComposeService{},
	}

	result := MergeComposeFiles(base, override)
	if result.Version != "3.9" {
		t.Errorf("Version = %q, want %q", result.Version, "3.9")
	}
}

func TestMergeServices_AllFields(t *testing.T) {
	base := ComposeService{
		Image:      "base:v1",
		WorkingDir: "/base",
		Platform:   "linux/amd64",
		Command:    "start",
		User:       "root",
	}
	override := ComposeService{
		Build: &ComposeBuild{
			Context:    ".",
			Dockerfile: "Dockerfile.prod",
		},
		Platform: "linux/arm64",
		Command:  "serve",
		User:     "app",
	}

	result := mergeServices(base, override)

	// Build should be overridden
	if result.Build == nil || result.Build.Dockerfile != "Dockerfile.prod" {
		t.Error("Build should be overridden")
	}
	// Platform should be overridden
	if result.Platform != "linux/arm64" {
		t.Errorf("Platform = %q, want %q", result.Platform, "linux/arm64")
	}
	// Command should be overridden
	if result.Command != "serve" {
		t.Errorf("Command = %q, want %q", result.Command, "serve")
	}
	// User should be overridden
	if result.User != "app" {
		t.Errorf("User = %q, want %q", result.User, "app")
	}
	// Image should be preserved (not overridden with empty)
	if result.Image != "base:v1" {
		t.Errorf("Image = %q, want %q (preserved)", result.Image, "base:v1")
	}
}
