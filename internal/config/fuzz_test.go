package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

// FuzzUnmarshalConfig tests JSON unmarshaling of Config with arbitrary input.
// Run: go test -fuzz=FuzzUnmarshalConfig -fuzztime=30s ./internal/config
func FuzzUnmarshalConfig(f *testing.F) {
	// Seed corpus with representative inputs
	seeds := []string{
		// Valid minimal config
		`{"project": {"name": "test"}}`,
		// Valid config with targets
		`{"project": {"name": "myproject"}, "targets": {"go": {"type": "language", "title": "Go"}}}`,
		// Valid config with all top-level fields
		`{"project": {"name": "full"}, "version": {"source": "VERSION"}, "targets": {}, "mise": {}, "tests": {}, "docker": {}, "release": {}, "ci": {}, "artifacts": {}}`,
		// Edge cases: empty object
		`{}`,
		// Edge cases: empty string
		``,
		// Edge cases: null
		`null`,
		// Edge cases: array (invalid root type)
		`[]`,
		// Edge cases: string (invalid root type)
		`"string"`,
		// Edge cases: number (invalid root type)
		`123`,
		// Edge cases: boolean (invalid root type)
		`true`,
		// Edge cases: deeply nested
		`{"project": {"name": "deep"}, "targets": {"a": {"type": "language", "title": "A", "commands": {"build": ["nested", "commands"]}}}}`,
		// Edge cases: Unicode in values
		`{"project": {"name": "test", "description": "项目描述 プロジェクト проект"}}`,
		// Edge cases: special characters in strings
		`{"project": {"name": "test", "description": "line1\nline2\ttab"}}`,
		// Edge cases: escaped characters
		`{"project": {"name": "test", "description": "quote\"slash\\null\u0000"}}`,
		// Edge cases: large numbers
		`{"tests": {"comparison": {"float_tolerance": 1e308}}}`,
		// Edge cases: negative numbers
		`{"tests": {"comparison": {"float_tolerance": -1.5}}}`,
		// Edge cases: very small numbers
		`{"tests": {"comparison": {"float_tolerance": 1e-308}}}`,
		// Edge cases: NaN/Infinity-like strings (JSON doesn't support these as numbers)
		`{"project": {"name": "nan", "description": "NaN Infinity -Infinity"}}`,
		// Malformed: trailing comma
		`{"project": {"name": "test",}}`,
		// Malformed: single quotes
		`{'project': {'name': 'test'}}`,
		// Malformed: unquoted keys
		`{project: {name: "test"}}`,
		// Malformed: missing closing brace
		`{"project": {"name": "test"}`,
		// Malformed: missing colon
		`{"project" {"name": "test"}}`,
		// Malformed: extra comma
		`{"project": {"name": "test"},,}`,
		// Edge case: empty string values
		`{"project": {"name": "", "description": ""}}`,
		// Edge case: whitespace-only values
		`{"project": {"name": "   ", "description": "\t\n"}}`,
		// Edge case: very long string
		`{"project": {"name": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`,
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var cfg Config

		// The unmarshaler should never panic on any input
		err1 := json.Unmarshal(data, &cfg)

		// Determinism: unmarshaling the same input twice must produce identical results
		var cfg2 Config
		err2 := json.Unmarshal(data, &cfg2)

		// Both should either succeed or fail
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("non-deterministic error: first=%v, second=%v", err1, err2)
		}

		// If both succeed, results should be identical
		if err1 == nil && err2 == nil {
			if !reflect.DeepEqual(cfg, cfg2) {
				t.Errorf("non-deterministic result: first=%+v, second=%+v", cfg, cfg2)
			}
		}

		// If unmarshaling succeeded, validate that we can re-marshal
		if err1 == nil {
			_, marshalErr := json.Marshal(cfg)
			if marshalErr != nil {
				t.Errorf("failed to re-marshal successfully unmarshaled config: %v", marshalErr)
			}
		}
	})
}

