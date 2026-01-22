package testhelper

// Note: internal/tests/compare.go contains similar comparison logic but uses
// different types (ComparisonConfig vs CompareOptions) and error message formats.
// The duplication is intentional to maintain API stability for external consumers.

import (
	"fmt"
	"math"
	"strings"
)

// CompareOptions configures output comparison behavior.
//
// # Zero Value vs DefaultOptions
//
// The zero value of CompareOptions is valid and usable, but differs from
// [DefaultOptions]:
//
//	| Field          | Zero Value       | DefaultOptions()  |
//	|----------------|------------------|-------------------|
//	| FloatTolerance | 0 (exact match)  | 1e-9              |
//	| ToleranceMode  | "" (→ relative)  | "relative"        |
//	| NaNEqualsNaN   | false            | true              |
//	| ArrayOrder     | "" (→ strict)    | "strict"          |
//
// The zero value requires exact float equality, which is rarely appropriate
// for computed results. For typical use cases, prefer [DefaultOptions] or
// [NewCompareOptions].
//
// Empty strings for ToleranceMode and ArrayOrder are treated as defaults:
//   - ToleranceMode "" → ToleranceModeRelative
//   - ArrayOrder "" → ArrayOrderStrict
//
// While empty strings work, prefer explicit values for clarity. Use [DefaultOptions]
// as the canonical way to get defaults, then override specific fields as needed:
//
//	opts := testhelper.DefaultOptions()
//	opts.FloatTolerance = 1e-6  // customize one field
//
// # Construction
//
// Three ways to create CompareOptions:
//
//  1. Default options (recommended): opts := testhelper.DefaultOptions()
//  2. Validated custom options: opts, err := testhelper.NewCompareOptions(...)
//  3. Zero value (exact equality): var opts CompareOptions
//
// Direct struct construction is valid but NOT validated:
//
//	opts := CompareOptions{ToleranceMode: "invalid"}  // compiles but panics on use
//	opts := CompareOptions{FloatTolerance: -1.0}      // compiles but panics on use
//
// Use [NewCompareOptionsOrdered] for validated construction, or call [ValidateOptions]
// before comparison to check for invalid values. Direct struct construction with
// invalid fields (negative tolerance, unknown mode strings) will panic when passed
// to comparison functions.
//
// # String Fields
//
// Important: Use the provided constants (ToleranceModeRelative, ArrayOrderStrict,
// etc.) for ToleranceMode and ArrayOrder fields. Arbitrary strings are rejected
// by [ValidateOptions] with an error listing valid values.
//
// # Panic vs Error API Design
//
// The comparison functions ([Equal], [Compare], [FormatComparisonResult]) panic
// on invalid options. This is intentional: options are typically compile-time
// constants or loaded from static configuration, so invalid options represent
// programmer errors rather than runtime conditions. Panicking fails fast during
// development.
//
// For dynamic or user-provided options, use the error-returning variants
// ([EqualE], [CompareE]) or call [ValidateOptions] before comparison:
//
//	// Option 1: Validate upfront
//	if err := testhelper.ValidateOptions(opts); err != nil {
//	    return err
//	}
//	result := testhelper.Equal(expected, actual, opts)  // safe, won't panic
//
//	// Option 2: Use error-returning variant
//	result, err := testhelper.EqualE(expected, actual, opts)
//
// The panic functions are the primary API; the *E variants are escape hatches.
//
// # JSON Schema Field Mapping
//
// When using CompareOptions in JSON configuration (e.g., structyl.json tests section),
// field names use snake_case per JSON conventions:
//
//	| Go Field       | JSON Field       |
//	|----------------|------------------|
//	| FloatTolerance | float_tolerance  |
//	| ToleranceMode  | tolerance_mode   |
//	| NaNEqualsNaN   | nan_equals_nan   |
//	| ArrayOrder     | array_order      |
type CompareOptions struct {
	// FloatTolerance specifies the tolerance for float comparisons.
	// For "relative" and "absolute" modes, this is the tolerance threshold.
	// For "ulp" mode, this value is truncated to an integer representing the
	// maximum allowed ULP (Units in Last Place) difference. For example,
	// a tolerance of 1.9 allows 1 ULP difference, not 2.
	FloatTolerance float64 `json:"float_tolerance"`

	// ToleranceMode specifies how tolerance is applied.
	// Use the ToleranceMode* constants: ToleranceModeRelative (default),
	// ToleranceModeAbsolute, or ToleranceModeULP.
	// Empty string ("") is treated as ToleranceModeRelative.
	//
	// For "relative" mode: comparison passes if |expected - actual| / |expected| <= tolerance.
	// Edge case: when expected == 0, the formula changes to |actual| <= tolerance,
	// since division by zero is undefined.
	//
	// For "ulp" mode, FloatTolerance is truncated to int64 for ULP distance
	// calculation. Practical limit: FloatTolerance values should be less than
	// 2^63-1 (approximately 9.2e18). For tolerances exceeding this limit,
	// comparison results may be incorrect due to integer overflow. In practice,
	// ULP tolerances above 1e15 are rarely meaningful since the total number of
	// representable floats between 1.0 and 2.0 is approximately 4.5e15.
	ToleranceMode string `json:"tolerance_mode"`

	// NaNEqualsNaN treats NaN values as equal when true.
	// This applies to both float64 NaN values and special string representations
	// ("NaN") used in JSON test case outputs. See [SpecialFloatNaN].
	NaNEqualsNaN bool `json:"nan_equals_nan"`

	// ArrayOrder specifies array comparison order.
	// Use the ArrayOrder* constants: ArrayOrderStrict (default) or
	// ArrayOrderUnordered. Empty string ("") is treated as ArrayOrderStrict.
	ArrayOrder string `json:"array_order"`
}

