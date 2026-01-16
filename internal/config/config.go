package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses a config.json configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadWithDefaults reads a config file and applies default values.
func LoadWithDefaults(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	applyDefaults(cfg)
	return cfg, nil
}

// LoadAndValidate reads a config file, applies defaults, validates, and returns warnings.
func LoadAndValidate(path string) (*Config, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, unknownWarnings, err := LoadWithWarnings(path, data)
	if err != nil {
		return nil, nil, err
	}

	applyDefaults(cfg)

	validationWarnings, err := Validate(cfg)
	if err != nil {
		return nil, append(unknownWarnings, validationWarnings...), err
	}

	allWarnings := append(unknownWarnings, validationWarnings...)
	return cfg, allWarnings, nil
}
