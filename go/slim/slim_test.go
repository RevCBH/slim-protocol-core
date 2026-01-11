package slim

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEncodePrimitives(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"null", nil, "!null"},
		{"true", true, "?T"},
		{"false", false, "?F"},
		{"number", float64(42), "#42"},
		{"string", "hello", "hello"},
		{"empty string", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Encode(tt.input, DefaultEncodeOptions())
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEncodeObject(t *testing.T) {
	data := map[string]interface{}{
		"name": "Mario",
		"age":  float64(30),
	}
	result, err := Encode(data, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	// Check that it contains both fields (order may vary)
	if !contains(result, "name:Mario") || !contains(result, "age:#30") {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestEncodeArray(t *testing.T) {
	data := []interface{}{float64(1), float64(2), float64(3)}
	result, err := Encode(data, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	expected := "@#[1,2,3]"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeTable(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Mario"},
		map[string]interface{}{"id": float64(2), "name": "Luigi"},
	}
	result, err := Encode(data, DefaultEncodeOptions())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if !contains(result, "|2|") {
		t.Errorf("Expected table format, got: %s", result)
	}
}

func TestDecodePrimitives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"null", "!null", nil},
		{"true", "?T", true},
		{"false", "?F", false},
		{"number", "#42", float64(42)},
		{"string", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decode(tt.input, DefaultDecodeOptions())
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecodeObject(t *testing.T) {
	slim := "{name:Mario,age:#30}"
	result, err := Decode(slim, DefaultDecodeOptions())
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}
	if obj["name"] != "Mario" || obj["age"] != float64(30) {
		t.Errorf("Unexpected result: %v", obj)
	}
}

func TestDecodeArray(t *testing.T) {
	slim := "@#[1,2,3]"
	result, err := Decode(slim, DefaultDecodeOptions())
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected array, got %T", result)
	}
	expected := []interface{}{float64(1), float64(2), float64(3)}
	if !reflect.DeepEqual(arr, expected) {
		t.Errorf("Expected %v, got %v", expected, arr)
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []interface{}{
		nil,
		true,
		false,
		float64(42),
		"hello",
		[]interface{}{float64(1), float64(2), float64(3)},
		map[string]interface{}{"name": "Mario", "age": float64(30)},
		[]interface{}{
			map[string]interface{}{"id": float64(1), "name": "Mario"},
			map[string]interface{}{"id": float64(2), "name": "Luigi"},
		},
	}

	for i, original := range tests {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			// Encode
			slim, err := Encode(original, DefaultEncodeOptions())
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			// Decode
			decoded, err := Decode(slim, DefaultDecodeOptions())
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Compare
			if !deepEqual(original, decoded) {
				t.Errorf("Round-trip failed:\n  Original: %v\n  Decoded:  %v\n  SLIM: %s",
					original, decoded, slim)
			}
		})
	}
}

func TestInferSchema(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Mario", "active": true},
		map[string]interface{}{"id": float64(2), "name": "Luigi", "active": false},
	}
	schema := InferSchema(data)

	// Check that schema contains expected type markers
	if !contains(schema, "id#") || !contains(schema, "name$") || !contains(schema, "active?") {
		t.Errorf("Unexpected schema: %s", schema)
	}
}

func TestValidateSchema(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Mario"},
	}

	// Valid schema
	result := ValidateSchema(data, "id#,name$")
	if !result.Valid {
		t.Errorf("Validation should pass")
	}

	// Missing required field
	result = ValidateSchema(data, "id#,name$,email$")
	if result.Valid {
		t.Errorf("Validation should fail for missing field")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func deepEqual(a, b interface{}) bool {
	// Handle nil
	if a == nil || b == nil {
		return a == b
	}

	// Use JSON marshaling for deep comparison to normalize types
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)

	var aNorm, bNorm interface{}
	json.Unmarshal(aJSON, &aNorm)
	json.Unmarshal(bJSON, &bNorm)

	return reflect.DeepEqual(aNorm, bNorm)
}
