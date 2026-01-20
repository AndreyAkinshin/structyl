package config

import (
	"fmt"
	"regexp"
)

// Validation limits.
const (
	maxProjectNameLength = 128
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

	if err := validateCI(cfg); err != nil {
		return nil, err
	}

	if err := validateTests(cfg); err != nil {
		return nil, err
	}

	return nil, nil
}

func validateCI(cfg *Config) error {
	if cfg.CI == nil || len(cfg.CI.Steps) == 0 {
		return nil
	}

	// Build set of defined step names and check for duplicates/empty names
	stepNames := make(map[string]bool)
	for i, step := range cfg.CI.Steps {
		// Step name must not be empty
		if step.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("ci.steps[%d].name", i),
				Message: "required",
			}
		}

		// Step name must be unique
		if stepNames[step.Name] {
			return &ValidationError{
				Field:   fmt.Sprintf("ci.%s.name", step.Name),
				Message: "duplicate step name",
			}
		}
		stepNames[step.Name] = true

		// Step target must not be empty
		if step.Target == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("ci.steps[%d].target", i),
				Message: "required",
			}
		}

		// Step target must reference a defined target (unless it's "all")
		if step.Target != "all" {
			if _, ok := cfg.Targets[step.Target]; !ok {
				return &ValidationError{
					Field:   fmt.Sprintf("ci.%s.target", step.Name),
					Message: fmt.Sprintf("references undefined target %q", step.Target),
				}
			}
		}
	}

	// Validate DependsOn references
	for _, step := range cfg.CI.Steps {
		for _, dep := range step.DependsOn {
			if !stepNames[dep] {
				return &ValidationError{
					Field:   fmt.Sprintf("ci.%s.depends_on", step.Name),
					Message: fmt.Sprintf("references undefined step %q", dep),
				}
			}
		}
	}

	return nil
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
			Message: "required",
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
			Message: "required",
		}
	}

	// Validate command definitions
	if err := validateCommands(name, target.Commands); err != nil {
		return err
	}

	return nil
}

// validateCommands checks that all command definitions use supported types.
// Supported: string, nil, []interface{} (command list).
// NOT supported: map/object form (e.g., {run, cwd, env}) - reject at load time.
func validateCommands(targetName string, commands map[string]interface{}) error {
	for cmdName, cmdDef := range commands {
		if err := validateCommandDef(targetName, cmdName, cmdDef); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandDef(targetName, cmdName string, cmdDef interface{}) error {
	switch v := cmdDef.(type) {
	case nil, string:
		// Valid types
		return nil
	case []interface{}:
		// Validate each element in the command list
		for i, elem := range v {
			switch elem.(type) {
			case string:
				// Valid - command reference
			default:
				return &ValidationError{
					Field:   fmt.Sprintf("targets.%s.commands.%s[%d]", targetName, cmdName, i),
					Message: fmt.Sprintf("command list elements must be strings, got %T", elem),
				}
			}
		}
		return nil
	case map[string]interface{}:
		// Object-form commands ({run, cwd, env}) are not implemented
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s.commands.%s", targetName, cmdName),
			Message: "object-form commands are not supported; use string or array syntax",
		}
	default:
		return &ValidationError{
			Field:   fmt.Sprintf("targets.%s.commands.%s", targetName, cmdName),
			Message: fmt.Sprintf("invalid command type %T; must be string, null, or array", cmdDef),
		}
	}
}

// ValidateProjectName checks if a project name is valid.
// Returns a ValidationError if the name is empty, too long (>128 chars),
// or doesn't match the required pattern.
func ValidateProjectName(name string) error {
	if name == "" {
		return &ValidationError{Field: "project.name", Message: "required"}
	}
	if len(name) > maxProjectNameLength {
		return &ValidationError{Field: "project.name", Message: fmt.Sprintf("must be %d characters or less", maxProjectNameLength)}
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
		return &ValidationError{Field: "target name", Message: "required"}
	}
	if !targetNamePattern.MatchString(name) {
		return &ValidationError{
			Field:   "target name",
			Message: "must match pattern ^[a-z][a-z0-9-]*$",
		}
	}
	return nil
}

// validateTests checks tests configuration for errors.
func validateTests(cfg *Config) error {
	if cfg.Tests == nil || cfg.Tests.Comparison == nil {
		return nil
	}
	c := cfg.Tests.Comparison

	// Validate tolerance_mode
	switch c.ToleranceMode {
	case "", "relative", "absolute", "ulp":
		// Valid values
	default:
		return &ValidationError{
			Field:   "tests.comparison.tolerance_mode",
			Message: fmt.Sprintf("must be \"relative\", \"absolute\", or \"ulp\", got %q", c.ToleranceMode),
		}
	}

	// Validate array_order
	switch c.ArrayOrder {
	case "", "strict", "unordered":
		// Valid values
	default:
		return &ValidationError{
			Field:   "tests.comparison.array_order",
			Message: fmt.Sprintf("must be \"strict\" or \"unordered\", got %q", c.ArrayOrder),
		}
	}

	return nil
}
