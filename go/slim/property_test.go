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
