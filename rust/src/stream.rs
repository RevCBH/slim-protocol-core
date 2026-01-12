//! SLIM Streaming API
//!
//! Provides incremental encoding/decoding and chunked processing.

use crate::decoder::decode;
use crate::encoder::encode;
use crate::error::Result;
use crate::types::{DecodeOptions, EncodeOptions};
use serde_json::Value;

/// Incremental encoder that buffers objects and encodes on end().
///
/// # Examples
///
/// ```
/// use slim_protocol::stream::SlimEncoder;
/// use slim_protocol::EncodeOptions;
/// use serde_json::json;
///
/// let mut encoder = SlimEncoder::new(Default::default());
/// encoder.write(json!({"id": 1, "name": "Mario"}));
/// encoder.write(json!({"id": 2, "name": "Luigi"}));
/// let slim = encoder.end().unwrap();
/// ```
pub struct SlimEncoder {
    buffer: Vec<Value>,
    options: EncodeOptions,
}

impl SlimEncoder {
    /// Creates a new incremental encoder.
    pub fn new(options: EncodeOptions) -> Self {
        Self {
            buffer: Vec::new(),
            options,
        }
    }

    /// Writes an object to the encoder buffer.
    pub fn write(&mut self, obj: Value) {
        self.buffer.push(obj);
    }

    /// Returns the number of buffered objects.
    pub fn size(&self) -> usize {
        self.buffer.len()
    }

    /// Encodes all buffered objects and returns the SLIM string.
    pub fn end(&self) -> Result<String> {
        if self.buffer.is_empty() {
            return Ok("@[]".to_string());
        }
        encode(&Value::Array(self.buffer.clone()), self.options.clone())
    }

    /// Clears the buffer without encoding.
    pub fn clear(&mut self) {
        self.buffer.clear();
    }
}

/// Incremental decoder that accumulates chunks and decodes on end().
///
/// # Examples
///
/// ```
/// use slim_protocol::stream::SlimDecoder;
/// use slim_protocol::DecodeOptions;
///
/// let mut decoder = SlimDecoder::new(Default::default());
/// decoder.write("|2|id#,name$|");
/// decoder.write("\n1,Mario\n2,Luigi");
/// let data = decoder.end().unwrap();
/// ```
pub struct SlimDecoder {
    buffer: String,
    options: DecodeOptions,
}

impl SlimDecoder {
    /// Creates a new incremental decoder.
    pub fn new(options: DecodeOptions) -> Self {
        Self {
            buffer: String::new(),
            options,
        }
    }

    /// Writes a string chunk to the decoder buffer.
    pub fn write(&mut self, chunk: &str) {
        self.buffer.push_str(chunk);
    }

    /// Returns true if there is pending data in the buffer.
    pub fn has_pending(&self) -> bool {
        !self.buffer.trim().is_empty()
    }

    /// Decodes the accumulated buffer and returns the value.
    pub fn end(&self) -> Result<Value> {
        if !self.has_pending() {
            return Ok(Value::Null);
        }
        decode(&self.buffer, self.options.clone())
    }

    /// Clears the buffer without decoding.
    pub fn clear(&mut self) {
        self.buffer.clear();
    }
}

/// Encodes objects in chunks, returning an iterator of SLIM strings.
///
/// # Arguments
///
/// * `objects` - Slice of objects to encode
/// * `chunk_size` - Number of objects per chunk
/// * `options` - Encoding options
///
/// # Examples
///
/// ```
/// use slim_protocol::stream::encode_chunked;
/// use slim_protocol::EncodeOptions;
/// use serde_json::json;
///
/// let objects = vec![
///     json!({"id": 1}),
///     json!({"id": 2}),
///     json!({"id": 3}),
///     json!({"id": 4}),
/// ];
///
/// for chunk in encode_chunked(&objects, 2, Default::default()) {
///     println!("{}", chunk.unwrap());
/// }
/// ```
pub fn encode_chunked(
    objects: &[Value],
    chunk_size: usize,
    options: EncodeOptions,
) -> impl Iterator<Item = Result<String>> + '_ {
    objects.chunks(chunk_size).map(move |chunk| {
        encode(&Value::Array(chunk.to_vec()), options.clone())
    })
}

