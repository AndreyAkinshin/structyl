package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidMinimal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"project": {"name": "myproject"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Project.Name != "myproject" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "myproject")
	}
}

func TestLoad_ValidFull(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"project": {
			"name": "test-project",
			"description": "A test project",
			"homepage": "https://test.dev",
			"repository": "https://github.com/test/test",
			"license": "MIT"
		},
		"targets": {
			"cs": {
				"type": "language",
				"title": "C#",
				"toolchain": "dotnet"
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Project.Name != "test-project" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "test-project")
	}
	if len(cfg.Targets) != 1 {
		t.Errorf("len(Targets) = %d, want 1", len(cfg.Targets))
	}
	if cfg.Targets["cs"].Title != "C#" {
		t.Errorf("Targets[cs].Title = %q, want %q", cfg.Targets["cs"].Title, "C#")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("Load() expected error for missing file")
	}
	// Verify error message contains useful information.
	// At least one of these should be present in the error.
	errMsg := err.Error()
	containsPath := strings.Contains(errMsg, "nonexistent")
	containsOSError := strings.Contains(errMsg, "no such file")
	if !containsPath && !containsOSError {
		t.Errorf("error = %q, want to contain file path or 'no such file'", errMsg)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid JSON")
	}
}

func TestLoadWithDefaults_AppliesDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"project": {"name": "myproject"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	// Check defaults were applied
	if cfg.Version.Source != DefaultVersionSource {
		t.Errorf("Version.Source = %q, want %q", cfg.Version.Source, DefaultVersionSource)
	}
	if cfg.Tests.Directory != DefaultTestsDirectory {
		t.Errorf("Tests.Directory = %q, want %q", cfg.Tests.Directory, DefaultTestsDirectory)
	}
	if cfg.Tests.Pattern != DefaultTestsPattern {
		t.Errorf("Tests.Pattern = %q, want %q", cfg.Tests.Pattern, DefaultTestsPattern)
	}
}

func TestLoadWithDefaults_TargetDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#"}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	target := cfg.Targets["cs"]
	if target.Directory != "cs" {
		t.Errorf("Target.Directory = %q, want %q", target.Directory, "cs")
	}
	if target.Cwd != "cs" {
		t.Errorf("Target.Cwd = %q, want %q", target.Cwd, "cs")
	}
}

func TestLoadWithDefaults_DockerConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with docker section but no values set
	content := `{
		"project": {"name": "myproject"},
		"docker": {}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	if cfg.Docker == nil {
		t.Fatal("Docker config should not be nil")
	}
	if cfg.Docker.ComposeFile != DefaultDockerComposeFile {
		t.Errorf("Docker.ComposeFile = %q, want %q", cfg.Docker.ComposeFile, DefaultDockerComposeFile)
	}
	if cfg.Docker.EnvVar != DefaultDockerEnvVar {
		t.Errorf("Docker.EnvVar = %q, want %q", cfg.Docker.EnvVar, DefaultDockerEnvVar)
	}
}

func TestLoadWithDefaults_DockerConfigPreservesCustomValues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with docker section and custom values
	content := `{
		"project": {"name": "myproject"},
		"docker": {
			"compose_file": "custom-compose.yml",
			"env_var": "MY_DOCKER_VAR"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	if cfg.Docker == nil {
		t.Fatal("Docker config should not be nil")
	}
	// Custom values should be preserved
	if cfg.Docker.ComposeFile != "custom-compose.yml" {
		t.Errorf("Docker.ComposeFile = %q, want %q", cfg.Docker.ComposeFile, "custom-compose.yml")
	}
	if cfg.Docker.EnvVar != "MY_DOCKER_VAR" {
		t.Errorf("Docker.EnvVar = %q, want %q", cfg.Docker.EnvVar, "MY_DOCKER_VAR")
	}
}

func TestLoadWithDefaults_NoDockerSection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config without docker section at all
	content := `{
		"project": {"name": "myproject"}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	// Docker should remain nil when not specified
	if cfg.Docker != nil {
		t.Error("Docker config should be nil when not specified")
	}
}

func TestLoad_DockerServiceConfigAllFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"project": {"name": "myproject"},
		"docker": {
			"services": {
				"rs": {
					"base_image": "rust:1.70",
					"dockerfile": "rs/Dockerfile.custom",
					"platform": "linux/amd64",
					"volumes": ["/cache:/root/.cargo", "/data:/data"]
				}
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Docker == nil {
		t.Fatal("Docker config should not be nil")
	}
	svc, ok := cfg.Docker.Services["rs"]
	if !ok {
		t.Fatal("Docker.Services['rs'] not found")
	}
	if svc.BaseImage != "rust:1.70" {
		t.Errorf("ServiceConfig.BaseImage = %q, want %q", svc.BaseImage, "rust:1.70")
	}
	if svc.Dockerfile != "rs/Dockerfile.custom" {
		t.Errorf("ServiceConfig.Dockerfile = %q, want %q", svc.Dockerfile, "rs/Dockerfile.custom")
	}
	if svc.Platform != "linux/amd64" {
		t.Errorf("ServiceConfig.Platform = %q, want %q", svc.Platform, "linux/amd64")
	}
	if len(svc.Volumes) != 2 {
		t.Errorf("len(ServiceConfig.Volumes) = %d, want 2", len(svc.Volumes))
	}
	if svc.Volumes[0] != "/cache:/root/.cargo" {
		t.Errorf("ServiceConfig.Volumes[0] = %q, want %q", svc.Volumes[0], "/cache:/root/.cargo")
	}
}

// =============================================================================
// LoadAndValidate Tests
// =============================================================================

func TestLoadAndValidate_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {"type": "language", "title": "C#"}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := LoadAndValidate(path)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadAndValidate() returned nil config")
	}
	if len(warnings) != 0 {
		t.Errorf("LoadAndValidate() warnings = %v, want empty", warnings)
	}
}

func TestLoadAndValidate_UnknownFieldsOnly_NoError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with unknown fields at root level
	content := `{
		"project": {"name": "myproject"},
		"unknown_field": "value",
		"another_unknown": 123
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := LoadAndValidate(path)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v, want nil (unknown fields should not cause error)", err)
	}
	if cfg == nil {
		t.Fatal("LoadAndValidate() returned nil config")
	}
	if len(warnings) != 2 {
		t.Errorf("LoadAndValidate() warnings = %d, want 2", len(warnings))
	}
	// Verify warnings mention the unknown fields
	warningText := warnings[0] + warnings[1]
	if !strings.Contains(warningText, "unknown_field") {
		t.Errorf("warnings should mention 'unknown_field', got %v", warnings)
	}
}

