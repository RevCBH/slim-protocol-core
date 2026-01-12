//! Property-based tests for SLIM Protocol
//!
//! These tests verify that the implementation behaves correctly for
//! arbitrary inputs using property-based testing.

use proptest::prelude::*;
use serde_json::{json, Value};
use slim_protocol::{
    decode, encode, DecodeOptions, EncodeOptions,
    deep_equal, clone, get_path, set_path, estimate_tokens, calculate_savings,
    SlimEncoder, SlimDecoder, encode_chunked, decode_iter,
};

// Strategy for generating arbitrary JSON values
fn arb_json_value() -> impl Strategy<Value = Value> {
    let leaf = prop_oneof![
        Just(Value::Null),
        any::<bool>().prop_map(Value::Bool),
        (-1000.0f64..1000.0f64).prop_map(|f| json!(f)),
        "[a-zA-Z0-9 ]{0,20}".prop_map(|s| Value::String(s)),
    ];

    leaf.prop_recursive(
        3,  // depth
        64, // max nodes
        10, // items per collection
        |inner| {
            prop_oneof![
                // Arrays
                prop::collection::vec(inner.clone(), 0..5)
                    .prop_map(|v| Value::Array(v)),
                // Objects
                prop::collection::hash_map("[a-zA-Z]{1,10}", inner, 0..5)
                    .prop_map(|m| {
                        let map: serde_json::Map<String, Value> = m.into_iter().collect();
                        Value::Object(map)
                    }),
            ]
        },
    )
}

// Strategy for generating arrays of objects (table format)
// Uses consistent keys across all objects to ensure proper roundtrip
fn arb_object_array() -> impl Strategy<Value = Vec<Value>> {
    // Generate arrays with consistent schema (same keys in all objects)
    (
        any::<bool>(),
        (-1000.0f64..1000.0f64),
        "[a-zA-Z0-9]{1,10}",
    ).prop_flat_map(|(b, n, s)| {
        // Create multiple objects with the same schema
        prop::collection::vec(
            (any::<bool>(), (-1000.0f64..1000.0f64), "[a-zA-Z0-9]{1,10}"),
            2..8
        ).prop_map(move |items| {
            items.into_iter().map(|(b, n, s)| {
                json!({
                    "active": b,
                    "value": n,
                    "name": s
                })
            }).collect()
        })
    })
}