/// Decodes a SLIM string and returns an iterator over the objects.
///
/// # Arguments
///
/// * `slim` - SLIM string to decode
/// * `options` - Decoding options
///
/// # Examples
///
/// ```
/// use slim_protocol::stream::decode_iter;
/// use slim_protocol::DecodeOptions;
///
/// let slim = "|2|id#,name$|\n1,Mario\n2,Luigi";
/// for obj in decode_iter(slim, Default::default()).unwrap() {
///     println!("{:?}", obj);
/// }
/// ```
pub fn decode_iter(slim: &str, options: DecodeOptions) -> Result<impl Iterator<Item = Value>> {
    let data = decode(slim, options)?;

    let items: Vec<Value> = match data {
        Value::Array(arr) => arr.into_iter()
            .filter(|v| v.is_object())
            .collect(),
        obj if obj.is_object() => vec![obj],
        _ => vec![],
    };

    Ok(items.into_iter())
}

/// Collects an iterator into a Vec.
///
/// # Arguments
///
/// * `iter` - Iterator to collect
///
/// # Examples
///
/// ```
/// use slim_protocol::stream::collect;
///
/// let items = vec![1, 2, 3];
/// let collected: Vec<i32> = collect(items.into_iter());
/// ```
pub fn collect<T, I: Iterator<Item = T>>(iter: I) -> Vec<T> {
    iter.collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_slim_encoder() {
        let mut encoder = SlimEncoder::new(Default::default());
        assert_eq!(encoder.size(), 0);

        encoder.write(json!({"id": 1, "name": "Mario"}));
        encoder.write(json!({"id": 2, "name": "Luigi"}));
        assert_eq!(encoder.size(), 2);

        let slim = encoder.end().unwrap();
        assert!(slim.starts_with("|2|"));
    }

    #[test]
    fn test_slim_encoder_empty() {
        let encoder = SlimEncoder::new(Default::default());
        let slim = encoder.end().unwrap();
        assert_eq!(slim, "@[]");
    }

    #[test]
    fn test_slim_decoder() {
        let mut decoder = SlimDecoder::new(Default::default());
        assert!(!decoder.has_pending());

        decoder.write("|2|id#,name$|");
        decoder.write("\n1,Mario\n2,Luigi");
        assert!(decoder.has_pending());

        let data = decoder.end().unwrap();
        assert!(data.is_array());
        assert_eq!(data.as_array().unwrap().len(), 2);
    }

    #[test]
    fn test_slim_decoder_empty() {
        let decoder = SlimDecoder::new(Default::default());
        let data = decoder.end().unwrap();
        assert!(data.is_null());
    }

    #[test]
    fn test_encode_chunked() {
        let objects = vec![
            json!({"id": 1}),
            json!({"id": 2}),
            json!({"id": 3}),
            json!({"id": 4}),
        ];

        let chunks: Vec<_> = encode_chunked(&objects, 2, Default::default())
            .collect::<Result<Vec<_>>>()
            .unwrap();

        assert_eq!(chunks.len(), 2);
        assert!(chunks[0].starts_with("|2|"));
        assert!(chunks[1].starts_with("|2|"));
    }

    #[test]
    fn test_decode_iter() {
        let slim = "|2|id#,name$|\n1,Mario\n2,Luigi";
        let objects: Vec<_> = decode_iter(slim, Default::default())
            .unwrap()
            .collect();

        assert_eq!(objects.len(), 2);
        assert!(objects[0].is_object());
        assert!(objects[1].is_object());
    }

    #[test]
    fn test_collect() {
        let items = vec![1, 2, 3];
        let collected: Vec<i32> = collect(items.into_iter());
        assert_eq!(collected, vec![1, 2, 3]);
    }
}
