package pyjson

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

const maxDepth = 1000 // CPython's default recursion limit stops deeper documents too

// Decode parses data like json.loads: objects keep insertion order (a repeated key keeps its
// first place and its last value), numbers without fraction or exponent are integers
// (int64, or *big.Int when larger), others float64 (overflow gives ±Inf), and NaN, Infinity
// and -Infinity are accepted. Control characters inside strings are rejected (strict mode).
func Decode(data []byte) (any, error) {
	p := &parser{data: data}
	p.space()
	value, err := p.value(0)
	if err != nil {
		return nil, err
	}
	p.space()
	if p.pos != len(p.data) {
		return nil, p.fail("Extra data")
	}
	return value, nil
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) fail(message string) error {
	return fmt.Errorf("pyjson: %s at char %d", message, p.pos)
}

func (p *parser) space() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *parser) literal(word string) bool {
	if len(p.data)-p.pos >= len(word) && string(p.data[p.pos:p.pos+len(word)]) == word {
		p.pos += len(word)
		return true
	}
	return false
}

func (p *parser) value(depth int) (any, error) {
	if depth > maxDepth {
		return nil, p.fail("maximum recursion depth exceeded")
	}
	if p.pos >= len(p.data) {
		return nil, p.fail("Expecting value")
	}
	switch c := p.data[p.pos]; {
	case c == '{':
		return p.object(depth)
	case c == '[':
		return p.array(depth)
	case c == '"':
		return p.string()
	case p.literal("null"):
		return nil, nil
	case p.literal("true"):
		return true, nil
	case p.literal("false"):
		return false, nil
	case p.literal("NaN"):
		return math.NaN(), nil
	case p.literal("Infinity"):
		return math.Inf(1), nil
	case p.literal("-Infinity"):
		return math.Inf(-1), nil
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	}
	return nil, p.fail("Expecting value")
}

func (p *parser) object(depth int) (any, error) {
	p.pos++
	object := NewObject()
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		return object, nil
	}
	for {
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return nil, p.fail("Expecting property name enclosed in double quotes")
		}
		key, err := p.string()
		if err != nil {
			return nil, err
		}
		p.space()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, p.fail("Expecting ':' delimiter")
		}
		p.pos++
		p.space()
		item, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		object.Set(key, item)
		p.space()
		if p.pos < len(p.data) && p.data[p.pos] == ',' {
			p.pos++
			p.space()
			continue
		}
		if p.pos < len(p.data) && p.data[p.pos] == '}' {
			p.pos++
			return object, nil
		}
		return nil, p.fail("Expecting ',' delimiter")
	}
}

func (p *parser) array(depth int) (any, error) {
	p.pos++
	items := []any{}
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		return items, nil
	}
	for {
		item, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		p.space()
		if p.pos < len(p.data) && p.data[p.pos] == ',' {
			p.pos++
			p.space()
			continue
		}
		if p.pos < len(p.data) && p.data[p.pos] == ']' {
			p.pos++
			return items, nil
		}
		return nil, p.fail("Expecting ',' delimiter")
	}
}

func (p *parser) number() (any, error) {
	start := p.pos
	if p.data[p.pos] == '-' {
		p.pos++
	}
	if p.pos < len(p.data) && p.data[p.pos] == '0' {
		p.pos++
	} else if !p.digits() {
		p.pos = start
		return nil, p.fail("Expecting value")
	}
	integer := true
	if p.pos+1 < len(p.data) && p.data[p.pos] == '.' && isDigit(p.data[p.pos+1]) {
		p.pos++
		p.digits()
		integer = false
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		mark := p.pos
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.digits() {
			integer = false
		} else {
			p.pos = mark // like Python's scanner: the number ends before a bare "e"
		}
	}
	text := string(p.data[start:p.pos])
	if integer {
		if small, err := strconv.ParseInt(text, 10, 64); err == nil {
			return small, nil
		}
		large, _ := new(big.Int).SetString(text, 10)
		return large, nil
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return nil, p.fail("Invalid number")
	}
	return f, nil
}

func (p *parser) digits() bool {
	start := p.pos
	for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
		p.pos++
	}
	return p.pos > start
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
