package slim

import (
	"math"
	"testing"
)

func TestDeepEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{"nil == nil", nil, nil, true},
		{"nil != bool", nil, true, false},
		{"true == true", true, true, true},
		{"true != false", true, false, false},
		{"number == number", float64(42), float64(42), true},
		{"number != number", float64(42), float64(43), false},
		{"string == string", "hello", "hello", true},
		{"string != string", "hello", "world", false},
		{"NaN == NaN", math.NaN(), math.NaN(), true},
		{"empty array == empty array", []interface{}{}, []interface{}{}, true},
		{"array == array", []interface{}{float64(1), float64(2)}, []interface{}{float64(1), float64(2)}, true},
		{"array != array (different length)", []interface{}{float64(1)}, []interface{}{float64(1), float64(2)}, false},
		{"array != array (different values)", []interface{}{float64(1)}, []interface{}{float64(2)}, false},
		{"object == object", map[string]interface{}{"a": float64(1)}, map[string]interface{}{"a": float64(1)}, true},
		{"object != object (different keys)", map[string]interface{}{"a": float64(1)}, map[string]interface{}{"b": float64(1)}, false},
		{"object != object (different values)", map[string]interface{}{"a": float64(1)}, map[string]interface{}{"a": float64(2)}, false},
		{"nested objects",
			map[string]interface{}{"user": map[string]interface{}{"name": "Mario"}},
			map[string]interface{}{"user": map[string]interface{}{"name": "Mario"}},
			true},
		{"type mismatch", float64(1), "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("DeepEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestDeepEqualSymmetry(t *testing.T) {
	values := []interface{}{
		nil,
		true,
		float64(42),
		"hello",
		[]interface{}{float64(1), float64(2)},
		map[string]interface{}{"a": float64(1)},
	}

	for i, a := range values {
		for j, b := range values {
			ab := DeepEqual(a, b)
			ba := DeepEqual(b, a)
			if ab != ba {
				t.Errorf("DeepEqual not symmetric for values[%d], values[%d]: %v vs %v", i, j, ab, ba)
			}
		}
	}
}

func TestClone(t *testing.T) {
	t.Run("clones nil", func(t *testing.T) {
		result := Clone(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("clones primitives", func(t *testing.T) {
		if Clone(true) != true {
			t.Error("bool clone failed")
		}
		if Clone(float64(42)) != float64(42) {
			t.Error("number clone failed")
		}
		if Clone("hello") != "hello" {
			t.Error("string clone failed")
		}
	})

	t.Run("clones arrays", func(t *testing.T) {
		original := []interface{}{float64(1), float64(2), float64(3)}
		cloned := Clone(original).([]interface{})

		if !DeepEqual(original, cloned) {
			t.Error("cloned array not equal to original")
		}

		// Modify original, cloned should be unaffected
		original[0] = float64(99)
		if cloned[0] == float64(99) {
			t.Error("clone is not independent")
		}
	})

	t.Run("clones objects", func(t *testing.T) {
		original := map[string]interface{}{
			"name": "Mario",
			"age":  float64(30),
		}
		cloned := Clone(original).(map[string]interface{})

		if !DeepEqual(original, cloned) {
			t.Error("cloned object not equal to original")
		}

		// Modify original, cloned should be unaffected
		original["name"] = "Luigi"
		if cloned["name"] == "Luigi" {
			t.Error("clone is not independent")
		}
	})

	t.Run("deep clones nested structures", func(t *testing.T) {
		original := map[string]interface{}{
			"user": map[string]interface{}{
				"name":   "Mario",
				"scores": []interface{}{float64(10), float64(20)},
			},
		}
		cloned := Clone(original).(map[string]interface{})

		// Modify nested value in original
		user := original["user"].(map[string]interface{})
		user["name"] = "Luigi"

		// Cloned should be unaffected
		clonedUser := cloned["user"].(map[string]interface{})
		if clonedUser["name"] == "Luigi" {
			t.Error("deep clone is not independent")
		}
	})
}

func TestGetPath(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":   "Mario",
			"age":    float64(30),
			"scores": []interface{}{float64(10), float64(20), float64(30)},
		},
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			map[string]interface{}{"id": float64(2)},
		},
	}

	tests := []struct {
		path     string
		expected interface{}
	}{
		{"user.name", "Mario"},
		{"user.age", float64(30)},
		{"user.scores.0", float64(10)},
		{"user.scores.1", float64(20)},
		{"user.scores.2", float64(30)},
		{"items.0.id", float64(1)},
		{"items.1.id", float64(2)},
		{"nonexistent", nil},
		{"user.nonexistent", nil},
		{"user.scores.99", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetPath(data, tt.path)
			if !DeepEqual(result, tt.expected) {
				t.Errorf("GetPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestSetPath(t *testing.T) {
	t.Run("sets simple path", func(t *testing.T) {
		data := map[string]interface{}{"name": "Mario"}
		result := SetPath(data, "name", "Luigi").(map[string]interface{})

		if result["name"] != "Luigi" {
			t.Errorf("expected Luigi, got %v", result["name"])
		}

		// Original should be unchanged
		if data["name"] != "Mario" {
			t.Error("original was mutated")
		}
	})

	t.Run("sets nested path", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{"name": "Mario"},
		}
		result := SetPath(data, "user.name", "Luigi").(map[string]interface{})

		user := result["user"].(map[string]interface{})
		if user["name"] != "Luigi" {
			t.Errorf("expected Luigi, got %v", user["name"])
		}
	})

	t.Run("creates intermediate objects", func(t *testing.T) {
		data := map[string]interface{}{}
		result := SetPath(data, "a.b.c", "value").(map[string]interface{})

		a := result["a"].(map[string]interface{})
		b := a["b"].(map[string]interface{})
		if b["c"] != "value" {
			t.Errorf("expected value, got %v", b["c"])
		}
	})

	t.Run("handles nil input", func(t *testing.T) {
		result := SetPath(nil, "key", "value").(map[string]interface{})
		if result["key"] != "value" {
			t.Errorf("expected value, got %v", result["key"])
		}
	})
}

func TestGetSetPathRoundtrip(t *testing.T) {
	paths := []string{"a", "a.b", "a.b.c", "user.name", "data.items"}
	values := []interface{}{
		float64(42),
		"hello",
		true,
		[]interface{}{float64(1), float64(2)},
		map[string]interface{}{"nested": "value"},
	}

	for _, path := range paths {
		for _, value := range values {
			obj := map[string]interface{}{}
			updated := SetPath(obj, path, value)
			retrieved := GetPath(updated, path)

			if !DeepEqual(retrieved, value) {
				t.Errorf("roundtrip failed for path=%q, value=%v: got %v", path, value, retrieved)
			}
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"a", 1},
		{"test", 1},
		{"hello", 2},
		{"12345678", 2},
		{"this is a longer string", 6},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateSavings(t *testing.T) {
	t.Run("calculates positive savings", func(t *testing.T) {
		slim := "|2|id#|\n1\n2"
		json := `[{"id":1},{"id":2}]`

		savings := CalculateSavings(slim, json)
		if savings <= 0 {
			t.Errorf("expected positive savings, got %d", savings)
		}
	})

	t.Run("handles equal length", func(t *testing.T) {
		savings := CalculateSavings("test", "test")
		if savings != 0 {
			t.Errorf("expected 0 savings for equal strings, got %d", savings)
		}
	})

	t.Run("handles empty json", func(t *testing.T) {
		savings := CalculateSavings("slim", "")
		if savings != 0 {
			t.Errorf("expected 0 for empty json, got %d", savings)
		}
	})

	t.Run("handles longer slim", func(t *testing.T) {
		// When SLIM is longer than JSON, savings should be negative
		savings := CalculateSavings("this is a very long slim string", "short")
		if savings >= 0 {
			t.Errorf("expected negative savings when slim is longer, got %d", savings)
		}
	})
}

func TestUtilityFunctionsCrossImpl(t *testing.T) {
	// These tests verify behavior that should be identical across all implementations

	t.Run("deep_equal handles complex nested structures", func(t *testing.T) {
		a := map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"id": float64(1), "name": "Mario"},
				map[string]interface{}{"id": float64(2), "name": "Luigi"},
			},
			"meta": map[string]interface{}{
				"count":  float64(2),
				"active": true,
			},
		}
		b := Clone(a)

		if !DeepEqual(a, b) {
			t.Error("cloned complex structure should be equal")
		}
	})

	t.Run("path operations on nested arrays", func(t *testing.T) {
		data := map[string]interface{}{
			"matrix": []interface{}{
				[]interface{}{float64(1), float64(2)},
				[]interface{}{float64(3), float64(4)},
			},
		}

		// Getting nested array elements
		val := GetPath(data, "matrix.0")
		arr, ok := val.([]interface{})
		if !ok {
			t.Fatalf("expected array, got %T", val)
		}
		if len(arr) != 2 {
			t.Errorf("expected 2 elements, got %d", len(arr))
		}
	})
}
