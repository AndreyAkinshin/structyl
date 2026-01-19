package target

import (
	"fmt"
	"sort"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
	"github.com/AndreyAkinshin/structyl/internal/topsort"
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
	return topsort.Validate(r.buildGraph())
}

// buildGraph creates a topsort.Graph from the target registry.
func (r *Registry) buildGraph() topsort.Graph {
	g := make(topsort.Graph, len(r.targets))
	for name, target := range r.targets {
		g[name] = target.DependsOn()
	}
	return g
}

// TopologicalOrder returns targets in dependency order.
func (r *Registry) TopologicalOrder() ([]Target, error) {
	// Sort in deterministic order by using sorted names
	sortedNames, err := topsort.Sort(r.buildGraph(), r.Names())
	if err != nil {
		return nil, err
	}

	result := make([]Target, len(sortedNames))
	for i, name := range sortedNames {
		result[i] = r.targets[name]
	}
	return result, nil
}
