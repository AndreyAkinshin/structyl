package topsort

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/quick"
)

// indexOfIn returns the index of s in slice, or -1 if not found.
// This helper avoids duplicating the indexOf closure in multiple tests.
func indexOfIn(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

func TestSort_Empty(t *testing.T) {
	t.Parallel()
	g := Graph{}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}
	if len(result) != 0 {
		t.Errorf("Sort() = %v, want empty", result)
	}
}

func TestSort_SingleNode(t *testing.T) {
	t.Parallel()
	g := Graph{"a": nil}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(result, []string{"a"}) {
		t.Errorf("Sort() = %v, want [a]", result)
	}
}

func TestSort_LinearChain(t *testing.T) {
	t.Parallel()
	// c depends on b, b depends on a
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": {"b"},
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}

	// Verify order: a before b, b before c
	if indexOfIn(result, "a") >= indexOfIn(result, "b") {
		t.Errorf("Sort() a should come before b: %v", result)
	}
	if indexOfIn(result, "b") >= indexOfIn(result, "c") {
		t.Errorf("Sort() b should come before c: %v", result)
	}
}

func TestSort_Diamond(t *testing.T) {
	t.Parallel()
	// d depends on b and c, b and c depend on a
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": {"a"},
		"d": {"b", "c"},
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}

	// a must come before b and c
	if indexOfIn(result, "a") >= indexOfIn(result, "b") || indexOfIn(result, "a") >= indexOfIn(result, "c") {
		t.Errorf("Sort() a should come before b and c: %v", result)
	}
	// b and c must come before d
	if indexOfIn(result, "b") >= indexOfIn(result, "d") || indexOfIn(result, "c") >= indexOfIn(result, "d") {
		t.Errorf("Sort() b and c should come before d: %v", result)
	}
}

func TestSort_Cycle(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"b"},
		"b": {"a"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for cycle, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
	// Verify error identifies at least one cycle participant for debugging
	if !strings.Contains(errStr, "a") && !strings.Contains(errStr, "b") {
		t.Errorf("Sort() error should identify cycle node, got: %v", err)
	}
}

func TestSort_LongCycle(t *testing.T) {
	t.Parallel()
	// 4-node cycle: a -> b -> c -> d -> a
	g := Graph{
		"a": {"b"},
		"b": {"c"},
		"c": {"d"},
		"d": {"a"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for 4-node cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_CycleWithBranch(t *testing.T) {
	t.Parallel()
	// Graph with a cycle and a branch:
	//   a -> b -> c -> d -> b (cycle: b -> c -> d -> b)
	//        |
	//        v
	//        e
	g := Graph{
		"a": {"b"},
		"b": {"c", "e"},
		"c": {"d"},
		"d": {"b"},
		"e": nil,
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for cycle with branch, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_SelfReference(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"a"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for self-reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_UndefinedDependency(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"undefined"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for undefined dependency, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Sort() error = %v, want to contain 'not found'", err)
	}
}

func TestSort_SelectedNodes(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": {"a"},
		"d": nil,
	}

	// Sort only b and its dependencies
	result, err := Sort(g, []string{"b"})
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}

	// Result should contain a and b only (d and c not reachable from b)
	sort.Strings(result)
	if !reflect.DeepEqual(result, []string{"a", "b"}) {
		t.Errorf("Sort() = %v, want [a, b]", result)
	}
}

func TestSort_SelectedNodes_InvalidReference(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": nil,
		"b": {"a"},
	}

	// Sort with a selected node that doesn't exist in the graph
	_, err := Sort(g, []string{"nonexistent"})
	if err == nil {
		t.Error("Sort() expected error for nonexistent selected node, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Sort() error = %v, want to contain 'not found'", err)
	}
}

func TestSort_NilVsEmptyNodes(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": nil,
	}

	// nil nodes: sort all nodes in graph
	resultNil, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort(nil) error = %v, want nil", err)
	}
	if len(resultNil) != 3 {
		t.Errorf("Sort(nil) returned %d nodes, want 3", len(resultNil))
	}

	// empty slice: return empty result
	resultEmpty, err := Sort(g, []string{})
	if err != nil {
		t.Errorf("Sort([]) error = %v, want nil", err)
	}
	if len(resultEmpty) != 0 {
		t.Errorf("Sort([]) = %v, want empty slice", resultEmpty)
	}
}