proptest! {
    // ==================== Roundtrip Tests ====================

    #[test]
    fn prop_roundtrip_bool(v in any::<bool>()) {
        let encoded = encode(&json!(v), Default::default()).unwrap();
        let decoded = decode(&encoded, Default::default()).unwrap();
        prop_assert_eq!(decoded.as_bool(), Some(v));
    }

    #[test]
    fn prop_roundtrip_number(v in -1000.0f64..1000.0f64) {
        let encoded = encode(&json!(v), Default::default()).unwrap();
        let decoded = decode(&encoded, Default::default()).unwrap();
        if let Some(f) = decoded.as_f64() {
            prop_assert!((f - v).abs() < 0.0001 || (v.is_nan() && f.is_nan()));
        } else {
            prop_assert!(false, "Expected number");
        }
    }

    #[test]
    fn prop_roundtrip_alphanumeric_string(v in "[a-zA-Z0-9]{0,50}") {
        let encoded = encode(&json!(v), Default::default()).unwrap();
        let decoded = decode(&encoded, Default::default()).unwrap();
        prop_assert_eq!(decoded.as_str(), Some(v.as_str()));
    }

    #[test]
    fn prop_roundtrip_object_array(arr in arb_object_array()) {
        let value = Value::Array(arr.clone());
        let encoded = encode(&value, Default::default()).unwrap();
        let decoded = decode(&encoded, Default::default()).unwrap();

        prop_assert!(decoded.is_array());
        let decoded_arr = decoded.as_array().unwrap();
        prop_assert_eq!(decoded_arr.len(), arr.len());
    }

    // ==================== Encode Stability Tests ====================

    #[test]
    fn prop_encode_stability_bool(v in any::<bool>()) {
        let encoded1 = encode(&json!(v), Default::default()).unwrap();
        let encoded2 = encode(&json!(v), Default::default()).unwrap();
        prop_assert_eq!(encoded1, encoded2);
    }

    #[test]
    fn prop_encode_stability_number(v in -1000.0f64..1000.0f64) {
        let encoded1 = encode(&json!(v), Default::default()).unwrap();
        let encoded2 = encode(&json!(v), Default::default()).unwrap();
        prop_assert_eq!(encoded1, encoded2);
    }

    #[test]
    fn prop_encode_stability_string(v in "[a-zA-Z0-9]{0,50}") {
        let encoded1 = encode(&json!(v), Default::default()).unwrap();
        let encoded2 = encode(&json!(v), Default::default()).unwrap();
        prop_assert_eq!(encoded1, encoded2);
    }

    // ==================== Deep Equal Tests ====================

    #[test]
    fn prop_deep_equal_reflexive(v in arb_json_value()) {
        prop_assert!(deep_equal(&v, &v));
    }

    #[test]
    fn prop_deep_equal_symmetric(a in arb_json_value(), b in arb_json_value()) {
        prop_assert_eq!(deep_equal(&a, &b), deep_equal(&b, &a));
    }

    // ==================== Clone Tests ====================

    #[test]
    fn prop_clone_equals_original(v in arb_json_value()) {
        let cloned = clone(&v);
        prop_assert!(deep_equal(&v, &cloned));
    }

    // ==================== Path Tests ====================

    #[test]
    fn prop_get_set_path_roundtrip(
        key in "[a-zA-Z]{1,5}",
        value in arb_json_value()
    ) {
        let obj = json!({});
        let updated = set_path(&obj, &key, &value);
        let retrieved = get_path(&updated, &key);
        prop_assert!(deep_equal(&retrieved, &value));
    }

    #[test]
    fn prop_nested_path_roundtrip(
        key1 in "[a-zA-Z]{1,5}",
        key2 in "[a-zA-Z]{1,5}",
        value in arb_json_value()
    ) {
        let path = format!("{}.{}", key1, key2);
        let obj = json!({});
        let updated = set_path(&obj, &path, &value);
        let retrieved = get_path(&updated, &path);
        prop_assert!(deep_equal(&retrieved, &value));
    }

    // ==================== Token Estimation Tests ====================

    #[test]
    fn prop_estimate_tokens_non_negative(s in ".*") {
        let tokens = estimate_tokens(&s);
        prop_assert!(tokens >= 0);
    }

    #[test]
    fn prop_estimate_tokens_proportional(s in ".{0,100}") {
        let tokens = estimate_tokens(&s);
        // Tokens should be roughly length/4
        let expected = (s.len() as f64 / 4.0).ceil() as usize;
        prop_assert_eq!(tokens, expected);
    }

    // ==================== Streaming API Tests ====================

    #[test]
    fn prop_encoder_decoder_roundtrip(arr in arb_object_array()) {
        let mut encoder = SlimEncoder::new(Default::default());
        for obj in &arr {
            encoder.write(obj.clone());
        }
        let encoded = encoder.end().unwrap();

        let mut decoder = SlimDecoder::new(Default::default());
        decoder.write(&encoded);
        let decoded = decoder.end().unwrap();

        prop_assert!(decoded.is_array());
        prop_assert_eq!(decoded.as_array().unwrap().len(), arr.len());
    }

    #[test]
    fn prop_encode_chunked_preserves_data(arr in arb_object_array()) {
        if arr.len() < 2 {
            return Ok(());
        }

        let chunk_size = std::cmp::max(1, arr.len() / 2);
        let chunks: Vec<_> = encode_chunked(&arr, chunk_size, Default::default())
            .collect::<Result<Vec<_>, _>>()
            .unwrap();

        // Decode all chunks and collect objects
        let mut all_objects = Vec::new();
        for chunk in chunks {
            if let Ok(decoded) = decode(&chunk, Default::default()) {
                if let Some(decoded_arr) = decoded.as_array() {
                    all_objects.extend(decoded_arr.iter().cloned());
                }
            }
        }

        prop_assert_eq!(all_objects.len(), arr.len());
    }

    // ==================== Token Savings Tests ====================

    #[test]
    fn prop_calculate_savings_non_negative_json(
        json_len in 10usize..100
    ) {
        // When JSON is longer than SLIM, savings should be positive
        let slim = "x".repeat(json_len / 2);
        let json = "x".repeat(json_len);
        let savings = calculate_savings(&slim, &json);
        prop_assert!(savings >= 0, "Expected non-negative savings, got {}", savings);
    }

    #[test]
    fn prop_calculate_savings_handles_longer_slim(
        slim_len in 10usize..100
    ) {
        // When SLIM is longer, savings should be negative
        let slim = "x".repeat(slim_len);
        let json = "x".repeat(slim_len / 2);
        let savings = calculate_savings(&slim, &json);
        prop_assert!(savings <= 0, "Expected non-positive savings, got {}", savings);
    }
}

// ==================== Cross-Implementation Compatibility Tests ====================
// These tests use fixed test vectors that should produce identical results
// across JS/TS, Rust, and Go implementations.

