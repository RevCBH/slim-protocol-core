package slim

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Encode encodes a value to SLIM format
func Encode(data interface{}, options EncodeOptions) (string, error) {
	return encodeValue(data, 0, options)
}

func encodeValue(data interface{}, depth int, opts EncodeOptions) (string, error) {
	// Depth check
	if depth > opts.MaxDepth {
		return SlimDeep, nil
	}

	switch v := data.(type) {
	case nil:
		return SlimNull, nil
	case bool:
		if v {
			return SlimTrue, nil
		}
		return SlimFalse, nil
	case float64:
		return encodeNumber(v), nil
	case int:
		return encodeNumber(float64(v)), nil
	case int64:
		return encodeNumber(float64(v)), nil
	case string:
		return encodeString(v), nil
	case []interface{}:
		return encodeArray(v, depth, opts)
	case map[string]interface{}:
		return encodeObject(v, depth, opts)
	default:
		// Try to convert to JSON-compatible types
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("unsupported type: %T", data)
		}
		var jsonData interface{}
		json.Unmarshal(jsonBytes, &jsonData)
		return encodeValue(jsonData, depth, opts)
	}
}

func encodeNumber(n float64) string {
	if math.IsNaN(n) {
		return SlimNaN
	}
	if math.IsInf(n, 1) {
		return SlimInf
	}
	if math.IsInf(n, -1) {
		return SlimNegInf
	}
	// Format number without unnecessary decimals
	if n == float64(int64(n)) {
		return fmt.Sprintf("#%d", int64(n))
	}
	return fmt.Sprintf("#%g", n)
}

