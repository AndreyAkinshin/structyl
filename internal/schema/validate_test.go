package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaValidConfig(t *testing.T) {
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
	err := ValidateConfig([]byte("{}"))
	if err == nil {
		t.Error("expected validation error for empty object, got nil")
	}
}

func TestSchemaInvalidConfigNotObject(t *testing.T) {
	err := ValidateConfig([]byte(`"string"`))
	if err == nil {
		t.Error("expected validation error for non-object, got nil")
	}
}

func TestSchemaValidToolchains(t *testing.T) {
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
	data := []byte(`{"toolchains": {}}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for missing version, got nil")
	}
}

func TestSchemaInvalidToolchainsMissingToolchains(t *testing.T) {
	data := []byte(`{"version": "1.0"}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for missing toolchains, got nil")
	}
}

func TestSchemaInvalidToolchainsWrongVersion(t *testing.T) {
	data := []byte(`{"version": "2.0", "toolchains": {}}`)
	err := ValidateToolchains(data)
	if err == nil {
		t.Error("expected validation error for wrong version, got nil")
	}
}

func TestSchemaToolchainsMinimal(t *testing.T) {
	data := []byte(`{"version": "1.0", "toolchains": {}}`)
	if err := ValidateToolchains(data); err != nil {
		t.Errorf("expected valid minimal toolchains, got error: %v", err)
	}
}

func TestSchemaToolchainsWithSimpleToolchain(t *testing.T) {
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