// ToleranceMode constants for CompareOptions.ToleranceMode.
// Using these constants prevents typos and enables IDE autocomplete.
const (
	// ToleranceModeRelative compares floats using relative tolerance.
	// The comparison passes if |expected - actual| / |expected| <= tolerance.
	ToleranceModeRelative = "relative"

	// ToleranceModeAbsolute compares floats using absolute tolerance.
	// The comparison passes if |expected - actual| <= tolerance.
	ToleranceModeAbsolute = "absolute"

	// ToleranceModeULP compares floats using ULP (Units in Last Place) distance.
	// FloatTolerance is truncated to int64 for ULP calculation.
	ToleranceModeULP = "ulp"
)

// ArrayOrder constants for CompareOptions.ArrayOrder.
// Using these constants prevents typos and enables IDE autocomplete.
const (
	// ArrayOrderStrict requires array elements to match in order.
	ArrayOrderStrict = "strict"

	// ArrayOrderUnordered allows array elements to match in any order.
	//
	// Performance: Uses O(n²) matching algorithm. For arrays with >1000 elements,
	// consider using ArrayOrderStrict with sorted/deterministic output, or
	// breaking large outputs into smaller assertions.
	ArrayOrderUnordered = "unordered"
)

// Special float string constants for JSON test cases.
// JSON cannot represent NaN or Infinity directly, so these strings are used
// in expected output to indicate special float values.
// Using these constants prevents typos and enables IDE autocomplete.
//
// IMPORTANT: These strings are case-sensitive. Only the exact strings defined
// below trigger special handling. For example, "NaN" works but "nan", "NAN",
// and "Nan" are treated as regular strings.
const (
	// SpecialFloatNaN matches actual NaN values (per NaNEqualsNaN option).
	// Case-sensitive: only "NaN" triggers special handling.
	SpecialFloatNaN = "NaN"

	// SpecialFloatInfinity matches actual positive infinity (+Inf).
	// This is the canonical representation for positive infinity; prefer this
	// over SpecialFloatPosInfinity for consistency. Case-sensitive: only
	// "Infinity" triggers special handling.
	SpecialFloatInfinity = "Infinity"

	// SpecialFloatPosInfinity matches actual positive infinity (+Inf).
	// Equivalent to SpecialFloatInfinity; use when explicit "+" is desired
	// for clarity. Prefer SpecialFloatInfinity as the canonical form.
	// Case-sensitive: only "+Infinity" triggers special handling.
	//
	// Deprecated: Use [SpecialFloatInfinity] instead. SpecialFloatPosInfinity
	// will be removed in v2.0.0.
	SpecialFloatPosInfinity = "+Infinity"

	// SpecialFloatNegInfinity matches actual negative infinity (-Inf).
	// Case-sensitive: only "-Infinity" triggers special handling.
	SpecialFloatNegInfinity = "-Infinity"
)

