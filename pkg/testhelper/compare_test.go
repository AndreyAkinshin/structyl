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

func TestEqual_Primitives(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if Equal(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("Equal() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestEqual_Floats(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if Equal(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("Equal() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestEqual_AbsoluteTolerance(t *testing.T) {
	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  "absolute",
	}

	if !Equal(1.0, 1.005, opts) {
		t.Error("expected pass with absolute tolerance")
	}

	if Equal(1.0, 1.02, opts) {
		t.Error("expected fail outside absolute tolerance")
	}
}

func TestEqual_SpecialFloats(t *testing.T) {
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
			if Equal(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("Equal() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestEqual_NaNWithoutFlag(t *testing.T) {
	opts := CompareOptions{
		NaNEqualsNaN: false,
	}

	if Equal("NaN", math.NaN(), opts) {
		t.Error("NaN should not equal NaN when NaNEqualsNaN is false")
	}
}

func TestEqual_Maps(t *testing.T) {
	opts := DefaultOptions()

	expected := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}
	actual := map[string]interface{}{
		"a": float64(1),
		"b": "hello",
	}

	if !Equal(expected, actual, opts) {
		t.Error("expected equal maps to pass")
	}

	// Missing key
	delete(actual, "b")
	if Equal(expected, actual, opts) {
		t.Error("expected missing key to fail")
	}

	// Extra key
	actual["b"] = "hello"
	actual["c"] = "extra"
	if Equal(expected, actual, opts) {
		t.Error("expected extra key to fail")
	}
}

func TestEqual_Arrays(t *testing.T) {
	opts := DefaultOptions()

	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(1), float64(2), float64(3)}

	if !Equal(expected, actual, opts) {
		t.Error("expected equal arrays to pass")
	}

	// Wrong order with strict mode
	actual = []interface{}{float64(3), float64(2), float64(1)}
	if Equal(expected, actual, opts) {
		t.Error("expected wrong order to fail in strict mode")
	}

	// Wrong order with unordered mode
	opts.ArrayOrder = "unordered"
	if !Equal(expected, actual, opts) {
		t.Error("expected wrong order to pass in unordered mode")
	}
}

func TestEqual_NestedStructures(t *testing.T) {
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

	if !Equal(expected, actual, opts) {
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

func TestNewCompareOptions(t *testing.T) {
	t.Parallel()

	t.Run("valid options", func(t *testing.T) {
		t.Parallel()
		opts, err := NewCompareOptions(ToleranceModeRelative, ArrayOrderStrict, 1e-6, true)
		if err != nil {
			t.Fatalf("NewCompareOptions() error = %v", err)
		}
		if opts.ToleranceMode != ToleranceModeRelative {
			t.Errorf("ToleranceMode = %q, want %q", opts.ToleranceMode, ToleranceModeRelative)
		}
		if opts.ArrayOrder != ArrayOrderStrict {
			t.Errorf("ArrayOrder = %q, want %q", opts.ArrayOrder, ArrayOrderStrict)
		}
		if opts.FloatTolerance != 1e-6 {
			t.Errorf("FloatTolerance = %v, want 1e-6", opts.FloatTolerance)
		}
		if !opts.NaNEqualsNaN {
			t.Error("NaNEqualsNaN should be true")
		}
	})

	t.Run("invalid tolerance mode", func(t *testing.T) {
		t.Parallel()
		_, err := NewCompareOptions("invalid", ArrayOrderStrict, 1e-6, true)
		if err == nil {
			t.Error("NewCompareOptions() expected error for invalid ToleranceMode")
		}
		if !strings.Contains(err.Error(), "ToleranceMode") {
			t.Errorf("error should mention ToleranceMode, got: %v", err)
		}
	})

	t.Run("invalid array order", func(t *testing.T) {
		t.Parallel()
		_, err := NewCompareOptions(ToleranceModeRelative, "invalid", 1e-6, true)
		if err == nil {
			t.Error("NewCompareOptions() expected error for invalid ArrayOrder")
		}
		if !strings.Contains(err.Error(), "ArrayOrder") {
			t.Errorf("error should mention ArrayOrder, got: %v", err)
		}
	})

	t.Run("negative tolerance", func(t *testing.T) {
		t.Parallel()
		_, err := NewCompareOptions(ToleranceModeRelative, ArrayOrderStrict, -1, true)
		if err == nil {
			t.Error("NewCompareOptions() expected error for negative tolerance")
		}
		if !strings.Contains(err.Error(), "FloatTolerance") {
			t.Errorf("error should mention FloatTolerance, got: %v", err)
		}
	})

	t.Run("empty strings use defaults", func(t *testing.T) {
		t.Parallel()
		// Empty strings are valid and default to relative/strict
		opts, err := NewCompareOptions("", "", 1e-6, false)
		if err != nil {
			t.Fatalf("NewCompareOptions() error = %v", err)
		}
		// Empty strings are stored as-is; they're handled at comparison time
		if opts.ToleranceMode != "" {
			t.Errorf("ToleranceMode = %q, want empty string", opts.ToleranceMode)
		}
	})
}

func TestCompareOptions_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     CompareOptions
		contains []string
	}{
		{
			name: "default options",
			opts: DefaultOptions(),
			contains: []string{
				"ToleranceMode:relative",
				"FloatTolerance:1e-09",
				"NaNEqualsNaN:true",
				"ArrayOrder:strict",
			},
		},
		{
			name: "custom options",
			opts: CompareOptions{
				FloatTolerance: 0.01,
				ToleranceMode:  ToleranceModeAbsolute,
				NaNEqualsNaN:   false,
				ArrayOrder:     ArrayOrderUnordered,
			},
			contains: []string{
				"ToleranceMode:absolute",
				"FloatTolerance:0.01",
				"NaNEqualsNaN:false",
				"ArrayOrder:unordered",
			},
		},
		{
			name: "empty mode defaults to relative",
			opts: CompareOptions{
				ToleranceMode: "",
				ArrayOrder:    "",
			},
			contains: []string{
				"ToleranceMode:relative",
				"ArrayOrder:strict",
			},
		},
		{
			name: "ulp mode",
			opts: CompareOptions{
				FloatTolerance: 5,
				ToleranceMode:  ToleranceModeULP,
			},
			contains: []string{
				"ToleranceMode:ulp",
				"FloatTolerance:5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.opts.String()
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("String() = %q, want to contain %q", result, want)
				}
			}
		})
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

	// ULP mode with overflow tolerance
	ulpOverflow := CompareOptions{ToleranceMode: "ulp", FloatTolerance: 1e19}
	if err := ValidateOptions(ulpOverflow); err == nil {
		t.Error("ValidateOptions with ULP tolerance exceeding MaxInt64 should return error")
	}

	// ULP mode with valid large tolerance (just under MaxInt64)
	ulpValid := CompareOptions{ToleranceMode: "ulp", FloatTolerance: float64(math.MaxInt64 - 1)}
	if err := ValidateOptions(ulpValid); err != nil {
		t.Errorf("ValidateOptions with valid ULP tolerance should not return error: %v", err)
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

func TestEqual_UlpTolerance(t *testing.T) {
	// Test ULP tolerance mode
	a := 1.0
	b := math.Nextafter(a, math.Inf(1)) // 1 ULP away

	// With ULP tolerance of 1, should pass
	opts := CompareOptions{
		FloatTolerance: 1,
		ToleranceMode:  "ulp",
	}
	if !Equal(a, b, opts) {
		t.Error("expected 1 ULP difference to pass with tolerance 1")
	}

	// With ULP tolerance of 0, should fail
	opts.FloatTolerance = 0
	if Equal(a, b, opts) {
		t.Error("expected 1 ULP difference to fail with tolerance 0")
	}

	// 3 ULPs away with tolerance of 2 should fail
	c := math.Nextafter(math.Nextafter(math.Nextafter(a, math.Inf(1)), math.Inf(1)), math.Inf(1))
	opts.FloatTolerance = 2
	if Equal(a, c, opts) {
		t.Error("expected 3 ULP difference to fail with tolerance 2")
	}

	// 3 ULPs away with tolerance of 3 should pass
	opts.FloatTolerance = 3
	if !Equal(a, c, opts) {
		t.Error("expected 3 ULP difference to pass with tolerance 3")
	}
}

func TestEqual_TypeMismatch(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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

func TestEqual_FloatTypeMismatch(t *testing.T) {
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

func TestEqual_SpecialFloatTypeMismatch(t *testing.T) {
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

func TestEqual_SpecialFloatMismatch(t *testing.T) {
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
	if Equal(math.Inf(1), float64(42), opts) {
		t.Error("expected +Inf vs regular to fail")
	}

	// -Inf vs regular number
	if Equal(math.Inf(-1), float64(42), opts) {
		t.Error("expected -Inf vs regular to fail")
	}

	// +Inf vs -Inf
	if Equal(math.Inf(1), math.Inf(-1), opts) {
		t.Error("expected +Inf vs -Inf to fail")
	}

	// NaN vs regular (with NaNEqualsNaN=false)
	opts.NaNEqualsNaN = false
	if Equal(math.NaN(), float64(42), opts) {
		t.Error("expected NaN vs regular to fail")
	}

	// Regular vs NaN
	if Equal(float64(42), math.NaN(), opts) {
		t.Error("expected regular vs NaN to fail")
	}
}

func TestFloatsEqual_NaNWithoutFlag(t *testing.T) {
	opts := CompareOptions{
		NaNEqualsNaN: false,
	}

	// NaN vs NaN with flag disabled
	if Equal(math.NaN(), math.NaN(), opts) {
		t.Error("expected NaN vs NaN to fail when NaNEqualsNaN is false")
	}
}

func TestEqual_ArrayLengthMismatch(t *testing.T) {
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

func TestEqual_UnorderedArrayNoMatch(t *testing.T) {
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

func TestEqual_UnorderedArray_Duplicates(t *testing.T) {
	t.Parallel()
	opts := CompareOptions{
		ArrayOrder: ArrayOrderUnordered,
	}

	tests := []struct {
		name     string
		expected []interface{}
		actual   []interface{}
		pass     bool
	}{
		{
			name:     "same duplicates same order",
			expected: []interface{}{float64(1), float64(1), float64(2)},
			actual:   []interface{}{float64(1), float64(1), float64(2)},
			pass:     true,
		},
		{
			name:     "same duplicates different order",
			expected: []interface{}{float64(1), float64(1), float64(2)},
			actual:   []interface{}{float64(2), float64(1), float64(1)},
			pass:     true,
		},
		{
			name:     "different duplicate counts",
			expected: []interface{}{float64(1), float64(1), float64(2)},
			actual:   []interface{}{float64(1), float64(2), float64(2)},
			pass:     false,
		},
		{
			name:     "all duplicates matching",
			expected: []interface{}{float64(5), float64(5), float64(5)},
			actual:   []interface{}{float64(5), float64(5), float64(5)},
			pass:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if Equal(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("Equal() = %v, want %v", !tt.pass, tt.pass)
			}
		})
	}
}

func TestEqual_UnorderedArray_Empty(t *testing.T) {
	t.Parallel()
	opts := CompareOptions{
		ArrayOrder: ArrayOrderUnordered,
	}

	// Both empty should match
	if !Equal([]interface{}{}, []interface{}{}, opts) {
		t.Error("expected empty arrays to match")
	}

	// Empty vs non-empty should not match
	if Equal([]interface{}{}, []interface{}{float64(1)}, opts) {
		t.Error("expected empty vs non-empty to not match")
	}
}

func TestEqual_UnorderedArray_NestedObjects(t *testing.T) {
	t.Parallel()
	opts := CompareOptions{
		ArrayOrder: ArrayOrderUnordered,
	}

	tests := []struct {
		name     string
		expected []interface{}
		actual   []interface{}
		pass     bool
	}{
		{
			name: "nested objects same order",
			expected: []interface{}{
				map[string]interface{}{"a": float64(1)},
				map[string]interface{}{"b": float64(2)},
			},
			actual: []interface{}{
				map[string]interface{}{"a": float64(1)},
				map[string]interface{}{"b": float64(2)},
			},
			pass: true,
		},
		{
			name: "nested objects different order",
			expected: []interface{}{
				map[string]interface{}{"a": float64(1)},
				map[string]interface{}{"b": float64(2)},
			},
			actual: []interface{}{
				map[string]interface{}{"b": float64(2)},
				map[string]interface{}{"a": float64(1)},
			},
			pass: true,
		},
		{
			name: "nested objects with mismatch",
			expected: []interface{}{
				map[string]interface{}{"a": float64(1)},
				map[string]interface{}{"b": float64(2)},
			},
			actual: []interface{}{
				map[string]interface{}{"a": float64(1)},
				map[string]interface{}{"b": float64(3)}, // different value
			},
			pass: false,
		},
		{
			name: "nested arrays in objects",
			expected: []interface{}{
				map[string]interface{}{"values": []interface{}{float64(1), float64(2)}},
			},
			actual: []interface{}{
				map[string]interface{}{"values": []interface{}{float64(1), float64(2)}},
			},
			pass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if Equal(tt.expected, tt.actual, opts) != tt.pass {
				t.Errorf("Equal() = %v, want %v", !tt.pass, tt.pass)
			}
		})
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

func TestEqual_NilMismatch(t *testing.T) {
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

// TestEqual_InvalidOptions_Panics verifies that Equal panics with invalid options.
// This is important because Equal's signature (returning only bool) doesn't hint
// at panic behavior, but it delegates to Compare which panics on invalid options.
func TestEqual_InvalidOptions_Panics(t *testing.T) {
	tests := []struct {
		name string
		opts CompareOptions
	}{
		{"invalid ToleranceMode", CompareOptions{ToleranceMode: "fuzzy"}},
		{"invalid ArrayOrder", CompareOptions{ArrayOrder: "scrambled"}},
		{"negative tolerance", CompareOptions{FloatTolerance: -0.1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Equal should panic with %s", tt.name)
				}
			}()

			Equal(1.0, 1.0, tt.opts)
		})
	}
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
	if !Equal(1.0, 1.005, opts) {
		t.Error("expected absolute tolerance to pass")
	}

	// Array comparison with unordered mode
	expected := []interface{}{float64(1), float64(2), float64(3)}
	actual := []interface{}{float64(3), float64(1), float64(2)}
	if !Equal(expected, actual, opts) {
		t.Error("expected unordered array comparison to pass")
	}
}

func TestEqual_UlpTolerance_Truncation(t *testing.T) {
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
	if !Equal(a, b, opts) {
		t.Error("expected 1 ULP to pass with tolerance 1.9 (truncated to 1)")
	}

	// With tolerance 1.9, truncated to 1, should fail for 2 ULPs
	if Equal(a, c, opts) {
		t.Error("expected 2 ULPs to fail with tolerance 1.9 (truncated to 1)")
	}
}

func TestEqual_UlpTolerance_ZeroTolerance(t *testing.T) {
	// Test ULP mode with zero tolerance (exact match required)
	a := 1.0
	b := math.Nextafter(a, math.Inf(1)) // 1 ULP away

	opts := CompareOptions{
		FloatTolerance: 0,
		ToleranceMode:  ToleranceModeULP,
	}

	// Same value should pass
	if !Equal(a, a, opts) {
		t.Error("expected identical values to pass with ULP tolerance 0")
	}

	// 1 ULP away should fail
	if Equal(a, b, opts) {
		t.Error("expected 1 ULP difference to fail with tolerance 0")
	}
}

func TestEqual_UlpTolerance_LargeTolerance(t *testing.T) {
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
	if !Equal(a, b, opts) {
		t.Error("expected large ULP tolerance to pass for 1.0 vs 2.0")
	}
}

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

func TestCompare_PlusInfinityString(t *testing.T) {
	opts := DefaultOptions()

	// "+Infinity" should match positive infinity (same as "Infinity")
	if !Equal("+Infinity", math.Inf(1), opts) {
		t.Error("expected \"+Infinity\" string to match math.Inf(1)")
	}

	// "+Infinity" should NOT match negative infinity
	if Equal("+Infinity", math.Inf(-1), opts) {
		t.Error("expected \"+Infinity\" string to NOT match math.Inf(-1)")
	}

	// "+Infinity" should NOT match regular numbers
	if Equal("+Infinity", float64(42), opts) {
		t.Error("expected \"+Infinity\" string to NOT match regular number")
	}

	// "+Infinity" should NOT match NaN
	if Equal("+Infinity", math.NaN(), opts) {
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

func TestULPDiff_BasicCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    float64
		b    float64
		want int64
	}{
		{"identical", 1.0, 1.0, 0},
		{"adjacent", 1.0, math.Nextafter(1.0, 2.0), 1},
		{"symmetric", 1.0, 1.5, ULPDiff(1.5, 1.0)}, // ULPDiff(a,b) == ULPDiff(b,a)
		{"zero", 0.0, 0.0, 0},
		{"negative zero", 0.0, math.Copysign(0, -1), 0}, // -0 and +0 differ by 0 ULPs in practice
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ULPDiff(tt.a, tt.b)
			if tt.name == "symmetric" {
				// For symmetric test, verify a == b
				if got != tt.want {
					t.Errorf("ULPDiff(%v, %v) != ULPDiff(%v, %v)", tt.a, tt.b, tt.b, tt.a)
				}
			} else if got != tt.want {
				t.Errorf("ULPDiff(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestULPDiff_Symmetry(t *testing.T) {
	t.Parallel()

	// Verify ULPDiff(a, b) == ULPDiff(b, a) for various value pairs
	pairs := [][2]float64{
		{1.0, 2.0},
		{-1.0, 1.0},
		{0.0, 1.0},
		{math.SmallestNonzeroFloat64, 0.0},
		{1.0, math.Nextafter(1.0, 2.0)},
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		ab := ULPDiff(a, b)
		ba := ULPDiff(b, a)
		if ab != ba {
			t.Errorf("ULPDiff(%v, %v) = %d != ULPDiff(%v, %v) = %d", a, b, ab, b, a, ba)
		}
	}
}

func TestULPDiff_AdjacentValues(t *testing.T) {
	t.Parallel()

	// Adjacent representable values should differ by exactly 1 ULP
	base := 1.0
	next := math.Nextafter(base, 2.0)
	prev := math.Nextafter(base, 0.0)

	if got := ULPDiff(base, next); got != 1 {
		t.Errorf("ULPDiff(1.0, nextafter(1.0, 2.0)) = %d, want 1", got)
	}

	if got := ULPDiff(base, prev); got != 1 {
		t.Errorf("ULPDiff(1.0, nextafter(1.0, 0.0)) = %d, want 1", got)
	}
}

func TestULPDiff_SpecialValues(t *testing.T) {
	t.Parallel()

	nan := math.NaN()
	posInf := math.Inf(1)
	negInf := math.Inf(-1)

	// Identical special values return 0
	if got := ULPDiff(nan, nan); got != 0 {
		t.Errorf("ULPDiff(NaN, NaN) = %d, want 0", got)
	}
	if got := ULPDiff(posInf, posInf); got != 0 {
		t.Errorf("ULPDiff(+Inf, +Inf) = %d, want 0", got)
	}
	if got := ULPDiff(negInf, negInf); got != 0 {
		t.Errorf("ULPDiff(-Inf, -Inf) = %d, want 0", got)
	}

	// NaN vs finite returns large value (not meaningful, but predictable)
	if got := ULPDiff(nan, 0); got <= 0 {
		t.Errorf("ULPDiff(NaN, 0) = %d, expected positive value", got)
	}

	// +Inf vs -Inf returns large value
	if got := ULPDiff(posInf, negInf); got <= 0 {
		t.Errorf("ULPDiff(+Inf, -Inf) = %d, expected positive value", got)
	}

	// Symmetry still holds for special values
	if ULPDiff(nan, 0) != ULPDiff(0, nan) {
		t.Error("ULPDiff symmetry violated for NaN")
	}
	if ULPDiff(posInf, negInf) != ULPDiff(negInf, posInf) {
		t.Error("ULPDiff symmetry violated for infinities")
	}
}

// Tests for CompareE

func TestCompareE_ValidOptions_MatchesCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		opts     CompareOptions
	}{
		{"equal floats", 1.0, 1.0, DefaultOptions()},
		{"unequal floats within tolerance", 1.0, 1.0000000001, DefaultOptions()},
		{"equal strings", "hello", "hello", DefaultOptions()},
		{"unequal strings", "hello", "world", DefaultOptions()},
		{"equal arrays", []interface{}{1, 2, 3}, []interface{}{1, 2, 3}, DefaultOptions()},
		{"equal objects", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1}, DefaultOptions()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantEqual, wantDiff := Compare(tt.expected, tt.actual, tt.opts)
			gotEqual, gotDiff, err := CompareE(tt.expected, tt.actual, tt.opts)

			if err != nil {
				t.Errorf("CompareE() returned unexpected error: %v", err)
			}
			if gotEqual != wantEqual {
				t.Errorf("CompareE() equal = %v, Compare() equal = %v", gotEqual, wantEqual)
			}
			if gotDiff != wantDiff {
				t.Errorf("CompareE() diff = %q, Compare() diff = %q", gotDiff, wantDiff)
			}
		})
	}
}

func TestCompareE_InvalidToleranceMode_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  "invalid",
	}

	equal, diff, err := CompareE(1.0, 1.0, opts)

	if err == nil {
		t.Error("CompareE() should return error for invalid ToleranceMode")
	}
	if equal {
		t.Error("CompareE() should return equal=false when err is non-nil")
	}
	if diff != "" {
		t.Errorf("CompareE() should return empty diff when err is non-nil, got %q", diff)
	}
	if !strings.Contains(err.Error(), "invalid ToleranceMode") {
		t.Errorf("error should mention invalid ToleranceMode, got: %v", err)
	}
}

func TestCompareE_InvalidArrayOrder_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		ArrayOrder: "random",
	}

	equal, diff, err := CompareE([]interface{}{1}, []interface{}{1}, opts)

	if err == nil {
		t.Error("CompareE() should return error for invalid ArrayOrder")
	}
	if equal {
		t.Error("CompareE() should return equal=false when err is non-nil")
	}
	if diff != "" {
		t.Errorf("CompareE() should return empty diff when err is non-nil, got %q", diff)
	}
	if !strings.Contains(err.Error(), "invalid ArrayOrder") {
		t.Errorf("error should mention invalid ArrayOrder, got: %v", err)
	}
}

func TestCompareE_NegativeTolerance_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		FloatTolerance: -1.0,
	}

	equal, diff, err := CompareE(1.0, 1.0, opts)

	if err == nil {
		t.Error("CompareE() should return error for negative tolerance")
	}
	if equal {
		t.Error("CompareE() should return equal=false when err is non-nil")
	}
	if diff != "" {
		t.Errorf("CompareE() should return empty diff when err is non-nil, got %q", diff)
	}
	if !strings.Contains(err.Error(), "FloatTolerance") {
		t.Errorf("error should mention FloatTolerance, got: %v", err)
	}
}

