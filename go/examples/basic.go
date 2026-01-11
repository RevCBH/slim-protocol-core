package main

import (
	"encoding/json"
	"fmt"

	"github.com/RevCBH/slim-protocol-core/go/slim"
)

func main() {
	fmt.Println("=== SLIM Protocol - Go Implementation ===\n")

	// Example 1: Simple object
	fmt.Println("Example 1: Simple object")
	data := map[string]interface{}{
		"name":   "Mario",
		"age":    float64(30),
		"active": true,
	}
	jsonBytes, _ := json.Marshal(data)
	slimStr, _ := slim.Encode(data, slim.DefaultEncodeOptions())
	fmt.Printf("JSON: %s\n", string(jsonBytes))
	fmt.Printf("SLIM: %s\n\n", slimStr)

	// Example 2: Array of objects (table format)
	fmt.Println("Example 2: Array of objects (table format)")
	users := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Mario", "active": true},
		map[string]interface{}{"id": float64(2), "name": "Luigi", "active": false},
		map[string]interface{}{"id": float64(3), "name": "Peach", "active": true},
	}
	jsonBytes, _ = json.Marshal(users)
	slimStr, _ = slim.Encode(users, slim.DefaultEncodeOptions())
	fmt.Printf("JSON: %s\n", string(jsonBytes))
	fmt.Printf("SLIM:\n%s\n\n", slimStr)

	// Example 3: Decode back
	fmt.Println("Example 3: Decode SLIM back to data")
	slimInput := "|2|id#,name$|\n1,Mario\n2,Luigi"
	decoded, _ := slim.Decode(slimInput, slim.DefaultDecodeOptions())
	decodedJSON, _ := json.MarshalIndent(decoded, "", "  ")
	fmt.Printf("SLIM: %s\n", slimInput)
	fmt.Printf("Decoded:\n%s\n\n", string(decodedJSON))

	// Example 4: Token savings
	fmt.Println("Example 4: Token savings comparison")
	data2 := []interface{}{
		map[string]interface{}{"id": float64(1), "score": 95.5, "passed": true},
		map[string]interface{}{"id": float64(2), "score": 87.2, "passed": true},
		map[string]interface{}{"id": float64(3), "score": 62.8, "passed": false},
	}
	jsonStr, _ := json.Marshal(data2)
	slimStr2, _ := slim.Encode(data2, slim.DefaultEncodeOptions())
	fmt.Printf("JSON length: %d chars\n", len(jsonStr))
	fmt.Printf("SLIM length: %d chars\n", len(slimStr2))
	fmt.Printf("Savings:     %.1f%%\n",
		float64(len(jsonStr)-len(slimStr2))/float64(len(jsonStr))*100)

	// Example 5: Schema inference
	fmt.Println("\nExample 5: Schema inference")
	schema := slim.InferSchema(users)
	fmt.Printf("Inferred schema: %s\n", schema)

	// Validate data
	result := slim.ValidateSchema(users, schema)
	if result.Valid {
		fmt.Println("✓ Data is valid according to schema")
	} else {
		fmt.Println("✗ Validation errors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s: %s\n", err.Path, err.Message)
		}
	}
}