func TestValidate_Valid(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": {"a", "b"},
	}
	if err := Validate(g); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_SelfReference(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"a"},
	}
	err := Validate(g)
	if err == nil {
		t.Error("Validate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "itself") {
		t.Errorf("Validate() error = %v, want to contain 'itself'", err)
	}
}

func TestValidate_UndefinedDependency(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"missing"},
	}
	err := Validate(g)
	if err == nil {
		t.Error("Validate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "undefined") {
		t.Errorf("Validate() error = %v, want to contain 'undefined'", err)
	}
}

func TestValidate_Cycle(t *testing.T) {
	t.Parallel()
	g := Graph{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}
	err := Validate(g)
	if err == nil {
		t.Error("Validate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Validate() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_DisconnectedWithCycle(t *testing.T) {
	t.Parallel()
	// Graph with two disconnected components:
	// Component 1: a -> b (valid)
	// Component 2: c -> d -> c (cycle)
	g := Graph{
		"a": {"b"},
		"b": nil,
		"c": {"d"},
		"d": {"c"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for cycle in disconnected component, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
}

func TestValidate_DisconnectedWithCycle(t *testing.T) {
	t.Parallel()
	// Graph with two disconnected components where one has a cycle
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": {"d"},
		"d": {"e"},
		"e": {"c"}, // cycle in second component
	}
	err := Validate(g)
	if err == nil {
		t.Error("Validate() expected error for cycle in disconnected component, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Validate() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_MultipleDisconnectedComponents(t *testing.T) {
	t.Parallel()
	// Graph with three disconnected components, all valid
	g := Graph{
		"a": nil,
		"b": {"a"},
		"c": nil,
		"d": {"c"},
		"e": nil,
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}
	if len(result) != 5 {
		t.Errorf("Sort() returned %d nodes, want 5", len(result))
	}

	// Verify ordering within components
	// a must come before b
	if indexOfIn(result, "a") >= indexOfIn(result, "b") {
		t.Errorf("Sort() a should come before b: %v", result)
	}
	// c must come before d
	if indexOfIn(result, "c") >= indexOfIn(result, "d") {
		t.Errorf("Sort() c should come before d: %v", result)
	}
}

func TestSort_DuplicateDependencies(t *testing.T) {
	t.Parallel()
	// Graph with duplicate dependencies: a depends on b twice
	g := Graph{
		"a": {"b", "b"},
		"b": nil,
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}
	// Should still produce valid ordering despite duplicates
	if len(result) != 2 {
		t.Errorf("Sort() returned %d nodes, want 2", len(result))
	}

	// b must come before a
	if indexOfIn(result, "b") >= indexOfIn(result, "a") {
		t.Errorf("Sort() b should come before a: %v", result)
	}
}

func TestSort_EmptyNodeName(t *testing.T) {
	t.Parallel()
	// Graph with empty node name
	g := Graph{
		"":  nil,
		"a": {""},
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Errorf("Sort() error = %v, want nil", err)
	}
	// Empty string is a valid node name (though unusual)
	if len(result) != 2 {
		t.Errorf("Sort() returned %d nodes, want 2", len(result))
	}
}

func TestSort_LargeGraph(t *testing.T) {
	t.Parallel()
	// Build a linear chain of 100 nodes: n000 <- n001 <- n002 <- ... <- n099
	const nodeCount = 100
	g := make(Graph, nodeCount)

	nodeName := func(i int) string {
		return fmt.Sprintf("n%03d", i)
	}

	for i := 0; i < nodeCount; i++ {
		name := nodeName(i)
		if i == 0 {
			g[name] = nil
		} else {
			g[name] = []string{nodeName(i - 1)}
		}
	}

	result, err := Sort(g, nil)
	if err != nil {
		t.Fatalf("Sort() error = %v, want nil", err)
	}

	if len(result) != nodeCount {
		t.Errorf("Sort() returned %d nodes, want %d", len(result), nodeCount)
	}

	// Verify ordering: each node should appear after its dependency
	indexOf := make(map[string]int, nodeCount)
	for i, name := range result {
		indexOf[name] = i
	}

	for name, deps := range g {
		for _, dep := range deps {
			if indexOf[dep] >= indexOf[name] {
				t.Errorf("Sort() dependency %s should come before %s", dep, name)
			}
		}
	}
}

func TestSort_Deterministic(t *testing.T) {
	t.Parallel()
	// Graph with multiple valid orderings (a, b, c are independent)
	g := Graph{
		"a": nil,
		"b": nil,
		"c": nil,
		"d": {"a", "b", "c"},
	}

	// Run Sort multiple times and verify identical results
	result1, err1 := Sort(g, nil)
	if err1 != nil {
		t.Fatalf("Sort() error = %v", err1)
	}

	for i := 0; i < 10; i++ {
		result2, err2 := Sort(g, nil)
		if err2 != nil {
			t.Fatalf("Sort() iteration %d error = %v", i, err2)
		}
		if !reflect.DeepEqual(result1, result2) {
			t.Errorf("Sort() is non-deterministic: iteration %d got %v, want %v", i, result2, result1)
		}
	}
}

func TestSort_UnicodeNodeNames(t *testing.T) {
	t.Parallel()
	// Graph with Unicode node names (various scripts)
	g := Graph{
		"日本語":      nil,          // Japanese
		"中文":       {"日本語"},      // Chinese depends on Japanese
		"한국어":      {"中文"},       // Korean depends on Chinese
		"Ελληνικά": {"한국어"},      // Greek depends on Korean
		"العربية":  {"Ελληνικά"}, // Arabic depends on Greek
		"🚀":        {"العربية"},  // Emoji depends on Arabic
	}
	result, err := Sort(g, nil)
	if err != nil {
		t.Fatalf("Sort() error = %v, want nil", err)
	}
	if len(result) != 6 {
		t.Errorf("Sort() returned %d nodes, want 6", len(result))
	}

	// Verify ordering
	indexOf := make(map[string]int)
	for i, name := range result {
		indexOf[name] = i
	}

	// Each node must come after its dependencies
	for name, deps := range g {
		for _, dep := range deps {
			if indexOf[dep] >= indexOf[name] {
				t.Errorf("Sort() dependency %q should come before %q", dep, name)
			}
		}
	}
}

func TestSort_VeryLongNodeNames(t *testing.T) {
	t.Parallel()
	// Create node names with >1000 characters
	longName1 := strings.Repeat("a", 1001)
	longName2 := strings.Repeat("b", 1500)
	longName3 := strings.Repeat("c", 2000)

	g := Graph{
		longName1: nil,
		longName2: {longName1},
		longName3: {longName2},
	}

	result, err := Sort(g, nil)
	if err != nil {
		t.Fatalf("Sort() error = %v, want nil", err)
	}
	if len(result) != 3 {
		t.Errorf("Sort() returned %d nodes, want 3", len(result))
	}

	// Verify ordering
	indexOf := make(map[string]int, 3)
	for i, name := range result {
		indexOf[name] = i
	}

	if indexOf[longName1] >= indexOf[longName2] {
		t.Error("Sort() longName1 should come before longName2")
	}
	if indexOf[longName2] >= indexOf[longName3] {
		t.Error("Sort() longName2 should come before longName3")
	}
}

func TestSort_VeryLargeGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}
	t.Parallel()

	// Build a graph with 10,000 nodes in a complex structure:
	// - First 5000 nodes form independent chains of length 5
	// - Last 5000 nodes depend on multiple earlier nodes
	const nodeCount = 10000
	const chainLength = 5
	g := make(Graph, nodeCount)

	nodeName := func(i int) string {
		return fmt.Sprintf("node_%05d", i)
	}

	// Create chains of length 5
	for i := 0; i < nodeCount/2; i++ {
		name := nodeName(i)
		if i%chainLength == 0 {
			g[name] = nil
		} else {
			g[name] = []string{nodeName(i - 1)}
		}
	}

	// Create nodes that depend on multiple chain heads
	for i := nodeCount / 2; i < nodeCount; i++ {
		name := nodeName(i)
		// Each node depends on a few chain heads
		deps := make([]string, 0, 3)
		for j := 0; j < 3; j++ {
			chainHead := ((i + j*13) % (nodeCount / 2 / chainLength)) * chainLength
			deps = append(deps, nodeName(chainHead))
		}
		g[name] = deps
	}

	result, err := Sort(g, nil)
	if err != nil {
		t.Fatalf("Sort() error = %v, want nil", err)
	}

	if len(result) != nodeCount {
		t.Errorf("Sort() returned %d nodes, want %d", len(result), nodeCount)
	}

	// Verify ordering constraint: each node appears after all its dependencies
	indexOf := make(map[string]int, nodeCount)
	for i, name := range result {
		indexOf[name] = i
	}

	for name, deps := range g {
		for _, dep := range deps {
			if indexOf[dep] >= indexOf[name] {
				t.Errorf("Sort() dependency %s should come before %s", dep, name)
			}
		}
	}
}

// TestSort_PropertyDependencyOrder uses property-based testing to verify
// that for any valid DAG, the sorted result satisfies all dependency constraints.
func TestSort_PropertyDependencyOrder(t *testing.T) {
	t.Parallel()

	// Property: For any valid DAG, each node appears after all its dependencies
	property := func(nodeCount uint8, seed uint64) bool {
		// Limit graph size to avoid slow tests
		n := int(nodeCount%20) + 1

		// Generate a valid DAG by only allowing edges from higher to lower indices
		// This guarantees no cycles
		g := make(Graph, n)
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("n%d", i)
			var deps []string
			// Node i can depend on nodes 0..i-1 (lower indices)
			for j := 0; j < i; j++ {
				// Use seed to deterministically decide if edge exists
				if (seed>>(uint(j)%64))&1 == 1 {
					deps = append(deps, fmt.Sprintf("n%d", j))
				}
			}
			g[name] = deps
		}

		result, err := Sort(g, nil)
		if err != nil {
			// Should never happen for a valid DAG
			return false
		}

		if len(result) != n {
			return false
		}

		// Build index map
		indexOf := make(map[string]int, n)
		for i, name := range result {
			indexOf[name] = i
		}

		// Verify property: every dependency appears before its dependent
		for name, deps := range g {
			for _, dep := range deps {
				if indexOf[dep] >= indexOf[name] {
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestSort_PropertyIdempotent verifies that sorting the same graph twice
// produces identical results (determinism).
func TestSort_PropertyIdempotent(t *testing.T) {
	t.Parallel()

	property := func(nodeCount uint8, seed uint64) bool {
		n := int(nodeCount%15) + 1

		g := make(Graph, n)
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("n%d", i)
			var deps []string
			for j := 0; j < i; j++ {
				if (seed>>(uint(j)%64))&1 == 1 {
					deps = append(deps, fmt.Sprintf("n%d", j))
				}
			}
			g[name] = deps
		}

		result1, err1 := Sort(g, nil)
		result2, err2 := Sort(g, nil)

		if err1 != nil || err2 != nil {
			return false
		}

		return reflect.DeepEqual(result1, result2)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestSort_PropertyMinimality verifies that when sorting with specified roots,
// the result contains exactly the reachable nodes (no extras, no missing).
func TestSort_PropertyMinimality(t *testing.T) {
	t.Parallel()

	// computeReachable computes all nodes reachable from roots via dependencies
	computeReachable := func(g Graph, roots []string) map[string]bool {
		reachable := make(map[string]bool)
		var visit func(name string)
		visit = func(name string) {
			if reachable[name] {
				return
			}
			reachable[name] = true
			for _, dep := range g[name] {
				visit(dep)
			}
		}
		for _, root := range roots {
			visit(root)
		}
		return reachable
	}

	property := func(nodeCount uint8, seed uint64, rootMask uint8) bool {
		n := int(nodeCount%15) + 2 // At least 2 nodes to have interesting root selection

		// Generate a valid DAG (edges only from higher to lower indices)
		g := make(Graph, n)
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("n%d", i)
			var deps []string
			for j := 0; j < i; j++ {
				if (seed>>(uint(j)%64))&1 == 1 {
					deps = append(deps, fmt.Sprintf("n%d", j))
				}
			}
			g[name] = deps
		}

		// Select roots based on rootMask (use some nodes as roots)
		var roots []string
		for i := 0; i < n; i++ {
			if (rootMask>>(uint(i)%8))&1 == 1 {
				roots = append(roots, fmt.Sprintf("n%d", i))
			}
		}
		if len(roots) == 0 {
			// Ensure at least one root
			roots = []string{"n0"}
		}

		result, err := Sort(g, roots)
		if err != nil {
			return false
		}

		// Compute expected reachable nodes
		expected := computeReachable(g, roots)

		// Property 1: result contains exactly len(expected) nodes
		if len(result) != len(expected) {
			return false
		}

		// Property 2: every node in result is in expected (no extras)
		for _, name := range result {
			if !expected[name] {
				return false
			}
		}

		// Property 3: every node in expected is in result (no missing)
		resultSet := make(map[string]bool, len(result))
		for _, name := range result {
			resultSet[name] = true
		}
		for name := range expected {
			if !resultSet[name] {
				return false
			}
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
