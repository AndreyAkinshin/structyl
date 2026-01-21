// Package topsort provides topological sorting with cycle detection.
package topsort

import (
	"fmt"
	"sort"
)

// Graph represents a directed graph for topological sorting.
// The keys are node names, values are lists of dependencies (edges point to dependencies).
type Graph map[string][]string

// Sort performs topological sort on the graph, returning nodes in dependency order.
// Dependencies appear before dependents in the result.
// Returns an error if a cycle is detected or a dependency is undefined.
//
// The nodes parameter specifies which nodes to sort:
//   - nil: sort all nodes in the graph (sorted alphabetically before processing)
//   - empty slice: return empty result (no nodes to sort)
//   - non-empty slice: sort only those nodes and their transitive dependencies
func Sort(g Graph, nodes []string) ([]string, error) {
	if nodes == nil {
		nodes = make([]string, 0, len(g))
		for name := range g {
			nodes = append(nodes, name)
		}
		sort.Strings(nodes)
	}

	var result []string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected involving %q", name)
		}
		if visited[name] {
			return nil
		}

		deps, exists := g[name]
		if !exists {
			return fmt.Errorf("node %q not found in graph", name)
		}

		inStack[name] = true

		for _, dep := range deps {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visited[name] = true
		inStack[name] = false
		result = append(result, name)

		return nil
	}

	for _, name := range nodes {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Validate checks the graph for self-references, undefined dependencies, and cycles.
func Validate(g Graph) error {
	for name, deps := range g {
		for _, dep := range deps {
			if dep == name {
				return fmt.Errorf("%q depends on itself", name)
			}
			if _, ok := g[dep]; !ok {
				return fmt.Errorf("%q depends on undefined node %q", name, dep)
			}
		}
	}

	_, err := Sort(g, nil)
	return err
}
