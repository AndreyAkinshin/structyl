package toolchain

import (
	"fmt"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// Resolver handles toolchain resolution including custom toolchains and extension.
type Resolver struct {
	custom map[string]*Toolchain
}

// NewResolver creates a resolver with custom toolchains from configuration.
func NewResolver(cfg *config.Config) (*Resolver, error) {
	r := &Resolver{
		custom: make(map[string]*Toolchain),
	}

	// Process custom toolchains from config
	for name, tcConfig := range cfg.Toolchains {
		tc, err := r.buildCustomToolchain(name, tcConfig)
		if err != nil {
			return nil, fmt.Errorf("toolchain %q: %w", name, err)
		}
		r.custom[name] = tc
	}

	return r, nil
}

// Resolve gets a toolchain by name, checking custom toolchains first.
func (r *Resolver) Resolve(name string) (*Toolchain, error) {
	// Check custom toolchains first
	if tc, ok := r.custom[name]; ok {
		return tc, nil
	}

	// Check built-in toolchains
	if tc, ok := builtinToolchains[name]; ok {
		return tc, nil
	}

	return nil, fmt.Errorf("unknown toolchain: %q", name)
}

// Exists checks if a toolchain exists (custom or built-in).
func (r *Resolver) Exists(name string) bool {
	if _, ok := r.custom[name]; ok {
		return true
	}
	_, ok := builtinToolchains[name]
	return ok
}

// buildCustomToolchain creates a Toolchain from configuration.
func (r *Resolver) buildCustomToolchain(name string, cfg config.ToolchainConfig) (*Toolchain, error) {
	tc := &Toolchain{
		Name:     name,
		Extends:  cfg.Extends,
		Commands: make(map[string]interface{}),
	}

	// If extending another toolchain, copy its commands first
	if cfg.Extends != "" {
		base, err := r.resolveBase(cfg.Extends)
		if err != nil {
			return nil, fmt.Errorf("extends %q: %w", cfg.Extends, err)
		}
		// Copy base commands
		for k, v := range base.Commands {
			tc.Commands[k] = v
		}
	}

	// Override/add commands from config
	for k, v := range cfg.Commands {
		tc.Commands[k] = v
	}

	return tc, nil
}

// resolveBase resolves a base toolchain for extension.
func (r *Resolver) resolveBase(name string) (*Toolchain, error) {
	// Check built-in first (most common case)
	if tc, ok := builtinToolchains[name]; ok {
		return tc, nil
	}

	// Check custom toolchains (for chained extensions)
	if tc, ok := r.custom[name]; ok {
		return tc, nil
	}

	return nil, fmt.Errorf("base toolchain %q not found", name)
}

// ValidateTargetToolchains validates all target toolchain references.
func (r *Resolver) ValidateTargetToolchains(targets map[string]config.TargetConfig) error {
	for name, target := range targets {
		if target.Toolchain == "" {
			continue // Auto-detect or no commands
		}
		if !r.Exists(target.Toolchain) {
			return fmt.Errorf("target %q references unknown toolchain %q", name, target.Toolchain)
		}
	}
	return nil
}

// GetResolvedCommands returns the resolved commands for a target.
// It merges toolchain commands with target-specific overrides.
func (r *Resolver) GetResolvedCommands(target config.TargetConfig) (map[string]interface{}, error) {
	commands := make(map[string]interface{})

	// Get toolchain commands if specified
	if target.Toolchain != "" {
		tc, err := r.Resolve(target.Toolchain)
		if err != nil {
			return nil, err
		}
		for k, v := range tc.Commands {
			commands[k] = v
		}
	}

	// Override with target-specific commands
	for k, v := range target.Commands {
		commands[k] = v
	}

	return commands, nil
}
