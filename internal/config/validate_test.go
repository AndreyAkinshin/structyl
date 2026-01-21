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

func TestValidateProjectName_LengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		length  int
		wantErr bool
		desc    string
	}{
		{127, false, "one below max"},
		{128, false, "exactly max"},
		{129, true, "one above max"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			name := ""
			for i := 0; i < tt.length; i++ {
				name += "a"
			}
			if len(name) != tt.length {
				t.Fatalf("test setup error: name length = %d, want %d", len(name), tt.length)
			}

			err := ValidateProjectName(name)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateProjectName(%d chars) = nil, want error", tt.length)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateProjectName(%d chars) = %v, want nil", tt.length, err)
			}
		})
	}
}

func TestValidateTargetName_Valid(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if err := ValidateTargetName(name); err != nil {
				t.Errorf("ValidateTargetName(%q) = %v, want nil", name, err)
			}
		})
	}
}

func TestValidateTargetName_Invalid(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if err := ValidateTargetName(tt.name); err == nil {
				t.Errorf("ValidateTargetName(%q) = nil, want error", tt.name)
			}
		})
	}
}

func TestValidate_MissingProjectName(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// Note: Target names allow consecutive hyphens and trailing hyphens per the regex
// pattern ^[a-z][a-z0-9-]*$. This is intentionally more permissive than project names.
func TestValidateTargetName_AllowsConsecutiveHyphens(t *testing.T) {
	t.Parallel()
	// Target names permit consecutive hyphens unlike project names
	if err := ValidateTargetName("my--target"); err != nil {
		t.Errorf("ValidateTargetName(\"my--target\") = %v, want nil (consecutive hyphens allowed)", err)
	}
}

func TestValidateTargetName_AllowsTrailingHyphen(t *testing.T) {
	t.Parallel()
	// Target names permit trailing hyphens unlike project names
	if err := ValidateTargetName("target-"); err != nil {
		t.Errorf("ValidateTargetName(\"target-\") = %v, want nil (trailing hyphen allowed)", err)
	}
}

func TestValidateTargetName_LengthBoundaries(t *testing.T) {
	t.Parallel()

	// Generate names of specific lengths (all 'a' characters)
	makeName := func(length int) string {
		name := make([]byte, length)
		for i := range name {
			name[i] = 'a'
		}
		return string(name)
	}

	tests := []struct {
		length  int
		wantErr bool
		desc    string
	}{
		{63, false, "one below max"},
		{64, false, "exactly max"},
		{65, true, "one above max"},
		{100, true, "well above max"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			name := makeName(tt.length)
			err := ValidateTargetName(name)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateTargetName(%d chars) = nil, want error", tt.length)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateTargetName(%d chars) = %v, want nil", tt.length, err)
			}
		})
	}
}

func TestValidate_EmptyTargetType(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	err := &ValidationError{
		Field:   "project.name",
		Message: "required",
	}

	expected := "project.name: required"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestValidate_ObjectFormCommand_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {
				Type:  "language",
				Title: "Rust",
				Commands: map[string]interface{}{
					"build": map[string]interface{}{
						"run": "cargo build",
						"cwd": "src",
					},
				},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for object-form command")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "targets.rs.commands.build" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "targets.rs.commands.build")
	}
}

func TestValidate_InvalidCommandListElement_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {
				Type:  "language",
				Title: "Rust",
				Commands: map[string]interface{}{
					"check": []interface{}{"lint", 123}, // 123 is invalid
				},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for invalid command list element")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "targets.rs.commands.check[1]" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "targets.rs.commands.check[1]")
	}
}

func TestValidate_ValidCommandTypes_Succeeds(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {
				Type:  "language",
				Title: "Rust",
				Commands: map[string]interface{}{
					"build":   "cargo build",                // string
					"restore": nil,                          // nil (disabled)
					"check":   []interface{}{"lint", "vet"}, // array of strings
				},
			},
		},
	}

	_, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for valid command types", err)
	}
}

func TestValidate_CIStepDependsOnUndefined_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "rs", Command: "build"},
				{Name: "test", Target: "rs", Command: "test", DependsOn: []string{"nonexistent"}},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for undefined CI step dependency")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "ci.test.depends_on" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "ci.test.depends_on")
	}
}

func TestValidate_CIStepDependsOnValid_Succeeds(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "rs", Command: "build"},
				{Name: "test", Target: "rs", Command: "test", DependsOn: []string{"build"}},
			},
		},
	}

	_, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for valid CI step dependencies", err)
	}
}

func TestValidate_CIStepTargetUndefined_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "nonexistent", Command: "build"},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for undefined CI step target")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "ci.build.target" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "ci.build.target")
	}
}

func TestValidate_CIStepEmptyName_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "", Target: "rs", Command: "build"},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for empty CI step name")
	}
}

