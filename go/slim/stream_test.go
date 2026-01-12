package slim

import (
	"testing"
)

func TestSlimEncoder(t *testing.T) {
	t.Run("empty encoder returns @[]", func(t *testing.T) {
		encoder := NewEncoder(DefaultEncodeOptions())
		result, err := encoder.End()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "@[]" {
			t.Errorf("expected @[], got %s", result)
		}
	})

	t.Run("encoder buffers and encodes objects", func(t *testing.T) {
		encoder := NewEncoder(DefaultEncodeOptions())

		encoder.Write(map[string]interface{}{"id": float64(1), "name": "Mario"})
		encoder.Write(map[string]interface{}{"id": float64(2), "name": "Luigi"})

		if encoder.Size() != 2 {
			t.Errorf("expected size 2, got %d", encoder.Size())
		}

		result, err := encoder.End()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Table format should start with |2|
		if len(result) < 3 || result[:3] != "|2|" {
			t.Errorf("expected table format starting with |2|, got %s", result)
		}
	})

	t.Run("clear removes buffered objects", func(t *testing.T) {
		encoder := NewEncoder(DefaultEncodeOptions())
		encoder.Write(map[string]interface{}{"id": float64(1)})
		encoder.Clear()

		if encoder.Size() != 0 {
			t.Errorf("expected size 0 after clear, got %d", encoder.Size())
		}

		result, err := encoder.End()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "@[]" {
			t.Errorf("expected @[], got %s", result)
		}
	})
}

func TestSlimDecoder(t *testing.T) {
	t.Run("empty decoder returns nil", func(t *testing.T) {
		decoder := NewDecoder(DefaultDecodeOptions())

		if decoder.HasPending() {
			t.Error("expected no pending data")
		}

		result, err := decoder.End()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("decoder accumulates chunks", func(t *testing.T) {
		decoder := NewDecoder(DefaultDecodeOptions())

		decoder.Write("|2|id#,name$|")
		decoder.Write("\n1,Mario\n2,Luigi")

		if !decoder.HasPending() {
			t.Error("expected pending data")
		}

		result, err := decoder.End()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		arr, ok := result.([]interface{})
		if !ok {
			t.Fatalf("expected array, got %T", result)
		}
		if len(arr) != 2 {
			t.Errorf("expected 2 items, got %d", len(arr))
		}
	})

	t.Run("clear removes buffered data", func(t *testing.T) {
		decoder := NewDecoder(DefaultDecodeOptions())
		decoder.Write("|2|id#|\n1\n2")
		decoder.Clear()

		if decoder.HasPending() {
			t.Error("expected no pending data after clear")
		}
	})
}

func TestEncodeChunked(t *testing.T) {
	objects := []interface{}{
		map[string]interface{}{"id": float64(1)},
		map[string]interface{}{"id": float64(2)},
		map[string]interface{}{"id": float64(3)},
		map[string]interface{}{"id": float64(4)},
	}

	t.Run("chunks objects correctly", func(t *testing.T) {
		chunks, err := EncodeChunked(objects, 2, DefaultEncodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(chunks) != 2 {
			t.Errorf("expected 2 chunks, got %d", len(chunks))
		}

		// Each chunk should be a table with 2 rows
		for i, chunk := range chunks {
			if len(chunk) < 3 || chunk[:3] != "|2|" {
				t.Errorf("chunk %d should start with |2|, got %s", i, chunk)
			}
		}
	})

	t.Run("handles uneven chunks", func(t *testing.T) {
		chunks, err := EncodeChunked(objects, 3, DefaultEncodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(chunks) != 2 {
			t.Errorf("expected 2 chunks, got %d", len(chunks))
		}

		// First chunk should have 3 items, second should have 1
		if len(chunks[0]) < 3 || chunks[0][:3] != "|3|" {
			t.Errorf("first chunk should start with |3|, got %s", chunks[0])
		}
		if len(chunks[1]) < 3 || chunks[1][:3] != "|1|" {
			t.Errorf("second chunk should start with |1|, got %s", chunks[1])
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		chunks, err := EncodeChunked([]interface{}{}, 2, DefaultEncodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(chunks) != 0 {
			t.Errorf("expected 0 chunks for empty input, got %d", len(chunks))
		}
	})
}

func TestDecodeIter(t *testing.T) {
	t.Run("decodes table to objects", func(t *testing.T) {
		slim := "|2|id#,name$|\n1,Mario\n2,Luigi"

		objects, err := DecodeIter(slim, DefaultDecodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(objects) != 2 {
			t.Errorf("expected 2 objects, got %d", len(objects))
		}

		if objects[0]["name"] != "Mario" {
			t.Errorf("expected Mario, got %v", objects[0]["name"])
		}
	})

	t.Run("decodes single object", func(t *testing.T) {
		slim := "{name:Mario,age:#30}"

		objects, err := DecodeIter(slim, DefaultDecodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(objects) != 1 {
			t.Errorf("expected 1 object, got %d", len(objects))
		}
	})

	t.Run("filters non-objects", func(t *testing.T) {
		// Number array should return empty
		slim := "@#[1,2,3]"

		objects, err := DecodeIter(slim, DefaultDecodeOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(objects) != 0 {
			t.Errorf("expected 0 objects for number array, got %d", len(objects))
		}
	})
}

func TestDecodeStreamChan(t *testing.T) {
	slim := "|2|id#,name$|\n1,Mario\n2,Luigi"

	ch := DecodeStreamChan(slim, DefaultDecodeOptions())
	objects := Collect(ch)

	if len(objects) != 2 {
		t.Errorf("expected 2 objects, got %d", len(objects))
	}

	// Verify objects
	if obj, ok := objects[0].(map[string]interface{}); ok {
		if obj["name"] != "Mario" {
			t.Errorf("expected Mario, got %v", obj["name"])
		}
	} else {
		t.Error("expected map[string]interface{}")
	}
}

func TestEncoderDecoderRoundtrip(t *testing.T) {
	// Encode some objects
	encoder := NewEncoder(DefaultEncodeOptions())
	encoder.Write(map[string]interface{}{"id": float64(1), "name": "Mario", "active": true})
	encoder.Write(map[string]interface{}{"id": float64(2), "name": "Luigi", "active": false})

	encoded, err := encoder.End()
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	// Decode them back
	decoder := NewDecoder(DefaultDecodeOptions())
	decoder.Write(encoded)

	decoded, err := decoder.End()
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	arr, ok := decoded.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", decoded)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 items, got %d", len(arr))
	}
}