// String returns a human-readable representation of CompareOptions for debugging.
// Empty fields are normalized to their default values in the output
// (e.g., empty ToleranceMode displays as "relative").
//
// The output format is not stable and may change between versions.
// Do not parse or rely on this output in tests or production code.
func (o CompareOptions) String() string {
	mode := o.ToleranceMode
	if mode == "" {
		mode = "relative"
	}
	order := o.ArrayOrder
	if order == "" {
		order = "strict"
	}
	return fmt.Sprintf("CompareOptions{ToleranceMode:%s, FloatTolerance:%g, NaNEqualsNaN:%v, ArrayOrder:%s}",
		mode, o.FloatTolerance, o.NaNEqualsNaN, order)
}

// DefaultOptions returns the default comparison options.
func DefaultOptions() CompareOptions {
	return CompareOptions{
		FloatTolerance: 1e-9,
		ToleranceMode:  ToleranceModeRelative,
		NaNEqualsNaN:   true,
		ArrayOrder:     ArrayOrderStrict,
	}
}

// NewCompareOptions creates validated CompareOptions.
// Returns an error if any parameter is invalid.
// This constructor ensures options are valid at creation time, avoiding panics
// when options are passed to Equal, Compare, or FormatComparisonResult.
//
// Deprecated: Use [NewCompareOptionsOrdered] instead. NewCompareOptions has
// parameter order that differs from the [CompareOptions] struct field order,
// which is confusing and error-prone. NewCompareOptions will be removed in v2.0.0.
//
// Parameters:
//   - toleranceMode: Use ToleranceModeRelative, ToleranceModeAbsolute, or ToleranceModeULP
//   - arrayOrder: Use ArrayOrderStrict or ArrayOrderUnordered
//   - tolerance: Must be >= 0; for ULP mode, must fit in int64
//   - nanEqualsNaN: Whether NaN values should be considered equal
//
// Example:
//
//	opts, err := testhelper.NewCompareOptions(
//	    testhelper.ToleranceModeRelative,
//	    testhelper.ArrayOrderStrict,
//	    1e-9,
//	    true,
//	)
//	if err != nil {
//	    // handle invalid options
//	}
//
// For a constructor with parameter order matching the struct field order,
// see [NewCompareOptionsOrdered].
func NewCompareOptions(toleranceMode, arrayOrder string, tolerance float64, nanEqualsNaN bool) (CompareOptions, error) {
	opts := CompareOptions{
		ToleranceMode:  toleranceMode,
		ArrayOrder:     arrayOrder,
		FloatTolerance: tolerance,
		NaNEqualsNaN:   nanEqualsNaN,
	}
	if err := ValidateOptions(opts); err != nil {
		return CompareOptions{}, err
	}
	return opts, nil
}

// NewCompareOptionsOrdered creates validated CompareOptions with parameters in
// struct field order. This constructor is preferred over [NewCompareOptions]
// because the parameter order matches the [CompareOptions] struct definition,
// reducing confusion when constructing options.
//
// Parameters (in struct field order):
//   - tolerance: FloatTolerance value; must be >= 0; for ULP mode, must fit in int64
//   - toleranceMode: Use ToleranceModeRelative, ToleranceModeAbsolute, or ToleranceModeULP
//   - nanEqualsNaN: Whether NaN values should be considered equal
//   - arrayOrder: Use ArrayOrderStrict or ArrayOrderUnordered
//
// Returns an error if any parameter is invalid.
//
// Example:
//
//	opts, err := testhelper.NewCompareOptionsOrdered(
//	    1e-9,                              // FloatTolerance
//	    testhelper.ToleranceModeRelative,  // ToleranceMode
//	    true,                              // NaNEqualsNaN
//	    testhelper.ArrayOrderStrict,       // ArrayOrder
//	)
//	if err != nil {
//	    // handle invalid options
//	}
func NewCompareOptionsOrdered(tolerance float64, toleranceMode string, nanEqualsNaN bool, arrayOrder string) (CompareOptions, error) {
	return NewCompareOptions(toleranceMode, arrayOrder, tolerance, nanEqualsNaN)
}

