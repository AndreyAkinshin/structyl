package testparser

import "strings"

// Registry maps toolchain identifiers to their parsers.
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates a new parser registry with all built-in parsers.
func NewRegistry() *Registry {
	r := &Registry{
		parsers: make(map[string]Parser),
	}

	// Register all built-in parsers
	goParser := &GoParser{}
	cargoParser := &CargoParser{}
	pytestParser := &PytestParser{}
	dotnetParser := &DotnetParser{}
	bunParser := &BunParser{}
	denoParser := &DenoParser{}

	// Map toolchain identifiers to parsers
	r.parsers["go"] = goParser
	r.parsers["cargo"] = cargoParser
	r.parsers["rs"] = cargoParser
	r.parsers["rust"] = cargoParser
	r.parsers["python"] = pytestParser
	r.parsers["py"] = pytestParser
	r.parsers["uv"] = pytestParser
	r.parsers["poetry"] = pytestParser
	r.parsers["pytest"] = pytestParser
	r.parsers["dotnet"] = dotnetParser
	r.parsers["cs"] = dotnetParser
	r.parsers["csharp"] = dotnetParser
	r.parsers["bun"] = bunParser
	r.parsers["deno"] = denoParser

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
