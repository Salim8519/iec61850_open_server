package pyjson

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
)

type style struct {
	sortKeys  bool
	itemSep   string
	keySep    string
	finalLine bool
}

var (
	// Flask 3.1 jsonify outside debug mode: sorted keys, compact separators, trailing newline.
	flaskStyle = style{sortKeys: true, itemSep: ",", keySep: ":", finalLine: true}
	// json.dumps defaults: insertion order, ", " and ": " (events, run logs, project files).
	dumpsStyle = style{itemSep: ", ", keySep: ": "}
	// json.dumps(separators=(",", ":")): insertion order, compact.
	compactStyle = style{itemSep: ",", keySep: ":"}
)

// Flask returns the body Flask's jsonify writes for value.
func Flask(value any) ([]byte, error) { return encodeWith(value, flaskStyle) }

// Dumps returns json.dumps(value) with default arguments.
func Dumps(value any) ([]byte, error) { return encodeWith(value, dumpsStyle) }

// Compact returns json.dumps(value, separators=(",", ":")).
func Compact(value any) ([]byte, error) { return encodeWith(value, compactStyle) }

func encodeWith(value any, s style) ([]byte, error) {
	out, err := appendValue(nil, value, s)
	if err != nil {
		return nil, err
	}
	if s.finalLine {
		out = append(out, '\n')
	}
	return out, nil
}

func appendValue(dst []byte, value any, s style) ([]byte, error) {
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
		return AppendFloat(dst, v), nil
	case string:
		return AppendString(dst, v), nil
	case []any:
		return appendArray(dst, v, s)
	case *Object:
		return appendObject(dst, v, s)
	}
	return nil, fmt.Errorf("pyjson: cannot encode %T", value)
}

func appendArray(dst []byte, items []any, s style) ([]byte, error) {
	dst = append(dst, '[')
	for index, item := range items {
		if index > 0 {
			dst = append(dst, s.itemSep...)
		}
		var err error
		if dst, err = appendValue(dst, item, s); err != nil {
			return nil, err
		}
	}
	return append(dst, ']'), nil
}

func appendObject(dst []byte, object *Object, s style) ([]byte, error) {
	entries := object.entries
	if s.sortKeys {
		// Python sorts str keys by code point; UTF-8 byte order is the same order.
		entries = append([]entry(nil), entries...)
		sort.Slice(entries, func(a, b int) bool { return entries[a].key < entries[b].key })
	}
	dst = append(dst, '{')
	for index := range entries {
		if index > 0 {
			dst = append(dst, s.itemSep...)
		}
		dst = AppendString(dst, entries[index].key)
		dst = append(dst, s.keySep...)
		var err error
		if dst, err = appendValue(dst, entries[index].value, s); err != nil {
			return nil, err
		}
	}
	return append(dst, '}'), nil
}
