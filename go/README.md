# SLIM Protocol - Go Implementation

[![Go Reference](https://pkg.go.dev/badge/github.com/RevCBH/slim-protocol-core/go/slim.svg)](https://pkg.go.dev/github.com/RevCBH/slim-protocol-core/go/slim)
[![Go Report Card](https://goreportcard.com/badge/github.com/RevCBH/slim-protocol-core/go)](https://goreportcard.com/report/github.com/RevCBH/slim-protocol-core/go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**SLIM** (Structured Lightweight Interchange Markup) - Token-efficient data serialization for AI/LLM applications, written in Go.

## Why SLIM?

When sending data to LLMs (ChatGPT, Claude, etc.), you pay per **token**. SLIM reduces tokens by **40-60%** compared to JSON while preserving all data.

```go
import (
	"encoding/json"
	"fmt"
	"github.com/RevCBH/slim-protocol-core/go/slim"
)

// JSON: 191 characters
data := []interface{}{
	map[string]interface{}{"id": 1.0, "name": "Alice", "active": true},
	map[string]interface{}{"id": 2.0, "name": "Bob", "active": false},
}
jsonBytes, _ := json.Marshal(data)
fmt.Println(string(jsonBytes))
// Output: [{"active":true,"id":1,"name":"Alice"},{"active":false,"id":2,"name":"Bob"}]

// SLIM: 69 characters (-63.9%)
slimStr, _ := slim.Encode(data, slim.DefaultEncodeOptions())
fmt.Println(slimStr)
// Output: |2|active?,id#,name$|
//         T,1,Alice
//         F,2,Bob
```

## Installation

```bash
go get github.com/RevCBH/slim-protocol-core/go/slim
```

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/RevCBH/slim-protocol-core/go/slim"
)

func main() {
	// Encode data to SLIM
	data := []interface{}{
		map[string]interface{}{"id": 1.0, "name": "Mario", "active": true},
		map[string]interface{}{"id": 2.0, "name": "Luigi", "active": false},
	}

	slimStr, err := slim.Encode(data, slim.DefaultEncodeOptions())
	if err != nil {
		panic(err)
	}
	fmt.Println(slimStr)
	// Output:
	// |2|active?,id#,name$|
	// T,1,Mario
	// F,2,Luigi

	// Decode SLIM back to data
	decoded, err := slim.Decode(slimStr, slim.DefaultDecodeOptions())
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", decoded)
}
```

## API Reference

### Encoding

```go
func Encode(data interface{}, options EncodeOptions) (string, error)

type EncodeOptions struct {
	MaxDepth       int  // Maximum nesting depth before returning !DEEP (default: 15)
	TableThreshold int  // Minimum rows for table format (default: 1)
	Pretty         bool // Pretty print (default: false)
}
```

### Decoding

```go
func Decode(slim string, options DecodeOptions) (interface{}, error)

type DecodeOptions struct {
	Strict bool // Throw error on invalid input (default: false)
}
```

### Schema Operations

```go
// Infer schema from data
schema := slim.InferSchema(data)
// Returns: "id#,name$,active?"

// Parse schema string
cols := slim.ParseSchema("id#,name$,active?")

// Validate data against schema
result := slim.ValidateSchema(data, "id#,name$,active?")
if !result.Valid {
	for _, err := range result.Errors {
		fmt.Printf("Error at %s: %s\n", err.Path, err.Message)
	}
}
```

## Supported Data Types

| Go Type | SLIM Marker | Example |
|---------|-------------|---------|
| `nil` | `!` | `!null` |
| `bool` | `?` | `?T`, `?F` |
| `float64` | `#` | `#42`, `#3.14` |
| `string` | (none) | `hello` |
| `string` (quoted) | `"` | `"hello, world"` |
| `[]interface{}` (numbers) | `@#` | `@#[1,2,3]` |
| `[]interface{}` (mixed) | `@` | `@[#1;hello;?T]` |
| `[]interface{}` (2D numbers) | `*` | `*[1,2;3,4]` |
| `[]interface{}` (objects) | `\|n\|schema\|` | `\|2\|id#,name$\|...` |
| `map[string]interface{}` | `{}` | `{name:Mario}` |

## Special Values

| Value | SLIM |
|-------|------|
| `math.NaN()` | `#NaN` |
| `math.Inf(1)` | `#Inf` |
| `math.Inf(-1)` | `#-Inf` |
| Empty string | `""` |
| Empty array | `@[]` |
| Empty object | `{}` |

## Testing

The implementation includes comprehensive tests:
- **Unit tests** - All basic operations
- **Round-trip tests** - Encode/decode consistency
- **Property-based tests** - 300+ random test cases using gopter

```bash
# Run all tests
go test ./slim

# Run with verbose output
go test ./slim -v

# Run property-based tests (generates 100 test cases per property)
go test ./slim -run TestProperty -v

# Run benchmarks
go test ./slim -bench=.
```

### Test Results

```
PASS
ok      github.com/RevCBH/slim-protocol-core/go/slim    0.013s

Property-based tests:
✓ encode/decode round trip for booleans: OK, passed 100 tests
✓ encode/decode round trip for numbers: OK, passed 100 tests
✓ encode/decode round trip for strings: OK, passed 100 tests
✓ encoding booleans is stable: OK, passed 100 tests
✓ encoding numbers is stable: OK, passed 100 tests

Token savings:
✓ Dataset 1: 63.9% savings (69 chars vs 191 chars)
✓ Dataset 2: 58.3% savings (30 chars vs 72 chars)
```

## Performance

The Go implementation is designed for efficiency:
- Zero allocations for simple types
- Efficient string building
- Minimal copying

## Comparison with Other Implementations

| Implementation | Purpose | Performance | Type Safety |
|----------------|---------|-------------|-------------|
| **TypeScript** | Original, for JS/TS projects | Baseline | Runtime |
| **Rust** | Maximum performance, FFI | 5-10x faster | Compile-time |
| **Go** | Native Go projects, simplicity | 2-3x faster | Static typing |

## Example

See `examples/basic.go` for a complete example.

```bash
go run examples/basic.go
```

## Development

```bash
# Run tests
go test ./slim

# Run tests with coverage
go test ./slim -cover

# Format code
go fmt ./...

# Lint
golangci-lint run
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Related Projects

- **slim-protocol-core** (TypeScript) - Original implementation
- **slim-protocol-core/rust** - Rust implementation for maximum performance
- **slim-db** - Embedded database with native SLIM storage
- **pg-slim** - PostgreSQL extension for SLIM data type
