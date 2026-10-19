package pytext

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// DumpsIndent returns json.dumps(value, indent=2): insertion order, ensure_ascii, "," at
// line ends, ": " between key and value, "[]" and "{}" for empty containers, no final
// newline. pyjson has no indented writer, so this one builds on its string and float rules.
func DumpsIndent(value any) ([]byte, error) {
	return appendIndented(nil, value, 0)
}

func appendIndented(dst []byte, value any, level int) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return append(dst, "null"...), nil
	case bool:
		return strconv.AppendBool(dst, v), nil
	case int:
		return strconv.AppendInt(dst, int64(v), 10), nil
	case int64:
		return strconv.AppendInt(dst, v, 10), nil
	case *big.Int:
		return v.Append(dst, 10), nil
	case float64:
		return pyjson.AppendFloat(dst, v), nil
	case string:
		return pyjson.AppendString(dst, v), nil
	case []any:
		return appendList(dst, v, level)
	case *pyjson.Object:
		return appendDict(dst, v, level)
	}
	return nil, fmt.Errorf("pytext: cannot encode %T", value)
}

func newline(dst []byte, level int) []byte {
	dst = append(dst, '\n')
	return append(dst, strings.Repeat("  ", level)...)
}

func appendList(dst []byte, items []any, level int) ([]byte, error) {
	if len(items) == 0 {
		return append(dst, "[]"...), nil
	}
	dst = append(dst, '[')
	for index, item := range items {
		if index > 0 {
			dst = append(dst, ',')
		}
		dst = newline(dst, level+1)
		var err error
		if dst, err = appendIndented(dst, item, level+1); err != nil {
			return nil, err
		}
	}
	return append(newline(dst, level), ']'), nil
}

func appendDict(dst []byte, object *pyjson.Object, level int) ([]byte, error) {
	if object.Len() == 0 {
		return append(dst, "{}"...), nil
	}
	dst = append(dst, '{')
	for index, key := range object.Keys() {
		if index > 0 {
			dst = append(dst, ',')
		}
		dst = newline(dst, level+1)
		dst = append(pyjson.AppendString(dst, key), ": "...)
		item, _ := object.Get(key)
		var err error
		if dst, err = appendIndented(dst, item, level+1); err != nil {
			return nil, err
		}
	}
	return append(newline(dst, level), '}'), nil
}
