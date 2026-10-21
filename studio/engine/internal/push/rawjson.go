package push

import (
	"bytes"

	"studio-engine/internal/pyjson"
)

// The scanner below splits a route body into raw byte ranges without decoding it, so a
// delta row is the exact bytes the GET route wrote. It accepts any valid JSON; malformed
// input makes the callers fall back to a full frame.

func space(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return i
}

// skipString returns the index after the string starting at b[i] == '"'.
func skipString(b []byte, i int) (int, bool) {
	for i++; i < len(b); i++ {
		switch b[i] {
		case '\\':
			i++
		case '"':
			return i + 1, true
		}
	}
	return 0, false
}

// skipValue returns the index after the value starting at b[i] (no leading space).
func skipValue(b []byte, i int) (int, bool) {
	if i >= len(b) {
		return 0, false
	}
	if b[i] == '"' {
		return skipString(b, i)
	}
	if b[i] != '{' && b[i] != '[' {
		start := i
		for i < len(b) && b[i] != ',' && b[i] != '}' && b[i] != ']' && b[i] != ' ' && b[i] != '\n' &&
			b[i] != '\r' && b[i] != '\t' {
			i++
		}
		return i, i > start
	}
	depth := 0
	for ; i < len(b); i++ {
		switch b[i] {
		case '"':
			end, ok := skipString(b, i)
			if !ok {
				return 0, false
			}
			i = end - 1
		case '{', '[':
			depth++
		case '}', ']':
			if depth--; depth == 0 {
				return i + 1, true
			}
		}
	}
	return 0, false
}

// field is one member of a JSON object: the raw key token (with quotes) and raw value.
type field struct {
	key, value []byte
}

// objectFields splits the object b into its members, in order.
func objectFields(b []byte, into []field) ([]field, bool) {
	i := space(b, 0)
	if i >= len(b) || b[i] != '{' {
		return nil, false
	}
	i = space(b, i+1)
	if i < len(b) && b[i] == '}' {
		return into, space(b, i+1) == len(b)
	}
	for i < len(b) {
		if b[i] != '"' {
			return nil, false
		}
		end, ok := skipString(b, i)
		if !ok {
			return nil, false
		}
		key := b[i:end]
		i = space(b, end)
		if i >= len(b) || b[i] != ':' {
			return nil, false
		}
		i = space(b, i+1)
		stop, ok := skipValue(b, i)
		if !ok {
			return nil, false
		}
		into = append(into, field{key: key, value: b[i:stop]})
		i = space(b, stop)
		if i < len(b) && b[i] == ',' {
			i = space(b, i+1)
			continue
		}
		if i < len(b) && b[i] == '}' {
			return into, space(b, i+1) == len(b)
		}
		return nil, false
	}
	return nil, false
}

// arrayItems splits the array b into its raw items.
func arrayItems(b []byte, into [][]byte) ([][]byte, bool) {
	i := space(b, 0)
	if i >= len(b) || b[i] != '[' {
		return nil, false
	}
	i = space(b, i+1)
	if i < len(b) && b[i] == ']' {
		return into, true
	}
	for i < len(b) {
		stop, ok := skipValue(b, i)
		if !ok {
			return nil, false
		}
		into = append(into, b[i:stop])
		i = space(b, stop)
		if i < len(b) && b[i] == ',' {
			i = space(b, i+1)
			continue
		}
		return into, i < len(b) && b[i] == ']'
	}
	return nil, false
}

// memberValue finds the raw value of the top-level member named by the quoted key token
// (for example `"id"`) in the object item, without allocating.
func memberValue(item, quoted []byte) ([]byte, bool) {
	i := space(item, 0)
	if i >= len(item) || item[i] != '{' {
		return nil, false
	}
	i = space(item, i+1)
	for i < len(item) && item[i] == '"' {
		end, ok := skipString(item, i)
		if !ok {
			return nil, false
		}
		match := bytes.Equal(item[i:end], quoted)
		i = space(item, end)
		if i >= len(item) || item[i] != ':' {
			return nil, false
		}
		i = space(item, i+1)
		stop, ok := skipValue(item, i)
		if !ok {
			return nil, false
		}
		if match {
			return item[i:stop], true
		}
		i = space(item, stop)
		if i < len(item) && item[i] == ',' {
			i = space(item, i+1)
		}
	}
	return nil, false
}

// keyText is the row key as JavaScript's String(row[field]) gives it: a string's text, or a
// number's literal. Other kinds (objects, null, true) are not keys.
func keyText(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	if raw[0] == '"' {
		if bytes.IndexByte(raw, '\\') < 0 {
			return string(raw[1 : len(raw)-1]), true
		}
		value, err := pyjson.Decode(raw)
		text, ok := value.(string)
		return text, err == nil && ok
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			if c != '-' {
				return "", false
			}
		}
	}
	return string(raw), true
}

// quote returns the JSON string token for name.
func quote(name string) []byte { return pyjson.AppendString(nil, name) }
