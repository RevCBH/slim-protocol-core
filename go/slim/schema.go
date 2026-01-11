package slim

import (
	"fmt"
	"strings"
)

// InferSchema infers a SLIM schema from an array of objects
func InferSchema(data []interface{}) string {
	if len(data) == 0 {
		return ""
	}

	// Collect all unique keys
	keysSet := make(map[string]bool)
	var keysOrder []string

	for _, row := range data {
		if obj, ok := row.(map[string]interface{}); ok {
			for key := range obj {
				if !keysSet[key] {
					keysSet[key] = true
					keysOrder = append(keysOrder, key)
				}
			}
		}
	}

	// Determine types
	var colTypes []struct {
		key  string
		mark string
	}

	for _, key := range keysOrder {
		var values []interface{}
		for _, row := range data {
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
		for _, row := range data {
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

		colTypes = append(colTypes, struct {
			key  string
			mark string
		}{key, typeMarker})
	}

	var parts []string
	for _, ct := range colTypes {
		parts = append(parts, ct.key+ct.mark)
	}
	return strings.Join(parts, ",")
}

// ParseSchema parses a schema string into column definitions
func ParseSchema(schema string) []ColumnDef {
	if schema == "" {
		return []ColumnDef{}
	}

	var cols []ColumnDef
	for _, col := range strings.Split(schema, ",") {
		// Find where type markers start
		nameEnd := len(col)
		colType := ColumnTypeString
		nullable := false

		for i, ch := range col {
			if ch == '#' || ch == '?' || ch == '@' || ch == '~' || ch == '$' || ch == '!' {
				if nameEnd == len(col) {
					nameEnd = i
				}
				if ch == '!' {
					nullable = true
				} else if colType == ColumnTypeString {
					colType = ColumnType(string(ch))
				}
			}
		}

		name := col[:nameEnd]
		cols = append(cols, ColumnDef{
			Name:     name,
			Type:     colType,
			Nullable: nullable,
		})
	}

	return cols
}

// ValidateSchema validates data against a schema
func ValidateSchema(data interface{}, schema string) ValidationResult {
	columns := ParseSchema(schema)
	var errors []ValidationError

	// Handle array of objects
	if arr, ok := data.([]interface{}); ok {
		for index, row := range arr {
			if obj, ok := row.(map[string]interface{}); ok {
				validateRow(obj, columns, fmt.Sprintf("[%d]", index), &errors)
			} else {
				errors = append(errors, ValidationError{
					Path:     fmt.Sprintf("[%d]", index),
					Message:  "Expected object",
					Expected: "object",
					Actual:   typeName(row),
				})
			}
		}
	} else if obj, ok := data.(map[string]interface{}); ok {
		validateRow(obj, columns, "", &errors)
	} else {
		errors = append(errors, ValidationError{
			Path:     "",
			Message:  "Expected object or array of objects",
			Expected: "object | object[]",
			Actual:   typeName(data),
		})
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func validateRow(row map[string]interface{}, columns []ColumnDef, pathPrefix string, errors *[]ValidationError) {
	for _, col := range columns {
		path := col.Name
		if pathPrefix != "" {
			path = pathPrefix + "." + col.Name
		}

		value, exists := row[col.Name]

		// Check required
		if (!exists || value == nil) && !col.Nullable {
			*errors = append(*errors, ValidationError{
				Path:     path,
				Message:  "Missing required field",
				Expected: string(col.Type),
			})
			continue
		}

		// Skip null/undefined for nullable columns
		if !exists || value == nil {
			continue
		}

		// Type check
		if !isCorrectType(value, col.Type) {
			*errors = append(*errors, ValidationError{
				Path:     path,
				Message:  "Type mismatch",
				Expected: string(col.Type),
				Actual:   typeName(value),
			})
		}
	}
}

func isCorrectType(value interface{}, colType ColumnType) bool {
	switch colType {
	case ColumnTypeNumber:
		_, ok := value.(float64)
		return ok
	case ColumnTypeBoolean:
		_, ok := value.(bool)
		return ok
	case ColumnTypeString:
		_, ok := value.(string)
		return ok
	case ColumnTypeArray:
		_, ok := value.([]interface{})
		return ok
	case ColumnTypeObject:
		_, ok := value.(map[string]interface{})
		return ok
	case ColumnTypeNullable:
		return true
	default:
		return true
	}
}

func typeName(value interface{}) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}
