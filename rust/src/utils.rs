//! SLIM Utility Functions

use serde_json::Value;

/// Counts the approximate number of tokens in a string.
/// Uses a simple heuristic (characters / 4) as a rough estimate.
///
/// # Arguments
///
/// * `s` - String to count tokens in
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::estimate_tokens;
///
/// let tokens = estimate_tokens("Hello, world!");
/// ```
pub fn estimate_tokens(s: &str) -> usize {
    (s.len() as f64 / 4.0).ceil() as usize
}

/// Calculates token savings compared to JSON.
///
/// # Arguments
///
/// * `slim_str` - SLIM-encoded string
/// * `json_str` - JSON-encoded string
///
/// # Returns
///
/// Percentage savings (0-100)
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::calculate_savings;
///
/// let savings = calculate_savings("|2|id#|\\n1\\n2", "[{\"id\":1},{\"id\":2}]");
/// ```
pub fn calculate_savings(slim_str: &str, json_str: &str) -> i32 {
    let slim_tokens = estimate_tokens(slim_str) as i32;
    let json_tokens = estimate_tokens(json_str) as i32;

    if json_tokens == 0 {
        return 0;
    }

    (((json_tokens - slim_tokens) as f64 / json_tokens as f64) * 100.0).round() as i32
}

/// Deep equality check for JSON values (handles NaN correctly).
///
/// # Arguments
///
/// * `a` - First value
/// * `b` - Second value
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::deep_equal;
/// use serde_json::json;
///
/// assert!(deep_equal(&json!({"a": 1}), &json!({"a": 1})));
/// ```
pub fn deep_equal(a: &Value, b: &Value) -> bool {
    match (a, b) {
        (Value::Null, Value::Null) => true,
        (Value::Bool(a), Value::Bool(b)) => a == b,
        (Value::Number(a), Value::Number(b)) => {
            // Handle NaN case
            if let (Some(a_f), Some(b_f)) = (a.as_f64(), b.as_f64()) {
                if a_f.is_nan() && b_f.is_nan() {
                    return true;
                }
            }
            a == b
        }
        (Value::String(a), Value::String(b)) => a == b,
        (Value::Array(a), Value::Array(b)) => {
            a.len() == b.len() && a.iter().zip(b.iter()).all(|(a, b)| deep_equal(a, b))
        }
        (Value::Object(a), Value::Object(b)) => {
            a.len() == b.len()
                && a.iter()
                    .all(|(k, v)| b.get(k).map(|bv| deep_equal(v, bv)).unwrap_or(false))
        }
        _ => false,
    }
}

/// Deep clone a JSON value.
///
/// # Arguments
///
/// * `value` - Value to clone
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::clone;
/// use serde_json::json;
///
/// let original = json!({"name": "Mario"});
/// let cloned = clone(&original);
/// ```
pub fn clone(value: &Value) -> Value {
    value.clone()
}

/// Gets a value at a dot-separated path.
///
/// # Arguments
///
/// * `value` - Object to traverse
/// * `path` - Dot-separated path (e.g., "user.name")
///
/// # Returns
///
/// Value at path, or Null if not found
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::get_path;
/// use serde_json::json;
///
/// let data = json!({"user": {"name": "Mario"}});
/// let name = get_path(&data, "user.name");
/// ```
pub fn get_path(value: &Value, path: &str) -> Value {
    let parts: Vec<&str> = path.split('.').collect();
    let mut current = value;

    for part in parts {
        match current {
            Value::Object(obj) => {
                if let Some(next) = obj.get(part) {
                    current = next;
                } else {
                    return Value::Null;
                }
            }
            Value::Array(arr) => {
                if let Ok(index) = part.parse::<usize>() {
                    if let Some(next) = arr.get(index) {
                        current = next;
                    } else {
                        return Value::Null;
                    }
                } else {
                    return Value::Null;
                }
            }
            _ => return Value::Null,
        }
    }

    current.clone()
}

/// Sets a value at a dot-separated path, returning a new value.
///
/// # Arguments
///
/// * `value` - Object to modify
/// * `path` - Dot-separated path
/// * `new_value` - Value to set
///
/// # Returns
///
/// New value with the path set
///
/// # Examples
///
/// ```
/// use slim_protocol::utils::set_path;
/// use serde_json::json;
///
/// let data = json!({"user": {"name": "Mario"}});
/// let updated = set_path(&data, "user.name", &json!("Luigi"));
/// ```
pub fn set_path(value: &Value, path: &str, new_value: &Value) -> Value {
    let parts: Vec<&str> = path.split('.').collect();
    set_path_recursive(value, &parts, new_value)
}

fn set_path_recursive(value: &Value, parts: &[&str], new_value: &Value) -> Value {
    if parts.is_empty() {
        return new_value.clone();
    }

    let part = parts[0];
    let remaining = &parts[1..];

    match value {
        Value::Object(obj) => {
            let mut new_obj = obj.clone();
            let current = obj.get(part).unwrap_or(&Value::Null);
            let updated = set_path_recursive(current, remaining, new_value);
            new_obj.insert(part.to_string(), updated);
            Value::Object(new_obj)
        }
        _ => {
            let mut new_obj = serde_json::Map::new();
            let updated = set_path_recursive(&Value::Null, remaining, new_value);
            new_obj.insert(part.to_string(), updated);
            Value::Object(new_obj)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_estimate_tokens() {
        assert_eq!(estimate_tokens("test"), 1);
        assert_eq!(estimate_tokens("hello world"), 3);
    }

    #[test]
    fn test_calculate_savings() {
        let savings = calculate_savings("short", "much longer string");
        assert!(savings > 0);
    }

    #[test]
    fn test_deep_equal() {
        assert!(deep_equal(&json!({"a": 1}), &json!({"a": 1})));
        assert!(!deep_equal(&json!({"a": 1}), &json!({"a": 2})));
    }

    #[test]
    fn test_get_path() {
        let data = json!({"user": {"name": "Mario"}});
        assert_eq!(get_path(&data, "user.name"), json!("Mario"));
        assert_eq!(get_path(&data, "user.age"), Value::Null);
    }

    #[test]
    fn test_set_path() {
        let data = json!({"user": {"name": "Mario"}});
        let updated = set_path(&data, "user.name", &json!("Luigi"));
        assert_eq!(get_path(&updated, "user.name"), json!("Luigi"));
    }
}
