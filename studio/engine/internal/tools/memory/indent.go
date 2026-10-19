package memory

import (
	"strings"

	"studio-engine/internal/pyjson"
)

// Indent is json.dumps(value, indent=2): insertion order, "," at line ends, ": " between
// key and value, "{}"/"[]" for empty containers, no final newline. Scalars use pyjson.
func Indent(value any) ([]byte, error) {
	return appendIndented(nil, value, 0)
}

func appendIndented(dst []byte, value any, depth int) ([]byte, error) {
	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			return append(dst, "[]"...), nil
		}
		dst = append(dst, '[')
		for index, item := range v {
			if index > 0 {
				dst = append(dst, ',')
			}
			dst = newline(dst, depth+1)
			var err error
			if dst, err = appendIndented(dst, item, depth+1); err != nil {
				return nil, err
			}
		}
		return append(newline(dst, depth), ']'), nil
	case *pyjson.Object:
		if v.Len() == 0 {
			return append(dst, "{}"...), nil
		}
		dst = append(dst, '{')
		for index, key := range v.Keys() {
			if index > 0 {
				dst = append(dst, ',')
			}
			dst = newline(dst, depth+1)
			dst = pyjson.AppendString(dst, key)
			dst = append(dst, ": "...)
			item, _ := v.Get(key)
			var err error
			if dst, err = appendIndented(dst, item, depth+1); err != nil {
				return nil, err
			}
		}
		return append(newline(dst, depth), '}'), nil
	}
	scalar, err := pyjson.Compact(value)
	if err != nil {
		return nil, err
	}
	return append(dst, scalar...), nil
}

func newline(dst []byte, depth int) []byte {
	dst = append(dst, '\n')
	return append(dst, strings.Repeat("  ", depth)...)
}

// crlf is what Path.write_text does on Windows: every "\n" becomes "\r\n".
func crlf(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\n", "\r\n"))
}
