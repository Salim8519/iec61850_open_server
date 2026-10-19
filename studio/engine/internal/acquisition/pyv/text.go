package pyv

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// PyStr is Python's str(value), as an f-string writes it.
func PyStr(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return Repr(value)
}

// Repr is Python's repr(value) for pyjson values.
func Repr(value any) string {
	switch v := value.(type) {
	case nil:
		return "None"
	case bool:
		if v {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case *big.Int:
		return v.String()
	case float64:
		return floatRepr(v)
	case string:
		return stringRepr(v)
	case []any:
		parts := make([]string, len(v))
		for index, item := range v {
			parts[index] = Repr(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, v.Len())
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			parts = append(parts, stringRepr(key)+": "+Repr(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(value)
}

func floatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	return pyjson.Repr(f)
}

// stringRepr quotes like CPython: single quotes unless the text has ' and no ".
func stringRepr(text string) string {
	quote := byte('\'')
	if strings.Contains(text, "'") && !strings.Contains(text, "\"") {
		quote = '"'
	}
	var out strings.Builder
	out.WriteByte(quote)
	for _, r := range text {
		switch {
		case r == rune(quote) || r == '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&out, `\x%02x`, r)
		case r < 0x7f || strconv.IsPrint(r):
			out.WriteRune(r)
		case r <= 0xff:
			fmt.Fprintf(&out, `\x%02x`, r)
		case r <= 0xffff:
			fmt.Fprintf(&out, `\u%04x`, r)
		default:
			fmt.Fprintf(&out, `\U%08x`, r)
		}
	}
	out.WriteByte(quote)
	return out.String()
}

// Format04X is Python's f"{value:04X}" for an integer.
func Format04X(value any) string {
	if number, ok := Int64(value); ok {
		if number < 0 {
			return "-" + fmt.Sprintf("%04X", -number)
		}
		return fmt.Sprintf("%04X", number)
	}
	if b, ok := value.(bool); ok {
		if b {
			return "0001"
		}
		return "0000"
	}
	return PyStr(value)
}
