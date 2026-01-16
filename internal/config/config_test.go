package config

import (
	"os"
	"path/filepath"
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

func TestLoadWithDefaults_DockerConfigDefaults(t *testing.T) {
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
