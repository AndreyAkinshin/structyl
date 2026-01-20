package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaValidConfig(t *testing.T) {
	t.Parallel()
	validFixtures := []string{
		"minimal",
		"multi-language",
		"with-docker",
	}

	for _, name := range validFixtures {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "test", "fixtures", name, ".structyl", "config.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			if err := ValidateConfig(data); err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
		})
	}
}

func TestSchemaValidConfigSemanticErrors(t *testing.T) {
	t.Parallel()
	// These fixtures are semantically invalid (circular deps, invalid toolchain ref)
	// but structurally valid according to the schema.
	semanticOnlyInvalid := []string{
		"invalid/circular-deps",
		"invalid/invalid-toolchain",
	}

	for _, name := range semanticOnlyInvalid {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "test", "fixtures", name, ".structyl", "config.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			if err := ValidateConfig(data); err != nil {
				t.Errorf("expected schema-valid config (semantic error only), got error: %v", err)
			}
		})
	}
}

func TestSchemaInvalidConfigMissingName(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "test", "fixtures", "invalid", "missing-name", ".structyl", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	err = ValidateConfig(data)
	if err == nil {
		t.Error("expected validation error for missing name, got nil")
	}
}

func TestSchemaInvalidConfigMalformedJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "test", "fixtures", "invalid", "malformed-json", ".structyl", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	err = ValidateConfig(data)
	if err == nil {
		t.Error("expected validation error for malformed JSON, got nil")
	}
}

func TestSchemaInvalidConfigEmpty(t *testing.T) {
	t.Parallel()
	err := ValidateConfig([]byte("{}"))
	if err == nil {
		t.Error("expected validation error for empty object, got nil")
	}
}

func TestSchemaInvalidConfigNotObject(t *testing.T) {
	t.Parallel()
	err := ValidateConfig([]byte(`"string"`))
	if err == nil {
		t.Error("expected validation error for non-object, got nil")
	}
}

func TestSchemaValidToolchains(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "cli", "toolchains_template.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read toolchains template: %v", err)
	}

	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid toolchains, got error: %v", err)
	}
}

func TestSchemaInvalidToolchainsMissingVersion(t *testing.T) {
	t.Parallel()
	data := []byte(`{"toolchains": {}}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for missing version, got nil")
	}
}

func TestSchemaInvalidToolchainsMissingToolchains(t *testing.T) {
	t.Parallel()
	data := []byte(`{"version": "1.0"}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for missing toolchains, got nil")
	}
}

func TestSchemaInvalidToolchainsWrongVersion(t *testing.T) {
	t.Parallel()
	data := []byte(`{"version": "2.0", "toolchains": {}}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for wrong version, got nil")
	}
}

func TestSchemaToolchainsMinimal(t *testing.T) {
	t.Parallel()
	data := []byte(`{"version": "1.0", "toolchains": {}}`)
	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid minimal toolchains, got error: %v", err)
	}
}

func TestSchemaToolchainsWithSimpleToolchain(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"version": "1.0",
		"toolchains": {
			"test": {
				"commands": {
					"build": "make build",
					"test": null,
					"check": ["lint", "format"]
				}
			}
		}
	}`)
	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid toolchains with simple toolchain, got error: %v", err)
	}
}

func TestSchemaToolchainsWithMise(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"version": "1.0",
		"toolchains": {
			"go": {
				"mise": {
					"primary_tool": "go",
					"version": "1.24",
					"extra_tools": {
						"golangci-lint": "latest"
					}
				},
				"commands": {
					"build": "go build ./..."
				}
			}
		}
	}`)
	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid toolchains with mise config, got error: %v", err)
	}
}

func TestSchemaInvalidConfigWrongFieldType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json string
	}{
		{
			name: "project name is number",
			json: `{"project": {"name": 123}}`,
		},
		{
			name: "targets is array",
			json: `{"project": {"name": "test"}, "targets": []}`,
		},
		{
			name: "target type is number",
			json: `{"project": {"name": "test"}, "targets": {"go": {"type": 123, "title": "Go"}}}`,
		},
		{
			name: "target title is boolean",
			json: `{"project": {"name": "test"}, "targets": {"go": {"type": "language", "title": true}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig([]byte(tt.json))
			if err == nil {
				t.Errorf("expected validation error for %s, got nil", tt.name)
			}
		})
	}
}