// FuzzLoadWithWarnings tests LoadWithWarnings with arbitrary JSON input.
// Run: go test -fuzz=FuzzLoadWithWarnings -fuzztime=30s ./internal/config
func FuzzLoadWithWarnings(f *testing.F) {
	// Seed corpus with inputs that exercise warning detection
	seeds := []string{
		// Valid config with no warnings
		`{"project": {"name": "test"}}`,
		// Config with unknown root field
		`{"project": {"name": "test"}, "unknown_field": "value"}`,
		// Config with $schema (should not warn)
		`{"$schema": "config.schema.json", "project": {"name": "test"}}`,
		// Config with unknown target field
		`{"project": {"name": "test"}, "targets": {"go": {"type": "language", "title": "Go", "unknown_target_field": true}}}`,
		// Config with multiple unknown fields
		`{"project": {"name": "test"}, "foo": 1, "bar": 2, "baz": 3}`,
		// Valid complex config
		`{"project": {"name": "complex"}, "targets": {"a": {"type": "language", "title": "A"}, "b": {"type": "auxiliary", "title": "B"}}}`,
		// Edge case: empty targets
		`{"project": {"name": "test"}, "targets": {}}`,
		// Edge case: null targets
		`{"project": {"name": "test"}, "targets": null}`,
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// LoadWithWarnings should never panic
		cfg, warnings, err1 := LoadWithWarnings("fuzz.json", data)

		// Determinism check
		cfg2, warnings2, err2 := LoadWithWarnings("fuzz.json", data)

		// Both should either succeed or fail
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("non-deterministic error: first=%v, second=%v", err1, err2)
		}

		// If both succeed, results should be identical
		if err1 == nil && err2 == nil {
			if !reflect.DeepEqual(cfg, cfg2) {
				t.Errorf("non-deterministic config: first=%+v, second=%+v", cfg, cfg2)
			}
			// Warning order might differ for unknown fields in maps (non-deterministic iteration)
			// So we check length rather than exact equality
			if len(warnings) != len(warnings2) {
				t.Errorf("non-deterministic warning count: first=%d, second=%d", len(warnings), len(warnings2))
			}
		}

		// If parsing succeeded, verify invariants
		if err1 == nil && cfg != nil {
			// Project name should be unchanged from JSON
			var raw struct {
				Project struct {
					Name string `json:"name"`
				} `json:"project"`
			}
			if json.Unmarshal(data, &raw) == nil {
				if cfg.Project.Name != raw.Project.Name {
					t.Errorf("project name mismatch: got %q, want %q", cfg.Project.Name, raw.Project.Name)
				}
			}
		}
	})
}

// FuzzValidate tests the Validate function with arbitrary Config values.
// Run: go test -fuzz=FuzzValidate -fuzztime=30s ./internal/config
func FuzzValidate(f *testing.F) {
	// Seed corpus with JSON configs that will be unmarshaled and validated
	seeds := []string{
		// Valid minimal
		`{"project": {"name": "test"}}`,
		// Valid with targets
		`{"project": {"name": "test"}, "targets": {"go": {"type": "language", "title": "Go"}}}`,
		// Invalid: missing project name
		`{"project": {}}`,
		// Invalid: bad project name
		`{"project": {"name": "TEST"}}`,
		// Invalid: circular deps
		`{"project": {"name": "test"}, "targets": {"a": {"type": "language", "title": "A", "depends_on": ["b"]}, "b": {"type": "language", "title": "B", "depends_on": ["a"]}}}`,
		// Invalid: unknown toolchain
		`{"project": {"name": "test"}, "targets": {"go": {"type": "language", "title": "Go", "toolchain": "nonexistent"}}}`,
		// Valid with CI steps
		`{"project": {"name": "test"}, "ci": {"steps": [{"name": "Build", "target": "all", "command": "build"}]}}`,
		// Invalid: CI step with unknown dependency
		`{"project": {"name": "test"}, "ci": {"steps": [{"name": "Build", "target": "all", "command": "build", "depends_on": ["nonexistent"]}]}}`,
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return // Invalid JSON, skip validation test
		}

		// Validate should never panic
		warnings1, err1 := Validate(&cfg)

		// Determinism check
		warnings2, err2 := Validate(&cfg)

		// Both should either succeed or fail
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("non-deterministic error: first=%v, second=%v", err1, err2)
		}

		// Warning counts should match
		if len(warnings1) != len(warnings2) {
			t.Errorf("non-deterministic warning count: first=%d, second=%d", len(warnings1), len(warnings2))
		}
	})
}
