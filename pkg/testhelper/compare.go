package testhelper

import (
	"fmt"
	"math"
	"strings"
)

// CompareOptions configures output comparison behavior.
type CompareOptions struct {
	// FloatTolerance specifies the tolerance for float comparisons.
	// For "relative" and "absolute" modes, this is the tolerance threshold.
	// For "ulp" mode, this value is truncated to an integer representing the
	// maximum allowed ULP (Units in Last Place) difference. For example,
	// a tolerance of 1.9 allows 1 ULP difference, not 2.
	FloatTolerance float64

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
	ToleranceMode string

	// NaNEqualsNaN treats NaN values as equal when true.
	NaNEqualsNaN bool

	// ArrayOrder specifies array comparison order.
	// Use the ArrayOrder* constants: ArrayOrderStrict (default) or
	// ArrayOrderUnordered. Empty string ("") is treated as ArrayOrderStrict.
	ArrayOrder string
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
	ArrayOrderUnordered = "unordered"
)

// String returns a human-readable representation of CompareOptions for debugging.
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

// ValidateOptions validates that CompareOptions has valid enum values.
// Returns nil if valid, or an error describing the invalid field.
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
	switch opts.ArrayOrder {
	case "", ArrayOrderStrict, ArrayOrderUnordered:
		// valid (empty defaults to strict)
	default:
		return fmt.Errorf("invalid ArrayOrder: %q (must be \"strict\" or \"unordered\")", opts.ArrayOrder)
	}
	return nil
}

// CompareOutput compares expected and actual outputs.
// Returns true if they match according to the options.
// Panics if opts contains invalid enum values (use ValidateOptions to check beforehand).
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
func CompareOutput(expected, actual interface{}, opts CompareOptions) bool {
	ok, _ := Compare(expected, actual, opts)
	return ok
}

// Compare compares expected and actual outputs with detailed diff.
// Returns true if they match, and a diff string if they don't.
// Panics if opts contains invalid enum values (use ValidateOptions to check beforehand).
// This fail-fast behavior ensures invalid options are caught immediately rather than
// silently producing incorrect comparison results.
//
// Special string values in expected trigger float comparisons:
//   - "NaN" matches actual NaN (per NaNEqualsNaN option)
//   - "Infinity" or "+Infinity" matches actual +Inf
//   - "-Infinity" matches actual -Inf
func Compare(expected, actual interface{}, opts CompareOptions) (bool, string) {
	if err := ValidateOptions(opts); err != nil {
		panic("testhelper.Compare: " + err.Error())
	}
	return compareValues(expected, actual, opts, "")
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
		if expStr == "NaN" || expStr == "Infinity" || expStr == "+Infinity" || expStr == "-Infinity" {
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
	case "absolute":
		return math.Abs(expected-actual) <= opts.FloatTolerance
	case "ulp":
		return ulpDiff(expected, actual) <= int64(opts.FloatTolerance)
	case "relative":
		fallthrough
	default:
		if expected == 0 {
			return math.Abs(actual) <= opts.FloatTolerance
		}
		return math.Abs((expected-actual)/expected) <= opts.FloatTolerance
	}
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
	case "NaN":
		if math.IsNaN(a) {
			if opts.NaNEqualsNaN {
				return true, ""
			}
			return false, fmt.Sprintf("%s: NaN mismatch (NaNEqualsNaN is false)", pathStr(path))
		}
		return false, fmt.Sprintf("%s: expected NaN, got %v", pathStr(path), a)
	case "Infinity", "+Infinity":
		if math.IsInf(a, 1) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected +Infinity, got %v", pathStr(path), a)
	case "-Infinity":
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

	if opts.ArrayOrder == "unordered" {
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

// Deprecated: FormatDiff is deprecated since v0.1.0 and will be removed in v1.0.
// Use FormatComparisonResult instead. FormatDiff returns "values are equal"
// when values match, which is semantically inconsistent. FormatComparisonResult
// has clearer semantics: empty string on match, descriptive diff on mismatch.
func FormatDiff(expected, actual interface{}, opts CompareOptions) string {
	_, diff := Compare(expected, actual, opts)
	if diff == "" {
		return "values are equal"
	}
	return diff
}

// FormatComparisonResult compares expected and actual values, returning a
// human-readable description of any differences.
//
// Returns:
//   - "" (empty string) if values match
//   - A descriptive diff string if values differ
//
// This function has clearer semantics than FormatDiff: an empty result means
// "no differences" rather than returning affirmative text on match.
func FormatComparisonResult(expected, actual interface{}, opts CompareOptions) string {
	_, diff := Compare(expected, actual, opts)
	return diff
}
