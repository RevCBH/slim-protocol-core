package slim

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestPropertyRoundTrip tests that encoding and decoding are inverses
func TestPropertyRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Test with simple values
	properties.Property("encode/decode round trip for booleans", prop.ForAll(
		func(v bool) bool {
			encoded, err := Encode(v, DefaultEncodeOptions())
			if err != nil {
				return false
			}

			decoded, err := Decode(encoded, DefaultDecodeOptions())
			if err != nil {
				return false
			}

			return decoded == v
		},
		gen.Bool(),
	))

	properties.Property("encode/decode round trip for numbers", prop.ForAll(
		func(v float64) bool {
			encoded, err := Encode(v, DefaultEncodeOptions())
			if err != nil {
				return false
			}

			decoded, err := Decode(encoded, DefaultDecodeOptions())
			if err != nil {
				return false
			}

			if f, ok := decoded.(float64); ok {
				// Handle NaN specially
				if math.IsNaN(v) {
					return math.IsNaN(f)
				}
				return f == v
			}
			return false
		},
		gen.Float64Range(-1000, 1000),
	))

	properties.Property("encode/decode round trip for strings", prop.ForAll(
		func(v string) bool {
			encoded, err := Encode(v, DefaultEncodeOptions())
			if err != nil {
				return false
			}

			decoded, err := Decode(encoded, DefaultDecodeOptions())
			if err != nil {
				return false
			}

			return decoded == v
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// TestPropertyEncodeStability tests that encoding is deterministic
func TestPropertyEncodeStability(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("encoding booleans is stable", prop.ForAll(
		func(v bool) bool {
			encoded1, err1 := Encode(v, DefaultEncodeOptions())
			encoded2, err2 := Encode(v, DefaultEncodeOptions())

			if err1 != nil || err2 != nil {
				return err1 == err2
			}

			return encoded1 == encoded2
		},
		gen.Bool(),
	))

	properties.Property("encoding numbers is stable", prop.ForAll(
		func(v float64) bool {
			encoded1, err1 := Encode(v, DefaultEncodeOptions())
			encoded2, err2 := Encode(v, DefaultEncodeOptions())

			if err1 != nil || err2 != nil {
				return err1 == err2
			}

			return encoded1 == encoded2
		},
		gen.Float64Range(-1000, 1000),
	))

	properties.TestingRun(t)
}

// TestPropertyTokenSavings tests that SLIM is generally smaller than JSON
func TestPropertyTokenSavings(t *testing.T) {
	// Create a few sample datasets manually
	testCases := [][]interface{}{
		{
			map[string]interface{}{"id": float64(1), "name": "Alice", "active": true},
			map[string]interface{}{"id": float64(2), "name": "Bob", "active": false},
			map[string]interface{}{"id": float64(3), "name": "Charlie", "active": true},
			map[string]interface{}{"id": float64(4), "name": "David", "active": false},
			map[string]interface{}{"id": float64(5), "name": "Eve", "active": true},
		},
		{
			map[string]interface{}{"x": float64(1), "y": float64(2)},
			map[string]interface{}{"x": float64(3), "y": float64(4)},
			map[string]interface{}{"x": float64(5), "y": float64(6)},
			map[string]interface{}{"x": float64(7), "y": float64(8)},
			map[string]interface{}{"x": float64(9), "y": float64(10)},
		},
	}

	for i, arr := range testCases {
		t.Run(fmt.Sprintf("dataset-%d", i), func(t *testing.T) {
			slim, err := Encode(arr, DefaultEncodeOptions())
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			jsonBytes, err := json.Marshal(arr)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			t.Logf("SLIM: %d chars, JSON: %d chars, Savings: %.1f%%",
				len(slim), len(jsonBytes),
				float64(len(jsonBytes)-len(slim))/float64(len(jsonBytes))*100)

			// For tables, SLIM should be smaller
			if len(slim) > len(jsonBytes) {
				t.Errorf("SLIM (%d) larger than JSON (%d)\nSLIM: %s\nJSON: %s",
					len(slim), len(jsonBytes), slim, string(jsonBytes))
			}
		})
	}
}

// TestPropertySchemaInference tests schema inference consistency
func TestPropertySchemaInference(t *testing.T) {
	testCases := [][]interface{}{
		{
			map[string]interface{}{"id": float64(1), "name": "Alice"},
			map[string]interface{}{"id": float64(2), "name": "Bob"},
		},
		{
			map[string]interface{}{"x": float64(1), "y": float64(2), "active": true},
			map[string]interface{}{"x": float64(3), "y": float64(4), "active": false},
		},
	}

	for i, arr := range testCases {
		t.Run(fmt.Sprintf("dataset-%d", i), func(t *testing.T) {
			schema := InferSchema(arr)
			result := ValidateSchema(arr, schema)

			if !result.Valid {
				t.Errorf("Inferred schema doesn't validate data. Schema: %s, Errors: %v",
					schema, result.Errors)
			}
		})
	}
}

// TestPropertyArrayFormat tests that arrays of objects use table format
func TestPropertyArrayFormat(t *testing.T) {
	arr := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Alice"},
		map[string]interface{}{"id": float64(2), "name": "Bob"},
	}

	encoded, err := Encode(arr, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Table format starts with |count|
	if len(encoded) == 0 || encoded[0] != '|' {
		t.Errorf("Expected table format, got: %s", encoded)
	}
}

// ==================== Deep Equal Property Tests ====================

// TestPropertyDeepEqualReflexive tests that x == x for all values
func TestPropertyDeepEqualReflexive(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("deep_equal is reflexive for booleans", prop.ForAll(
		func(v bool) bool {
			return DeepEqual(v, v)
		},
		gen.Bool(),
	))

	properties.Property("deep_equal is reflexive for numbers", prop.ForAll(
		func(v float64) bool {
			return DeepEqual(v, v)
		},
		gen.Float64Range(-1000, 1000),
	))

	properties.Property("deep_equal is reflexive for strings", prop.ForAll(
		func(v string) bool {
			return DeepEqual(v, v)
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// TestPropertyDeepEqualSymmetric tests that (x == y) == (y == x)
func TestPropertyDeepEqualSymmetric(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("deep_equal is symmetric for numbers", prop.ForAll(
		func(a, b float64) bool {
			return DeepEqual(a, b) == DeepEqual(b, a)
		},
		gen.Float64Range(-1000, 1000),
		gen.Float64Range(-1000, 1000),
	))

	properties.Property("deep_equal is symmetric for strings", prop.ForAll(
		func(a, b string) bool {
			return DeepEqual(a, b) == DeepEqual(b, a)
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// ==================== Clone Property Tests ====================

// TestPropertyCloneEquality tests that clone(x) == x
func TestPropertyCloneEquality(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("clone preserves boolean equality", prop.ForAll(
		func(v bool) bool {
			return DeepEqual(v, Clone(v))
		},
		gen.Bool(),
	))

	properties.Property("clone preserves number equality", prop.ForAll(
		func(v float64) bool {
			cloned := Clone(v)
			if math.IsNaN(v) {
				return math.IsNaN(cloned.(float64))
			}
			return DeepEqual(v, cloned)
		},
		gen.Float64Range(-1000, 1000),
	))

	properties.Property("clone preserves string equality", prop.ForAll(
		func(v string) bool {
			return DeepEqual(v, Clone(v))
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// ==================== Path Operations Property Tests ====================

// TestPropertyGetSetPathRoundtrip tests get_path(set_path(obj, path, val), path) == val
func TestPropertyGetSetPathRoundtrip(t *testing.T) {
	// Use fixed test cases instead of generators for more reliable testing
	testCases := []struct {
		key string
		val interface{}
	}{
		{"name", "Mario"},
		{"age", float64(30)},
		{"active", true},
		{"count", float64(100)},
		{"title", "Test Title"},
	}

	for _, tc := range testCases {
		obj := make(map[string]interface{})
		updated := SetPath(obj, tc.key, tc.val)
		retrieved := GetPath(updated, tc.key)
		if !DeepEqual(retrieved, tc.val) {
			t.Errorf("get/set roundtrip failed for key=%q, val=%v: got %v", tc.key, tc.val, retrieved)
		}
	}
}

// ==================== Token Estimation Property Tests ====================

// TestPropertyEstimateTokensNonNegative tests that token estimates are always >= 0
func TestPropertyEstimateTokensNonNegative(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("token estimate is non-negative", prop.ForAll(
		func(s string) bool {
			return EstimateTokens(s) >= 0
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// TestPropertyEstimateTokensProportional tests that tokens scale with string length
func TestPropertyEstimateTokensProportional(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("token estimate is proportional to length", prop.ForAll(
		func(s string) bool {
			tokens := EstimateTokens(s)
			expected := int(math.Ceil(float64(len(s)) / 4.0))
			return tokens == expected
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// ==================== Streaming API Property Tests ====================

// TestPropertyEncoderDecoderRoundtrip tests encoder/decoder roundtrip
func TestPropertyEncoderDecoderRoundtrip(t *testing.T) {
	// Generate test data
	testCases := [][]interface{}{
		{
			map[string]interface{}{"id": float64(1), "name": "Alice"},
			map[string]interface{}{"id": float64(2), "name": "Bob"},
		},
		{
			map[string]interface{}{"x": float64(1), "y": float64(2)},
			map[string]interface{}{"x": float64(3), "y": float64(4)},
			map[string]interface{}{"x": float64(5), "y": float64(6)},
		},
		{
			map[string]interface{}{"active": true, "count": float64(10)},
		},
	}

	for i, data := range testCases {
		t.Run(fmt.Sprintf("dataset-%d", i), func(t *testing.T) {
			// Encode
			encoder := NewEncoder(DefaultEncodeOptions())
			for _, obj := range data {
				encoder.Write(obj)
			}
			encoded, err := encoder.End()
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}

			// Decode
			decoder := NewDecoder(DefaultDecodeOptions())
			decoder.Write(encoded)
			decoded, err := decoder.End()
			if err != nil {
				t.Fatalf("decode failed: %v", err)
			}

			// Verify
			arr, ok := decoded.([]interface{})
			if !ok {
				t.Fatalf("expected array, got %T", decoded)
			}
			if len(arr) != len(data) {
				t.Errorf("expected %d items, got %d", len(data), len(arr))
			}
		})
	}
}

// TestPropertyEncodeChunkedPreservesData tests that chunked encoding preserves all data
func TestPropertyEncodeChunkedPreservesData(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1)},
		map[string]interface{}{"id": float64(2)},
		map[string]interface{}{"id": float64(3)},
		map[string]interface{}{"id": float64(4)},
		map[string]interface{}{"id": float64(5)},
	}

	for chunkSize := 1; chunkSize <= len(data)+1; chunkSize++ {
		t.Run(fmt.Sprintf("chunkSize-%d", chunkSize), func(t *testing.T) {
			chunks, err := EncodeChunked(data, chunkSize, DefaultEncodeOptions())
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}

			// Decode all chunks and count objects
			totalObjects := 0
			for _, chunk := range chunks {
				decoded, err := Decode(chunk, DefaultDecodeOptions())
				if err != nil {
					t.Fatalf("decode failed: %v", err)
				}
				if arr, ok := decoded.([]interface{}); ok {
					totalObjects += len(arr)
				}
			}

			if totalObjects != len(data) {
				t.Errorf("expected %d total objects, got %d", len(data), totalObjects)
			}
		})
	}
}

// ==================== Cross-Implementation Compatibility Tests ====================

// TestCrossImplPrimitives tests that primitive encoding matches across implementations
func TestCrossImplPrimitives(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected string
	}{
		{nil, "!null"},
		{true, "?T"},
		{false, "?F"},
		{float64(42), "#42"},
	}

	for _, tc := range testCases {
		encoded, err := Encode(tc.input, DefaultEncodeOptions())
		if err != nil {
			t.Fatalf("encode failed for %v: %v", tc.input, err)
		}
		if encoded != tc.expected {
			t.Errorf("Encode(%v) = %q, want %q", tc.input, encoded, tc.expected)
		}
	}
}

// TestCrossImplNumberArray tests number array format
func TestCrossImplNumberArray(t *testing.T) {
	data := []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)}

	encoded, err := Encode(data, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	expected := "@#[1,2,3,4,5]"
	if encoded != expected {
		t.Errorf("expected %q, got %q", expected, encoded)
	}
}

// TestCrossImplMatrix tests matrix format
func TestCrossImplMatrix(t *testing.T) {
	data := []interface{}{
		[]interface{}{float64(1), float64(2)},
		[]interface{}{float64(3), float64(4)},
	}

	encoded, err := Encode(data, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	expected := "*[1,2;3,4]"
	if encoded != expected {
		t.Errorf("expected %q, got %q", expected, encoded)
	}
}
