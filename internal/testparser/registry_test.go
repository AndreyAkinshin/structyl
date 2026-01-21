package testparser

import "testing"

func TestRegistry(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	tests := []struct {
		toolchain    string
		expectedName string
	}{
		{"go", "go"},
		{"cargo", "cargo"},
		{"rs", "cargo"},
		{"rust", "cargo"},
		{"python", "pytest"},
		{"py", "pytest"},
		{"uv", "pytest"},
		{"poetry", "pytest"},
		{"pytest", "pytest"},
		{"dotnet", "dotnet"},
		{"cs", "dotnet"},
		{"csharp", "dotnet"},
		{"bun", "bun"},
		{"deno", "deno"},
	}

	for _, tt := range tests {
		t.Run(tt.toolchain, func(t *testing.T) {
			t.Parallel()
			parser := registry.GetParser(tt.toolchain)
			if parser == nil {
				t.Errorf("GetParser(%s): got nil, want parser", tt.toolchain)
				return
			}
			if parser.Name() != tt.expectedName {
				t.Errorf("GetParser(%s).Name(): got %s, want %s", tt.toolchain, parser.Name(), tt.expectedName)
			}
		})
	}
}

func TestRegistryUnknownToolchain(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	parser := registry.GetParser("unknown")
	if parser != nil {
		t.Errorf("GetParser(unknown): got parser, want nil")
	}
}

func TestRegistryGetParserForTask(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	tests := []struct {
		taskName     string
		expectedName string
		expectNil    bool
	}{
		{"test:go", "go", false},
		{"test:rs", "cargo", false},
		{"test:py", "pytest", false},
		{"ci:test:go", "go", false},
		{"build:go", "go", false},
		{"test", "", true},         // No toolchain suffix
		{"singleword", "", true},   // No colon
		{"test:unknown", "", true}, // Unknown toolchain
	}

	for _, tt := range tests {
		t.Run(tt.taskName, func(t *testing.T) {
			t.Parallel()
			parser := registry.GetParserForTask(tt.taskName)
			if tt.expectNil {
				if parser != nil {
					t.Errorf("GetParserForTask(%s): got parser, want nil", tt.taskName)
				}
			} else {
				if parser == nil {
					t.Errorf("GetParserForTask(%s): got nil, want parser", tt.taskName)
					return
				}
				if parser.Name() != tt.expectedName {
					t.Errorf("GetParserForTask(%s).Name(): got %s, want %s", tt.taskName, parser.Name(), tt.expectedName)
				}
			}
		})
	}
}

func TestRegistryRegisterParser(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	// Register a custom parser
	customParser := &GoParser{} // Using GoParser as a stand-in
	registry.RegisterParser("custom", customParser)

	parser := registry.GetParser("custom")
	if parser == nil {
		t.Errorf("GetParser(custom): got nil after registration")
	}
}

func TestRegistryCaseInsensitive(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	tests := []string{"Go", "GO", "go", "gO"}
	for _, tc := range tests {
		parser := registry.GetParser(tc)
		if parser == nil {
			t.Errorf("GetParser(%s): got nil, want parser", tc)
		}
	}
}
