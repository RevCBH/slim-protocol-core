//! Basic SLIM encoding/decoding example

use slim_protocol::{encode, decode};
use serde_json::json;

fn main() {
    println!("=== SLIM Protocol - Basic Example ===\n");

    // Example 1: Simple object
    println!("Example 1: Simple object");
    let data = json!({"name": "Mario", "age": 30, "active": true});
    let slim = encode(&data, Default::default()).unwrap();
    println!("Original JSON: {}", serde_json::to_string(&data).unwrap());
    println!("SLIM format:   {}", slim);
    println!();

    // Example 2: Array of objects (table format)
    println!("Example 2: Array of objects (table format)");
    let users = json!([
        {"id": 1, "name": "Mario", "active": true},
        {"id": 2, "name": "Luigi", "active": false},
        {"id": 3, "name": "Peach", "active": true}
    ]);
    let slim = encode(&users, Default::default()).unwrap();
    println!("Original JSON: {}", serde_json::to_string(&users).unwrap());
    println!("SLIM format:\n{}", slim);
    println!();

    // Example 3: Decode back
    println!("Example 3: Decode SLIM back to JSON");
    let slim_str = "|2|id#,name$|\n1,Mario\n2,Luigi";
    let decoded = decode(slim_str, Default::default()).unwrap();
    println!("SLIM:    {}", slim_str.replace('\n', "\\n"));
    println!("Decoded: {}", serde_json::to_string_pretty(&decoded).unwrap());
    println!();

    // Example 4: Token savings
    println!("Example 4: Token savings comparison");
    let data = json!([
        {"id": 1, "score": 95.5, "passed": true},
        {"id": 2, "score": 87.2, "passed": true},
        {"id": 3, "score": 62.8, "passed": false}
    ]);
    let json_str = serde_json::to_string(&data).unwrap();
    let slim_str = encode(&data, Default::default()).unwrap();

    println!("JSON length: {} chars", json_str.len());
    println!("SLIM length: {} chars", slim_str.len());
    println!("Savings:     {:.1}%",
        ((json_str.len() - slim_str.len()) as f64 / json_str.len() as f64) * 100.0);
}
