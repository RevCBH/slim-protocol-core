package slim

import (
	"math"
	"strings"
)

// DeepEqual performs a deep equality check on two values.
// Handles NaN correctly (NaN == NaN returns true).
func DeepEqual(a, b interface{}) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle booleans
	if aBool, ok := a.(bool); ok {
		if bBool, ok := b.(bool); ok {
			return aBool == bBool
		}
		return false
	}

	// Handle numbers (float64)
	if aNum, ok := a.(float64); ok {
		if bNum, ok := b.(float64); ok {
			// Handle NaN
			if math.IsNaN(aNum) && math.IsNaN(bNum) {
				return true
			}
			return aNum == bNum
		}
		return false
	}

	// Handle strings
	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			return aStr == bStr
		}
		return false
	}

	// Handle arrays
	if aArr, ok := a.([]interface{}); ok {
		if bArr, ok := b.([]interface{}); ok {
			if len(aArr) != len(bArr) {
				return false
			}
			for i := range aArr {
				if !DeepEqual(aArr[i], bArr[i]) {
					return false
				}
			}
			return true
		}
		return false
	}

	// Handle objects
	if aObj, ok := a.(map[string]interface{}); ok {
		if bObj, ok := b.(map[string]interface{}); ok {
			if len(aObj) != len(bObj) {
				return false
			}
			for k, v := range aObj {
				if bv, exists := bObj[k]; !exists || !DeepEqual(v, bv) {
					return false
				}
			}
			return true
		}
		return false
	}

	return false
}

// Clone creates a deep copy of a value.
func Clone(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Primitives are immutable, return as-is
	switch v := value.(type) {
	case bool, float64, string:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}

	// Clone array
	if arr, ok := value.([]interface{}); ok {
		result := make([]interface{}, len(arr))
		for i, item := range arr {
			result[i] = Clone(item)
		}
		return result
	}

	// Clone object
	if obj, ok := value.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for k, v := range obj {
			result[k] = Clone(v)
		}
		return result
	}

	return value
}

// GetPath retrieves a value at a dot-separated path.
// Returns nil if the path doesn't exist.
func GetPath(value interface{}, path string) interface{} {
	if value == nil || path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	current := value

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// Handle array access
		if arr, ok := current.([]interface{}); ok {
			index := 0
			for i, c := range part {
				if c < '0' || c > '9' {
					return nil
				}
				index = index*10 + int(c-'0')
				if i > 0 && part[0] == '0' {
					// Leading zeros not allowed
					return nil
				}
			}
			if index >= len(arr) {
				return nil
			}
			current = arr[index]
			continue
		}

		// Handle object access
		if obj, ok := current.(map[string]interface{}); ok {
			val, exists := obj[part]
			if !exists {
				return nil
			}
			current = val
			continue
		}

		return nil
	}

	return current
}

// SetPath sets a value at a dot-separated path, returning a new value.
// Creates intermediate objects as needed.
func SetPath(value interface{}, path string, newValue interface{}) interface{} {
	if path == "" {
		return newValue
	}

	// Clone the original to avoid mutations
	if value == nil {
		value = make(map[string]interface{})
	}
	result := Clone(value)

	parts := strings.Split(path, ".")
	setPathRecursive(result, parts, newValue)

	return result
}

func setPathRecursive(value interface{}, parts []string, newValue interface{}) {
	if len(parts) == 0 {
		return
	}

	part := parts[0]
	remaining := parts[1:]

	obj, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	if len(remaining) == 0 {
		obj[part] = newValue
		return
	}

	// Create intermediate object if needed
	if _, exists := obj[part]; !exists {
		obj[part] = make(map[string]interface{})
	} else if _, isObj := obj[part].(map[string]interface{}); !isObj {
		obj[part] = make(map[string]interface{})
	}

	setPathRecursive(obj[part], remaining, newValue)
}

// EstimateTokens estimates the number of tokens in a string.
// Uses a simple heuristic (length / 4) rounded up.
func EstimateTokens(s string) int {
	return int(math.Ceil(float64(len(s)) / 4.0))
}

// CalculateSavings calculates the percentage of token savings.
// Returns (jsonTokens - slimTokens) / jsonTokens * 100.
func CalculateSavings(slim, json string) int {
	slimTokens := EstimateTokens(slim)
	jsonTokens := EstimateTokens(json)

	if jsonTokens == 0 {
		return 0
	}

	return int(math.Round(float64(jsonTokens-slimTokens) / float64(jsonTokens) * 100))
}
