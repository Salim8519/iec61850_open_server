package pytext

import (
	"errors"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

const bom = "\xef\xbb\xbf" // U+FEFF in UTF-8

// ErrBOM is json.loads' "Unexpected UTF-8 BOM" error for str input.
var ErrBOM = errors.New("Unexpected UTF-8 BOM (decode using utf-8-sig)")

// ErrUTF8 is the UnicodeDecodeError raised for bytes that are not UTF-8.
var ErrUTF8 = errors.New("'utf-8' codec can't decode bytes")

// LoadsString is json.loads(text) for a str: a leading BOM is refused.
func LoadsString(text string) (any, error) {
	if strings.HasPrefix(text, bom) {
		return nil, ErrBOM
	}
	return pyjson.Decode([]byte(text))
}

// LoadsBytes is json.loads(data) for bytes: UTF-8 with an optional BOM ("utf-8-sig");
// bytes that are not UTF-8 fail like CPython's UnicodeDecodeError (also a ValueError).
// The UTF-16/32 detection CPython applies to NUL-led input is not reproduced: such input
// never comes from the license server or a ScadaBench file.
func LoadsBytes(data []byte) (any, error) {
	if !utf8.Valid(data) {
		return nil, ErrUTF8
	}
	return pyjson.Decode([]byte(strings.TrimPrefix(string(data), bom)))
}

// ReadText is Path.read_text(encoding="utf-8"): the file must exist and be UTF-8.
func ReadText(data []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", ErrUTF8
	}
	return string(data), nil
}
