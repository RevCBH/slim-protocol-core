//! SLIM Encoder
//!
//! Converts JSON values to SLIM format.

use crate::error::{Result, SlimError};
use crate::types::*;
use serde_json::{Map, Value};
use std::collections::HashSet;

/// Encodes a JSON value to SLIM format.
///
/// # Arguments
///
/// * `data` - The value to encode
/// * `options` - Encoding options
///
/// # Examples
///
/// ```
/// use slim_protocol::{encode, EncodeOptions};
/// use serde_json::json;
///
/// let data = json!([1, 2, 3]);
/// let slim = encode(&data, Default::default()).unwrap();
/// assert_eq!(slim, "@#[1,2,3]");
/// ```
pub fn encode(data: &Value, options: EncodeOptions) -> Result<String> {
    encode_value(data, 0, &options)
}

/// Internal recursive encoder
fn encode_value(data: &Value, depth: usize, opts: &EncodeOptions) -> Result<String> {
    // Depth check
    if depth > opts.max_depth {
        return Ok(SLIM_DEEP.to_string());
    }

    match data {
        Value::Null => Ok(SLIM_NULL.to_string()),
        Value::Bool(b) => Ok(if *b { SLIM_TRUE.to_string() } else { SLIM_FALSE.to_string() }),
        Value::Number(n) => encode_number(n),
        Value::String(s) => Ok(encode_string(s)),
        Value::Array(arr) => encode_array(arr, depth, opts),
        Value::Object(obj) => encode_object(obj, depth, opts),
    }
}

/// Encode a number
fn encode_number(num: &serde_json::Number) -> Result<String> {
    if let Some(n) = num.as_f64() {
        if n.is_nan() {
            return Ok(SLIM_NAN.to_string());
        }
        if n.is_infinite() {
            return Ok(if n > 0.0 {
                SLIM_INF.to_string()
            } else {
                SLIM_NEG_INF.to_string()
            });
        }
        Ok(format!("#{}", n))
    } else if let Some(n) = num.as_i64() {
        Ok(format!("#{}", n))
    } else if let Some(n) = num.as_u64() {
        Ok(format!("#{}", n))
    } else {
        Ok(format!("#{}", num))
    }
}

/// Encode a string, quoting if necessary
fn encode_string(s: &str) -> String {
    if s.is_empty() {
        return "\"\"".to_string();
    }

    // Check if quoting is needed
    let needs_quoting = s.chars().any(|c| matches!(c, ',' | ';' | '\n' | '|' | '{' | '}' | '[' | ']' | '"' | '#' | '?' | '!' | '*' | '@'))
        || s != s.trim();

    if needs_quoting {
        // Escape quotes and newlines
        let escaped = s.replace('"', "\"\"").replace('\n', "\\n");
        format!("\"{}\"", escaped)
    } else {
        s.to_string()
    }
}

/// Encode a string for use in object keys
fn encode_key(s: &str) -> String {
    let needs_quoting = s.chars().any(|c| matches!(c, ',' | ':' | '{' | '}' | '[' | ']'));

    if needs_quoting {
        let escaped = s.replace('"', "\"\"");
        format!("\"{}\"", escaped)
    } else {
        s.to_string()
    }
}

