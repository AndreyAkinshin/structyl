package topsort

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

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
	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}

	if indexOf("a") >= indexOf("b") {
		t.Errorf("Sort() a should come before b: %v", result)
	}
	if indexOf("b") >= indexOf("c") {
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

	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}

	// a must come before b and c
	if indexOf("a") >= indexOf("b") || indexOf("a") >= indexOf("c") {
		t.Errorf("Sort() a should come before b and c: %v", result)
	}
	// b and c must come before d
	if indexOf("b") >= indexOf("d") || indexOf("c") >= indexOf("d") {
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
	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}

	// a must come before b
	if indexOf("a") >= indexOf("b") {
		t.Errorf("Sort() a should come before b: %v", result)
	}
	// c must come before d
	if indexOf("c") >= indexOf("d") {
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

	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}

	// b must come before a
	if indexOf("b") >= indexOf("a") {
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