// ValidateOptions validates that CompareOptions has valid enum values.
// Returns nil if valid, or an error describing the invalid field.
//
// # Panic vs Error Design
//
// The comparison functions (Equal, Compare, FormatComparisonResult) panic on
// invalid options rather than returning an error. This design follows the
// principle that invalid options represent programmer errors, not runtime
// conditions:
//
//   - Options are typically hardcoded constants or loaded from static config
//   - Invalid options indicate a bug in the calling code, not bad input data
//   - Panics fail fast and loudly during development/testing
//   - Callers who need graceful handling can call ValidateOptions first
//
// For callers who want to validate options before calling comparison functions
// (e.g., when options come from user input), call ValidateOptions explicitly:
//
//	if err := testhelper.ValidateOptions(opts); err != nil {
//	    // handle error
//	}
//	// Safe to call Equal/Compare now
func ValidateOptions(opts CompareOptions) error {
	if opts.FloatTolerance < 0 {
		return fmt.Errorf("invalid FloatTolerance: %v (must be >= 0)", opts.FloatTolerance)
	}
	switch opts.ToleranceMode {
	case "", ToleranceModeRelative, ToleranceModeAbsolute, ToleranceModeULP:
		// valid (empty defaults to relative)
	default:
		return fmt.Errorf("invalid ToleranceMode: %q (must be \"relative\", \"absolute\", or \"ulp\")", opts.ToleranceMode)
	}
	// ULP mode: validate tolerance fits in int64 to prevent overflow
	if opts.ToleranceMode == ToleranceModeULP && opts.FloatTolerance > float64(math.MaxInt64) {
		return fmt.Errorf("invalid FloatTolerance for ULP mode: %v exceeds max int64 (%d)", opts.FloatTolerance, int64(math.MaxInt64))
	}
	switch opts.ArrayOrder {
	case "", ArrayOrderStrict, ArrayOrderUnordered:
		// valid (empty defaults to strict)
	default:
		return fmt.Errorf("invalid ArrayOrder: %q (must be \"strict\" or \"unordered\")", opts.ArrayOrder)
	}
	return nil
}

// IsValid returns true if opts contains valid configuration values.
// This is a convenience wrapper around [ValidateOptions] for cases where
// boolean checking is preferred over error handling.
//
// Use IsValid for conditional fallback:
//
//	if !opts.IsValid() {
//	    opts = DefaultOptions()
//	}
//
// Use [ValidateOptions] when you need error details:
//
//	if err := ValidateOptions(opts); err != nil {
//	    return fmt.Errorf("invalid options: %w", err)
//	}
func (o CompareOptions) IsValid() bool {
	return ValidateOptions(o) == nil
}

// IsZero reports whether opts is the zero value of CompareOptions.
//
// Note: The zero value differs from [DefaultOptions]:
//
//	| Field          | Zero Value       | DefaultOptions()  |
//	|----------------|------------------|-------------------|
//	| FloatTolerance | 0 (exact match)  | 1e-9              |
//	| ToleranceMode  | "" (→ relative)  | "relative"        |
//	| NaNEqualsNaN   | false            | true              |
//	| ArrayOrder     | "" (→ strict)    | "strict"          |
//
// Use IsZero to detect uninitialized options and provide defaults:
//
//	if opts.IsZero() {
//	    opts = DefaultOptions()
//	}
func (o CompareOptions) IsZero() bool {
	return o == CompareOptions{}
}

// IsDefault reports whether opts equals [DefaultOptions].
//
// Use IsDefault to detect whether options have been customized:
//
//	if opts.IsDefault() {
//	    // Using standard comparison settings
//	}
func (o CompareOptions) IsDefault() bool {
	return o == DefaultOptions()
}

// WithFloatTolerance returns a copy of CompareOptions with the FloatTolerance field set.
// This enables fluent configuration:
//
//	opts := testhelper.DefaultOptions().WithFloatTolerance(1e-6)
func (o CompareOptions) WithFloatTolerance(tolerance float64) CompareOptions {
	o.FloatTolerance = tolerance
	return o
}

// WithToleranceMode returns a copy of CompareOptions with the ToleranceMode field set.
// Use the ToleranceMode* constants: [ToleranceModeRelative], [ToleranceModeAbsolute], [ToleranceModeULP].
// This enables fluent configuration:
//
//	opts := testhelper.DefaultOptions().WithToleranceMode(testhelper.ToleranceModeAbsolute)
func (o CompareOptions) WithToleranceMode(mode string) CompareOptions {
	o.ToleranceMode = mode
	return o
}