func TestValidate_CIStepDuplicateName_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "rs", Command: "build"},
				{Name: "build", Target: "rs", Command: "test"}, // Duplicate name
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for duplicate CI step name")
	}
}

func TestValidate_CIStepEmptyTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "", Command: "build"},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for empty CI step target")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "ci.steps[0].target" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "ci.steps[0].target")
	}
}

func TestValidate_CIStepTargetAll_Succeeds(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build-all", Target: "all", Command: "build"},
			},
		},
	}

	_, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for target 'all'", err)
	}
}

func TestValidate_CIStepEmptyCommand_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Targets: map[string]TargetConfig{
			"rs": {Type: "language", Title: "Rust"},
		},
		CI: &CIConfig{
			Steps: []CIStep{
				{Name: "build", Target: "rs", Command: ""},
			},
		},
	}

	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for empty CI step command")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "ci.steps[0].command" {
		t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "ci.steps[0].command")
	}
}

func TestValidate_ToleranceMode_Valid(t *testing.T) {
	t.Parallel()
	validModes := []string{"", "relative", "absolute", "ulp"}
	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Project: ProjectConfig{Name: "myproject"},
				Tests: &TestsConfig{
					Comparison: &ComparisonConfig{
						ToleranceMode: mode,
					},
				},
			}
			_, err := Validate(cfg)
			if err != nil {
				t.Errorf("Validate() with tolerance_mode=%q error = %v, want nil", mode, err)
			}
		})
	}
}

func TestValidate_ToleranceMode_Invalid(t *testing.T) {
	t.Parallel()
	invalidModes := []string{"relativ", "RELATIVE", "percent", "unknown"}
	for _, mode := range invalidModes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Project: ProjectConfig{Name: "myproject"},
				Tests: &TestsConfig{
					Comparison: &ComparisonConfig{
						ToleranceMode: mode,
					},
				},
			}
			_, err := Validate(cfg)
			if err == nil {
				t.Errorf("Validate() with tolerance_mode=%q expected error, got nil", mode)
				return
			}
			valErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}
			if valErr.Field != "tests.comparison.tolerance_mode" {
				t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "tests.comparison.tolerance_mode")
			}
		})
	}
}

func TestValidate_ArrayOrder_Valid(t *testing.T) {
	t.Parallel()
	validOrders := []string{"", "strict", "unordered"}
	for _, order := range validOrders {
		t.Run(order, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Project: ProjectConfig{Name: "myproject"},
				Tests: &TestsConfig{
					Comparison: &ComparisonConfig{
						ArrayOrder: order,
					},
				},
			}
			_, err := Validate(cfg)
			if err != nil {
				t.Errorf("Validate() with array_order=%q error = %v, want nil", order, err)
			}
		})
	}
}

func TestValidate_ArrayOrder_Invalid(t *testing.T) {
	t.Parallel()
	invalidOrders := []string{"STRICT", "ordered", "random", "unknown"}
	for _, order := range invalidOrders {
		t.Run(order, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Project: ProjectConfig{Name: "myproject"},
				Tests: &TestsConfig{
					Comparison: &ComparisonConfig{
						ArrayOrder: order,
					},
				},
			}
			_, err := Validate(cfg)
			if err == nil {
				t.Errorf("Validate() with array_order=%q expected error, got nil", order)
				return
			}
			valErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}
			if valErr.Field != "tests.comparison.array_order" {
				t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, "tests.comparison.array_order")
			}
		})
	}
}

func TestValidate_Tests_NilComparison_Succeeds(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Project: ProjectConfig{Name: "myproject"},
		Tests: &TestsConfig{
			Directory: "tests",
		},
	}
	_, err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() with nil comparison error = %v, want nil", err)
	}
}

// TestValidate_TargetNames_CaseSensitive verifies that target names differing
// only by case are both valid (the validator doesn't reject case variants).
// Note: Target names must be lowercase per the validation regex, so "Rs" and "RS"
// are invalid anyway. This test documents the expected behavior.
func TestValidate_TargetNames_CaseSensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets map[string]TargetConfig
		wantErr bool
	}{
		{
			name: "lowercase_valid",
			targets: map[string]TargetConfig{
				"rs": {Type: "language", Title: "Rust"},
			},
			wantErr: false,
		},
		{
			name: "uppercase_invalid",
			targets: map[string]TargetConfig{
				"RS": {Type: "language", Title: "Rust"},
			},
			wantErr: true,
		},
		{
			name: "mixed_case_invalid",
			targets: map[string]TargetConfig{
				"Rs": {Type: "language", Title: "Rust"},
			},
			wantErr: true,
		},
		{
			name: "similar_lowercase_names_valid",
			targets: map[string]TargetConfig{
				"rs":    {Type: "language", Title: "Rust"},
				"rs-v2": {Type: "language", Title: "Rust v2"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Project: ProjectConfig{Name: "myproject"},
				Targets: tt.targets,
			}
			_, err := Validate(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for targets %v, got nil", tt.targets)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}
