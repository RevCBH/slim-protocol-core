//! SLIM Decoder
//!
//! Parses SLIM format strings back to JSON values.

use crate::error::{Result, SlimError};
use crate::types::*;
use serde_json::{Map, Value};

/// Decodes a SLIM string to a JSON value.
///
/// # Arguments
///
/// * `slim` - The SLIM string to decode
/// * `options` - Decoding options
///
/// # Examples
///
/// ```
/// use slim_protocol::{decode, DecodeOptions};
///
/// let slim = "{name:Mario,age:#30}";
/// let value = decode(slim, Default::default()).unwrap();
/// ```
pub fn decode(slim: &str, _options: DecodeOptions) -> Result<Value> {
    let mut parser = SlimParser::new(slim);
    parser.parse_value()
}

/// Internal parser structure
struct SlimParser {
    input: Vec<char>,
    pos: usize,
}

impl SlimParser {
    fn new(input: &str) -> Self {
        Self {
            input: input.chars().collect(),
            pos: 0,
        }
    }

    /// Peek at the next n characters without consuming
    fn peek(&self, n: usize) -> String {
        self.input
            .iter()
            .skip(self.pos)
            .take(n)
            .collect()
    }

    /// Peek at a single character
    fn peek_char(&self) -> Option<char> {
        self.input.get(self.pos).copied()
    }

    /// Consume and return the next n characters
    fn consume(&mut self, n: usize) -> String {
        let s: String = self.input.iter().skip(self.pos).take(n).collect();
        self.pos += n;
        s
    }

    /// Consume a single character
    fn consume_char(&mut self) -> Option<char> {
        if self.pos < self.input.len() {
            let ch = self.input[self.pos];
            self.pos += 1;
            Some(ch)
        } else {
            None
        }
    }

    /// Try to match a string, consuming if matched
    fn match_str(&mut self, s: &str) -> bool {
        if self.peek(s.len()) == s {
            self.consume(s.len());
            true
        } else {
            false
        }
    }

    /// Skip whitespace
    fn skip_ws(&mut self) {
        while self.pos < self.input.len() && self.peek_char() == Some(' ') {
            self.pos += 1;
        }
    }

    /// Check if we've reached the end
    fn is_end(&self) -> bool {
        self.pos >= self.input.len()
    }

    /// Parse any SLIM value
    fn parse_value(&mut self) -> Result<Value> {
        self.skip_ws();

        if self.is_end() {
            return Ok(Value::Null);
        }

        let ch = self.peek_char().unwrap();

        match ch {
            '!' => {
                self.consume_char();
                self.parse_null()
            }
            '?' => {
                self.consume_char();
                Ok(Value::Bool(self.consume_char() == Some('T')))
            }
            '#' => {
                self.consume_char();
                self.parse_number()
            }
            '@' => {
                self.consume_char();
                self.parse_array()
            }
            '*' => {
                self.consume_char();
                self.parse_matrix()
            }
            '{' => self.parse_object(),
            '|' => self.parse_table(),
            '"' => self.parse_quoted_string(),
            _ => Ok(Value::String(self.parse_unquoted())),
        }
    }

    /// Parse null/undefined/DEEP
    fn parse_null(&mut self) -> Result<Value> {
        if self.match_str("null") || self.match_str("undef") || self.match_str("DEEP") {
            Ok(Value::Null)
        } else {
            Ok(Value::Null)
        }
    }

    /// Parse a number
    fn parse_number(&mut self) -> Result<Value> {
        if self.match_str("NaN") {
            return Ok(Value::Number(serde_json::Number::from_f64(f64::NAN).unwrap()));
        }
        if self.match_str("-Inf") {
            return Ok(Value::Number(serde_json::Number::from_f64(f64::NEG_INFINITY).unwrap()));
        }
        if self.match_str("Inf") {
            return Ok(Value::Number(serde_json::Number::from_f64(f64::INFINITY).unwrap()));
        }

        let mut num_str = String::new();
        while !self.is_end() {
            if let Some(ch) = self.peek_char() {
                if ch.is_ascii_digit() || matches!(ch, '.' | 'e' | 'E' | '+' | '-') {
                    num_str.push(ch);
                    self.consume_char();
                } else {
                    break;
                }
            } else {
                break;
            }
        }

        if let Ok(n) = num_str.parse::<f64>() {
            Ok(Value::Number(serde_json::Number::from_f64(n).unwrap()))
        } else {
            Err(SlimError::ParseError(format!("Invalid number: {}", num_str)))
        }
    }

    /// Parse a quoted string
    fn parse_quoted_string(&mut self) -> Result<Value> {
        self.consume_char(); // Opening "
        let mut val = String::new();

        while !self.is_end() {
            if self.peek_char() == Some('"') {
                self.consume_char();
                if self.peek_char() == Some('"') {
                    // Escaped quote
                    val.push('"');
                    self.consume_char();
                } else {
                    // End of string
                    break;
                }
            } else if self.peek(2) == "\\n" {
                self.consume(2);
                val.push('\n');
            } else if let Some(ch) = self.consume_char() {
                val.push(ch);
            }
        }

        Ok(Value::String(val))
    }

