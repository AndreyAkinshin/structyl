package topsort

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestSort_Empty(t *testing.T) {
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
	g := Graph{
		"a": {"b"},
		"b": {"a"},
	}
	_, err := Sort(g, nil)
	if err == nil {
		t.Error("Sort() expected error for cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Sort() error = %v, want to contain 'circular'", err)
	}
}

func TestSort_LongCycle(t *testing.T) {
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

func TestSort_LargeGraph(t *testing.T) {
	// Build a linear chain of 100 nodes: n0 <- n1 <- n2 <- ... <- n99
	const nodeCount = 100
	g := make(Graph, nodeCount)

	for i := 0; i < nodeCount; i++ {
		name := "n" + string(rune('0'+i/100)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i%10))
		if i == 0 {
			g[name] = nil
		} else {
			prevName := "n" + string(rune('0'+(i-1)/100)) + string(rune('0'+((i-1)/10)%10)) + string(rune('0'+(i-1)%10))
			g[name] = []string{prevName}
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
