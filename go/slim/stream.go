package slim

// SlimEncoder provides incremental encoding of objects.
// Objects are buffered and encoded when End() is called.
type SlimEncoder struct {
	buffer  []interface{}
	options EncodeOptions
}

// NewEncoder creates a new incremental encoder.
func NewEncoder(options EncodeOptions) *SlimEncoder {
	return &SlimEncoder{
		buffer:  make([]interface{}, 0),
		options: options,
	}
}

// Write adds an object to the encoder buffer.
func (e *SlimEncoder) Write(obj interface{}) {
	e.buffer = append(e.buffer, obj)
}

// Size returns the number of buffered objects.
func (e *SlimEncoder) Size() int {
	return len(e.buffer)
}

// End encodes all buffered objects and returns the SLIM string.
func (e *SlimEncoder) End() (string, error) {
	if len(e.buffer) == 0 {
		return "@[]", nil
	}
	return Encode(e.buffer, e.options)
}

// Clear removes all objects from the buffer without encoding.
func (e *SlimEncoder) Clear() {
	e.buffer = make([]interface{}, 0)
}

// SlimDecoder provides incremental decoding of SLIM strings.
// String chunks are accumulated and decoded when End() is called.
type SlimDecoder struct {
	buffer  string
	options DecodeOptions
}

// NewDecoder creates a new incremental decoder.
func NewDecoder(options DecodeOptions) *SlimDecoder {
	return &SlimDecoder{
		buffer:  "",
		options: options,
	}
}

// Write adds a string chunk to the decoder buffer.
func (d *SlimDecoder) Write(chunk string) {
	d.buffer += chunk
}

// HasPending returns true if there is data in the buffer.
func (d *SlimDecoder) HasPending() bool {
	return len(d.buffer) > 0 && len(trimWhitespace(d.buffer)) > 0
}

// End decodes the accumulated buffer and returns the value.
func (d *SlimDecoder) End() (interface{}, error) {
	if !d.HasPending() {
		return nil, nil
	}
	return Decode(d.buffer, d.options)
}

// Clear removes all data from the buffer without decoding.
func (d *SlimDecoder) Clear() {
	d.buffer = ""
}

func trimWhitespace(s string) string {
	result := ""
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			result += string(r)
		}
	}
	return result
}

// EncodeChunked encodes objects in chunks, returning a slice of SLIM strings.
func EncodeChunked(objects []interface{}, chunkSize int, options EncodeOptions) ([]string, error) {
	if chunkSize <= 0 {
		chunkSize = 1
	}

	var chunks []string
	for i := 0; i < len(objects); i += chunkSize {
		end := i + chunkSize
		if end > len(objects) {
			end = len(objects)
		}
		chunk := objects[i:end]
		encoded, err := Encode(chunk, options)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, encoded)
	}
	return chunks, nil
}

// DecodeIter decodes a SLIM string and returns a slice of objects.
// Only objects (not primitives or arrays) are returned.
func DecodeIter(slim string, options DecodeOptions) ([]map[string]interface{}, error) {
	data, err := Decode(slim, options)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if obj, ok := item.(map[string]interface{}); ok {
				result = append(result, obj)
			}
		}
	case map[string]interface{}:
		result = append(result, v)
	}

	return result, nil
}

// DecodeStreamChan decodes a SLIM string and returns a channel that yields objects.
// The channel is closed when all objects have been sent.
func DecodeStreamChan(slim string, options DecodeOptions) <-chan interface{} {
	ch := make(chan interface{})

	go func() {
		defer close(ch)

		data, err := Decode(slim, options)
		if err != nil {
			return
		}

		switch v := data.(type) {
		case []interface{}:
			for _, item := range v {
				if obj, ok := item.(map[string]interface{}); ok {
					ch <- obj
				}
			}
		case map[string]interface{}:
			ch <- v
		}
	}()

	return ch
}

// Collect collects all items from a channel into a slice.
func Collect(ch <-chan interface{}) []interface{} {
	var result []interface{}
	for item := range ch {
		result = append(result, item)
	}
	return result
}
