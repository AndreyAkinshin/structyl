package target

import (
	"fmt"
	"sort"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// Registry manages a collection of targets.
type Registry struct {
	targets map[string]Target
}

// NewRegistry creates a registry from configuration.
func NewRegistry(cfg *config.Config, rootDir string) (*Registry, error) {
	resolver, err := toolchain.NewResolver(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolchain resolver: %w", err)
	}

	r := &Registry{
		targets: make(map[string]Target),
	}

	for name, targetCfg := range cfg.Targets {
		target, err := NewTarget(name, targetCfg, rootDir, resolver)
		if err != nil {
			return nil, fmt.Errorf("target %q: %w", name, err)
		}
		r.targets[name] = target
	}

	// Validate dependencies
	if err := r.validateDependencies(); err != nil {
		return nil, err
	}

	return r, nil
}

// Get retrieves a target by name.
func (r *Registry) Get(name string) (Target, bool) {
	t, ok := r.targets[name]
	return t, ok
}

// All returns all targets sorted by name.
func (r *Registry) All() []Target {
	targets := make([]Target, 0, len(r.targets))
	for _, t := range r.targets {
		targets = append(targets, t)
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name() < targets[j].Name()
	})
	return targets
}

// ByType returns targets of a specific type sorted by name.
func (r *Registry) ByType(targetType TargetType) []Target {
	var targets []Target
	for _, t := range r.targets {
		if t.Type() == targetType {
			targets = append(targets, t)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name() < targets[j].Name()
	})
	return targets
}

// Languages returns all language targets.
func (r *Registry) Languages() []Target {
	return r.ByType(TypeLanguage)
}

// Auxiliary returns all auxiliary targets.
func (r *Registry) Auxiliary() []Target {
	return r.ByType(TypeAuxiliary)
}

// Names returns all target names sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.targets))
	for name := range r.targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// validateDependencies checks for undefined and circular dependencies.
func (r *Registry) validateDependencies() error {
	// Check for undefined dependencies
	for name, target := range r.targets {
		for _, dep := range target.DependsOn() {
			if dep == name {
				return fmt.Errorf("target %q depends on itself", name)
			}
			if _, ok := r.targets[dep]; !ok {
				return fmt.Errorf("target %q depends on undefined target %q", name, dep)
			}
		}
	}

	// Check for circular dependencies
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected involving target %q", name)
		}
		if visited[name] {
			return nil
		}

		visited[name] = true
		inStack[name] = true

		target := r.targets[name]
		for _, dep := range target.DependsOn() {
			if err := visit(dep); err != nil {
				return err
			}
		}

		inStack[name] = false
		return nil
	}

	for name := range r.targets {
		if err := visit(name); err != nil {
			return err
		}
	}

	return nil
}

// TopologicalOrder returns targets in dependency order.
func (r *Registry) TopologicalOrder() ([]Target, error) {
	var result []Target
	visited := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		visited[name] = true

		target := r.targets[name]
		for _, dep := range target.DependsOn() {
			if err := visit(dep); err != nil {
				return err
			}
		}

		result = append(result, target)
		return nil
	}

	// Visit in sorted order for deterministic output
	for _, name := range r.Names() {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}
