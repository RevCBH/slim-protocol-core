//! SLIM Schema Utilities
//!
//! Functions for inferring, parsing, and validating SLIM schemas.

use crate::types::*;
use serde_json::Value;
use std::collections::HashSet;

/// Infers a SLIM schema from an array of objects.
///
/// # Arguments
///
/// * `data` - Array of objects to infer schema from
///
/// # Examples
///
/// ```
/// use slim_protocol::infer_schema;
/// use serde_json::json;
///
/// let data = vec![
///     json!({"id": 1, "name": "Mario", "active": true}),
///     json!({"id": 2, "name": "Luigi", "active": false}),
/// ];
/// let schema = infer_schema(&data);
/// ```
pub fn infer_schema(data: &[Value]) -> String {
    if data.is_empty() {
        return String::new();
    }

    // Collect all unique keys
    let mut keys_set = HashSet::new();
    let mut keys_order = Vec::new();

    for row in data {
        if let Some(obj) = row.as_object() {
            for key in obj.keys() {
                if keys_set.insert(key.clone()) {
                    keys_order.push(key.clone());
                }
            }
        }
    }

    // Determine types
    let mut col_types: Vec<(String, String)> = Vec::new();

    for key in &keys_order {
        let values: Vec<&Value> = data
            .iter()
            .filter_map(|r| r.as_object())
            .filter_map(|obj| obj.get(key))
            .filter(|v| !v.is_null())
            .collect();

        let type_marker = if values.is_empty() {
            "!".to_string()
        } else if values.iter().all(|v| v.is_boolean()) {
            "?".to_string()
        } else if values.iter().all(|v| v.is_number()) {
            "#".to_string()
        } else if values.iter().all(|v| v.is_array()) {
            "@".to_string()
        } else if values.iter().all(|v| v.is_object()) {
            "~".to_string()
        } else {
            "$".to_string()
        };

        // Mark nullable
        let mut final_type = type_marker;
        if data
            .iter()
            .filter_map(|r| r.as_object())
            .any(|obj| obj.get(key).map(|v| v.is_null()).unwrap_or(true))
        {
            final_type.push('!');
        }

        col_types.push((key.clone(), final_type));
    }

    col_types
        .iter()
        .map(|(k, t)| format!("{}{}", k, t))
        .collect::<Vec<_>>()
        .join(",")
}

/// Parses a schema string into column definitions.
///
/// # Arguments
///
/// * `schema` - Schema string (e.g., "id#,name$,active?")
///
/// # Examples
///
/// ```
/// use slim_protocol::parse_schema;
///
/// let cols = parse_schema("id#,name$,active?!");
/// ```
pub fn parse_schema(schema: &str) -> Vec<ColumnDef> {
    if schema.is_empty() {
        return vec![];
    }

    schema
        .split(',')
        .map(|col| {
            // Find where the type markers start
            let mut name_end = col.len();
            let mut col_type = ColumnType::String;
            let mut nullable = false;

            for (i, ch) in col.char_indices() {
                if let Some(ct) = ColumnType::from_char(ch) {
                    if name_end == col.len() {
                        name_end = i;
                    }
                    if ct == ColumnType::Nullable {
                        nullable = true;
                    } else if col_type == ColumnType::String {
                        col_type = ct;
                    }
                }
            }

            let name = col[..name_end].to_string();

            ColumnDef {
                name,
                col_type,
                nullable,
            }
        })
        .collect()
}

