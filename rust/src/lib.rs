//! # SLIM Protocol
//!
//! SLIM (Structured Lightweight Interchange Markup) is a token-efficient data serialization format
//! designed for AI/LLM applications. It reduces token usage by 40-50% compared to JSON while
//! preserving all data.
//!
//! ## Quick Start
//!
//! ```rust
//! use slim_protocol::{encode, decode, Value};
//! use serde_json::json;
//!
//! // Encode data to SLIM
//! let data = json!([
//!     {"id": 1, "name": "Mario", "active": true},
//!     {"id": 2, "name": "Luigi", "active": false}
//! ]);
//!
//! let slim = encode(&data, Default::default()).unwrap();
//! // Output: |2|id#,name$,active?|
//! //         1,Mario,T
//! //         2,Luigi,F
//!
//! // Decode SLIM back to data
//! let restored = decode(&slim, Default::default()).unwrap();
//! // Verify it's an array with 2 elements
//! assert!(restored.is_array());
//! assert_eq!(restored.as_array().unwrap().len(), 2);
//! ```

pub mod types;
pub mod encoder;
pub mod decoder;
pub mod schema;
pub mod utils;
pub mod error;
pub mod stream;

pub use types::{Value, EncodeOptions, DecodeOptions};
pub use encoder::encode;
pub use decoder::decode;
pub use schema::{infer_schema, parse_schema, validate_schema};
pub use error::{SlimError, Result};
pub use stream::{SlimEncoder, SlimDecoder, encode_chunked, decode_iter, collect};
pub use utils::{deep_equal, clone, get_path, set_path, estimate_tokens, calculate_savings};
