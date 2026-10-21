package pyval

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// DumpsSorted is json.dumps(v, sort_keys=True): default ", " and ": " separators,
// ensure_ascii, NaN and Infinity literals.
func DumpsSorted(v any) ([]byte, error) {
	return appendStyled(nil, v, true, -1, 0)
}

// DumpsIndent is json.dumps(v, indent=n): insertion order, "," at line ends and ": ".
func DumpsIndent(v any, n int) ([]byte, error) {
	return appendStyled(nil, v, false, n, 0)
}

func appendStyled(dst []byte, v any, sorted bool, indent, depth int) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return append(dst, "null"...), nil
	case bool:
		return strconv.AppendBool(dst, x), nil
	case int:
		return strconv.AppendInt(dst, int64(x), 10), nil
	case int64:
		return strconv.AppendInt(dst, x, 10), nil
	case *big.Int:
		return x.Append(dst, 10), nil
	case float64:
		return pyjson.AppendFloat(dst, x), nil
	case string:
		return pyjson.AppendString(dst, x), nil
	case []any:
		if len(x) == 0 {
			return append(dst, "[]"...), nil
		}
		dst = append(dst, '[')
		for index, item := range x {
			dst = separator(dst, index, indent, depth+1)
			var err error
			if dst, err = appendStyled(dst, item, sorted, indent, depth+1); err != nil {
				return nil, err
			}
		}
		return append(closing(dst, indent, depth), ']'), nil
	case *pyjson.Object:
		if x.Len() == 0 {
			return append(dst, "{}"...), nil
		}
		keys := x.Keys()
		if sorted {
			sort.Strings(keys)
		}
		dst = append(dst, '{')
		for index, key := range keys {
			dst = separator(dst, index, indent, depth+1)
			dst = append(pyjson.AppendString(dst, key), ": "...)
			value, _ := x.Get(key)
			var err error
			if dst, err = appendStyled(dst, value, sorted, indent, depth+1); err != nil {
				return nil, err
			}
		}
		return append(closing(dst, indent, depth), '}'), nil
	}
	return nil, fmt.Errorf("pyval: cannot encode %T", v)
}

func separator(dst []byte, index, indent, depth int) []byte {
	if indent < 0 {
		if index > 0 {
			dst = append(dst, ", "...)
		}
		return dst
	}
	if index > 0 {
		dst = append(dst, ',')
	}
	return append(append(dst, '\n'), strings.Repeat(" ", indent*depth)...)
}

func closing(dst []byte, indent, depth int) []byte {
	if indent < 0 {
		return dst
	}
	return append(append(dst, '\n'), strings.Repeat(" ", indent*depth)...)
}