// WithNaNEqualsNaN returns a copy of CompareOptions with the NaNEqualsNaN field set.
// This enables fluent configuration:
//
//	opts := testhelper.DefaultOptions().WithNaNEqualsNaN(false)
func (o CompareOptions) WithNaNEqualsNaN(nanEqualsNaN bool) CompareOptions {
	o.NaNEqualsNaN = nanEqualsNaN
	return o
}

// WithArrayOrder returns a copy of CompareOptions with the ArrayOrder field set.
// Use the ArrayOrder* constants: [ArrayOrderStrict], [ArrayOrderUnordered].
// This enables fluent configuration:
//
//	opts := testhelper.DefaultOptions().WithArrayOrder(testhelper.ArrayOrderUnordered)
func (o CompareOptions) WithArrayOrder(order string) CompareOptions {
	o.ArrayOrder = order
	return o
}

// Equal compares expected and actual outputs for equality.
// Panics on invalid opts; use [ValidateOptions] or [EqualE] for error handling.
//
// Returns true if values match according to opts.
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
func Equal(expected, actual interface{}, opts CompareOptions) bool {
	ok, _ := Compare(expected, actual, opts)
	return ok
}

// CompareOutput compares expected and actual outputs.
// Returns true if they match according to the options.
// Panics if opts contains invalid enum values (use ValidateOptions to check beforehand).
//
// Deprecated: Use [Equal] instead. CompareOutput will be removed in v2.0.0.
func CompareOutput(expected, actual interface{}, opts CompareOptions) bool {
	return Equal(expected, actual, opts)
}

// Compare compares expected and actual outputs with detailed diff.
// Panics on invalid opts; use [ValidateOptions] or [CompareE] for error handling.
//
// Returns true if values match, and a diff string describing mismatches.
//
// Panic conditions (use [ValidateOptions] to check beforehand):
//   - ToleranceMode not in {"", "relative", "absolute", "ulp"}
//   - ArrayOrder not in {"", "strict", "unordered"}
//   - FloatTolerance < 0
//   - ToleranceMode == "ulp" && FloatTolerance > math.MaxInt64
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
//
// The diff string uses JSON Path notation to identify mismatched locations:
//   - "$" represents the root value
//   - "$.foo" represents the "foo" key in a root object
//   - "$.foo[0]" represents the first element of array "foo"
//   - "$.foo.bar[2].baz" for deeply nested paths
//
// Note: For programmatic use, int actual values are converted to float64 for
// comparison. This accommodates callers who construct test data without JSON
// unmarshaling (JSON always produces float64 for numbers).
func Compare(expected, actual interface{}, opts CompareOptions) (bool, string) {
	if err := ValidateOptions(opts); err != nil {
		panic("testhelper.Compare: " + err.Error() + "; use ValidateOptions() to check options before comparison")
	}
	return compareValues(expected, actual, opts, "")
}

// CompareE compares expected and actual outputs with detailed diff.
// Returns true if they match, a diff string if they don't, and an error if
// opts contains invalid values.
//
// This is an error-returning variant of [Compare]. Use CompareE when options
// come from user input or external configuration where validation errors
// should be handled gracefully rather than causing a panic.
//
// The return values are:
//   - equal: true if expected and actual match according to opts
//   - diff: empty string if equal, otherwise a description of the first difference
//   - err: non-nil if opts is invalid
//
// When err is non-nil, equal is false and diff is empty.
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
//
// Example:
//
//	opts := testhelper.CompareOptions{ToleranceMode: userInput}
//	equal, diff, err := testhelper.CompareE(expected, actual, opts)
//	if err != nil {
//	    // Handle invalid options from user input
//	    return fmt.Errorf("invalid comparison options: %w", err)
//	}
//	if !equal {
//	    fmt.Printf("Values differ: %s\n", diff)
//	}
func CompareE(expected, actual interface{}, opts CompareOptions) (bool, string, error) {
	if err := ValidateOptions(opts); err != nil {
		return false, "", err
	}
	equal, diff := compareValues(expected, actual, opts, "")
	return equal, diff, nil
}