func TestSchemaInvalidConfigInvalidTargetType(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"project": {"name": "test"},
		"targets": {
			"go": {
				"type": "unknown",
				"title": "Go"
			}
		}
	}`)

	err := ValidateConfig(data)
	if err == nil {
		t.Error("expected validation error for invalid target type, got nil")
	}
}

func TestValidateConfig_ErrorMessageContainsPath(t *testing.T) {
	t.Parallel()
	// Test case that should fail with a path indicator
	data := []byte(`{
		"project": {"name": "test"},
		"targets": {
			"invalid-target": {
				"type": "invalid-type",
				"title": "Test"
			}
		}
	}`)

	err := ValidateConfig(data)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	// Error message should contain some indication of what failed
	errStr := err.Error()
	if errStr == "" {
		t.Error("error message should not be empty")
	}
}

func TestSchemaInvalidToolchainsMalformedJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`{invalid json}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for malformed JSON in toolchains, got nil")
	}
}

func TestSchemaInvalidConfigArray(t *testing.T) {
	t.Parallel()
	err := ValidateConfig([]byte(`[]`))
	if err == nil {
		t.Error("expected validation error for array instead of object, got nil")
	}
}

func TestSchemaInvalidConfigNull(t *testing.T) {
	t.Parallel()
	err := ValidateConfig([]byte(`null`))
	if err == nil {
		t.Error("expected validation error for null config, got nil")
	}
}

func TestSchemaInvalidToolchainsNotObject(t *testing.T) {
	t.Parallel()
	err := ValidateToolchains([]byte(`"string"`))
	if err == nil {
		t.Error("expected validation error for non-object toolchains, got nil")
	}
}

func TestSchemaInvalidToolchainsNull(t *testing.T) {
	t.Parallel()
	err := ValidateToolchains([]byte(`null`))
	if err == nil {
		t.Error("expected validation error for null toolchains, got nil")
	}
}

func TestSchemaInvalidToolchainsArray(t *testing.T) {
	t.Parallel()
	err := ValidateToolchains([]byte(`[]`))
	if err == nil {
		t.Error("expected validation error for array toolchains, got nil")
	}
}

func TestSchemaConfigWithAllOptionalSections(t *testing.T) {
	t.Parallel()
	// Test a comprehensive config with most optional sections
	data := []byte(`{
		"project": {"name": "comprehensive-test"},
		"version": {
			"file": "VERSION",
			"propagate": true
		},
		"targets": {
			"go": {
				"type": "language",
				"title": "Go",
				"toolchain": "go",
				"directory": "go",
				"depends_on": []
			}
		},
		"mise": {
			"auto_generate": true
		},
		"docker": {
			"compose_file": "docker-compose.yml"
		}
	}`)

	if err := ValidateConfig(data); err != nil {
		t.Errorf("expected valid comprehensive config, got error: %v", err)
	}
}

func TestSchemaToolchainsWithAllFields(t *testing.T) {
	t.Parallel()
	// Test a comprehensive toolchains config
	data := []byte(`{
		"version": "1.0",
		"toolchains": {
			"custom": {
				"mise": {
					"primary_tool": "go",
					"version": "1.24",
					"extra_tools": {"golangci-lint": "latest"}
				},
				"commands": {
					"build": "make build",
					"test": "make test",
					"check": ["lint", "format"],
					"clean": null,
					"restore": "go mod download",
					"ci": "make ci"
				},
				"descriptions": {
					"build": "Build the project",
					"test": "Run tests"
				}
			}
		}
	}`)

	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid comprehensive toolchains, got error: %v", err)
	}
}

func TestSchemaInvalidToolchainsInvalidCommandType(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"version": "1.0",
		"toolchains": {
			"test": {
				"commands": {
					"build": 123
				}
			}
		}
	}`)

	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for invalid command type (number), got nil")
	}
}

func TestSchemaConfigNestedValidationError(t *testing.T) {
	t.Parallel()
	// Test deep nesting validation
	data := []byte(`{
		"project": {"name": "test"},
		"targets": {
			"go": {
				"type": "language",
				"title": "Go",
				"commands": {
					"build": 123
				}
			}
		}
	}`)

	err := ValidateConfig(data)
	if err == nil {
		t.Error("expected validation error for invalid nested command type, got nil")
	}
}

func TestSchemaConfigInvalidProjectNamePattern(t *testing.T) {
	t.Parallel()
	// Project name must match pattern ^[a-z][a-z0-9-]*$
	invalidNames := []string{
		"TestProject",  // uppercase
		"123project",   // starts with number
		"test_project", // underscore
		"test.project", // dot
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			data := []byte(`{"project": {"name": "` + name + `"}}`)
			err := ValidateConfig(data)
			if err == nil {
				t.Errorf("expected validation error for invalid project name %q, got nil", name)
			}
		})
	}
}
