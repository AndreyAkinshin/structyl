package schema

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// TestEmbeddedSchemasAreValidJSON verifies that all embedded schema files are valid JSON.
// This catches corrupted or malformed schema files at test time rather than runtime.
func TestEmbeddedSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("failed to read embedded FS: %v", err)
	}

	schemaCount := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		schemaCount++

		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			data, err := FS.ReadFile(entry.Name())
			if err != nil {
				t.Fatalf("failed to read %s: %v", entry.Name(), err)
			}

			var v interface{}
			if err := json.Unmarshal(data, &v); err != nil {
				t.Errorf("%s is not valid JSON: %v", entry.Name(), err)
			}

			// Verify it's an object (all JSON schemas should be objects)
			if _, ok := v.(map[string]interface{}); !ok {
				t.Errorf("%s root is not an object", entry.Name())
			}
		})
	}

	// Ensure we actually tested some schemas
	if schemaCount == 0 {
		t.Error("no schema files found in embedded FS")
	}
}

// TestExpectedSchemasExist verifies that all required schema files are embedded.
func TestExpectedSchemasExist(t *testing.T) {
	t.Parallel()

	expectedSchemas := []string{
		"config.schema.json",
		"toolchains.schema.json",
		"testcase.schema.json",
	}

	for _, name := range expectedSchemas {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := FS.ReadFile(name)
			if err != nil {
				t.Errorf("expected schema %s not found: %v", name, err)
			}
		})
	}
}

// TestSchemaStructure verifies that schemas have expected top-level fields.
func TestSchemaStructure(t *testing.T) {
	t.Parallel()

	schemas := []string{
		"config.schema.json",
		"toolchains.schema.json",
		"testcase.schema.json",
	}

	for _, name := range schemas {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := FS.ReadFile(name)
			if err != nil {
				t.Fatalf("failed to read %s: %v", name, err)
			}

			var schema map[string]interface{}
			if err := json.Unmarshal(data, &schema); err != nil {
				t.Fatalf("failed to parse %s: %v", name, err)
			}

			// All schemas should have $schema field
			if _, ok := schema["$schema"]; !ok {
				t.Errorf("%s missing $schema field", name)
			}

			// All schemas should have type field
			if _, ok := schema["type"]; !ok {
				t.Errorf("%s missing type field", name)
			}
		})
	}
}