// EqualE compares expected and actual outputs for equality.
// Returns true if they match, or an error if opts contains invalid values.
//
// This is an error-returning variant of [Equal]. Use EqualE when options
// come from user input or external configuration where validation errors
// should be handled gracefully rather than causing a panic.
//
// EqualE wraps [CompareE], discarding the diff string. If you need the diff
// for diagnostic output, call CompareE directly.
//
// The return values are:
//   - equal: true if expected and actual match according to opts
//   - err: non-nil if opts is invalid
//
// When err is non-nil, equal is false.
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
//
// Example:
//
//	opts := testhelper.CompareOptions{ToleranceMode: userInput}
//	equal, err := testhelper.EqualE(expected, actual, opts)
//	if err != nil {
//	    // Handle invalid options from user input
//	    return fmt.Errorf("invalid comparison options: %w", err)
//	}
//	if !equal {
//	    fmt.Println("Values differ")
//	}
func EqualE(expected, actual interface{}, opts CompareOptions) (bool, error) {
	equal, _, err := CompareE(expected, actual, opts)
	return equal, err
}

func compareValues(expected, actual interface{}, opts CompareOptions, path string) (bool, string) {
	// Handle nil
	if expected == nil && actual == nil {
		return true, ""
	}
	if expected == nil || actual == nil {
		return false, fmt.Sprintf("%s: nil mismatch (expected=%v, actual=%v)", pathStr(path), expected, actual)
	}

	// Handle special float strings
	if expStr, ok := expected.(string); ok {
		if expStr == SpecialFloatNaN || expStr == SpecialFloatInfinity ||
			expStr == SpecialFloatPosInfinity || expStr == SpecialFloatNegInfinity {
			return compareSpecialFloat(expStr, actual, opts, path)
		}
	}

	// Type-specific comparison
	switch e := expected.(type) {
	case float64:
		return compareFloat(e, actual, opts, path)
	case int:
		return compareFloat(float64(e), actual, opts, path)
	case []interface{}:
		return compareArray(e, actual, opts, path)
	case map[string]interface{}:
		return compareObject(e, actual, opts, path)
	case string:
		if a, ok := actual.(string); ok {
			if e == a {
				return true, ""
			}
			return false, fmt.Sprintf("%s: string mismatch (expected=%q, actual=%q)", pathStr(path), e, a)
		}
		return false, fmt.Sprintf("%s: type mismatch (expected=string, actual=%T)", pathStr(path), actual)
	case bool:
		if a, ok := actual.(bool); ok {
			if e == a {
				return true, ""
			}
			return false, fmt.Sprintf("%s: bool mismatch (expected=%v, actual=%v)", pathStr(path), e, a)
		}
		return false, fmt.Sprintf("%s: type mismatch (expected=bool, actual=%T)", pathStr(path), actual)
	default:
		if expected == actual {
			return true, ""
		}
		return false, fmt.Sprintf("%s: value mismatch (expected=%v, actual=%v)", pathStr(path), expected, actual)
	}
}

// compareFloat compares a float64 expected value against an actual value.
// Handles int as actual type for convenience (JSON integers like "expected: 1"
// are sometimes decoded as int rather than float64 depending on context).
func compareFloat(expected float64, actual interface{}, opts CompareOptions, path string) (bool, string) {
	var a float64
	switch v := actual.(type) {
	case float64:
		a = v
	case int:
		a = float64(v)
	default:
		return false, fmt.Sprintf("%s: type mismatch (expected=float64, actual=%T)", pathStr(path), actual)
	}

	if floatsEqual(expected, a, opts) {
		return true, ""
	}
	return false, fmt.Sprintf("%s: float mismatch (expected=%v, actual=%v)", pathStr(path), expected, a)
}

func floatsEqual(expected, actual float64, opts CompareOptions) bool {
	// Handle NaN
	if math.IsNaN(expected) && math.IsNaN(actual) {
		return opts.NaNEqualsNaN
	}

	// Handle infinity
	if math.IsInf(expected, 1) && math.IsInf(actual, 1) {
		return true
	}
	if math.IsInf(expected, -1) && math.IsInf(actual, -1) {
		return true
	}

	// Exact equality for special values
	if math.IsNaN(expected) || math.IsNaN(actual) ||
		math.IsInf(expected, 0) || math.IsInf(actual, 0) {
		return false
	}

	switch opts.ToleranceMode {
	case "", ToleranceModeRelative:
		// Relative mode (default when ToleranceMode is empty string).
		if expected == 0 {
			return math.Abs(actual) <= opts.FloatTolerance
		}
		return math.Abs((expected-actual)/expected) <= opts.FloatTolerance
	case ToleranceModeAbsolute:
		return math.Abs(expected-actual) <= opts.FloatTolerance
	case ToleranceModeULP:
		return ulpDiff(expected, actual) <= int64(opts.FloatTolerance)
	default:
		// ValidateOptions ensures this is unreachable for properly validated options.
		// Panic to catch programming errors (invalid options passed without validation).
		panic("testhelper.floatsEqual: invalid ToleranceMode: " + opts.ToleranceMode + "; use ValidateOptions() to check options before comparison")
	}
}

