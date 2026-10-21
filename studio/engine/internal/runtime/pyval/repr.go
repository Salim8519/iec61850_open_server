package pyval

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Repr is str() of a value: str of strings, repr of floats, True/False/None, and Python's
// repr form for lists and dicts.
func Repr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case *big.Int:
		return typed.String()
	case float64:
		switch {
		case math.IsNaN(typed):
			return "nan"
		case math.IsInf(typed, 1):
			return "inf"
		case math.IsInf(typed, -1):
			return "-inf"
		}
		return pyjson.Repr(typed)
	case []any:
		parts := make([]string, len(typed))
		for i, item := range typed {
			parts[i] = inner(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, typed.Len())
		for _, key := range typed.Keys() {
			item, _ := typed.Get(key)
			parts = append(parts, inner(key)+": "+inner(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return ""
}

func inner(value any) string {
	if text, ok := value.(string); ok {
		return quote(text)
	}
	return Repr(value)
}

// quote is repr() of a str: single quotes unless the text has ' and no ", escapes for
// backslash, the quote, \n \r \t and other control characters.
func quote(text string) string {
	mark := byte('\'')
	if strings.Contains(text, "'") && !strings.Contains(text, `"`) {
		mark = '"'
	}
	var out strings.Builder
	out.WriteByte(mark)
	for _, r := range text {
		switch {
		case r == '\\':
			out.WriteString(`\\`)
		case r == rune(mark):
			out.WriteByte('\\')
			out.WriteRune(r)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			out.WriteString(fmt.Sprintf(`\x%02x`, r))
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte(mark)
	return out.String()
}

// TypeName is type(value).__name__.
func TypeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case float64:
		return "float"
	case []any:
		return "list"
	case *pyjson.Object:
		return "dict"
	}
	return "int"
}