func TestCompareE_DoesNotPanic_InvalidOptions(t *testing.T) {
	t.Parallel()

	invalidOptionsList := []CompareOptions{
		{ToleranceMode: "fuzzy"},
		{ArrayOrder: "scrambled"},
		{FloatTolerance: -0.1},
	}

	for _, opts := range invalidOptionsList {
		t.Run(opts.String(), func(t *testing.T) {
			t.Parallel()
			// This should NOT panic
			_, _, err := CompareE(1.0, 1.0, opts)
			if err == nil {
				t.Error("CompareE() should return error for invalid options")
			}
		})
	}
}

// Tests for EqualE

func TestEqualE_ValidOptions_MatchesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		opts     CompareOptions
	}{
		{"equal floats", 1.0, 1.0, DefaultOptions()},
		{"unequal floats within tolerance", 1.0, 1.0000000001, DefaultOptions()},
		{"equal strings", "hello", "hello", DefaultOptions()},
		{"unequal strings", "hello", "world", DefaultOptions()},
		{"equal arrays", []interface{}{1, 2, 3}, []interface{}{1, 2, 3}, DefaultOptions()},
		{"equal objects", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1}, DefaultOptions()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantEqual := Equal(tt.expected, tt.actual, tt.opts)
			gotEqual, err := EqualE(tt.expected, tt.actual, tt.opts)

			if err != nil {
				t.Errorf("EqualE() returned unexpected error: %v", err)
			}
			if gotEqual != wantEqual {
				t.Errorf("EqualE() = %v, Equal() = %v", gotEqual, wantEqual)
			}
		})
	}
}

