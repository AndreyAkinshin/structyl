// Package tests provides the internal test comparison implementation for Structyl.
//
// Note: This package intentionally duplicates some comparison logic from pkg/testhelper.
// The duplication exists because:
//   - This package uses ComparisonConfig (maps to JSON config structure)
//   - pkg/testhelper uses CompareOptions (stable public API)
//   - Error messages differ: this uses "root" path prefix, testhelper uses "$" (JSON Path)
//   - ULP calculation delegates to testhelper.ULPDiff to avoid duplicating IEEE 754 logic
package tests

import (
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/pkg/testhelper"
)

// Compare compares expected and actual values using the given configuration.
func Compare(expected, actual interface{}, cfg ComparisonConfig) (bool, string) {
	return compareValues(expected, actual, cfg, "")
}

func compareValues(expected, actual interface{}, cfg ComparisonConfig, path string) (bool, string) {
	// Handle nil cases
	if expected == nil && actual == nil {
		return true, ""
	}
	if expected == nil || actual == nil {
		return false, fmt.Sprintf("%s: expected %v, got %v", pathStr(path), expected, actual)
	}

	// After nil guards above, both values are non-nil.
	// Handle special float values represented as strings
	if str, ok := expected.(string); ok {
		if isSpecialFloat(str) {
			return compareSpecialFloat(str, actual, cfg, path)
		}
	}

	switch exp := expected.(type) {
	case float64:
		return compareFloats(exp, actual, cfg, path)
	case int:
		return compareFloats(float64(exp), actual, cfg, path)
	case string:
		if act, ok := actual.(string); ok {
			if exp == act {
				return true, ""
			}
		}
		return false, fmt.Sprintf("%s: expected %q, got %v", pathStr(path), exp, actual)
	case bool:
		if act, ok := actual.(bool); ok && exp == act {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected %v, got %v", pathStr(path), exp, actual)
	case map[string]interface{}:
		return compareMaps(exp, actual, cfg, path)
	case []interface{}:
		return compareArrays(exp, actual, cfg, path)
	default:
		// Fallback for types not explicitly handled above (e.g., nested primitives,
		// custom types). reflect.DeepEqual is slower but provides correct semantics.
		if reflect.DeepEqual(expected, actual) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected %v (%T), got %v (%T)", pathStr(path), expected, expected, actual, actual)
	}
}

func compareFloats(expected float64, actual interface{}, cfg ComparisonConfig, path string) (bool, string) {
	var actFloat float64
	switch v := actual.(type) {
	case float64:
		actFloat = v
	case int:
		actFloat = float64(v)
	default:
		return false, fmt.Sprintf("%s: expected float, got %T", pathStr(path), actual)
	}

	// Handle special values
	if math.IsNaN(expected) && math.IsNaN(actFloat) {
		if cfg.NaNEqualsNaN {
			return true, ""
		}
		return false, fmt.Sprintf("%s: NaN != NaN (set nan_equals_nan to allow)", pathStr(path))
	}
	if math.IsInf(expected, 1) && math.IsInf(actFloat, 1) {
		return true, ""
	}
	if math.IsInf(expected, -1) && math.IsInf(actFloat, -1) {
		return true, ""
	}

	// Compare with tolerance
	var withinTolerance bool
	switch config.ToleranceMode(cfg.ToleranceMode) {
	case config.ToleranceModeAbsolute:
		withinTolerance = math.Abs(expected-actFloat) <= cfg.FloatTolerance
	case config.ToleranceModeULP:
		// ULP comparison using IEEE 754 bit representation
		withinTolerance = testhelper.ULPDiff(expected, actFloat) <= int64(cfg.FloatTolerance)
	default:
		// Relative tolerance: explicit, empty string, or unknown mode (for backward compatibility)
		withinTolerance = isWithinRelativeTolerance(expected, actFloat, cfg.FloatTolerance)
	}

	if withinTolerance {
		return true, ""
	}
	return false, fmt.Sprintf("%s: expected %v, got %v (tolerance: %v %s)", pathStr(path), expected, actFloat, cfg.FloatTolerance, cfg.ToleranceMode)
}

func compareMaps(expected map[string]interface{}, actual interface{}, cfg ComparisonConfig, path string) (bool, string) {
	actMap, ok := actual.(map[string]interface{})
	if !ok {
		return false, fmt.Sprintf("%s: expected object, got %T", pathStr(path), actual)
	}

	// Check for missing/extra keys
	for key := range expected {
		if _, ok := actMap[key]; !ok {
			return false, fmt.Sprintf("%s: missing key %q", pathStr(path), key)
		}
	}
	for key := range actMap {
		if _, ok := expected[key]; !ok {
			return false, fmt.Sprintf("%s: unexpected key %q", pathStr(path), key)
		}
	}

	// Compare values
	for key, expVal := range expected {
		actVal := actMap[key]
		keyPath := path + "." + key
		if path == "" {
			keyPath = key
		}
		if ok, diff := compareValues(expVal, actVal, cfg, keyPath); !ok {
			return false, diff
		}
	}

	return true, ""
}

func compareArrays(expected []interface{}, actual interface{}, cfg ComparisonConfig, path string) (bool, string) {
	actArr, ok := actual.([]interface{})
	if !ok {
		return false, fmt.Sprintf("%s: expected array, got %T", pathStr(path), actual)
	}

	if len(expected) != len(actArr) {
		return false, fmt.Sprintf("%s: expected %d elements, got %d", pathStr(path), len(expected), len(actArr))
	}

	if config.ArrayOrder(cfg.ArrayOrder) == config.ArrayOrderUnordered {
		return compareArraysUnordered(expected, actArr, cfg, path)
	}

	// Strict order comparison
	for i := range expected {
		indexPath := fmt.Sprintf("%s[%d]", path, i)
		if ok, diff := compareValues(expected[i], actArr[i], cfg, indexPath); !ok {
			return false, diff
		}
	}

	return true, ""
}

func compareArraysUnordered(expected, actual []interface{}, cfg ComparisonConfig, path string) (bool, string) {
	// Simple unordered comparison: try to match each expected element
	matched := make([]bool, len(actual))

	for i, exp := range expected {
		found := false
		for j, act := range actual {
			if matched[j] {
				continue
			}
			if ok, _ := compareValues(exp, act, cfg, ""); ok {
				matched[j] = true
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Sprintf("%s[%d]: no matching element found for %v", path, i, exp)
		}
	}

	return true, ""
}

func compareSpecialFloat(expected string, actual interface{}, cfg ComparisonConfig, path string) (bool, string) {
	actFloat, ok := toFloat(actual)
	if !ok {
		return false, fmt.Sprintf("%s: expected float, got %T", pathStr(path), actual)
	}

	switch expected {
	case "NaN":
		if math.IsNaN(actFloat) {
			if cfg.NaNEqualsNaN {
				return true, ""
			}
			return false, fmt.Sprintf("%s: NaN != NaN (set nan_equals_nan to allow)", pathStr(path))
		}
		return false, fmt.Sprintf("%s: expected NaN, got %v", pathStr(path), actFloat)
	case "Infinity", "+Infinity":
		if math.IsInf(actFloat, 1) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected +Infinity, got %v", pathStr(path), actFloat)
	case "-Infinity":
		if math.IsInf(actFloat, -1) {
			return true, ""
		}
		return false, fmt.Sprintf("%s: expected -Infinity, got %v", pathStr(path), actFloat)
	}

	return false, fmt.Sprintf("%s: unexpected special float %q", pathStr(path), expected)
}

func isSpecialFloat(s string) bool {
	return s == "NaN" || s == "Infinity" || s == "+Infinity" || s == "-Infinity"
}

func toFloat(v interface{}) (float64, bool) {
	switch f := v.(type) {
	case float64:
		return f, true
	case int:
		return float64(f), true
	default:
		return 0, false
	}
}

// isWithinRelativeTolerance checks if actual is within relative tolerance of expected.
// For expected == 0, uses absolute comparison to avoid division by zero.
func isWithinRelativeTolerance(expected, actual, tolerance float64) bool {
	if expected == 0 {
		return math.Abs(actual) <= tolerance
	}
	return math.Abs((expected-actual)/expected) <= tolerance
}

// pathStr formats a path for error messages.
// Returns "root" for empty path to indicate the top-level value.
// Note: The pkg/testhelper version uses "$" (JSON Path convention) instead of "root"
// because it's a public API targeting external consumers familiar with JSON Path.
func pathStr(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

// SortedKeys returns sorted keys of a map for deterministic iteration.
func SortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
