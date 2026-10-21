package native

import (
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// truthy is Python's bool(value) for JSON values.
func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case int64:
		return typed != 0
	case int:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case *pyjson.Object:
		return typed.Len() > 0
	case *big.Int:
		return typed.Sign() != 0
	}
	return true
}

// pyStr is Python's str() for the scalar JSON values an error field can hold.
func pyStr(value any) string {
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
	case float64:
		return pyjson.Repr(typed)
	case *big.Int:
		return typed.String()
	}
	encoded, _ := pyjson.Dumps(value)
	return string(encoded)
}

// decodeReplace is bytes.decode("utf-8", "replace").
func decodeReplace(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	return strings.ToValidUTF8(string(raw), "�")
}

// pyStrip is str.strip() with Python's whitespace set (str.isspace).
func pyStrip(text string) string {
	return strings.TrimFunc(text, isPySpace)
}

func isPySpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, 0x85, 0xa0, 0x1680,
		0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}
