package testhelper

import (
	"math"
	"testing"
)

// BenchmarkEqual benchmarks the Equal function with various input types.
// Run: go test -bench=BenchmarkEqual -benchmem ./pkg/testhelper

func BenchmarkEqual_SimpleFloat(b *testing.B) {
	opts := DefaultOptions()
	expected := 3.14159265358979
	actual := 3.14159265358979

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_FloatWithTolerance(b *testing.B) {
	opts := DefaultOptions()
	expected := 3.14159265358979
	actual := 3.14159265358980 // slightly different

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_ULPTolerance(b *testing.B) {
	opts := CompareOptions{
		FloatTolerance: 10,
		ToleranceMode:  ToleranceModeULP,
	}
	expected := 1.0
	actual := 1.0000000000000002 // 1 ULP away

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_SimpleString(b *testing.B) {
	opts := DefaultOptions()
	expected := "hello world"
	actual := "hello world"

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_SmallArray(b *testing.B) {
	opts := DefaultOptions()
	expected := []interface{}{1.0, 2.0, 3.0, 4.0, 5.0}
	actual := []interface{}{1.0, 2.0, 3.0, 4.0, 5.0}

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_LargeArray(b *testing.B) {
	opts := DefaultOptions()
	expected := make([]interface{}, 1000)
	actual := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		expected[i] = float64(i)
		actual[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_UnorderedArray(b *testing.B) {
	opts := CompareOptions{
		FloatTolerance: 1e-9,
		ToleranceMode:  ToleranceModeRelative,
		ArrayOrder:     ArrayOrderUnordered,
	}
	expected := []interface{}{5.0, 4.0, 3.0, 2.0, 1.0}
	actual := []interface{}{1.0, 2.0, 3.0, 4.0, 5.0}

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_SmallObject(b *testing.B) {
	opts := DefaultOptions()
	expected := map[string]interface{}{
		"name":  "test",
		"value": 42.0,
		"flag":  true,
	}
	actual := map[string]interface{}{
		"name":  "test",
		"value": 42.0,
		"flag":  true,
	}

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_DeepNested(b *testing.B) {
	opts := DefaultOptions()
	expected := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"value": 42.0,
					},
				},
			},
		},
	}
	actual := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"value": 42.0,
					},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkCompare_WithDiff(b *testing.B) {
	opts := DefaultOptions()
	expected := map[string]interface{}{
		"name":  "test",
		"value": 42.0,
	}
	actual := map[string]interface{}{
		"name":  "test",
		"value": 43.0, // different
	}

	b.ResetTimer()
	for b.Loop() {
		Compare(expected, actual, opts)
	}
}

func BenchmarkValidateOptions(b *testing.B) {
	opts := CompareOptions{
		FloatTolerance: 1e-9,
		ToleranceMode:  ToleranceModeRelative,
		NaNEqualsNaN:   true,
		ArrayOrder:     ArrayOrderStrict,
	}

	b.ResetTimer()
	for b.Loop() {
		ValidateOptions(opts)
	}
}

func BenchmarkULPDiff(b *testing.B) {
	a := 1.0
	c := 1.0000000000000002

	b.ResetTimer()
	for b.Loop() {
		ULPDiff(a, c)
	}
}

// BenchmarkSpecialFloats benchmarks comparison of special float values.

func BenchmarkEqual_NaN(b *testing.B) {
	opts := DefaultOptions()
	expected := SpecialFloatNaN
	actual := math.NaN()

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}

func BenchmarkEqual_Infinity(b *testing.B) {
	opts := DefaultOptions()
	expected := SpecialFloatInfinity
	actual := math.Inf(1)

	b.ResetTimer()
	for b.Loop() {
		Equal(expected, actual, opts)
	}
}
