// Package jsonfile reads and writes the small JSON stores of the data folder exactly as
// Python does on Windows (persistence-3 sections 4-6): json.dumps(value, indent=2) written
// by Path.write_text in text mode, so every "\n" becomes "\r\n", without a final newline
// or BOM, to "<name>.tmp" and then os.replace onto the file.
package jsonfile

import (
	"bytes"
	"os"
	"path/filepath"
	"unicode/utf8"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Encode returns the file bytes for value: json.dumps(value, indent=2) with CRLF line ends.
func Encode(value any) ([]byte, error) {
	text, err := pyval.DumpsIndent(value, 2)
	if err != nil {
		return nil, err
	}
	return bytes.ReplaceAll(text, []byte("\n"), []byte("\r\n")), nil
}

// Write stores value at path like the Python stores: parents created
// (mkdir(parents=True, exist_ok=True)), then <name>.tmp written and replaced onto path.
func Write(path string, value any) error {
	data, err := Encode(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	temporary := path + ".tmp" // path.with_name(path.name + ".tmp")
	if err := os.WriteFile(temporary, data, 0o666); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// Read is json.loads(path.read_text(encoding="utf-8")) with the stores' tolerance: it
// returns (nil, false) for every OSError or ValueError - a missing or unreadable file,
// invalid UTF-8 (surrogates included), a UTF-8 BOM (read_text keeps U+FEFF, which json.loads
// refuses) and invalid JSON. Line ends do not matter: outside strings they are whitespace
// and inside strings raw CR or LF is invalid either way.
func Read(path string) (any, bool) {
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) {
		return nil, false
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		return nil, false
	}
	return value, true
}