/// Encode an array
fn encode_array(arr: &[Value], depth: usize, opts: &EncodeOptions) -> Result<String> {
    // Empty array
    if arr.is_empty() {
        return Ok("@[]".to_string());
    }

    // Check if it's a 2D numeric matrix
    if is_matrix(arr) {
        return encode_matrix(arr);
    }

    // Check if all numbers
    if arr.iter().all(|v| v.is_number()) {
        let nums: Vec<String> = arr
            .iter()
            .filter_map(|v| v.as_f64().map(|n| n.to_string()))
            .collect();
        return Ok(format!("@#[{}]", nums.join(",")));
    }

    // Check if all simple strings (no commas/semicolons)
    if arr.iter().all(|v| {
        v.as_str()
            .map(|s| !s.chars().any(|c| matches!(c, ',' | ';' | '[' | ']')))
            .unwrap_or(false)
    }) {
        let strs: Vec<String> = arr
            .iter()
            .filter_map(|v| v.as_str().map(|s| s.to_string()))
            .collect();
        return Ok(format!("@[{}]", strs.join(",")));
    }

    // Check if array of objects -> use table format
    if arr.len() >= opts.table_threshold
        && arr.iter().all(|v| v.is_object())
    {
        return encode_table(arr, depth, opts);
    }

    // Generic array with semicolon separators
    let items: Result<Vec<String>> = arr
        .iter()
        .map(|v| encode_value(v, depth + 1, opts))
        .collect();
    Ok(format!("@[{}]", items?.join(";")))
}

/// Check if array is a 2D numeric matrix
fn is_matrix(arr: &[Value]) -> bool {
    arr.iter().all(|row| {
        row.as_array()
            .map(|r| r.iter().all(|v| v.is_number()))
            .unwrap_or(false)
    })
}

/// Encode a 2D numeric matrix
fn encode_matrix(matrix: &[Value]) -> Result<String> {
    let rows: Vec<String> = matrix
        .iter()
        .filter_map(|row| {
            row.as_array().map(|r| {
                r.iter()
                    .filter_map(|v| v.as_f64().map(|n| n.to_string()))
                    .collect::<Vec<_>>()
                    .join(",")
            })
        })
        .collect();
    Ok(format!("*[{}]", rows.join(";")))
}

/// Encode an object
fn encode_object(obj: &Map<String, Value>, depth: usize, opts: &EncodeOptions) -> Result<String> {
    if obj.is_empty() {
        return Ok("{}".to_string());
    }

    let parts: Result<Vec<String>> = obj
        .iter()
        .map(|(key, value)| {
            let safe_key = encode_key(key);

            // Check if value is array of objects -> inline table
            if let Value::Array(arr) = value {
                if arr.len() >= opts.table_threshold
                    && arr.iter().all(|v| v.is_object())
                {
                    let table = encode_table(arr, depth + 1, opts)?;
                    return Ok(format!("{}:{}", safe_key, table));
                }
            }

            let encoded_value = encode_value(value, depth + 1, opts)?;
            Ok(format!("{}:{}", safe_key, encoded_value))
        })
        .collect();

    Ok(format!("{{{}}}", parts?.join(",")))
}

/// Encode an array of objects as a SLIM table
fn encode_table(rows: &[Value], depth: usize, opts: &EncodeOptions) -> Result<String> {
    // Collect all unique keys
    let mut keys_set = HashSet::new();
    let mut keys_order = Vec::new();

    for row in rows {
        if let Some(obj) = row.as_object() {
            for key in obj.keys() {
                if keys_set.insert(key.clone()) {
                    keys_order.push(key.clone());
                }
            }
        }
    }

    // Determine column types
    let mut col_types: Vec<(String, String)> = Vec::new();

    for key in &keys_order {
        let values: Vec<&Value> = rows
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

        // Mark nullable if any null/undefined
        let mut final_type = type_marker;
        if rows
            .iter()
            .filter_map(|r| r.as_object())
            .any(|obj| obj.get(key).map(|v| v.is_null()).unwrap_or(true))
        {
            final_type.push('!');
        }

        col_types.push((key.clone(), final_type));
    }

    // Build schema
    let schema: Vec<String> = col_types
        .iter()
        .map(|(k, t)| format!("{}{}", k, t))
        .collect();

    // Build data rows
    let data_rows: Result<Vec<String>> = rows
        .iter()
        .map(|row| {
            let obj = row.as_object().ok_or_else(|| {
                SlimError::InvalidInput("Expected object in table".to_string())
            })?;

            let cells: Result<Vec<String>> = col_types
                .iter()
                .map(|(key, type_marker)| {
                    let value = obj.get(key);

                    if value.is_none() || value == Some(&Value::Null) {
                        return Ok(String::new());
                    }

                    let v = value.unwrap();

                    if type_marker.starts_with('?') {
                        return Ok(if v.as_bool().unwrap_or(false) {
                            "T".to_string()
                        } else {
                            "F".to_string()
                        });
                    }

                    if type_marker.starts_with('#') {
                        return Ok(v.as_f64().map(|n| n.to_string()).unwrap_or_default());
                    }

                    if type_marker.starts_with('@') {
                        return encode_inline_array(v);
                    }

                    if type_marker.starts_with('~') {
                        return encode_value(v, depth + 1, opts);
                    }

                    // String
                    if let Some(s) = v.as_str() {
                        if s.is_empty() {
                            return Ok("\"\"".to_string());
                        }
                        if s.chars().any(|c| matches!(c, ',' | '\n' | '|' | '"')) {
                            let escaped = s.replace('"', "\"\"").replace('\n', "\\n");
                            return Ok(format!("\"{}\"", escaped));
                        }
                        return Ok(s.to_string());
                    }

                    Ok(v.to_string())
                })
                .collect();

            Ok(cells?.join(","))
        })
        .collect();

    Ok(format!(
        "|{}|{}|\n{}",
        rows.len(),
        schema.join(","),
        data_rows?.join("\n")
    ))
}

