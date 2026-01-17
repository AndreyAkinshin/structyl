// Package toolchain provides built-in toolchain presets and command mappings.
package toolchain

// Toolchain represents a set of command mappings for a build ecosystem.
type Toolchain struct {
	Name     string
	Extends  string
	Commands map[string]interface{}
}

// GetCommand returns the command definition for a given command name.
// Commands can be:
// - string: a shell command
// - []string: a list of other commands to run in sequence
// - nil: command is not supported
func (t *Toolchain) GetCommand(name string) (interface{}, bool) {
	cmd, ok := t.Commands[name]
	return cmd, ok
}

// HasCommand checks if a command is defined (even if nil).
func (t *Toolchain) HasCommand(name string) bool {
	_, ok := t.Commands[name]
	return ok
}

// Get retrieves a toolchain by name from built-in toolchains.
// Deprecated: Use GetFromConfig for loaded toolchains configuration.
func Get(name string) (*Toolchain, bool) {
	tc, ok := builtinToolchains[name]
	return tc, ok
}

// GetFromConfig retrieves a toolchain by name using the loaded configuration.
// Falls back to built-in toolchains if loaded is nil or toolchain not found.
func GetFromConfig(name string, loaded *ToolchainsFile) (*Toolchain, bool) {
	// Try loaded config first
	if tc, ok := GetToolchainFromConfig(name, loaded); ok {
		return tc, true
	}

	// Fallback to built-in
	return Get(name)
}

// List returns a list of all built-in toolchain names.
func List() []string {
	names := make([]string, 0, len(builtinToolchains))
	for name := range builtinToolchains {
		names = append(names, name)
	}
	return names
}

// IsBuiltin checks if a toolchain name is a built-in toolchain.
func IsBuiltin(name string) bool {
	_, ok := builtinToolchains[name]
	return ok
}
