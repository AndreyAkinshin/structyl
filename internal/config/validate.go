package config

import (
	"fmt"
	"regexp"
)

// Validation patterns from the specification.
var (
	// Project name: must start with lowercase letter, may contain lowercase, digits, hyphens.
	// Hyphens must not be consecutive or trailing.
	projectNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

	// Target name: lowercase letters, digits, and hyphens.
	targetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks a configuration for errors and returns warnings for non-fatal issues.
// Note: warnings are reserved for future use (deprecated fields, migration hints, etc.)
func Validate(cfg *Config) (warnings []string, err error) {
	if err := validateProject(cfg); err != nil {
		return nil, err
	}

	if err := validateTargets(cfg); err != nil {
		return nil, err
	}

	return nil, nil
}

func validateProject(cfg *Config) error {
	return ValidateProjectName(cfg.Project.Name)
}

func validateTargets(cfg *Config) error {
	for name, target := range cfg.Targets {
		if err := validateTargetName(name); err != nil {
			return err
		}
		if err := validateTargetConfig(name, target); err != nil {
			return err
		}
	}
	return nil
}

func validateTargetName(name string) error {
	if !targetNamePattern.MatchString(name) {
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s", name),
			Message: "target name must match pattern ^[a-z][a-z0-9-]*$ (lowercase letters, digits, hyphens)",
		}
	}
	return nil
}

func validateTargetConfig(name string, target TargetConfig) error {
	if target.Type == "" {
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s.type", name),
			Message: "is required",
		}
	}

	if target.Type != "language" && target.Type != "auxiliary" {
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s.type", name),
			Message: `must be "language" or "auxiliary"`,
		}
	}

	if target.Title == "" {
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s.title", name),
			Message: "is required",
		}
	}

	return nil
}

// ValidateProjectName checks if a project name is valid.
// Returns a ValidationError if the name is empty, too long (>128 chars),
// or doesn't match the required pattern.
func ValidateProjectName(name string) error {
	if name == "" {
		return &ValidationError{Field: "project.name", Message: "is required"}
	}
	if len(name) > 128 {
		return &ValidationError{Field: "project.name", Message: "must be 128 characters or less"}
	}
	if !projectNamePattern.MatchString(name) {
		return &ValidationError{
			Field:   "project.name",
			Message: "must match pattern ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ (lowercase letters, digits, non-consecutive hyphens)",
		}
	}
	return nil
}

// ValidateTargetName checks if a target name is valid.
func ValidateTargetName(name string) error {
	if name == "" {
		return &ValidationError{Field: "target name", Message: "is required"}
	}
	if !targetNamePattern.MatchString(name) {
		return &ValidationError{
			Field:   "target name",
			Message: "must match pattern ^[a-z][a-z0-9-]*$",
		}
	}
	return nil
}