/// Encode an array for use in a table cell (+ separator)
fn encode_inline_array(arr: &Value) -> Result<String> {
    if let Some(arr) = arr.as_array() {
        if arr.is_empty() {
            return Ok("[]".to_string());
        }

        // Numbers
        if arr.iter().all(|v| v.is_number()) {
            let nums: Vec<String> = arr
                .iter()
                .filter_map(|v| v.as_f64().map(|n| n.to_string()))
                .collect();
            return Ok(nums.join("+"));
        }

        // Strings
        let strs: Vec<String> = arr
            .iter()
            .map(|v| {
                let s = v.as_str().unwrap_or("");
                if s.chars().any(|c| matches!(c, '+' | ',')) {
                    format!("\"{}\"", s.replace('"', "\"\""))
                } else {
                    s.to_string()
                }
            })
            .collect();
        Ok(strs.join("+"))
    } else {
        Ok(String::new())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_encode_primitives() {
        assert_eq!(encode(&Value::Null, Default::default()).unwrap(), "!null");
        assert_eq!(encode(&json!(true), Default::default()).unwrap(), "?T");
        assert_eq!(encode(&json!(false), Default::default()).unwrap(), "?F");
        assert_eq!(encode(&json!(42), Default::default()).unwrap(), "#42");
        assert_eq!(encode(&json!("hello"), Default::default()).unwrap(), "hello");
    }

    #[test]
    fn test_encode_string_quoting() {
        assert_eq!(encode(&json!(""), Default::default()).unwrap(), "\"\"");
        assert_eq!(encode(&json!("hello, world"), Default::default()).unwrap(), "\"hello, world\"");
    }

    #[test]
    fn test_encode_object() {
        let data = json!({"name": "Mario", "age": 30});
        let result = encode(&data, Default::default()).unwrap();
        // Order is not guaranteed in JSON objects, so check both possibilities
        assert!(result == "{name:Mario,age:#30}" || result == "{age:#30,name:Mario}");
    }

    #[test]
    fn test_encode_array_numbers() {
        let data = json!([1, 2, 3]);
        let result = encode(&data, Default::default()).unwrap();
        assert_eq!(result, "@#[1,2,3]");
    }

    #[test]
    fn test_encode_table() {
        let data = json!([
            {"id": 1, "name": "Mario"},
            {"id": 2, "name": "Luigi"}
        ]);
        let result = encode(&data, Default::default()).unwrap();
        assert!(result.starts_with("|2|"));
    }
}