    /// Parse an unquoted string
    fn parse_unquoted(&mut self) -> String {
        let mut val = String::new();
        while !self.is_end() {
            if let Some(ch) = self.peek_char() {
                if matches!(ch, ',' | ';' | '\n' | '|' | '{' | '}' | '[' | ']') {
                    break;
                }
                val.push(ch);
                self.consume_char();
            } else {
                break;
            }
        }
        val
    }

    /// Parse an array
    fn parse_array(&mut self) -> Result<Value> {
        let is_numeric = self.peek_char() == Some('#');
        if is_numeric {
            self.consume_char();
        }

        if !self.match_str("[") {
            return Ok(Value::Array(vec![]));
        }

        if self.peek_char() == Some(']') {
            self.consume_char();
            return Ok(Value::Array(vec![]));
        }

        let mut items = Vec::new();

        while !self.is_end() && self.peek_char() != Some(']') {
            if is_numeric {
                // Parse number directly
                let mut num_str = String::new();
                while !self.is_end() {
                    if let Some(ch) = self.peek_char() {
                        if ch.is_ascii_digit() || matches!(ch, '.' | 'e' | 'E' | '+' | '-') {
                            num_str.push(ch);
                            self.consume_char();
                        } else {
                            break;
                        }
                    } else {
                        break;
                    }
                }
                if !num_str.is_empty() {
                    if let Ok(n) = num_str.parse::<f64>() {
                        items.push(Value::Number(serde_json::Number::from_f64(n).unwrap()));
                    }
                }
            } else {
                items.push(self.parse_value()?);
            }

            // Skip separator
            if self.peek_char() == Some(',') || self.peek_char() == Some(';') {
                self.consume_char();
            }
        }

        self.match_str("]");
        Ok(Value::Array(items))
    }

    /// Parse a 2D matrix
    fn parse_matrix(&mut self) -> Result<Value> {
        if !self.match_str("[") {
            return Ok(Value::Array(vec![]));
        }

        let mut rows = Vec::new();
        let mut current_row = Vec::new();

        while !self.is_end() && self.peek_char() != Some(']') {
            if self.peek_char() == Some(';') {
                self.consume_char();
                if !current_row.is_empty() {
                    rows.push(Value::Array(current_row.clone()));
                    current_row.clear();
                }
            } else if self.peek_char() == Some(',') {
                self.consume_char();
            } else if let Some(ch) = self.peek_char() {
                if ch.is_ascii_digit() || ch == '-' || ch == '.' {
                    let mut num_str = String::new();
                    while !self.is_end() {
                        if let Some(ch) = self.peek_char() {
                            if ch.is_ascii_digit() || matches!(ch, '.' | 'e' | 'E' | '+' | '-') {
                                num_str.push(ch);
                                self.consume_char();
                            } else {
                                break;
                            }
                        } else {
                            break;
                        }
                    }
                    if let Ok(n) = num_str.parse::<f64>() {
                        current_row.push(Value::Number(serde_json::Number::from_f64(n).unwrap()));
                    }
                } else {
                    break;
                }
            } else {
                break;
            }
        }

        if !current_row.is_empty() {
            rows.push(Value::Array(current_row));
        }

        self.match_str("]");
        Ok(Value::Array(rows))
    }

    /// Parse an object
    fn parse_object(&mut self) -> Result<Value> {
        self.match_str("{");
        let mut obj = Map::new();

        while !self.is_end() && self.peek_char() != Some('}') {
            self.skip_ws();

            // Parse key
            let key = if self.peek_char() == Some('"') {
                if let Value::String(s) = self.parse_quoted_string()? {
                    s
                } else {
                    String::new()
                }
            } else {
                let mut k = String::new();
                while !self.is_end() {
                    if let Some(ch) = self.peek_char() {
                        if matches!(ch, ':' | ',' | '{' | '}') {
                            break;
                        }
                        k.push(ch);
                        self.consume_char();
                    } else {
                        break;
                    }
                }
                k
            };

            self.match_str(":");
            let value = self.parse_value()?;
            obj.insert(key, value);

            if self.peek_char() == Some(',') {
                self.consume_char();
            }
        }

        self.match_str("}");
        Ok(Value::Object(obj))
    }

