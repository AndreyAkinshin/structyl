package testparser

import "strings"

// Registry maps toolchain identifiers to their parsers.
type Registry struct {
	parsers map[string]Parser
}

// builtinParsers defines all built-in parsers and their toolchain aliases.
// Each parser is instantiated once and shared across all its aliases.
var builtinParsers = []struct {
	parser  Parser
	aliases []string
}{
	{&GoParser{}, []string{"go"}},
	{&CargoParser{}, []string{"cargo", "rs", "rust"}},
	{&PytestParser{}, []string{"python", "py", "uv", "poetry", "pytest"}},
	{&DotnetParser{}, []string{"dotnet", "cs", "csharp"}},
	{&BunParser{}, []string{"bun"}},
	{&DenoParser{}, []string{"deno"}},
}

// NewRegistry creates a new parser registry with all built-in parsers.
// Note: Aliases in builtinParsers must be unique; duplicate aliases will
// silently overwrite earlier registrations. This is validated by tests.
func NewRegistry() *Registry {
	r := &Registry{
		parsers: make(map[string]Parser),
	}

	for _, entry := range builtinParsers {
		for _, alias := range entry.aliases {
			r.parsers[alias] = entry.parser
		}
	}

	return r
}

// GetParser returns a parser for the given toolchain identifier.
// Returns nil if no parser is found.
func (r *Registry) GetParser(toolchain string) Parser {
	return r.parsers[strings.ToLower(toolchain)]
}

// GetParserForTask returns a parser based on task name.
// Task names like "test:go" will return the Go parser.
func (r *Registry) GetParserForTask(taskName string) Parser {
	// Extract toolchain from task name (e.g., "test:go" -> "go")
	parts := strings.Split(taskName, ":")
	if len(parts) < 2 {
		return nil
	}

	// The toolchain is the last part of the task name
	toolchain := parts[len(parts)-1]
	return r.GetParser(toolchain)
}

// RegisterParser adds a custom parser for a toolchain.
func (r *Registry) RegisterParser(toolchain string, parser Parser) {
	r.parsers[strings.ToLower(toolchain)] = parser
}
