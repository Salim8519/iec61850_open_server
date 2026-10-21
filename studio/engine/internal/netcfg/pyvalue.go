package netcfg

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// pyCrash stands for an exception the Python code did not catch (TypeError, AttributeError):
// callers pass it on unchanged instead of turning it into a user message.
type pyCrash struct{ message string }

func (e *pyCrash) Error() string { return e.message }

// pyStr returns Python's str() of a decoded JSON value.
func pyStr(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return pyRepr(value)
}

// pyRepr returns Python's repr() of a decoded JSON value.
func pyRepr(value any) string {
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
			parts[index] = pyRepr(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, v.Len())
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			parts = append(parts, stringRepr(key)+": "+pyRepr(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(value)
}

func floatRepr(f float64) string {
	switch text := string(pyjson.AppendFloat(nil, f)); text {
	case "NaN":
		return "nan"
	case "Infinity":
		return "inf"
	case "-Infinity":
		return "-inf"
	default:
		return text
	}
}

// stringRepr quotes like CPython: single quotes unless the text has ' and no ".
func stringRepr(text string) string {
	quote := byte('\'')
	if strings.Contains(text, "'") && !strings.Contains(text, "\"") {
		quote = '"'
	}
	out := []byte{quote}
	for _, r := range text {
		switch {
		case r == '\\' || r == rune(quote):
			out = append(out, '\\', byte(r))
		case r == '\t':
			out = append(out, `\t`...)
		case r == '\n':
			out = append(out, `\n`...)
		case r == '\r':
			out = append(out, `\r`...)
		case r < 0x20 || r == 0x7f:
			out = append(out, fmt.Sprintf(`\x%02x`, r)...)
		case r < 0x7f || unicode.IsPrint(r):
			out = utf8.AppendRune(out, r)
		case r < 0x100:
			out = append(out, fmt.Sprintf(`\x%02x`, r)...)
		case r < 0x10000:
			out = append(out, fmt.Sprintf(`\u%04x`, r)...)
		default:
			out = append(out, fmt.Sprintf(`\U%08x`, r)...)
		}
	}
	return string(append(out, quote))
}

// truthy is Python's bool() of a decoded JSON value.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case *big.Int:
		return v.Sign() != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case *pyjson.Object:
		return v.Len() > 0
	}
	return value != nil
}

// pyInt returns a JSON integer (never a bool) as *big.Int, like isinstance(value, int).
func pyInt(value any) (*big.Int, bool) {
	switch v := value.(type) {
	case int:
		return big.NewInt(int64(v)), true
	case int64:
		return big.NewInt(v), true
	case *big.Int:
		return v, true
	}
	return nil, false
}

// pyIntIn reports whether value is a JSON integer (not bool) with low <= value <= high.
func pyIntIn(value any, low, high int64) bool {
	number, ok := pyInt(value)
	return ok && number.Cmp(big.NewInt(low)) >= 0 && number.Cmp(big.NewInt(high)) <= 0
}