    /// Parse a table (array of objects)
    fn parse_table(&mut self) -> Result<Value> {
        self.match_str("|");

        // Parse row count
        let mut count_str = String::new();
        while self.peek_char() != Some('|') && !self.is_end() {
            if let Some(ch) = self.consume_char() {
                count_str.push(ch);
            }
        }
        self.match_str("|");

        // Parse schema
        let mut schema_str = String::new();
        while self.peek_char() != Some('|') && !self.is_end() {
            if let Some(ch) = self.consume_char() {
                schema_str.push(ch);
            }
        }
        self.match_str("|");

        // Parse column definitions
        let columns: Vec<(String, String)> = schema_str
            .split(',')
            .map(|col| {
                // Find where the type markers start
                let mut name_end = col.len();
                for (i, ch) in col.char_indices() {
                    if matches!(ch, '#' | '?' | '@' | '~' | '$' | '!') {
                        name_end = i;
                        break;
                    }
                }
                let name = col[..name_end].to_string();
                let type_marker = col[name_end..].to_string();
                (name, type_marker)
            })
            .collect();

        let mut rows = Vec::new();

        // Parse data rows
        while !self.is_end() {
            if self.peek_char() == Some('\n') {
                self.consume_char();
            }
            if self.is_end() || matches!(self.peek_char(), Some('}') | Some(',') | Some(']')) {
                break;
            }

            let mut obj = Map::new();

            for (i, (name, type_marker)) in columns.iter().enumerate() {
                if i > 0 && self.peek_char() == Some(',') {
                    self.consume_char();
                }

                // Empty cell
                if matches!(self.peek_char(), Some(',') | Some('\n') | Some('}')) || self.is_end() {
                    if type_marker.contains('!') {
                        obj.insert(name.clone(), Value::Null);
                    }
                    continue;
                }

                let val = if self.peek_char() == Some('"') {
                    self.parse_quoted_string()?
                } else if type_marker.starts_with('#') {
                    let mut num_str = String::new();
                    while !self.is_end() {
                        if let Some(ch) = self.peek_char() {
                            if ch.is_ascii_digit() || matches!(ch, '.' | 'e' | 'E' | '+' | '-') {
                                num_str.push(ch);
                                self.consume_char();
                            } else {
                                break;
                            }
                        } else {
                            break;
                        }
                    }
                    if num_str.is_empty() {
                        Value::Null
                    } else {
                        Value::Number(serde_json::Number::from_f64(num_str.parse().unwrap()).unwrap())
                    }
                } else if type_marker.starts_with('?') {
                    Value::Bool(self.consume_char() == Some('T'))
                } else if type_marker.starts_with('@') {
                    let mut arr_str = String::new();
                    while !self.is_end() {
                        if let Some(ch) = self.peek_char() {
                            if matches!(ch, ',' | '\n' | '}') {
                                break;
                            }
                            arr_str.push(ch);
                            self.consume_char();
                        } else {
                            break;
                        }
                    }
                    if arr_str == "[]" {
                        Value::Array(vec![])
                    } else {
                        let elements: Vec<&str> = arr_str.split('+').filter(|x| !x.is_empty()).collect();
                        if elements.iter().all(|x| x.parse::<f64>().is_ok()) {
                            Value::Array(
                                elements
                                    .iter()
                                    .filter_map(|x| x.parse::<f64>().ok())
                                    .map(|n| Value::Number(serde_json::Number::from_f64(n).unwrap()))
                                    .collect(),
                            )
                        } else {
                            Value::Array(elements.iter().map(|x| Value::String(x.to_string())).collect())
                        }
                    }
                } else if type_marker.starts_with('~') {
                    self.parse_value()?
                } else {
                    // String
                    if self.peek_char() == Some('"') {
                        self.parse_quoted_string()?
                    } else {
                        let mut s = String::new();
                        while !self.is_end() {
                            if let Some(ch) = self.peek_char() {
                                if matches!(ch, ',' | '\n' | '}') {
                                    break;
                                }
                                s.push(ch);
                                self.consume_char();
                            } else {
                                break;
                            }
                        }
                        Value::String(s)
                    }
                };

                // Only add if not empty/null
                match &val {
                    Value::String(s) if s.is_empty() => {
                        if type_marker.contains('!') {
                            obj.insert(name.clone(), Value::Null);
                        }
                    }
                    Value::Null => {
                        if type_marker.contains('!') {
                            obj.insert(name.clone(), Value::Null);
                        }
                    }
                    _ => {
                        obj.insert(name.clone(), val);
                    }
                }
            }

            rows.push(Value::Object(obj));

            if self.peek_char() == Some('\n') {
                self.consume_char();
            } else if !matches!(self.peek_char(), Some('}') | Some(',') | Some(']')) && !self.is_end() {
                break;
            }
        }

        Ok(Value::Array(rows))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_decode_primitives() {
        assert_eq!(decode("!null", Default::default()).unwrap(), Value::Null);
        assert_eq!(decode("?T", Default::default()).unwrap(), json!(true));
        assert_eq!(decode("?F", Default::default()).unwrap(), json!(false));
        assert_eq!(decode("#42", Default::default()).unwrap(), json!(42.0));
        assert_eq!(decode("hello", Default::default()).unwrap(), json!("hello"));
    }

    #[test]
    fn test_decode_object() {
        let result = decode("{name:Mario,age:#30}", Default::default()).unwrap();
        assert_eq!(result, json!({"name": "Mario", "age": 30.0}));
    }

    #[test]
    fn test_decode_array() {
        let result = decode("@#[1,2,3]", Default::default()).unwrap();
        assert_eq!(result, json!([1.0, 2.0, 3.0]));
    }
}
