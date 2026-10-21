package httpapi

import (
	"encoding/binary"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf16"

	"studio-engine/internal/pycompat"
	"studio-engine/internal/pyjson"
)

// MaxContentLength is Flask's MAX_CONTENT_LENGTH (100 MiB).
const MaxContentLength = 100 * 1024 * 1024

// ReadBody is request.get_data(): the whole body, or 413 above MaxContentLength.
func ReadBody(r *http.Request) ([]byte, error) {
	if r.ContentLength > MaxContentLength {
		return nil, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxContentLength+1))
	if err != nil {
		return nil, ErrBadRequest
	}
	if len(data) > MaxContentLength {
		return nil, ErrTooLarge
	}
	return data, nil
}

// ReadJSON is request.get_json(silent=silent): without a JSON Content-Type it is None
// (silent) or 415; a body that is not JSON is None (silent) or 400 with Werkzeug's generic
// text; a body above MaxContentLength is 413 either way. None is returned as nil.
func ReadJSON(r *http.Request, silent bool) (any, error) {
	if !isJSON(r.Header.Get("Content-Type")) {
		if silent {
			return nil, nil
		}
		return nil, ErrUnsupportedJSON
	}
	data, err := ReadBody(r)
	if err != nil {
		return nil, err
	}
	value, err := decodeJSONBytes(data)
	if err != nil {
		if silent {
			return nil, nil
		}
		return nil, ErrBadRequest
	}
	return value, nil
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType, _, _ = strings.Cut(contentType, ";")
		mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	}
	return mediaType == "application/json" ||
		(strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"))
}

// decodeJSONBytes is json.loads(bytes): the encoding is detected as json.detect_encoding
// does (UTF-8 with or without BOM, UTF-16 and UTF-32 by BOM or zero bytes).
func decodeJSONBytes(data []byte) (any, error) {
	text, err := detectAndDecode(data)
	if err != nil {
		return nil, err
	}
	return pyjson.Decode([]byte(text))
}

func detectAndDecode(b []byte) (string, error) {
	switch {
	case hasPrefix(b, "\x00\x00\xfe\xff"):
		return decodeUTF32(b[4:], binary.BigEndian), nil
	case hasPrefix(b, "\xff\xfe\x00\x00"):
		return decodeUTF32(b[4:], binary.LittleEndian), nil
	case hasPrefix(b, "\xfe\xff"):
		return decodeUTF16(b[2:], binary.BigEndian), nil
	case hasPrefix(b, "\xff\xfe"):
		return decodeUTF16(b[2:], binary.LittleEndian), nil
	case hasPrefix(b, pycompat.BOM):
		return pycompat.DecodeUTF8Sig(b)
	}
	if len(b) >= 4 {
		switch {
		case b[0] == 0 && b[1] != 0:
			return decodeUTF16(b, binary.BigEndian), nil
		case b[0] == 0:
			return decodeUTF32(b, binary.BigEndian), nil
		case b[1] == 0 && (b[2] != 0 || b[3] != 0):
			return decodeUTF16(b, binary.LittleEndian), nil
		case b[1] == 0:
			return decodeUTF32(b, binary.LittleEndian), nil
		}
	} else if len(b) == 2 {
		switch {
		case b[0] == 0:
			return decodeUTF16(b, binary.BigEndian), nil
		case b[1] == 0:
			return decodeUTF16(b, binary.LittleEndian), nil
		}
	}
	return pycompat.DecodeUTF8(b)
}

func hasPrefix(b []byte, prefix string) bool { return strings.HasPrefix(string(b), prefix) }

func decodeUTF16(b []byte, order binary.ByteOrder) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = order.Uint16(b[2*i:])
	}
	return string(utf16.Decode(units))
}

func decodeUTF32(b []byte, order binary.ByteOrder) string {
	runes := make([]rune, len(b)/4)
	for i := range runes {
		runes[i] = rune(order.Uint32(b[4*i:]))
	}
	return string(runes)
}

// orEmpty is Python's `value or {}` for a decoded JSON body.
func orEmpty(value any) any {
	if truthy(value) {
		return value
	}
	return pyjson.NewObject()
}

// pyGet is value.get(key, fallback): only dicts have .get; anything else is the
// AttributeError Flask answered with 500.
func pyGet(value any, key string, fallback any) (any, error) {
	object, isObject := value.(*pyjson.Object)
	if !isObject {
		return nil, &typeError{"'" + pyTypeName(value) + "' object has no attribute 'get'"}
	}
	if item, found := object.Get(key); found {
		return item, nil
	}
	return fallback, nil
}

func pyTypeName(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case string:
		return "str"
	case float64:
		return "float"
	case []any:
		return "list"
	}
	return "int"
}