/// Validates data against a schema.
///
/// # Arguments
///
/// * `data` - Data to validate (array of objects or single object)
/// * `schema` - Schema string to validate against
///
/// # Examples
///
/// ```
/// use slim_protocol::validate_schema;
/// use serde_json::json;
///
/// let data = vec![json!({"id": 1, "name": "Mario"})];
/// let result = validate_schema(&json!(data), "id#,name$,email$");
/// ```
pub fn validate_schema(data: &Value, schema: &str) -> ValidationResult {
    let columns = parse_schema(schema);
    let mut errors = Vec::new();

    // Handle array of objects
    if let Some(arr) = data.as_array() {
        for (index, row) in arr.iter().enumerate() {
            if let Some(obj) = row.as_object() {
                validate_row(obj, &columns, &format!("[{}]", index), &mut errors);
            } else {
                errors.push(ValidationError {
                    path: format!("[{}]", index),
                    message: "Expected object".to_string(),
                    expected: Some("object".to_string()),
                    actual: Some(type_name(row)),
                });
            }
        }
    }
    // Handle single object
    else if let Some(obj) = data.as_object() {
        validate_row(obj, &columns, "", &mut errors);
    } else {
        errors.push(ValidationError {
            path: String::new(),
            message: "Expected object or array of objects".to_string(),
            expected: Some("object | object[]".to_string()),
            actual: Some(type_name(data)),
        });
    }

    ValidationResult {
        valid: errors.is_empty(),
        errors,
    }
}

/// Validate a single row against column definitions
fn validate_row(
    row: &serde_json::Map<String, Value>,
    columns: &[ColumnDef],
    path_prefix: &str,
    errors: &mut Vec<ValidationError>,
) {
    for col in columns {
        let path = if path_prefix.is_empty() {
            col.name.clone()
        } else {
            format!("{}.{}", path_prefix, col.name)
        };

        let value = row.get(&col.name);

        // Check required
        if (value.is_none() || value == Some(&Value::Null)) && !col.nullable {
            errors.push(ValidationError {
                path,
                message: "Missing required field".to_string(),
                expected: Some(col.col_type.to_string_repr().to_string()),
                actual: None,
            });
            continue;
        }

        // Skip null/undefined for nullable columns
        if value.is_none() || value == Some(&Value::Null) {
            continue;
        }

        let v = value.unwrap();

        // Type check
        if !is_correct_type(v, col.col_type) {
            errors.push(ValidationError {
                path,
                message: "Type mismatch".to_string(),
                expected: Some(col.col_type.to_string_repr().to_string()),
                actual: Some(type_name(v)),
            });
        }
    }
}

/// Check if a value matches the expected column type
fn is_correct_type(value: &Value, col_type: ColumnType) -> bool {
    match col_type {
        ColumnType::Number => value.is_number(),
        ColumnType::Boolean => value.is_boolean(),
        ColumnType::String => value.is_string(),
        ColumnType::Array => value.is_array(),
        ColumnType::Object => value.is_object(),
        ColumnType::Nullable => true,
    }
}

/// Get type name of a value
fn type_name(value: &Value) -> String {
    match value {
        Value::Null => "null".to_string(),
        Value::Bool(_) => "boolean".to_string(),
        Value::Number(_) => "number".to_string(),
        Value::String(_) => "string".to_string(),
        Value::Array(_) => "array".to_string(),
        Value::Object(_) => "object".to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_infer_schema() {
        let data = vec![
            json!({"id": 1, "name": "Mario", "active": true}),
            json!({"id": 2, "name": "Luigi", "active": false}),
        ];
        let schema = infer_schema(&data);
        assert!(schema.contains("id#"));
        assert!(schema.contains("name$"));
        assert!(schema.contains("active?"));
    }

    #[test]
    fn test_parse_schema() {
        let cols = parse_schema("id#,name$,active?!");
        assert_eq!(cols.len(), 3);
        assert_eq!(cols[0].name, "id");
        assert_eq!(cols[0].col_type, ColumnType::Number);
        assert_eq!(cols[2].nullable, true);
    }

    #[test]
    fn test_validate_schema() {
        let data = vec![json!({"id": 1, "name": "Mario"})];
        let result = validate_schema(&json!(data), "id#,name$");
        assert!(result.valid);

        let result = validate_schema(&json!(data), "id#,name$,email$");
        assert!(!result.valid);
    }
}
