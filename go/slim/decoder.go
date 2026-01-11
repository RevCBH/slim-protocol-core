package slim

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Decode decodes a SLIM string to a Go value
func Decode(slim string, options DecodeOptions) (interface{}, error) {
	parser := newSlimParser(slim)
	return parser.parseValue()
}

type slimParser struct {
	input []rune
	pos   int
}

func newSlimParser(input string) *slimParser {
	return &slimParser{
		input: []rune(input),
		pos:   0,
	}
}

func (p *slimParser) peek(n int) string {
	if p.pos+n > len(p.input) {
		return string(p.input[p.pos:])
	}
	return string(p.input[p.pos : p.pos+n])
}

func (p *slimParser) peekChar() (rune, bool) {
	if p.pos < len(p.input) {
		return p.input[p.pos], true
	}
	return 0, false
}

func (p *slimParser) consume(n int) string {
	s := p.peek(n)
	p.pos += len([]rune(s))
	return s
}

func (p *slimParser) consumeChar() (rune, bool) {
	ch, ok := p.peekChar()
	if ok {
		p.pos++
	}
	return ch, ok
}

func (p *slimParser) match(s string) bool {
	if p.peek(len(s)) == s {
		p.consume(len(s))
		return true
	}
	return false
}

func (p *slimParser) skipWs() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *slimParser) isEnd() bool {
	return p.pos >= len(p.input)
}

func (p *slimParser) parseValue() (interface{}, error) {
	p.skipWs()

	if p.isEnd() {
		return nil, nil
	}

	ch, _ := p.peekChar()

	switch ch {
	case '!':
		p.consumeChar()
		return p.parseNull()
	case '?':
		p.consumeChar()
		ch, _ := p.consumeChar()
		return ch == 'T', nil
	case '#':
		p.consumeChar()
		return p.parseNumber()
	case '@':
		p.consumeChar()
		return p.parseArray()
	case '*':
		p.consumeChar()
		return p.parseMatrix()
	case '{':
		return p.parseObject()
	case '|':
		return p.parseTable()
	case '"':
		return p.parseQuotedString()
	default:
		return p.parseUnquoted(), nil
	}
}

func (p *slimParser) parseNull() (interface{}, error) {
	if p.match("null") || p.match("undef") || p.match("DEEP") {
		return nil, nil
	}
	return nil, nil
}

func (p *slimParser) parseNumber() (interface{}, error) {
	if p.match("NaN") {
		return math.NaN(), nil
	}
	if p.match("-Inf") {
		return math.Inf(-1), nil
	}
	if p.match("Inf") {
		return math.Inf(1), nil
	}

	var numStr string
	for !p.isEnd() {
		ch, _ := p.peekChar()
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
			numStr += string(ch)
			p.consumeChar()
		} else {
			break
		}
	}

	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %s", numStr)
	}
	return f, nil
}

func (p *slimParser) parseQuotedString() (interface{}, error) {
	p.consumeChar() // Opening "
	var val string

	for !p.isEnd() {
		ch, _ := p.peekChar()
		if ch == '"' {
			p.consumeChar()
			if nextCh, ok := p.peekChar(); ok && nextCh == '"' {
				// Escaped quote
				val += "\""
				p.consumeChar()
			} else {
				// End of string
				break
			}
		} else if p.peek(2) == "\\n" {
			p.consume(2)
			val += "\n"
		} else {
			ch, _ := p.consumeChar()
			val += string(ch)
		}
	}

	return val, nil
}

func (p *slimParser) parseUnquoted() string {
	var val string
	for !p.isEnd() {
		ch, _ := p.peekChar()
		if ch == ',' || ch == ';' || ch == '\n' || ch == '|' || ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			break
		}
		val += string(ch)
		p.consumeChar()
	}
	return val
}

func (p *slimParser) parseArray() (interface{}, error) {
	isNumeric := false
	if ch, ok := p.peekChar(); ok && ch == '#' {
		isNumeric = true
		p.consumeChar()
	}

	if !p.match("[") {
		return []interface{}{}, nil
	}

	if ch, ok := p.peekChar(); ok && ch == ']' {
		p.consumeChar()
		return []interface{}{}, nil
	}

	var items []interface{}

	for !p.isEnd() {
		if ch, ok := p.peekChar(); ok && ch == ']' {
			break
		}

		if isNumeric {
			// Parse number directly
			var numStr string
			for !p.isEnd() {
				ch, _ := p.peekChar()
				if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
					numStr += string(ch)
					p.consumeChar()
				} else {
					break
				}
			}
			if numStr != "" {
				if f, err := strconv.ParseFloat(numStr, 64); err == nil {
					items = append(items, f)
				}
			}
		} else {
			val, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			items = append(items, val)
		}

		// Skip separator
		if ch, ok := p.peekChar(); ok && (ch == ',' || ch == ';') {
			p.consumeChar()
		}
	}

	p.match("]")
	return items, nil
}

func (p *slimParser) parseMatrix() (interface{}, error) {
	if !p.match("[") {
		return []interface{}{}, nil
	}

	var rows []interface{}
	var currentRow []interface{}

	for !p.isEnd() {
		if ch, ok := p.peekChar(); ok && ch == ']' {
			break
		}

		if ch, ok := p.peekChar(); ok && ch == ';' {
			p.consumeChar()
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
				currentRow = []interface{}{}
			}
		} else if ch, ok := p.peekChar(); ok && ch == ',' {
			p.consumeChar()
		} else if ch, ok := p.peekChar(); ok && (ch >= '0' && ch <= '9' || ch == '-' || ch == '.') {
			var numStr string
			for !p.isEnd() {
				ch, _ := p.peekChar()
				if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
					numStr += string(ch)
					p.consumeChar()
				} else {
					break
				}
			}
			if f, err := strconv.ParseFloat(numStr, 64); err == nil {
				currentRow = append(currentRow, f)
			}
		} else {
			break
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	p.match("]")
	return rows, nil
}

