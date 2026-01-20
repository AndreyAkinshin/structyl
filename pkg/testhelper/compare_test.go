package testhelper

// NOTE: These tests intentionally mirror internal/tests/compare_test.go.
// Both packages provide comparison functionality but for different consumers:
// - internal/tests: Internal test framework with ComparisonConfig
// - pkg/testhelper: Public API for external test helpers with CompareOptions
// Each package maintains its own tests to ensure independent correctness.

import (
	"math"
	"strings"
	"testing"
)

func TestCompareOutput_Primitives(t *testing.T) {
	opts := DefaultOptions()

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
			if CompareOutput(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("CompareOutput() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestCompareOutput_Floats(t *testing.T) {
	opts := CompareOptions{
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
			if CompareOutput(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("CompareOutput() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestCompareOutput_AbsoluteTolerance(t *testing.T) {
	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  "absolute",
	}

	if !CompareOutput(1.0, 1.005, opts) {
		t.Error("expected pass with absolute tolerance")
	}

	if CompareOutput(1.0, 1.02, opts) {
		t.Error("expected fail outside absolute tolerance")
	}
}

func TestCompareOutput_SpecialFloats(t *testing.T) {
	opts := CompareOptions{
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
			if CompareOutput(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("CompareOutput() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestCompareOutput_NaNWithoutFlag(t *testing.T) {
	opts := CompareOptions{
		NaNEqualsNaN: false,
	}

	if CompareOutput("NaN", math.NaN(), opts) {
		t.Error("NaN should not equal NaN when NaNEqualsNaN is false")
	}
}

func TestCompareOutput_Maps(t *testing.T) {
	opts := DefaultOptions()

	expected := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}
	actual := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}

	if !CompareOutput(expected, actual, opts) {
		t.Error("expected equal maps to pass")
	}

	// Missing key
	delete(actual, "b")
	if CompareOutput(expected, actual, opts) {
		t.Error("expected missing key to fail")
	}

	// Extra key
	actual["b"] = "hello"
	actual["c"] = "extra"
	if CompareOutput(expected, actual, opts) {
		t.Error("expected extra key to fail")
	}
}

func TestCompareOutput_Arrays(t *testing.T) {
	opts := DefaultOptions()

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(1), float64(2), float64(3)}

	if !CompareOutput(expected, actual, opts) {
		t.Error("expected equal arrays to pass")
	}

	// Wrong order with strict mode
	actual = []interface{}{float64(3), float64(2), float64(1)}
	if CompareOutput(expected, actual, opts) {
		t.Error("expected wrong order to fail in strict mode")
	}

	// Wrong order with unordered mode
	opts.ArrayOrder = "unordered"
	if !CompareOutput(expected, actual, opts) {
		t.Error("expected wrong order to pass in unordered mode")
	}
}

func TestCompareOutput_NestedStructures(t *testing.T) {
	opts := DefaultOptions()

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

	if !CompareOutput(expected, actual, opts) {
		t.Error("expected nested structures to pass")
	}

	// Modify nested value
	actual["data"].(map[string]interface{})["values"] = []interface{}{float64(1), float64(3)}
	ok, diff := Compare(expected, actual, opts)
	if ok {
		t.Error("expected nested mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.FloatTolerance != 1e-9 {
		t.Errorf("FloatTolerance = %v, want 1e-9", opts.FloatTolerance)
	}
	if opts.ToleranceMode != "relative" {
		t.Errorf("ToleranceMode = %q, want %q", opts.ToleranceMode, "relative")
	}
	if !opts.NaNEqualsNaN {
		t.Error("NaNEqualsNaN should be true by default")
	}
	if opts.ArrayOrder != "strict" {
		t.Errorf("ArrayOrder = %q, want %q", opts.ArrayOrder, "strict")
	}
}

func TestValidateOptions(t *testing.T) {
	// Valid options
	validCases := []CompareOptions{
		DefaultOptions(),
		{ToleranceMode: "relative", ArrayOrder: "strict"},
		{ToleranceMode: "absolute", ArrayOrder: "strict"},
		{ToleranceMode: "ulp", ArrayOrder: "strict"},
		{ToleranceMode: "relative", ArrayOrder: "unordered"},
		{ToleranceMode: "", ArrayOrder: ""}, // empty defaults
		{FloatTolerance: 0},                 // zero tolerance is valid
	}
	for _, opts := range validCases {
		if err := ValidateOptions(opts); err != nil {
			t.Errorf("ValidateOptions(%+v) returned error: %v", opts, err)
		}
	}

	// Invalid ToleranceMode
	invalidMode := CompareOptions{ToleranceMode: "fuzzy"}
	if err := ValidateOptions(invalidMode); err == nil {
		t.Error("ValidateOptions with invalid ToleranceMode should return error")
	}

	// Invalid ArrayOrder
	invalidOrder := CompareOptions{ArrayOrder: "random"}
	if err := ValidateOptions(invalidOrder); err == nil {
		t.Error("ValidateOptions with invalid ArrayOrder should return error")
	}

	// Negative FloatTolerance
	negativeTolerance := CompareOptions{FloatTolerance: -0.01}
	if err := ValidateOptions(negativeTolerance); err == nil {
		t.Error("ValidateOptions with negative FloatTolerance should return error")
	}
}

func TestFormatDiff(t *testing.T) {
	opts := DefaultOptions()

	// Equal values
	diff := FormatDiff("hello", "hello", opts)
	if diff != "values are equal" {
		t.Errorf("expected 'values are equal', got %q", diff)
	}

	// Different values
	diff = FormatDiff("hello", "world", opts)
	if diff == "values are equal" {
		t.Error("expected diff message, got 'values are equal'")
	}
}

func TestFormatComparisonResult(t *testing.T) {
	opts := DefaultOptions()

	// Equal values - should return empty string
	result := FormatComparisonResult("hello", "hello", opts)
	if result != "" {
		t.Errorf("expected empty string for equal values, got %q", result)
	}

	// Different values - should return diff description
	result = FormatComparisonResult("hello", "world", opts)
	if result == "" {
		t.Error("expected diff message for different values, got empty string")
	}
	if !strings.Contains(result, "hello") || !strings.Contains(result, "world") {
		t.Errorf("diff should contain expected and actual values, got %q", result)
	}
}

func TestPathStr(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", "$"},
		{".foo", "foo"},
		{".foo.bar", "foo.bar"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := pathStr(tt.path)
			if result != tt.expected {
				t.Errorf("pathStr(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Work Item 8: ULP Diff Tests
// =============================================================================

func TestUlpDiff_IdenticalValues(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"zero", 0.0},
		{"one", 1.0},
		{"negative", -1.0},
		{"small", 1e-10},
		{"large", 1e10},
		{"pi", 3.14159265358979323846},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := ulpDiff(tt.value, tt.value)
			if diff != 0 {
				t.Errorf("ulpDiff(%v, %v) = %d, want 0", tt.value, tt.value, diff)
			}
		})
	}
}

func TestUlpDiff_AdjacentValues(t *testing.T) {
	// math.Nextafter returns the next representable float64 value
	// The ULP difference should be 1
	tests := []struct {
		name string
		a    float64
	}{
		{"one", 1.0},
		{"small", 1e-10},
		{"large", 1e10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := math.Nextafter(tt.a, math.Inf(1))
			diff := ulpDiff(tt.a, b)
			if diff != 1 {
				t.Errorf("ulpDiff(%v, nextafter) = %d, want 1", tt.a, diff)
			}
		})
	}
}

func TestUlpDiff_NegativeValues(t *testing.T) {
	// Test ULP difference with negative values
	a := -1.0
	b := math.Nextafter(a, math.Inf(-1)) // Next value toward -infinity

	diff := ulpDiff(a, b)
	if diff != 1 {
		t.Errorf("ulpDiff(-1.0, nextafter) = %d, want 1", diff)
	}

	// Same value negative should be 0
	diff = ulpDiff(-5.5, -5.5)
	if diff != 0 {
		t.Errorf("ulpDiff(-5.5, -5.5) = %d, want 0", diff)
	}
}

func TestUlpDiff_Symmetric(t *testing.T) {
	// ULP diff should be symmetric: ulpDiff(a, b) == ulpDiff(b, a)
	a := 1.0
	b := math.Nextafter(math.Nextafter(a, math.Inf(1)), math.Inf(1)) // 2 ULPs away

	diff1 := ulpDiff(a, b)
	diff2 := ulpDiff(b, a)

	if diff1 != diff2 {
		t.Errorf("ulpDiff not symmetric: ulpDiff(a,b)=%d, ulpDiff(b,a)=%d", diff1, diff2)
	}
	if diff1 != 2 {
		t.Errorf("ulpDiff = %d, want 2", diff1)
	}
}

func TestUlpDiff_MixedSigns(t *testing.T) {
	// Test ULP difference across zero crossing (positive vs negative)
	// This is a known edge case for ULP algorithms
	posSmall := math.SmallestNonzeroFloat64
	negSmall := -math.SmallestNonzeroFloat64

	// The ULP distance from smallest positive to smallest negative
	// should be 2 (through zero)
	diff := ulpDiff(posSmall, negSmall)
	if diff != 2 {
		t.Errorf("ulpDiff(SmallestPositive, SmallestNegative) = %d, want 2", diff)
	}

	// Test with larger values across zero
	a := 1.0
	b := -1.0
	diff = ulpDiff(a, b)
	// Diff should be positive and large (the number of representable floats between -1 and 1)
	if diff <= 0 {
		t.Errorf("ulpDiff(1.0, -1.0) = %d, should be positive", diff)
	}
}

func TestUlpDiff_ExtremeValues(t *testing.T) {
	// Test with extreme values to ensure no overflow
	maxVal := math.MaxFloat64
	negMaxVal := -math.MaxFloat64

	// Same extreme values should have 0 diff
	if diff := ulpDiff(maxVal, maxVal); diff != 0 {
		t.Errorf("ulpDiff(MaxFloat64, MaxFloat64) = %d, want 0", diff)
	}
	if diff := ulpDiff(negMaxVal, negMaxVal); diff != 0 {
		t.Errorf("ulpDiff(-MaxFloat64, -MaxFloat64) = %d, want 0", diff)
	}

	// Adjacent to max value
	almostMax := math.Nextafter(maxVal, 0)
	diff := ulpDiff(maxVal, almostMax)
	if diff != 1 {
		t.Errorf("ulpDiff(MaxFloat64, nextafter) = %d, want 1", diff)
	}
}

func TestCompareOutput_UlpTolerance(t *testing.T) {
	// Test ULP tolerance mode
	a := 1.0
	b := math.Nextafter(a, math.Inf(1)) // 1 ULP away

	// With ULP tolerance of 1, should pass
	opts := CompareOptions{
		FloatTolerance: 1,
		ToleranceMode:  "ulp",
	}
	if !CompareOutput(a, b, opts) {
		t.Error("expected 1 ULP difference to pass with tolerance 1")
	}

	// With ULP tolerance of 0, should fail
	opts.FloatTolerance = 0
	if CompareOutput(a, b, opts) {
		t.Error("expected 1 ULP difference to fail with tolerance 0")
	}

	// 3 ULPs away with tolerance of 2 should fail
	c := math.Nextafter(math.Nextafter(math.Nextafter(a, math.Inf(1)), math.Inf(1)), math.Inf(1))
	opts.FloatTolerance = 2
	if CompareOutput(a, c, opts) {
		t.Error("expected 3 ULP difference to fail with tolerance 2")
	}

	// 3 ULPs away with tolerance of 3 should pass
	opts.FloatTolerance = 3
	if !CompareOutput(a, c, opts) {
		t.Error("expected 3 ULP difference to pass with tolerance 3")
	}
}

// =============================================================================
// Work Item 5: Additional Coverage Tests
// =============================================================================

func TestCompareOutput_TypeMismatch(t *testing.T) {
	opts := DefaultOptions()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"string vs number", "hello", float64(42)},
		{"string vs bool", "hello", true},
		{"string vs array", "hello", []interface{}{1, 2}},
		{"string vs map", "hello", map[string]interface{}{"a": 1}},
		{"bool vs string", true, "true"},
		{"bool vs number", true, float64(1)},
		{"bool vs array", true, []interface{}{true}},
		{"float vs string", float64(42), "42"},
		{"float vs bool", float64(1), true},
		{"float vs array", float64(1), []interface{}{1}},
		{"int vs string", 42, "42"},
		{"array vs string", []interface{}{1}, "array"},
		{"array vs number", []interface{}{1}, float64(1)},
		{"map vs string", map[string]interface{}{"a": 1}, "object"},
		{"map vs array", map[string]interface{}{"a": 1}, []interface{}{1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, diff := Compare(tt.expected, tt.actual, opts)
			if ok {
				t.Errorf("expected type mismatch to fail")
			}
			if diff == "" {
				t.Errorf("expected diff message for type mismatch")
			}
		})
	}
}

func TestCompareOutput_FloatTypeMismatch(t *testing.T) {
	opts := DefaultOptions()

	// Expected float64, actual is string
	ok, diff := Compare(float64(42), "42", opts)
	if ok {
		t.Error("expected float64 vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected int, actual is string (int is converted to float64)
	ok, diff = Compare(42, "42", opts)
	if ok {
		t.Error("expected int vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompareOutput_SpecialFloatTypeMismatch(t *testing.T) {
	opts := DefaultOptions()

	// Expected "NaN" string (special), actual is not a number type
	ok, diff := Compare("NaN", "not a number", opts)
	if ok {
		t.Error("expected NaN vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected "Infinity" vs string
	ok, diff = Compare("Infinity", "infinity", opts)
	if ok {
		t.Error("expected Infinity vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected "-Infinity" vs string
	ok, diff = Compare("-Infinity", "negative infinity", opts)
	if ok {
		t.Error("expected -Infinity vs string to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompareOutput_SpecialFloatMismatch(t *testing.T) {
	opts := DefaultOptions()

	// Expected NaN, got regular number
	ok, diff := Compare("NaN", float64(42), opts)
	if ok {
		t.Error("expected NaN vs regular number to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected +Infinity, got regular number
	ok, diff = Compare("Infinity", float64(42), opts)
	if ok {
		t.Error("expected Infinity vs regular number to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected -Infinity, got regular number
	ok, diff = Compare("-Infinity", float64(42), opts)
	if ok {
		t.Error("expected -Infinity vs regular number to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}

	// Expected +Infinity, got -Infinity (already tested above, but with int)
	ok, _ = Compare("Infinity", int(42), opts)
	if ok {
		t.Error("expected Infinity vs int to fail")
	}

	// Expected -Infinity, got int
	ok, _ = Compare("-Infinity", int(-42), opts)
	if ok {
		t.Error("expected -Infinity vs int to fail")
	}
}

func TestFloatsEqual_InfinityMismatch(t *testing.T) {
	opts := DefaultOptions()

	// +Inf vs regular number
	if CompareOutput(math.Inf(1), float64(42), opts) {
		t.Error("expected +Inf vs regular to fail")
	}

	// -Inf vs regular number
	if CompareOutput(math.Inf(-1), float64(42), opts) {
		t.Error("expected -Inf vs regular to fail")
	}

	// +Inf vs -Inf
	if CompareOutput(math.Inf(1), math.Inf(-1), opts) {
		t.Error("expected +Inf vs -Inf to fail")
	}

	// NaN vs regular (with NaNEqualsNaN=false)
	opts.NaNEqualsNaN = false
	if CompareOutput(math.NaN(), float64(42), opts) {
		t.Error("expected NaN vs regular to fail")
	}

	// Regular vs NaN
	if CompareOutput(float64(42), math.NaN(), opts) {
		t.Error("expected regular vs NaN to fail")
	}
}

func TestFloatsEqual_NaNWithoutFlag(t *testing.T) {
	opts := CompareOptions{
		NaNEqualsNaN: false,
	}

	// NaN vs NaN with flag disabled
	if CompareOutput(math.NaN(), math.NaN(), opts) {
		t.Error("expected NaN vs NaN to fail when NaNEqualsNaN is false")
	}
}

func TestCompareOutput_ArrayLengthMismatch(t *testing.T) {
	opts := DefaultOptions()

	expected := []interface{}{1, 2, 3}
	actual := []interface{}{1, 2}

	ok, diff := Compare(expected, actual, opts)
	if ok {
		t.Error("expected array length mismatch to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompareOutput_UnorderedArrayNoMatch(t *testing.T) {
	opts := CompareOptions{
		ArrayOrder: "unordered",
	}

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(1), float64(2), float64(4)} // 3 is missing

	ok, diff := Compare(expected, actual, opts)
	if ok {
		t.Error("expected unordered array with missing element to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompareValues_DefaultCase(t *testing.T) {
	opts := DefaultOptions()

	// Test the default case with equal values of unknown type
	// Using a type that's not explicitly handled (struct pointer)
	type customType struct{ value int }
	a := &customType{value: 1}

	// Same pointer should be equal
	ok, _ := Compare(a, a, opts)
	if !ok {
		t.Error("expected same pointer to be equal")
	}

	// Different pointers with same value should not be equal
	b := &customType{value: 1}
	ok, diff := Compare(a, b, opts)
	if ok {
		t.Error("expected different pointers to fail")
	}
	if diff == "" {
		t.Error("expected diff message")
	}
}

func TestCompareOutput_NilMismatch(t *testing.T) {
	opts := DefaultOptions()

	// value vs nil
	ok, diff := Compare("value", nil, opts)
	if ok {
		t.Error("expected value vs nil to fail")
	}
	if diff == "" {
		t.Error("expected diff message for nil mismatch")
	}
}

// TestCompare_InvalidToleranceMode_Panics verifies that Compare panics with
// invalid ToleranceMode values to fail-fast rather than silently using defaults.
func TestCompare_InvalidToleranceMode_Panics(t *testing.T) {
	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  "invalid", // Not a valid mode
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Compare should panic with invalid ToleranceMode")
		}
		panicMsg, ok := r.(string)
		if !ok {
			t.Errorf("panic value should be string, got %T", r)
			return
		}
		if !strings.Contains(panicMsg, "testhelper.Compare") {
			t.Errorf("panic message should mention testhelper.Compare, got: %s", panicMsg)
		}
		if !strings.Contains(panicMsg, "invalid ToleranceMode") {
			t.Errorf("panic message should mention invalid ToleranceMode, got: %s", panicMsg)
		}
	}()

	Compare(1.0, 1.005, opts)
}

// TestCompareOutput_InvalidToleranceMode_Panics verifies CompareOutput also panics.
func TestCompareOutput_InvalidToleranceMode_Panics(t *testing.T) {
	opts := CompareOptions{
		ToleranceMode: "fuzzy", // Invalid
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("CompareOutput should panic with invalid ToleranceMode")
		}
	}()

	CompareOutput(1.0, 1.0, opts)
}

// TestCompare_InvalidArrayOrder_Panics verifies that Compare panics with invalid ArrayOrder.
func TestCompare_InvalidArrayOrder_Panics(t *testing.T) {
	opts := CompareOptions{
		ArrayOrder: "random", // Invalid
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Compare should panic with invalid ArrayOrder")
		}
		panicMsg, ok := r.(string)
		if ok && !strings.Contains(panicMsg, "invalid ArrayOrder") {
			t.Errorf("panic message should mention invalid ArrayOrder, got: %s", panicMsg)
		}
	}()

	Compare([]interface{}{1}, []interface{}{1}, opts)
}

// =============================================================================
// Enum Constants Tests
// =============================================================================

func TestToleranceModeConstants(t *testing.T) {
	// Verify constants match the validation switch cases
	validModes := []string{ToleranceModeRelative, ToleranceModeAbsolute, ToleranceModeULP}
	for _, mode := range validModes {
		opts := CompareOptions{ToleranceMode: mode}
		if err := ValidateOptions(opts); err != nil {
			t.Errorf("constant %q should be valid, got error: %v", mode, err)
		}
	}

	// Verify constant values
	if ToleranceModeRelative != "relative" {
		t.Errorf("ToleranceModeRelative = %q, want %q", ToleranceModeRelative, "relative")
	}
	if ToleranceModeAbsolute != "absolute" {
		t.Errorf("ToleranceModeAbsolute = %q, want %q", ToleranceModeAbsolute, "absolute")
	}
	if ToleranceModeULP != "ulp" {
		t.Errorf("ToleranceModeULP = %q, want %q", ToleranceModeULP, "ulp")
	}
}

func TestArrayOrderConstants(t *testing.T) {
	// Verify constants match the validation switch cases
	validOrders := []string{ArrayOrderStrict, ArrayOrderUnordered}
	for _, order := range validOrders {
		opts := CompareOptions{ArrayOrder: order}
		if err := ValidateOptions(opts); err != nil {
			t.Errorf("constant %q should be valid, got error: %v", order, err)
		}
	}

	// Verify constant values
	if ArrayOrderStrict != "strict" {
		t.Errorf("ArrayOrderStrict = %q, want %q", ArrayOrderStrict, "strict")
	}
	if ArrayOrderUnordered != "unordered" {
		t.Errorf("ArrayOrderUnordered = %q, want %q", ArrayOrderUnordered, "unordered")
	}
}

func TestConstantsWithCompare(t *testing.T) {
	// Test using constants in actual comparisons
	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  ToleranceModeAbsolute,
		ArrayOrder:     ArrayOrderUnordered,
	}

	// Float comparison with absolute tolerance
	if !CompareOutput(1.0, 1.005, opts) {
		t.Error("expected absolute tolerance to pass")
	}

	// Array comparison with unordered mode
	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(3), float64(1), float64(2)}
	if !CompareOutput(expected, actual, opts) {
		t.Error("expected unordered array comparison to pass")
	}
}

// =============================================================================
// ULP Edge Case Tests (documenting behavior for extreme tolerances)
// =============================================================================

func TestCompareOutput_UlpTolerance_Truncation(t *testing.T) {
	// Test that FloatTolerance is truncated to integer for ULP mode
	// (e.g., 1.9 allows 1 ULP, not 2)
	a := 1.0
	b := math.Nextafter(a, math.Inf(1)) // 1 ULP away
	c := math.Nextafter(b, math.Inf(1)) // 2 ULPs away
	_ = math.Nextafter(c, math.Inf(1))  // 3 ULPs away (unused)

	// With tolerance 1.9, truncated to 1, should pass for 1 ULP
	opts := CompareOptions{
		FloatTolerance: 1.9,
		ToleranceMode:  ToleranceModeULP,
	}
	if !CompareOutput(a, b, opts) {
		t.Error("expected 1 ULP to pass with tolerance 1.9 (truncated to 1)")
	}

	// With tolerance 1.9, truncated to 1, should fail for 2 ULPs
	if CompareOutput(a, c, opts) {
		t.Error("expected 2 ULPs to fail with tolerance 1.9 (truncated to 1)")
	}
}

func TestCompareOutput_UlpTolerance_ZeroTolerance(t *testing.T) {
	// Test ULP mode with zero tolerance (exact match required)
	a := 1.0
	b := math.Nextafter(a, math.Inf(1)) // 1 ULP away

	opts := CompareOptions{
		FloatTolerance: 0,
		ToleranceMode:  ToleranceModeULP,
	}

	// Same value should pass
	if !CompareOutput(a, a, opts) {
		t.Error("expected identical values to pass with ULP tolerance 0")
	}

	// 1 ULP away should fail
	if CompareOutput(a, b, opts) {
		t.Error("expected 1 ULP difference to fail with tolerance 0")
	}
}

func TestCompareOutput_UlpTolerance_LargeTolerance(t *testing.T) {
	// Test ULP mode with large tolerance value
	// Note: Behavior for tolerances >= 2^63 is undefined per documentation
	a := 1.0
	b := 2.0 // Large ULP distance

	// Large but reasonable tolerance
	opts := CompareOptions{
		FloatTolerance: 1e18, // Large but within int64 range
		ToleranceMode:  ToleranceModeULP,
	}

	// Should pass because tolerance is huge
	if !CompareOutput(a, b, opts) {
		t.Error("expected large ULP tolerance to pass for 1.0 vs 2.0")
	}
}

// =============================================================================
// Empty String Default Behavior Tests
// =============================================================================

func TestCompareOutput_EmptyToleranceMode_DefaultsToRelative(t *testing.T) {
	// Verify that empty ToleranceMode defaults to relative tolerance behavior
	opts := CompareOptions{
		FloatTolerance: 1e-9,
		ToleranceMode:  "", // Empty should default to relative
	}

	// This should behave like relative tolerance
	// |1.0 - 1.0000000005| / |1.0| = 5e-10 which is <= 1e-9
	if !CompareOutput(1.0, 1.0000000005, opts) {
		t.Error("empty ToleranceMode should default to relative and pass for small relative diff")
	}

	// This should fail for relative tolerance
	// |1.0 - 1.1| / |1.0| = 0.1 which is > 1e-9
	if CompareOutput(1.0, 1.1, opts) {
		t.Error("empty ToleranceMode should default to relative and fail for large relative diff")
	}

	// Verify by comparing to explicit relative mode
	optsExplicit := CompareOptions{
		FloatTolerance: 1e-9,
		ToleranceMode:  ToleranceModeRelative,
	}

	// Behavior should match explicit relative mode
	if CompareOutput(1.0, 1.0000000005, opts) != CompareOutput(1.0, 1.0000000005, optsExplicit) {
		t.Error("empty ToleranceMode should behave identically to explicit relative mode")
	}
}

func TestCompareOutput_EmptyArrayOrder_DefaultsToStrict(t *testing.T) {
	// Verify that empty ArrayOrder defaults to strict ordering
	opts := CompareOptions{
		ArrayOrder: "", // Empty should default to strict
	}

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actualOrdered := []interface{}{float64(1), float64(2), float64(3)}
	actualReordered := []interface{}{float64(3), float64(2), float64(1)}

	// Same order should pass
	if !CompareOutput(expected, actualOrdered, opts) {
		t.Error("empty ArrayOrder should default to strict and pass for same order")
	}

	// Different order should fail (strict mode)
	if CompareOutput(expected, actualReordered, opts) {
		t.Error("empty ArrayOrder should default to strict and fail for different order")
	}

	// Verify by comparing to explicit strict mode
	optsExplicit := CompareOptions{
		ArrayOrder: ArrayOrderStrict,
	}

	if CompareOutput(expected, actualReordered, opts) != CompareOutput(expected, actualReordered, optsExplicit) {
		t.Error("empty ArrayOrder should behave identically to explicit strict mode")
	}
}

// =============================================================================
// Work Item 6: +Infinity String Support Test
// =============================================================================

func TestCompare_PlusInfinityString(t *testing.T) {
	opts := DefaultOptions()

	// "+Infinity" should match positive infinity (same as "Infinity")
	if !CompareOutput("+Infinity", math.Inf(1), opts) {
		t.Error("expected \"+Infinity\" string to match math.Inf(1)")
	}

	// "+Infinity" should NOT match negative infinity
	if CompareOutput("+Infinity", math.Inf(-1), opts) {
		t.Error("expected \"+Infinity\" string to NOT match math.Inf(-1)")
	}

	// "+Infinity" should NOT match regular numbers
	if CompareOutput("+Infinity", float64(42), opts) {
		t.Error("expected \"+Infinity\" string to NOT match regular number")
	}

	// "+Infinity" should NOT match NaN
	if CompareOutput("+Infinity", math.NaN(), opts) {
		t.Error("expected \"+Infinity\" string to NOT match NaN")
	}

	// Type mismatch: "+Infinity" vs string (not treated as special float)
	ok, diff := Compare("+Infinity", "infinity", opts)
	if ok {
		t.Error("expected \"+Infinity\" vs string to fail (type mismatch)")
	}
	if diff == "" {
		t.Error("expected diff message for type mismatch")
	}
}