// ULPDiff returns the ULP (Units in Last Place) distance between two float64 values.
// This measures how many representable floating-point values exist between a and b.
// Returns 0 for identical values, 1 for adjacent representable values.
// The result is always non-negative and symmetric: ULPDiff(a, b) == ULPDiff(b, a).
//
// Special cases:
//   - ULPDiff(x, x) = 0 for any x, including NaN and ±Inf
//   - ULPDiff(NaN, y) for y ≠ NaN returns a large value (~9.2e18)
//   - ULPDiff(+Inf, -Inf) returns a large value (~9e15, roughly the number
//     of representable floats between -MaxFloat64 and +MaxFloat64)
//
// Note: The returned values for NaN and infinity comparisons are mathematically
// meaningless but are predictable and symmetric. For float comparison with
// tolerance, use the Equal function which handles special values explicitly
// before calling ULPDiff.
//
// Use this function for debugging float comparisons or implementing custom
// tolerance logic based on ULP distance.
func ULPDiff(a, b float64) int64 {
	return ulpDiff(a, b)
}

func ulpDiff(a, b float64) int64 {
	ai := int64(math.Float64bits(a))
	bi := int64(math.Float64bits(b))
	if ai < 0 {
		ai = math.MinInt64 - ai
	}
	if bi < 0 {
		bi = math.MinInt64 - bi
	}
	diff := ai - bi
	if diff < 0 {
		return -diff
	}
	return diff
}

func compareSpecialFloat(expected string, actual interface{}, opts CompareOptions, path string) (bool, string) {
	var a float64
	switch v := actual.(type) {
	case float64:
		a = v
	case int:
		a = float64(v)
	default:
		return false, fmt.Sprintf("%s: type mismatch (expected=float, actual=%T)", pathStr(path), actual)
	}

	switch expected {
	case SpecialFloatNaN:
		if math.IsNaN(a) {
			if opts.NaNEqualsNaN {
				return true, ""
			}
			return false, fmt.Sprintf("%s: NaN mismatch (NaNEqualsNaN is false)", pathStr(path))
		}
		return false, fmt.Sprintf("%s: expected NaN, got %v", pathStr(path), a)
	case SpecialFloatInfinity, SpecialFloatPosInfinity:
		// Both constants match positive infinity; they are equivalent.
		// SpecialFloatPosInfinity is deprecated but handled for backwards compatibility.
		if math.IsInf(a, 1) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected +Infinity, got %v", pathStr(path), a)
	case SpecialFloatNegInfinity:
		if math.IsInf(a, -1) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected -Infinity, got %v", pathStr(path), a)
	}

	return false, fmt.Sprintf("%s: unknown special float %q", pathStr(path), expected)
}

func compareArray(expected []interface{}, actual interface{}, opts CompareOptions, path string) (bool, string) {
	a, ok := actual.([]interface{})
	if !ok {
		return false, fmt.Sprintf("%s: type mismatch (expected=array, actual=%T)", pathStr(path), actual)
	}

	if len(expected) != len(a) {
		return false, fmt.Sprintf("%s: array length mismatch (expected=%d, actual=%d)", pathStr(path), len(expected), len(a))
	}

	if opts.ArrayOrder == ArrayOrderUnordered {
		return compareUnorderedArray(expected, a, opts, path)
	}

	// Strict order comparison
	for i := range expected {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		if ok, diff := compareValues(expected[i], a[i], opts, elemPath); !ok {
			return false, diff
		}
	}

	return true, ""
}

