package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// LoadWithWarnings reads a config file and returns any unknown field warnings.
func LoadWithWarnings(path string, data []byte) (*Config, []string, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Detect unknown fields
	warnings := detectUnknownFields(data)

	return &cfg, warnings, nil
}

// detectUnknownFields compares raw JSON with known struct fields.
// Note: Since this is called after successful Config parsing, a parse failure
// here would indicate an unexpected internal inconsistency.
func detectUnknownFields(data []byte) []string {
	var warnings []string

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// This should never happen since the data was already parsed successfully.
		// Return a warning so the condition is visible rather than silently ignored.
		return []string{"internal: failed to re-parse config for unknown field detection"}
	}

	knownTopLevel := getJSONFields(reflect.TypeOf(Config{}))
	for key := range raw {
		if key == "$schema" {
			continue // $schema is explicitly allowed and ignored
		}
		if !knownTopLevel[key] {
			warnings = append(warnings, fmt.Sprintf("unknown field %q at root level (ignored)", key))
		}
	}

	// Check nested unknown fields in targets
	if targetsRaw, ok := raw["targets"]; ok {
		targetWarnings := checkTargetsUnknownFields(targetsRaw)
		warnings = append(warnings, targetWarnings...)
	}

	return warnings
}

func checkTargetsUnknownFields(data json.RawMessage) []string {
	var warnings []string

	var targets map[string]json.RawMessage
	if err := json.Unmarshal(data, &targets); err != nil {
		// Should not happen since Config.Targets parsed successfully.
		return []string{"internal: failed to re-parse targets for unknown field detection"}
	}

	knownTargetFields := getJSONFields(reflect.TypeOf(TargetConfig{}))
	for targetName, targetRaw := range targets {
		var targetFields map[string]json.RawMessage
		if err := json.Unmarshal(targetRaw, &targetFields); err != nil {
			continue
		}
		for key := range targetFields {
			if !knownTargetFields[key] {
				warnings = append(warnings, fmt.Sprintf("unknown field %q in target %q (ignored)", key, targetName))
			}
		}
	}

	return warnings
}

// getJSONFields returns a map of known JSON field names for a struct type.
func getJSONFields(t reflect.Type) map[string]bool {
	fields := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// Extract field name from tag (before comma)
		name := strings.Split(tag, ",")[0]
		if name != "" {
			fields[name] = true
		}
	}
	return fields
}