func (p *slimParser) parseObject() (interface{}, error) {
	p.match("{")
	obj := make(map[string]interface{})

	for !p.isEnd() {
		if ch, ok := p.peekChar(); ok && ch == '}' {
			break
		}

		p.skipWs()

		// Parse key
		var key string
		if ch, ok := p.peekChar(); ok && ch == '"' {
			keyVal, err := p.parseQuotedString()
			if err != nil {
				return nil, err
			}
			key = keyVal.(string)
		} else {
			for !p.isEnd() {
				ch, _ := p.peekChar()
				if ch == ':' || ch == ',' || ch == '{' || ch == '}' {
					break
				}
				key += string(ch)
				p.consumeChar()
			}
		}

		p.match(":")
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		obj[key] = value

		if ch, ok := p.peekChar(); ok && ch == ',' {
			p.consumeChar()
		}
	}

	p.match("}")
	return obj, nil
}

func (p *slimParser) parseTable() (interface{}, error) {
	p.match("|")

	// Parse row count
	var countStr string
	for {
		ch, ok := p.peekChar()
		if !ok || ch == '|' {
			break
		}
		countStr += string(ch)
		p.consumeChar()
	}
	p.match("|")

	// Parse schema
	var schemaStr string
	for {
		ch, ok := p.peekChar()
		if !ok || ch == '|' {
			break
		}
		schemaStr += string(ch)
		p.consumeChar()
	}
	p.match("|")

	// Parse column definitions
	type Column struct {
		name       string
		typeMarker string
	}
	var columns []Column

	for _, col := range strings.Split(schemaStr, ",") {
		// Find where type markers start
		nameEnd := len(col)
		for i, ch := range col {
			if ch == '#' || ch == '?' || ch == '@' || ch == '~' || ch == '$' || ch == '!' {
				nameEnd = i
				break
			}
		}
		name := col[:nameEnd]
		typeMarker := col[nameEnd:]
		columns = append(columns, Column{name, typeMarker})
	}

	var rows []interface{}

	// Parse data rows
	for !p.isEnd() {
		if ch, ok := p.peekChar(); ok && ch == '\n' {
			p.consumeChar()
		}
		if p.isEnd() {
			break
		}
		if ch, ok := p.peekChar(); ok && (ch == '}' || ch == ',' || ch == ']') {
			break
		}

		obj := make(map[string]interface{})

		for i, col := range columns {
			if i > 0 {
				if ch, ok := p.peekChar(); ok && ch == ',' {
					p.consumeChar()
				}
			}

			// Empty cell
			ch, ok := p.peekChar()
			if !ok || ch == ',' || ch == '\n' || ch == '}' {
				if strings.Contains(col.typeMarker, "!") {
					obj[col.name] = nil
				}
				continue
			}

			var val interface{}
			var err error

			if ch == '"' {
				val, err = p.parseQuotedString()
				if err != nil {
					return nil, err
				}
			} else if strings.HasPrefix(col.typeMarker, "#") {
				val, err = p.parseNumber()
				if err != nil {
					val = nil
				}
			} else if strings.HasPrefix(col.typeMarker, "?") {
				ch, _ := p.consumeChar()
				val = ch == 'T'
			} else if strings.HasPrefix(col.typeMarker, "@") {
				var arrStr string
				for !p.isEnd() {
					ch, _ := p.peekChar()
					if ch == ',' || ch == '\n' || ch == '}' {
						break
					}
					arrStr += string(ch)
					p.consumeChar()
				}
				if arrStr == "[]" {
					val = []interface{}{}
				} else {
					elements := strings.Split(arrStr, "+")
					var arr []interface{}
					allNums := true
					for _, elem := range elements {
						if elem == "" {
							continue
						}
						if f, err := strconv.ParseFloat(elem, 64); err == nil {
							arr = append(arr, f)
						} else {
							allNums = false
							arr = append(arr, elem)
						}
					}
					if allNums {
						val = arr
					} else {
						val = arr
					}
				}
			} else if strings.HasPrefix(col.typeMarker, "~") {
				val, err = p.parseValue()
				if err != nil {
					return nil, err
				}
			} else {
				// String
				if ch == '"' {
					val, err = p.parseQuotedString()
					if err != nil {
						return nil, err
					}
				} else {
					var s string
					for !p.isEnd() {
						ch, _ := p.peekChar()
						if ch == ',' || ch == '\n' || ch == '}' {
							break
						}
						s += string(ch)
						p.consumeChar()
					}
					val = s
				}
			}

			// Only add if not empty/null
			switch v := val.(type) {
			case string:
				if v != "" || strings.Contains(col.typeMarker, "!") {
					obj[col.name] = val
				}
			case nil:
				if strings.Contains(col.typeMarker, "!") {
					obj[col.name] = nil
				}
			default:
				obj[col.name] = val
			}
		}

		rows = append(rows, obj)

		if ch, ok := p.peekChar(); ok && ch == '\n' {
			p.consumeChar()
		} else if ch, ok := p.peekChar(); ok && !(ch == '}' || ch == ',' || ch == ']') {
			if !p.isEnd() {
				break
			}
		}
	}

	return rows, nil
}
