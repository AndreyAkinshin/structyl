package config

import "testing"

func TestValidateProjectName_Valid(t *testing.T) {
	t.Parallel()
	tests := []string{
		"a",                     // minimum length
		"myproject",             // simple name
		"my-project",            // single hyphen
		"test-project-123",      // multiple hyphens
		"a1",                    // letter + digit
		"abc123",                // letters + digits
		"a-b-c-d",               // multiple single-char segments
		"project1-version2-rc3", // complex multi-segment
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateProjectName(name); err != nil {
				t.Errorf("ValidateProjectName(%q) = %v, want nil", name, err)
			}
		})
	}
}

func TestValidateProjectName_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		desc string
	}{
		{"", "empty"},
		{"1abc", "starts with digit"},
		{"ABC", "uppercase"},
		{"my_project", "underscore"},
		{"my--project", "consecutive hyphens"},
		{"my-project-", "trailing hyphen"},
		{"-myproject", "leading hyphen"},
		{"my project", "space"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			if err := ValidateProjectName(tt.name); err == nil {
				t.Errorf("ValidateProjectName(%q) = nil, want error", tt.name)
			}
		})
	}
}

func TestValidateProjectName_ExactlyMaxLength(t *testing.T) {
	// Exactly 128 characters - should be valid
	name := "a"
	for i := 1; i < 128; i++ {
		name += "a"
	}
	if len(name) != 128 {
		t.Fatalf("test setup error: name length = %d, want 128", len(name))
	}
	if err := ValidateProjectName(name); err != nil {
		t.Errorf("ValidateProjectName() = %v, want nil for exactly 128 chars", err)
	}
}

func TestValidateProjectName_TooLong(t *testing.T) {
	// 129 characters - should be invalid
	name := "a"
	for i := 1; i < 129; i++ {
		name += "a"
	}
	if len(name) != 129 {
		t.Fatalf("test setup error: name length = %d, want 129", len(name))
	}
	if err := ValidateProjectName(name); err == nil {
		t.Errorf("ValidateProjectName() = nil, want error for name > 128 chars")
	}
}

func TestValidateTargetName_Valid(t *testing.T) {
	tests := []string{
		"a",
		"cs",
		"py",
		"go",
		"my-target",
		"target123",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateTargetName(name); err != nil {
				t.Errorf("ValidateTargetName(%q) = %v, want nil", name, err)
			}
		})
	}
}

func TestValidateTargetName_Invalid(t *testing.T) {
	tests := []struct {
		name string
		desc string
	}{
		{"", "empty"},
		{"1abc", "starts with digit"},
		{"ABC", "uppercase"},
		{"my_target", "underscore"},
		{"-target", "leading hyphen"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := ValidateTargetName(tt.name); err == nil {
				t.Errorf("ValidateTargetName(%q) = nil, want error", tt.name)
			}
		})
	}
}

func TestValidate_MissingProjectName(t *testing.T) {
	cfg := &Config{}
	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for missing project.name")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Validate() error type = %T, want *ValidationError", err)
	}
	if ve.Field != "project.name" {
		t.Errorf("ValidationError.Field = %q, want %q", ve.Field, "project.name")
	}
}

func TestValidate_InvalidTargetType(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"cs": {
				Type:  "invalid",
				Title: "C#",
			},
		},
	}
	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for invalid target type")
	}
}

func TestValidate_MissingTargetTitle(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"cs": {
				Type: "language",
			},
		},
	}
	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for missing target title")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"cs": {
				Type:  "language",
				Title: "C#",
			},
			"py": {
				Type:  "language",
				Title: "Python",
			},
		},
	}
	warnings, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Validate() warnings = %v, want empty", warnings)
	}
}

func TestValidateProjectName_Boundary127Chars(t *testing.T) {
	// Exactly 127 characters - should be valid (one below max)
	name := ""
	for i := 0; i < 127; i++ {
		name += "a"
	}
	if len(name) != 127 {
		t.Fatalf("test setup error: name length = %d, want 127", len(name))
	}
	if err := ValidateProjectName(name); err != nil {
		t.Errorf("ValidateProjectName() = %v, want nil for 127 chars", err)
	}
}

// Note: Target names allow consecutive hyphens and trailing hyphens per the regex
// pattern ^[a-z][a-z0-9-]*$. This is intentionally more permissive than project names.
func TestValidateTargetName_AllowsConsecutiveHyphens(t *testing.T) {
	// Target names permit consecutive hyphens unlike project names
	if err := ValidateTargetName("my--target"); err != nil {
		t.Errorf("ValidateTargetName(\"my--target\") = %v, want nil (consecutive hyphens allowed)", err)
	}
}

func TestValidateTargetName_AllowsTrailingHyphen(t *testing.T) {
	// Target names permit trailing hyphens unlike project names
	if err := ValidateTargetName("target-"); err != nil {
		t.Errorf("ValidateTargetName(\"target-\") = %v, want nil (trailing hyphen allowed)", err)
	}
}

func TestValidate_EmptyTargetType(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"cs": {
				Type:  "", // Empty type should error
				Title: "C#",
			},
		},
	}
	// Empty type should cause an error (type is required)
	_, err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for empty target type")
	}
}

func TestValidate_AuxiliaryTargetType(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"img": {
				Type:  "auxiliary",
				Title: "Images",
			},
		},
	}
	warnings, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Validate() warnings = %v, want empty", warnings)
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "project.name",
		Message: "is required",
	}

	expected := "project.name: is required"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}