// compareUnorderedArray performs O(n²) comparison by checking each expected element
// against unmatched actual elements. This is acceptable for typical test output sizes
// (<1000 elements). For larger arrays, a set-based comparison with hashing would be
// more efficient but requires hashable/comparable values.
//
// Performance note: For arrays with >1000 elements, comparison may be noticeably
// slow. Consider breaking large test outputs into smaller, more targeted assertions
// or using ArrayOrderStrict when order is deterministic.
func compareUnorderedArray(expected, actual []interface{}, opts CompareOptions, path string) (bool, string) {
	// Track which actual elements have been matched
	matched := make([]bool, len(actual))

	for i, exp := range expected {
		found := false
		for j, act := range actual {
			if matched[j] {
				continue
			}
			if ok, _ := compareValues(exp, act, opts, ""); ok {
				matched[j] = true
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Sprintf("%s: element %d not found in actual array", pathStr(path), i)
		}
	}

	return true, ""
}

func compareObject(expected map[string]interface{}, actual interface{}, opts CompareOptions, path string) (bool, string) {
	a, ok := actual.(map[string]interface{})
	if !ok {
		return false, fmt.Sprintf("%s: type mismatch (expected=object, actual=%T)", pathStr(path), actual)
	}

	// Check for missing keys in actual
	for key := range expected {
		if _, ok := a[key]; !ok {
			return false, fmt.Sprintf("%s.%s: missing in actual", pathStr(path), key)
		}
	}

	// Check for extra keys in actual
	for key := range a {
		if _, ok := expected[key]; !ok {
			return false, fmt.Sprintf("%s.%s: unexpected in actual", pathStr(path), key)
		}
	}

	// Compare values
	for key, exp := range expected {
		keyPath := path + "." + key
		if ok, diff := compareValues(exp, a[key], opts, keyPath); !ok {
			return false, diff
		}
	}

	return true, ""
}

// pathStr formats a path for error messages using JSON Path conventions.
// Returns "$" for empty path (JSON Path root reference).
// Strips leading dots to normalize paths like ".foo.bar" to "foo.bar".
// Note: The internal/tests version uses "root" instead of "$" for internal use.
func pathStr(path string) string {
	if path == "" {
		return "$"
	}
	return strings.TrimPrefix(path, ".")
}

// FormatDiff compares expected and actual values, returning a description.
//
// Deprecated: Use [FormatComparisonResult] instead. FormatDiff will be removed in v2.0.0.
// FormatDiff returns "values are equal" when values match, which is
// semantically inconsistent. FormatComparisonResult has clearer semantics:
// empty string on match, descriptive diff on mismatch.
func FormatDiff(expected, actual interface{}, opts CompareOptions) string {
	_, diff := Compare(expected, actual, opts)
	if diff == "" {
		return "values are equal"
	}
	return diff
}

// FormatComparisonResult compares expected and actual values, returning a
// human-readable description of any differences.
// Panics on invalid opts; use [ValidateOptions] or [FormatComparisonResultE] for error handling.
//
// Returns "" (empty string) if values match, or a descriptive diff if they differ.
//
// Panic conditions (use [ValidateOptions] to check beforehand):
//   - ToleranceMode not in {"", "relative", "absolute", "ulp"}
//   - ArrayOrder not in {"", "strict", "unordered"}
//   - FloatTolerance < 0
//   - ToleranceMode == "ulp" && FloatTolerance > math.MaxInt64
//
// This function has clearer semantics than FormatDiff: an empty result means
// "no differences" rather than returning affirmative text on match.
func FormatComparisonResult(expected, actual interface{}, opts CompareOptions) string {
	_, diff := Compare(expected, actual, opts)
	return diff
}

// FormatComparisonResultE compares expected and actual values, returning a
// human-readable description of any differences, or an error if opts is invalid.
//
// This is an error-returning variant of [FormatComparisonResult]. Use
// FormatComparisonResultE when options come from user input or external
// configuration where validation errors should be handled gracefully rather
// than causing a panic.
//
// Returns:
//   - result: empty string if values match, otherwise a description of the first difference
//   - err: non-nil if opts is invalid
//
// When err is non-nil, result is empty.
//
// Example:
//
//	opts := testhelper.CompareOptions{ToleranceMode: userInput}
//	diff, err := testhelper.FormatComparisonResultE(expected, actual, opts)
//	if err != nil {
//	    // Handle invalid options from user input
//	    return fmt.Errorf("invalid comparison options: %w", err)
//	}
//	if diff != "" {
//	    fmt.Printf("Values differ: %s\n", diff)
//	}
func FormatComparisonResultE(expected, actual interface{}, opts CompareOptions) (string, error) {
	if err := ValidateOptions(opts); err != nil {
		return "", err
	}
	_, diff := compareValues(expected, actual, opts, "")
	return diff, nil
}
