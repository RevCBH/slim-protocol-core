# slim-protocol-core

[![npm version](https://img.shields.io/npm/v/slim-protocol-core.svg)](https://www.npmjs.com/package/slim-protocol-core)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**SLIM** (Structured Lightweight Interchange Markup) - Token-efficient data serialization for AI/LLM applications.

## Why SLIM?

When sending data to LLMs (ChatGPT, Claude, etc.), you pay per **token**. SLIM reduces tokens by **40-50%** compared to JSON while preserving all data.

```javascript
// JSON: 54 characters
JSON.stringify([{id: 1, name: "Mario"}, {id: 2, name: "Luigi"}])
// Output: [{"id":1,"name":"Mario"},{"id":2,"name":"Luigi"}]

// SLIM: 31 characters (-43%)
encode([{id: 1, name: "Mario"}, {id: 2, name: "Luigi"}])
// Output: |2|id#,name$|
//         1,Mario
//         2,Luigi
```

## Installation

```bash
npm install slim-protocol-core
```

## Quick Start

```typescript
import { encode, decode } from 'slim-protocol-core';

// Encode JavaScript data to SLIM
const data = [
  { id: 1, name: 'Mario', active: true },
  { id: 2, name: 'Luigi', active: false }
];

const slim = encode(data);
// Output:
// |2|id#,name$,active?|
// 1,Mario,T
// 2,Luigi,F

// Decode SLIM back to JavaScript
const restored = decode(slim);
// restored deep-equals data
```

## Benchmarks

### Token Savings vs JSON

| Data Type | JSON Tokens | SLIM Tokens | Savings |
|-----------|-------------|-------------|---------|
| User table (100 rows) | 2053 | 903 | **56%** |
| Nested config | 71 | 58 | **18%** |
| GPS track (50 points) | 300 | 273 | **9%** |
| **Average** | - | - | **49%** |

### When to Use SLIM

| Use Case | Expected Savings |
|----------|------------------|
| Arrays of objects (tables) | 40-60% |
| Objects with repeated keys | 30-50% |
| Mixed data | 15-30% |
| Numeric matrices | 5-15% |
| Single values | ~0% |

### Performance

| Operation | JSON | SLIM | Notes |
|-----------|------|------|-------|
| Encode (100 objects) | 17us | 61us | SLIM slower (table overhead) |
| Decode (100 objects) | 25us | 132us | SLIM slower (custom parser) |
| Output size | 8210B | 3610B | **SLIM -56%** |

**Trade-off**: You pay in CPU, you save in tokens/storage. For LLM workloads, token savings far outweigh the CPU cost.

## API Reference

### encode(data, options?)

Encodes JavaScript data to SLIM string.

```typescript
function encode(data: unknown, options?: EncodeOptions): string;

interface EncodeOptions {
  maxDepth?: number;        // Default: 15
  tableThreshold?: number;  // Min rows for table format, default: 1
}
```

### decode(slim, options?)

Decodes SLIM string to JavaScript data.

```typescript
function decode(slim: string, options?: DecodeOptions): unknown;

interface DecodeOptions {
  strict?: boolean;  // Throw on invalid input, default: false
}
```

### Schema Utilities

```typescript
import { inferSchema, parseSchema, validateSchema } from 'slim-protocol-core';

// Infer schema from data
const schema = inferSchema([{ id: 1, name: 'Mario' }]);
// Returns: 'id#,name$'

// Parse schema into column definitions
const cols = parseSchema('id#,name$,active?');
// Returns: [{name:'id', type:'#', nullable:false}, ...]

// Validate data against schema
const result = validateSchema(data, 'id#,name$,active?');
// Returns: { valid: true } or { valid: false, errors: [...] }
```

### Streaming API

For large datasets:

```typescript
import {
  createEncoder,
  createDecoder,
  encodeStream,
  decodeStream,
  encodeChunked,
  collect
} from 'slim-protocol-core';

// Incremental encoder
const encoder = createEncoder();
encoder.write({ id: 1, name: 'Mario' });
encoder.write({ id: 2, name: 'Luigi' });
const slim = encoder.end();

// Encode from async iterable
const slim = await encodeStream(asyncGenerator());

// Decode to async iterable
for await (const obj of decodeStream(slim)) {
  console.log(obj);
}

// Encode in chunks (for large data)
for (const chunk of encodeChunked(bigArray, 1000)) {
  await saveChunk(chunk);
}
```

### Utility Functions

```typescript
import {
  deepEqual,        // Deep comparison
  clone,            // Deep clone
  getPath,          // Access by path (e.g., 'user.name')
  setPath,          // Modify by path
  estimateTokens,   // Estimate token count
  calculateSavings  // Calculate savings percentage
} from 'slim-protocol-core';
```

## Supported Data Types

| JavaScript Type | SLIM Marker | Example |
|-----------------|-------------|---------|
| `null` | `!` | `!null` |
| `undefined` | `!` | `!undef` |
| `boolean` | `?` | `?T`, `?F` |
| `number` | `#` | `#42`, `#3.14` |
| `string` | (none) | `hello` |
| `string` (quoted) | `"` | `"hello, world"` |
| `Array` (numbers) | `@#` | `@#[1,2,3]` |
| `Array` (mixed) | `@` | `@[#1;hello;?T]` |
| `Array` (2D numbers) | `*` | `*[1,2;3,4]` |
| `Array` (objects) | `\|n\|schema\|` | `\|2\|id#,name$\|...` |
| `Object` | `{}` | `{name:Mario}` |

## Special Values

| Value | SLIM |
|-------|------|
| `NaN` | `#NaN` |
| `Infinity` | `#Inf` |
| `-Infinity` | `#-Inf` |
| Empty string | `""` |
| Empty array | `@[]` |
| Empty object | `{}` |

## Test Coverage

- **210 tests** passing
- **96.67%** code coverage
- Tested: primitives, unicode, emoji, CJK, edge cases, streaming, schemas

## Requirements

- Node.js >= 18.0.0
- TypeScript >= 5.0 (optional, for development)

## Zero Dependencies

This package has **zero runtime dependencies**. Only dev dependencies for testing/building.

## Development

```bash
# Install dependencies
npm install

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run benchmarks
npm run benchmark

# Build
npm run build

# Type check
npm run typecheck
```

## Other Language Implementations

### Rust

The Rust implementation provides a native, high-performance SLIM encoder/decoder.

**Installation:**

```toml
[dependencies]
slim-protocol = { path = "rust" }
```

**Quick Start:**

```rust
use slim_protocol::{encode, decode, EncodeOptions, DecodeOptions};
use serde_json::json;

// Encode data to SLIM
let data = json!([
    {"id": 1, "name": "Mario", "active": true},
    {"id": 2, "name": "Luigi", "active": false}
]);

let slim = encode(&data, Default::default()).unwrap();
// Output: |2|id#,name$,active?|
//         1,Mario,T
//         2,Luigi,F

// Decode SLIM back to data
let restored = decode(&slim, Default::default()).unwrap();
```

**Streaming API Examples:**

```rust
use slim_protocol::{SlimEncoder, SlimDecoder, encode_chunked, decode_iter};
use serde_json::json;

// === Incremental Encoding ===
// Buffer objects one at a time, encode when ready
let mut encoder = SlimEncoder::new(Default::default());

// Simulate receiving objects from a stream
for i in 1..=100 {
    encoder.write(json!({
        "id": i,
        "value": i * 10,
        "processed": true
    }));
}

println!("Buffered {} objects", encoder.size());
let slim = encoder.end().unwrap();
// All 100 objects encoded as a single SLIM table

// === Incremental Decoding ===
// Accumulate chunks, decode when complete
let mut decoder = SlimDecoder::new(Default::default());

// Simulate receiving SLIM data in chunks
decoder.write("|3|id#,name$|");
decoder.write("\n1,Alice");
decoder.write("\n2,Bob");
decoder.write("\n3,Charlie");

if decoder.has_pending() {
    let data = decoder.end().unwrap();
    println!("Decoded: {:?}", data);
}

// === Chunked Encoding ===
// Split large datasets into smaller encoded chunks
let large_dataset: Vec<_> = (1..=1000).map(|i| json!({
    "id": i,
    "data": format!("item-{}", i)
})).collect();

// Encode in batches of 100
for (i, chunk) in encode_chunked(&large_dataset, 100, Default::default()).enumerate() {
    let encoded = chunk.unwrap();
    println!("Chunk {}: {} bytes", i, encoded.len());
    // Save or transmit each chunk independently
}

// === Iterate Over Decoded Objects ===
// Process objects one at a time without loading all into memory
let slim_data = "|3|id#,name$|\n1,Alice\n2,Bob\n3,Charlie";

for obj in decode_iter(slim_data, Default::default()).unwrap() {
    println!("Processing: {:?}", obj);
}
```

**Utility Functions Examples:**

```rust
use slim_protocol::{deep_equal, clone, get_path, set_path, estimate_tokens, calculate_savings};
use serde_json::json;

// === Deep Equality ===
// Compare values including nested structures (handles NaN correctly)
let a = json!({"users": [{"id": 1}, {"id": 2}]});
let b = json!({"users": [{"id": 1}, {"id": 2}]});
assert!(deep_equal(&a, &b));

// === Deep Clone ===
// Create independent copies of complex structures
let original = json!({"nested": {"data": [1, 2, 3]}});
let cloned = clone(&original);
// Modifications to cloned won't affect original

// === Path-based Access ===
// Navigate nested structures with dot notation
let config = json!({
    "database": {
        "host": "localhost",
        "ports": [5432, 5433, 5434]
    }
});

let host = get_path(&config, "database.host");
assert_eq!(host, json!("localhost"));

let first_port = get_path(&config, "database.ports.0");
assert_eq!(first_port, json!(5432));

// === Path-based Updates ===
// Immutably update nested values
let updated = set_path(&config, "database.host", &json!("production.db"));
// Original unchanged, updated has new host

// Create nested paths that don't exist
let empty = json!({});
let with_nested = set_path(&empty, "a.b.c.d", &json!("deep value"));
// Creates: {"a": {"b": {"c": {"d": "deep value"}}}}

// === Token Estimation ===
// Estimate LLM token usage
let text = "Hello, this is a test string for token estimation.";
let tokens = estimate_tokens(text);
println!("Estimated {} tokens", tokens);

// === Savings Calculation ===
// Compare SLIM vs JSON efficiency
let data = json!([
    {"id": 1, "name": "Alice", "active": true},
    {"id": 2, "name": "Bob", "active": false},
]);
let slim_str = encode(&data, Default::default()).unwrap();
let json_str = serde_json::to_string(&data).unwrap();

let savings = calculate_savings(&slim_str, &json_str);
println!("SLIM saves {}% tokens vs JSON", savings);
```

**API Reference:**

```rust
// Core functions
fn encode(data: &Value, options: EncodeOptions) -> Result<String>;
fn decode(slim: &str, options: DecodeOptions) -> Result<Value>;

// Options
struct EncodeOptions {
    max_depth: usize,        // Default: 15
    table_threshold: usize,  // Default: 1
    pretty: bool,            // Default: false
}

struct DecodeOptions {
    strict: bool,  // Default: false
}

// Schema utilities
fn infer_schema(data: &[Value]) -> String;
fn parse_schema(schema: &str) -> Vec<ColumnDef>;
fn validate_schema(data: &Value, schema: &str) -> ValidationResult;

// Streaming API
struct SlimEncoder { ... }
impl SlimEncoder {
    fn new(options: EncodeOptions) -> Self;
    fn write(&mut self, obj: Value);
    fn end(&self) -> Result<String>;
    fn size(&self) -> usize;
}

struct SlimDecoder { ... }
impl SlimDecoder {
    fn new(options: DecodeOptions) -> Self;
    fn write(&mut self, chunk: &str);
    fn end(&self) -> Result<Value>;
    fn has_pending(&self) -> bool;
}

fn encode_chunked(objects: &[Value], chunk_size: usize, options: EncodeOptions)
    -> impl Iterator<Item = Result<String>>;
fn decode_iter(slim: &str, options: DecodeOptions)
    -> Result<impl Iterator<Item = Value>>;

// Utility functions
fn deep_equal(a: &Value, b: &Value) -> bool;
fn clone(value: &Value) -> Value;
fn get_path(value: &Value, path: &str) -> Value;
fn set_path(value: &Value, path: &str, new_value: &Value) -> Value;
fn estimate_tokens(s: &str) -> usize;
fn calculate_savings(slim: &str, json: &str) -> i32;
```

### Go

The Go implementation provides idiomatic Go APIs for SLIM encoding/decoding.

**Installation:**

```bash
go get github.com/your-org/slim-protocol-core/go/slim
```

**Quick Start:**

```go
package main

import (
    "fmt"
    "github.com/your-org/slim-protocol-core/go/slim"
)

func main() {
    // Encode data to SLIM
    data := []interface{}{
        map[string]interface{}{"id": 1, "name": "Mario", "active": true},
        map[string]interface{}{"id": 2, "name": "Luigi", "active": false},
    }

    encoded, _ := slim.Encode(data, slim.DefaultEncodeOptions())
    fmt.Println(encoded)
    // Output: |2|active?,id#,name$|
    //         T,1,Mario
    //         F,2,Luigi

    // Decode SLIM back to data
    decoded, _ := slim.Decode(encoded, slim.DefaultDecodeOptions())
    fmt.Printf("%v\n", decoded)
}
```

**Streaming API Examples:**

```go
package main

import (
    "fmt"
    "github.com/your-org/slim-protocol-core/go/slim"
)

func main() {
    // === Incremental Encoding ===
    // Buffer objects one at a time, encode when ready
    encoder := slim.NewEncoder(slim.DefaultEncodeOptions())

    // Simulate receiving objects from a stream
    for i := 1; i <= 100; i++ {
        encoder.Write(map[string]interface{}{
            "id":        float64(i),
            "value":     float64(i * 10),
            "processed": true,
        })
    }

    fmt.Printf("Buffered %d objects\n", encoder.Size())
    encoded, _ := encoder.End()
    // All 100 objects encoded as a single SLIM table

    // === Incremental Decoding ===
    // Accumulate chunks, decode when complete
    decoder := slim.NewDecoder(slim.DefaultDecodeOptions())

    // Simulate receiving SLIM data in chunks
    decoder.Write("|3|id#,name$|")
    decoder.Write("\n1,Alice")
    decoder.Write("\n2,Bob")
    decoder.Write("\n3,Charlie")

    if decoder.HasPending() {
        data, _ := decoder.End()
        fmt.Printf("Decoded: %v\n", data)
    }

    // === Chunked Encoding ===
    // Split large datasets into smaller encoded chunks
    largeDataset := make([]interface{}, 1000)
    for i := 0; i < 1000; i++ {
        largeDataset[i] = map[string]interface{}{
            "id":   float64(i + 1),
            "data": fmt.Sprintf("item-%d", i+1),
        }
    }

    // Encode in batches of 100
    chunks, _ := slim.EncodeChunked(largeDataset, 100, slim.DefaultEncodeOptions())
    for i, chunk := range chunks {
        fmt.Printf("Chunk %d: %d bytes\n", i, len(chunk))
        // Save or transmit each chunk independently
    }

    // === Iterate Over Decoded Objects ===
    // Get all objects as a slice
    slimData := "|3|id#,name$|\n1,Alice\n2,Bob\n3,Charlie"
    objects, _ := slim.DecodeIter(slimData, slim.DefaultDecodeOptions())
    for _, obj := range objects {
        fmt.Printf("Processing: %v\n", obj)
    }

    // === Channel-based Streaming ===
    // Process objects via Go channels (concurrent-friendly)
    ch := slim.DecodeStreamChan(slimData, slim.DefaultDecodeOptions())
    for obj := range ch {
        fmt.Printf("From channel: %v\n", obj)
    }

    // Or collect all at once
    ch2 := slim.DecodeStreamChan(slimData, slim.DefaultDecodeOptions())
    allObjects := slim.Collect(ch2)
    fmt.Printf("Collected %d objects\n", len(allObjects))
}
```

**Utility Functions Examples:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/your-org/slim-protocol-core/go/slim"
)

func main() {
    // === Deep Equality ===
    // Compare values including nested structures (handles NaN correctly)
    a := map[string]interface{}{
        "users": []interface{}{
            map[string]interface{}{"id": float64(1)},
            map[string]interface{}{"id": float64(2)},
        },
    }
    b := map[string]interface{}{
        "users": []interface{}{
            map[string]interface{}{"id": float64(1)},
            map[string]interface{}{"id": float64(2)},
        },
    }
    fmt.Printf("Equal: %v\n", slim.DeepEqual(a, b)) // true

    // === Deep Clone ===
    // Create independent copies of complex structures
    original := map[string]interface{}{
        "nested": map[string]interface{}{
            "data": []interface{}{float64(1), float64(2), float64(3)},
        },
    }
    cloned := slim.Clone(original)
    // Modifications to cloned won't affect original

    // === Path-based Access ===
    // Navigate nested structures with dot notation
    config := map[string]interface{}{
        "database": map[string]interface{}{
            "host":  "localhost",
            "ports": []interface{}{float64(5432), float64(5433), float64(5434)},
        },
    }

    host := slim.GetPath(config, "database.host")
    fmt.Printf("Host: %v\n", host) // localhost

    firstPort := slim.GetPath(config, "database.ports.0")
    fmt.Printf("First port: %v\n", firstPort) // 5432

    // === Path-based Updates ===
    // Immutably update nested values
    updated := slim.SetPath(config, "database.host", "production.db")
    // Original unchanged, updated has new host

    // Create nested paths that don't exist
    empty := map[string]interface{}{}
    withNested := slim.SetPath(empty, "a.b.c.d", "deep value")
    fmt.Printf("Created: %v\n", withNested)
    // Creates: {"a": {"b": {"c": {"d": "deep value"}}}}

    // === Token Estimation ===
    // Estimate LLM token usage
    text := "Hello, this is a test string for token estimation."
    tokens := slim.EstimateTokens(text)
    fmt.Printf("Estimated %d tokens\n", tokens)

    // === Savings Calculation ===
    // Compare SLIM vs JSON efficiency
    data := []interface{}{
        map[string]interface{}{"id": float64(1), "name": "Alice", "active": true},
        map[string]interface{}{"id": float64(2), "name": "Bob", "active": false},
    }
    slimStr, _ := slim.Encode(data, slim.DefaultEncodeOptions())
    jsonBytes, _ := json.Marshal(data)

    savings := slim.CalculateSavings(slimStr, string(jsonBytes))
    fmt.Printf("SLIM saves %d%% tokens vs JSON\n", savings)
}
```

**API Reference:**

```go
// Core functions
func Encode(data interface{}, options EncodeOptions) (string, error)
func Decode(slim string, options DecodeOptions) (interface{}, error)

// Options
type EncodeOptions struct {
    MaxDepth       int  // Default: 15
    TableThreshold int  // Default: 1
    Pretty         bool // Default: false
}

type DecodeOptions struct {
    Strict bool // Default: false
}

// Schema utilities
func InferSchema(data []interface{}) string
func ParseSchema(schema string) []ColumnDef
func ValidateSchema(data interface{}, schema string) ValidationResult

// Streaming API
type SlimEncoder struct { ... }
func NewEncoder(options EncodeOptions) *SlimEncoder
func (e *SlimEncoder) Write(obj interface{})
func (e *SlimEncoder) End() (string, error)
func (e *SlimEncoder) Size() int

type SlimDecoder struct { ... }
func NewDecoder(options DecodeOptions) *SlimDecoder
func (d *SlimDecoder) Write(chunk string)
func (d *SlimDecoder) End() (interface{}, error)
func (d *SlimDecoder) HasPending() bool

func EncodeChunked(objects []interface{}, chunkSize int, options EncodeOptions) ([]string, error)
func DecodeIter(slim string, options DecodeOptions) ([]map[string]interface{}, error)
func DecodeStreamChan(slim string, options DecodeOptions) <-chan interface{}
func Collect(ch <-chan interface{}) []interface{}

// Utility functions
func DeepEqual(a, b interface{}) bool
func Clone(value interface{}) interface{}
func GetPath(value interface{}, path string) interface{}
func SetPath(value interface{}, path string, newValue interface{}) interface{}
func EstimateTokens(s string) int
func CalculateSavings(slim, json string) int
```

## Roadmap

This is Phase 1 of the SLIM ecosystem:

1. **slim-core** (this package) - Core serialization library
2. **slim-db** - Embedded database with native SLIM storage
3. **pg-slim** - PostgreSQL extension for SLIM data type

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