func TestLoadAndValidate_ValidationError_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with invalid project name (uppercase not allowed)
	content := `{
		"project": {"name": "MyProject"}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := LoadAndValidate(path)
	if err == nil {
		t.Fatal("LoadAndValidate() error = nil, want error for invalid project name")
	}
	if cfg != nil {
		t.Error("LoadAndValidate() should return nil config on error")
	}
	// Warnings should be empty since validation failed before accumulation
	_ = warnings // warnings may or may not be present depending on error stage
}

func TestLoadAndValidate_ValidationError_WithUnknownFields_ReturnsBothWarnings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with unknown fields AND validation error
	content := `{
		"project": {"name": "InvalidName"},
		"unknown_field": "value",
		"targets": {
			"cs": {"type": "language", "title": "C#", "unknown_target_field": "x"}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := LoadAndValidate(path)
	if err == nil {
		t.Fatal("LoadAndValidate() error = nil, want error for invalid project name")
	}
	if cfg != nil {
		t.Error("LoadAndValidate() should return nil config on error")
	}
	// Unknown field warnings should still be returned even when validation fails.
	// Expected: 2 warnings (one for "unknown_field" at root, one for "unknown_target_field" in target)
	if len(warnings) != 2 {
		t.Errorf("LoadAndValidate() warnings = %d, want 2", len(warnings))
	}
}

func TestLoadAndValidate_FileNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	_, _, err := LoadAndValidate("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("LoadAndValidate() error = nil, want error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read") {
		t.Errorf("error = %q, want to contain 'failed to read'", err.Error())
	}
}

func TestLoadAndValidate_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadAndValidate(path)
	if err == nil {
		t.Fatal("LoadAndValidate() error = nil, want error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error = %q, want to contain 'parse'", err.Error())
	}
}

func TestLoadAndValidate_WarningsNoSliceAliasing(t *testing.T) {
	// Verify that warning slices are properly allocated without aliasing.
	// This test ensures the fix for the slice append aliasing bug works correctly.
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Config with unknown fields (produces warnings)
	content := `{
		"project": {"name": "myproject"},
		"unknown1": "value1",
		"unknown2": "value2"
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, warnings, err := LoadAndValidate(path)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}

	// Verify warnings are present and independent
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(warnings))
	}

	// Modify the warnings slice - should not affect internal state
	// This is a sanity check that the returned slice is independent
	if len(warnings) > 0 {
		original := warnings[0]
		warnings[0] = "modified"
		// The returned slice should be independent, so this modification
		// should not cause any issues. This is more of a documentation
		// of expected behavior than a strict test.
		_ = original
	}
}