func encodeString(s string) string {
	if s == "" {
		return `""`
	}

	// Check if quoting is needed
	needsQuoting := false
	for _, ch := range s {
		if strings.ContainsRune(`,;\n|{}[]"#?!*@`, ch) {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting && s != strings.TrimSpace(s) {
		needsQuoting = true
	}

	if needsQuoting {
		// Escape quotes and newlines
		escaped := strings.ReplaceAll(s, `"`, `""`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		return `"` + escaped + `"`
	}
	return s
}

func encodeKey(s string) string {
	needsQuoting := strings.ContainsAny(s, `,:{}[]`)
	if needsQuoting {
		escaped := strings.ReplaceAll(s, `"`, `""`)
		return `"` + escaped + `"`
	}
	return s
}

func encodeArray(arr []interface{}, depth int, opts EncodeOptions) (string, error) {
	if len(arr) == 0 {
		return "@[]", nil
	}

	// Check if it's a 2D numeric matrix
	if isMatrix(arr) {
		return encodeMatrix(arr)
	}

	// Check if all numbers
	if allNumbers(arr) {
		var nums []string
		for _, v := range arr {
			if f, ok := v.(float64); ok {
				nums = append(nums, fmt.Sprintf("%g", f))
			}
		}
		return "@#[" + strings.Join(nums, ",") + "]", nil
	}

	// Check if all simple strings
	if allSimpleStrings(arr) {
		var strs []string
		for _, v := range arr {
			if s, ok := v.(string); ok {
				strs = append(strs, s)
			}
		}
		return "@[" + strings.Join(strs, ",") + "]", nil
	}

	// Check if array of objects -> use table format
	if len(arr) >= opts.TableThreshold && allObjects(arr) {
		return encodeTable(arr, depth, opts)
	}

	// Generic array with semicolon separators
	var items []string
	for _, v := range arr {
		encoded, err := encodeValue(v, depth+1, opts)
		if err != nil {
			return "", err
		}
		items = append(items, encoded)
	}
	return "@[" + strings.Join(items, ";") + "]", nil
}

func isMatrix(arr []interface{}) bool {
	for _, row := range arr {
		if rowArr, ok := row.([]interface{}); ok {
			for _, v := range rowArr {
				if _, ok := v.(float64); !ok {
					return false
				}
			}
		} else {
			return false
		}
	}
	return true
}

func encodeMatrix(matrix []interface{}) (string, error) {
	var rows []string
	for _, row := range matrix {
		if rowArr, ok := row.([]interface{}); ok {
			var nums []string
			for _, v := range rowArr {
				if f, ok := v.(float64); ok {
					nums = append(nums, fmt.Sprintf("%g", f))
				}
			}
			rows = append(rows, strings.Join(nums, ","))
		}
	}
	return "*[" + strings.Join(rows, ";") + "]", nil
}

func allNumbers(arr []interface{}) bool {
	for _, v := range arr {
		if _, ok := v.(float64); !ok {
			return false
		}
	}
	return true
}

func allSimpleStrings(arr []interface{}) bool {
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return false
		}
		if strings.ContainsAny(s, `,;[]`) {
			return false
		}
	}
	return true
}

func allObjects(arr []interface{}) bool {
	for _, v := range arr {
		if _, ok := v.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

func encodeObject(obj map[string]interface{}, depth int, opts EncodeOptions) (string, error) {
	if len(obj) == 0 {
		return "{}", nil
	}

	// Get sorted keys for deterministic output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		value := obj[key]
		safeKey := encodeKey(key)

		// Check if value is array of objects -> inline table
		if arr, ok := value.([]interface{}); ok {
			if len(arr) >= opts.TableThreshold && allObjects(arr) {
				table, err := encodeTable(arr, depth+1, opts)
				if err != nil {
					return "", err
				}
				parts = append(parts, safeKey+":"+table)
				continue
			}
		}

		encodedValue, err := encodeValue(value, depth+1, opts)
		if err != nil {
			return "", err
		}
		parts = append(parts, safeKey+":"+encodedValue)
	}

	return "{" + strings.Join(parts, ",") + "}", nil
}

func encodeTable(rows []interface{}, depth int, opts EncodeOptions) (string, error) {
	// Collect all unique keys
	keysSet := make(map[string]bool)
	var keysOrder []string

	for _, row := range rows {
		if obj, ok := row.(map[string]interface{}); ok {
			for key := range obj {
				if !keysSet[key] {
					keysSet[key] = true
					keysOrder = append(keysOrder, key)
				}
			}
		}
	}

	// Determine column types
	type ColType struct {
		key        string
		typeMarker string
	}
	var colTypes []ColType

	for _, key := range keysOrder {
		var values []interface{}
		for _, row := range rows {
			if obj, ok := row.(map[string]interface{}); ok {
				if val, exists := obj[key]; exists && val != nil {
					values = append(values, val)
				}
			}
		}

		typeMarker := "$"
		if len(values) == 0 {
			typeMarker = "!"
		} else if allBooleans(values) {
			typeMarker = "?"
		} else if allNumberValues(values) {
			typeMarker = "#"
		} else if allArrayValues(values) {
			typeMarker = "@"
		} else if allObjectValues(values) {
			typeMarker = "~"
		}

		// Mark nullable
		hasNull := false
		for _, row := range rows {
			if obj, ok := row.(map[string]interface{}); ok {
				val, exists := obj[key]
				if !exists || val == nil {
					hasNull = true
					break
				}
			}
		}
		if hasNull {
			typeMarker += "!"
		}

		colTypes = append(colTypes, ColType{key, typeMarker})
	}

	// Build schema
	var schemaParts []string
	for _, ct := range colTypes {
		schemaParts = append(schemaParts, ct.key+ct.typeMarker)
	}

	// Build data rows
	var dataRows []string
	for _, row := range rows {
		obj, _ := row.(map[string]interface{})
		var cells []string

		for _, ct := range colTypes {
			value, exists := obj[ct.key]
			if !exists || value == nil {
				cells = append(cells, "")
				continue
			}

			if strings.HasPrefix(ct.typeMarker, "?") {
				if b, ok := value.(bool); ok {
					if b {
						cells = append(cells, "T")
					} else {
						cells = append(cells, "F")
					}
				} else {
					cells = append(cells, "")
				}
			} else if strings.HasPrefix(ct.typeMarker, "#") {
				if f, ok := value.(float64); ok {
					cells = append(cells, fmt.Sprintf("%g", f))
				} else {
					cells = append(cells, "")
				}
			} else if strings.HasPrefix(ct.typeMarker, "@") {
				encoded := encodeInlineArray(value)
				cells = append(cells, encoded)
			} else if strings.HasPrefix(ct.typeMarker, "~") {
				encoded, _ := encodeValue(value, depth+1, opts)
				cells = append(cells, encoded)
			} else {
				// String
				if s, ok := value.(string); ok {
					if s == "" {
						cells = append(cells, `""`)
					} else if strings.ContainsAny(s, ",\n|\"") {
						escaped := strings.ReplaceAll(s, `"`, `""`)
						escaped = strings.ReplaceAll(escaped, "\n", `\n`)
						cells = append(cells, `"`+escaped+`"`)
					} else {
						cells = append(cells, s)
					}
				} else {
					cells = append(cells, "")
				}
			}
		}
		dataRows = append(dataRows, strings.Join(cells, ","))
	}

	return fmt.Sprintf("|%d|%s|\n%s",
		len(rows),
		strings.Join(schemaParts, ","),
		strings.Join(dataRows, "\n")), nil
}

func encodeInlineArray(arr interface{}) string {
	if arrVal, ok := arr.([]interface{}); ok {
		if len(arrVal) == 0 {
			return "[]"
		}

		// Numbers
		if allNumbers(arrVal) {
			var nums []string
			for _, v := range arrVal {
				if f, ok := v.(float64); ok {
					nums = append(nums, fmt.Sprintf("%g", f))
				}
			}
			return strings.Join(nums, "+")
		}

		// Strings
		var strs []string
		for _, v := range arrVal {
			s := fmt.Sprintf("%v", v)
			if strings.ContainsAny(s, "+,") {
				strs = append(strs, `"`+strings.ReplaceAll(s, `"`, `""`)+`"`)
			} else {
				strs = append(strs, s)
			}
		}
		return strings.Join(strs, "+")
	}
	return ""
}

func allBooleans(values []interface{}) bool {
	for _, v := range values {
		if _, ok := v.(bool); !ok {
			return false
		}
	}
	return true
}

func allNumberValues(values []interface{}) bool {
	for _, v := range values {
		if _, ok := v.(float64); !ok {
			return false
		}
	}
	return true
}

func allArrayValues(values []interface{}) bool {
	for _, v := range values {
		if _, ok := v.([]interface{}); !ok {
			return false
		}
	}
	return true
}

func allObjectValues(values []interface{}) bool {
	for _, v := range values {
		if _, ok := v.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}
