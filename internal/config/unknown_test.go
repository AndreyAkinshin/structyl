package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWithWarnings_UnknownRootField(t *testing.T) {
	data := []byte(`{
		"project": {"name": "myproject"},
		"unknown_field": "value"
	}`)

	cfg, warnings, err := LoadWithWarnings("test.json", data)
	if err != nil {
		t.Fatalf("LoadWithWarnings() error = %v", err)
	}
	if cfg.Project.Name != "myproject" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "myproject")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unknown_field") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning about unknown_field, got %v", warnings)
	}
}

func TestLoadWithWarnings_SchemaFieldIgnored(t *testing.T) {
	data := []byte(`{
		"$schema": "https://structyl.dev/schemas/structyl.schema.json",
		"project": {"name": "myproject"}
	}`)

	_, warnings, err := LoadWithWarnings("test.json", data)
	if err != nil {
		t.Fatalf("LoadWithWarnings() error = %v", err)
	}

	for _, w := range warnings {
		if strings.Contains(w, "$schema") {
			t.Errorf("$schema should not produce warning, got: %s", w)
		}
	}
}

func TestLoadWithWarnings_UnknownTargetField(t *testing.T) {
	data := []byte(`{
		"project": {"name": "myproject"},
		"targets": {
			"cs": {
				"type": "language",
				"title": "C#",
				"unknown_target_field": "value"
			}
		}
	}`)

	_, warnings, err := LoadWithWarnings("test.json", data)
	if err != nil {
		t.Fatalf("LoadWithWarnings() error = %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unknown_target_field") && strings.Contains(w, "cs") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning about unknown_target_field in cs, got %v", warnings)
	}
}

func TestLoadAndValidate_WithUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"project": {"name": "myproject"},
		"future_feature": true
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := LoadAndValidate(path)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}
	if cfg.Project.Name != "myproject" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "myproject")
	}
	if len(warnings) == 0 {
		t.Error("Expected warnings for unknown field")
	}
}
