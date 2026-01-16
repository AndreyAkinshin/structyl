package tests

import (
	"math"
	"strings"
	"testing"
)

func TestCompare_Primitives(t *testing.T) {
	cfg := DefaultComparisonConfig()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		pass     bool
	}{
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal bools", true, true, true},
		{"different bools", true, false, false},
		{"equal ints", float64(42), float64(42), true},
		{"nil both", nil, nil, true},
		{"nil vs value", nil, "value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := Compare(tt.expected, tt.actual, cfg)
			if ok != tt.pass {
				t.Errorf("Compare() = %v, want %v", ok, tt.pass)
			}
		})
	}
}

func TestCompare_Floats(t *testing.T) {
	cfg := ComparisonConfig{
		FloatTolerance: 1e-9,
		ToleranceMode:  "relative",
	}

	tests := []struct {
		name     string
		expected float64
		actual   float64
		pass     bool
	}{
		{"equal", 1.0, 1.0, true},
		{"within tolerance", 1.0, 1.0 + 1e-10, true},
		{"outside tolerance", 1.0, 1.1, false},
		{"zero comparison", 0.0, 1e-10, true},
		{"zero vs large", 0.0, 1.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := Compare(tt.expected, tt.actual, cfg)
			if ok != tt.pass {
				t.Errorf("Compare() = %v, want %v", ok, tt.pass)
			}
		})
	}
}

func TestCompare_AbsoluteTolerance(t *testing.T) {
	cfg := ComparisonConfig{
		FloatTolerance: 0.01,
		ToleranceMode:  "absolute",
	}

	ok, _ := Compare(1.0, 1.005, cfg)
	if !ok {
		t.Error("expected pass with absolute tolerance")
	}

	ok, _ = Compare(1.0, 1.02, cfg)
	if ok {
		t.Error("expected fail outside absolute tolerance")
	}
}

func TestCompare_SpecialFloats(t *testing.T) {
	cfg := ComparisonConfig{
		NaNEqualsNaN: true,
	}

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		pass     bool
	}{
		{"NaN with NaNEqualsNaN", "NaN", math.NaN(), true},
		{"+Infinity", "Infinity", math.Inf(1), true},
		{"-Infinity", "-Infinity", math.Inf(-1), true},
		{"Infinity mismatch", "Infinity", math.Inf(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := Compare(tt.expected, tt.actual, cfg)
			if ok != tt.pass {
				t.Errorf("Compare() = %v, want %v", ok, tt.pass)
			}
		})
	}
}

func TestCompare_NaNWithoutFlag(t *testing.T) {
	cfg := ComparisonConfig{
		NaNEqualsNaN: false,
	}

	ok, _ := Compare("NaN", math.NaN(), cfg)
	if ok {
		t.Error("NaN should not equal NaN when NaNEqualsNaN is false")
	}
}

func TestCompare_NaNErrorMessageHint(t *testing.T) {
	cfg := ComparisonConfig{
		NaNEqualsNaN: false,
	}

	// When both values are NaN but flag is false, error should mention the config option
	ok, diff := Compare("NaN", math.NaN(), cfg)
	if ok {
		t.Error("NaN should not equal NaN when NaNEqualsNaN is false")
	}
	if !strings.Contains(diff, "nan_equals_nan") {
		t.Errorf("error message should mention nan_equals_nan config option, got: %s", diff)
	}
}

func TestCompare_Maps(t *testing.T) {
	cfg := DefaultComparisonConfig()

	expected := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}
	actual := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}

	ok, _ := Compare(expected, actual, cfg)
	if !ok {
		t.Error("expected equal maps to pass")
	}

	// Missing key
	delete(actual, "b")
	ok, _ = Compare(expected, actual, cfg)
	if ok {
		t.Error("expected missing key to fail")
	}

	// Extra key
	actual["b"] = "hello"
	actual["c"] = "extra"
	ok, _ = Compare(expected, actual, cfg)
	if ok {
		t.Error("expected extra key to fail")
	}
}

func TestCompare_Arrays(t *testing.T) {
	cfg := DefaultComparisonConfig()

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(1), float64(2), float64(3)}

	ok, _ := Compare(expected, actual, cfg)
	if !ok {
		t.Error("expected equal arrays to pass")
	}

	// Wrong order with strict mode
	actual = []interface{}{float64(3), float64(2), float64(1)}
	ok, _ = Compare(expected, actual, cfg)
	if ok {
		t.Error("expected wrong order to fail in strict mode")
	}

	// Wrong order with unordered mode
	cfg.ArrayOrder = "unordered"
	ok, _ = Compare(expected, actual, cfg)
	if !ok {
		t.Error("expected wrong order to pass in unordered mode")
	}
}