func TestEqualE_InvalidToleranceMode_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		FloatTolerance: 0.01,
		ToleranceMode:  "invalid",
	}

	equal, err := EqualE(1.0, 1.0, opts)

	if err == nil {
		t.Error("EqualE() should return error for invalid ToleranceMode")
	}
	if equal {
		t.Error("EqualE() should return equal=false when err is non-nil")
	}
	if !strings.Contains(err.Error(), "invalid ToleranceMode") {
		t.Errorf("error should mention invalid ToleranceMode, got: %v", err)
	}
}

func TestEqualE_InvalidArrayOrder_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		ArrayOrder: "random",
	}

	equal, err := EqualE([]interface{}{1}, []interface{}{1}, opts)

	if err == nil {
		t.Error("EqualE() should return error for invalid ArrayOrder")
	}
	if equal {
		t.Error("EqualE() should return equal=false when err is non-nil")
	}
	if !strings.Contains(err.Error(), "invalid ArrayOrder") {
		t.Errorf("error should mention invalid ArrayOrder, got: %v", err)
	}
}

func TestEqualE_NegativeTolerance_ReturnsError(t *testing.T) {
	t.Parallel()

	opts := CompareOptions{
		FloatTolerance: -1.0,
	}

	equal, err := EqualE(1.0, 1.0, opts)

	if err == nil {
		t.Error("EqualE() should return error for negative tolerance")
	}
	if equal {
		t.Error("EqualE() should return equal=false when err is non-nil")
	}
	if !strings.Contains(err.Error(), "FloatTolerance") {
		t.Errorf("error should mention FloatTolerance, got: %v", err)
	}
}

func TestEqualE_DoesNotPanic_InvalidOptions(t *testing.T) {
	t.Parallel()

	invalidOptionsList := []CompareOptions{
		{ToleranceMode: "fuzzy"},
		{ArrayOrder: "scrambled"},
		{FloatTolerance: -0.1},
	}

	for _, opts := range invalidOptionsList {
		t.Run(opts.String(), func(t *testing.T) {
			t.Parallel()
			// This should NOT panic
			_, err := EqualE(1.0, 1.0, opts)
			if err == nil {
				t.Error("EqualE() should return error for invalid options")
			}
		})
	}
}