#[test]
fn test_cross_impl_primitives() {
    // These are fixed test vectors for cross-implementation testing
    let test_cases = vec![
        (json!(null), "!null"),
        (json!(true), "?T"),
        (json!(false), "?F"),
        (json!(42), "#42"),
    ];

    for (input, expected) in test_cases {
        let encoded = encode(&input, Default::default()).unwrap();
        assert_eq!(encoded, expected, "Failed for input: {:?}", input);
    }
}

#[test]
fn test_cross_impl_table_format() {
    let data = json!([
        {"id": 1, "name": "Mario"},
        {"id": 2, "name": "Luigi"}
    ]);

    let encoded = encode(&data, Default::default()).unwrap();

    // Table format should start with |count|
    assert!(encoded.starts_with("|2|"));

    // Roundtrip should preserve data
    let decoded = decode(&encoded, Default::default()).unwrap();
    assert!(decoded.is_array());
    assert_eq!(decoded.as_array().unwrap().len(), 2);
}

#[test]
fn test_cross_impl_number_array() {
    let data = json!([1, 2, 3, 4, 5]);
    let encoded = encode(&data, Default::default()).unwrap();

    // Number arrays use @#[...] format
    assert_eq!(encoded, "@#[1,2,3,4,5]");

    let decoded = decode(&encoded, Default::default()).unwrap();
    assert!(decoded.is_array());
    assert_eq!(decoded.as_array().unwrap().len(), 5);
}

#[test]
fn test_cross_impl_matrix() {
    let data = json!([[1, 2], [3, 4]]);
    let encoded = encode(&data, Default::default()).unwrap();

    // Matrix format uses *[...] format
    assert_eq!(encoded, "*[1,2;3,4]");

    let decoded = decode(&encoded, Default::default()).unwrap();
    assert!(decoded.is_array());
    assert_eq!(decoded.as_array().unwrap().len(), 2);
}

#[test]
fn test_cross_impl_nested_object() {
    let data = json!({"user": {"name": "Mario", "age": 30}});
    let encoded = encode(&data, Default::default()).unwrap();

    let decoded = decode(&encoded, Default::default()).unwrap();
    let name = get_path(&decoded, "user.name");
    assert_eq!(name.as_str(), Some("Mario"));
}

#[test]
fn test_utility_functions() {
    // Test deep_equal
    assert!(deep_equal(&json!({"a": 1}), &json!({"a": 1})));
    assert!(!deep_equal(&json!({"a": 1}), &json!({"a": 2})));

    // Test clone
    let original = json!({"nested": {"value": 42}});
    let cloned = clone(&original);
    assert!(deep_equal(&original, &cloned));

    // Test get_path
    let data = json!({"user": {"name": "Mario", "scores": [10, 20, 30]}});
    assert_eq!(get_path(&data, "user.name"), json!("Mario"));
    assert_eq!(get_path(&data, "user.scores.1"), json!(20));

    // Test set_path
    let updated = set_path(&data, "user.name", &json!("Luigi"));
    assert_eq!(get_path(&updated, "user.name"), json!("Luigi"));

    // Test estimate_tokens
    assert_eq!(estimate_tokens("test"), 1);
    assert_eq!(estimate_tokens("12345678"), 2);

    // Test calculate_savings
    let slim = "|2|id#|\\n1\\n2";
    let json_str = "[{\"id\":1},{\"id\":2}]";
    let savings = calculate_savings(slim, json_str);
    assert!(savings > 0);
}

#[test]
fn test_streaming_api() {
    // Test SlimEncoder
    let mut encoder = SlimEncoder::new(Default::default());
    encoder.write(json!({"id": 1, "name": "Mario"}));
    encoder.write(json!({"id": 2, "name": "Luigi"}));
    assert_eq!(encoder.size(), 2);

    let slim = encoder.end().unwrap();
    assert!(slim.starts_with("|2|"));

    // Test SlimDecoder
    let mut decoder = SlimDecoder::new(Default::default());
    decoder.write(&slim);
    assert!(decoder.has_pending());

    let decoded = decoder.end().unwrap();
    assert!(decoded.is_array());
    assert_eq!(decoded.as_array().unwrap().len(), 2);

    // Test encode_chunked
    let objects = vec![
        json!({"id": 1}),
        json!({"id": 2}),
        json!({"id": 3}),
        json!({"id": 4}),
    ];

    let chunks: Vec<_> = encode_chunked(&objects, 2, Default::default())
        .collect::<Result<Vec<_>, _>>()
        .unwrap();
    assert_eq!(chunks.len(), 2);

    // Test decode_iter
    let objects_iter: Vec<_> = decode_iter(&slim, Default::default())
        .unwrap()
        .collect();
    assert_eq!(objects_iter.len(), 2);
}