func TestCompare_NestedStructures(t *testing.T) {
	cfg := DefaultComparisonConfig()

	expected := map[string]interface{}{
		"data": map[string]interface{}{
			"values": []interface{}{float64(1), float64(2)},
		},
	}
	actual := map[string]interface{}{
		"data": map[string]interface{}{
			"values": []interface{}{float64(1), float64(2)},
		},
	}

	ok, _ := Compare(expected, actual, cfg)
	if !ok {
		t.Error("expected nested structures to pass")
	}

	// Modify nested value
	actual["data"].(map[string]interface{})["values"] = []interface{}{float64(1), float64(3)}
	ok, diff := Compare(expected, actual, cfg)
	if ok {
		t.Error("expected nested mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompare_ULPTolerance(t *testing.T) {
	cfg := ComparisonConfig{
		FloatTolerance: 1.0,
		ToleranceMode:  "ulp",
	}

	// Values within 1 ULP should pass
	ok, _ := Compare(1.0, math.Nextafter(1.0, 2.0), cfg)
	if !ok {
		t.Error("expected adjacent floats to pass with ULP tolerance")
	}

	// Values very far apart should fail (using much larger gap)
	cfg.FloatTolerance = 1e-10 // Very small ULP tolerance
	ok, _ = Compare(1.0, 2.0, cfg)
	if ok {
		t.Error("expected far apart floats to fail with tiny ULP tolerance")
	}
}

func TestCompare_InfinityFromFloat64(t *testing.T) {
	cfg := DefaultComparisonConfig()

	// Test +Inf with +Inf (float64 to float64)
	ok, _ := Compare(math.Inf(1), math.Inf(1), cfg)
	if !ok {
		t.Error("expected +Inf == +Inf to pass")
	}

	// Test -Inf with -Inf (float64 to float64)
	ok, _ = Compare(math.Inf(-1), math.Inf(-1), cfg)
	if !ok {
		t.Error("expected -Inf == -Inf to pass")
	}

	// Test +Inf with -Inf should fail
	ok, _ = Compare(math.Inf(1), math.Inf(-1), cfg)
	if ok {
		t.Error("expected +Inf != -Inf")
	}
}

func TestCompare_NaNFloat64(t *testing.T) {
	cfg := ComparisonConfig{
		NaNEqualsNaN: true,
	}

	// Test NaN with NaN (float64 to float64)
	ok, _ := Compare(math.NaN(), math.NaN(), cfg)
	if !ok {
		t.Error("expected NaN == NaN with NaNEqualsNaN")
	}

	// Without flag
	cfg.NaNEqualsNaN = false
	ok, _ = Compare(math.NaN(), math.NaN(), cfg)
	if ok {
		t.Error("expected NaN != NaN without NaNEqualsNaN")
	}
}

func TestCompare_IntToFloat(t *testing.T) {
	cfg := DefaultComparisonConfig()

	// Int expected, float actual
	ok, _ := Compare(42, float64(42), cfg)
	if !ok {
		t.Error("expected int 42 == float64 42")
	}

	// Float expected, int actual
	ok, _ = Compare(float64(42), 42, cfg)
	if !ok {
		t.Error("expected float64 42 == int 42")
	}
}

func TestCompare_TypeMismatch(t *testing.T) {
	cfg := DefaultComparisonConfig()

	// Float expected, string actual
	ok, diff := Compare(1.0, "not a float", cfg)
	if ok {
		t.Error("expected type mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message for type mismatch")
	}
}

func TestCompare_ArrayLengthMismatch(t *testing.T) {
	cfg := DefaultComparisonConfig()

	expected := []interface{}{float64(1), float64(2)}
	actual := []interface{}{float64(1), float64(2), float64(3)}

	ok, diff := Compare(expected, actual, cfg)
	if ok {
		t.Error("expected array length mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message for length mismatch")
	}
}

func TestSortedKeys_Empty(t *testing.T) {
	m := map[string]interface{}{}
	keys := SortedKeys(m)
	if len(keys) != 0 {
		t.Errorf("SortedKeys() = %v, want empty", keys)
	}
}

func TestSortedKeys_Sorted(t *testing.T) {
	m := map[string]interface{}{
		"z": 1,
		"a": 2,
		"m": 3,
	}
	keys := SortedKeys(m)
	if len(keys) != 3 {
		t.Errorf("len(SortedKeys()) = %d, want 3", len(keys))
	}
	if keys[0] != "a" || keys[1] != "m" || keys[2] != "z" {
		t.Errorf("SortedKeys() = %v, want [a, m, z]", keys)
	}
}

func TestCompare_UnorderedArrayNoMatch(t *testing.T) {
	cfg := ComparisonConfig{
		ArrayOrder: "unordered",
	}

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(1), float64(2), float64(4)}

	ok, diff := Compare(expected, actual, cfg)
	if ok {
		t.Error("expected unordered mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompare_SpecialFloatTypeMismatch(t *testing.T) {
	cfg := ComparisonConfig{
		NaNEqualsNaN: true,
	}

	// String "NaN" vs non-float
	ok, diff := Compare("NaN", "not a float", cfg)
	if ok {
		t.Error("expected NaN vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompare_UnknownSpecialFloat(t *testing.T) {
	cfg := DefaultComparisonConfig()

	// isSpecialFloat should return false for non-special strings
	// but the code path should handle it
	ok, _ := Compare("SomeString", 1.0, cfg)
	if ok {
		t.Error("expected string vs float to fail")
	}
}

func TestIsSpecialFloat(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"NaN", true},
		{"Infinity", true},
		{"+Infinity", true},
		{"-Infinity", true},
		{"other", false},
		{"nan", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isSpecialFloat(tt.input)
			if got != tt.want {
				t.Errorf("isSpecialFloat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPathStr(t *testing.T) {
	if pathStr("") != "root" {
		t.Errorf("pathStr(\"\") = %q, want \"root\"", pathStr(""))
	}
	if pathStr("foo.bar") != "foo.bar" {
		t.Errorf("pathStr(\"foo.bar\") = %q, want \"foo.bar\"", pathStr("foo.bar"))
	}
}

func TestUlpDiff_IdenticalValues(t *testing.T) {
	if diff := ulpDiff(1.0, 1.0); diff != 0 {
		t.Errorf("ulpDiff(1.0, 1.0) = %d, want 0", diff)
	}
	if diff := ulpDiff(0.0, 0.0); diff != 0 {
		t.Errorf("ulpDiff(0.0, 0.0) = %d, want 0", diff)
	}
	if diff := ulpDiff(-1.0, -1.0); diff != 0 {
		t.Errorf("ulpDiff(-1.0, -1.0) = %d, want 0", diff)
	}
}

func TestUlpDiff_AdjacentValues(t *testing.T) {
	a := 1.0
	b := math.Nextafter(a, 2.0)
	if diff := ulpDiff(a, b); diff != 1 {
		t.Errorf("ulpDiff(1.0, nextafter(1.0)) = %d, want 1", diff)
	}
}

func TestUlpDiff_Symmetric(t *testing.T) {
	a := 1.0
	b := 1.5
	if ulpDiff(a, b) != ulpDiff(b, a) {
		t.Errorf("ulpDiff should be symmetric: ulpDiff(%v, %v) = %d, ulpDiff(%v, %v) = %d",
			a, b, ulpDiff(a, b), b, a, ulpDiff(b, a))
	}
}

func TestUlpDiff_NegativeValues(t *testing.T) {
	a := -1.0
	b := math.Nextafter(a, 0.0)
	if diff := ulpDiff(a, b); diff != 1 {
		t.Errorf("ulpDiff(-1.0, nextafter(-1.0, 0)) = %d, want 1", diff)
	}
}

func TestCompare_ULPTolerance_ExactBoundary(t *testing.T) {
	// Test exact ULP boundary: 1 ULP tolerance should pass for 1 ULP diff, fail for 2 ULP diff
	cfg := ComparisonConfig{
		ToleranceMode:  "ulp",
		FloatTolerance: 1, // Allow exactly 1 ULP difference
	}

	// Adjacent floats (1 ULP apart) should match with 1 ULP tolerance
	a := 1.0
	b := math.Nextafter(a, 2.0)
	ok, diff := Compare(a, b, cfg)
	if !ok {
		t.Errorf("Compare with ULP mode failed for adjacent floats (1 ULP apart): %s", diff)
	}

	// Two ULPs apart should fail with 1 ULP tolerance
	c := math.Nextafter(b, 2.0)
	ok, _ = Compare(a, c, cfg)
	if ok {
		t.Errorf("Compare with ULP mode should fail for floats 2 ULPs apart with tolerance 1")
	}
}

func TestToFloat_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"string", "not a float"},
		{"bool", true},
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"a": 1}},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat(tt.input)
			if ok {
				t.Errorf("toFloat(%v) = (%v, true), want (_, false)", tt.input, result)
			}
			if result != 0 {
				t.Errorf("toFloat(%v) = (%v, _), want (0, _)", tt.input, result)
			}
		})
	}
}

func TestToFloat_SupportedTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"float64", float64(3.14), 3.14},
		{"int", int(42), 42.0},
		{"zero float", float64(0), 0},
		{"zero int", int(0), 0},
		{"negative float", float64(-1.5), -1.5},
		{"negative int", int(-10), -10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat(tt.input)
			if !ok {
				t.Errorf("toFloat(%v) = (_, false), want (_, true)", tt.input)
			}
			if result != tt.expected {
				t.Errorf("toFloat(%v) = (%v, _), want (%v, _)", tt.input, result, tt.expected)
			}
		})
	}
}
