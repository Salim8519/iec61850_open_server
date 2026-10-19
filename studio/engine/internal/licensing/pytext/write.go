package pytext

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
)

// TextBytes returns what Path.write_text puts on disk for text: text mode turns every
// "\n" into os.linesep, which is "\r\n" on Windows (the indented JSON files are CRLF).
func TextBytes(text []byte) []byte {
	if runtime.GOOS != "windows" {
		return append([]byte(nil), text...)
	}
	return bytes.ReplaceAll(text, []byte("\n"), []byte("\r\n"))
}

// WriteReplace is the Python pattern used for every license and update file: create the
// parent folder, write_text to "<name>.tmp", then os.replace onto the real name, so a
// reader never sees a half-written file.
func WriteReplace(path string, text []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, TextBytes(text), 0o666); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// ReadJSON is read_text(encoding="utf-8") followed by json.loads; ok is false where
// Python's callers catch OSError or ValueError and fall back to a default.
func ReadJSON(path string) (value any, ok bool) {
	text, err := ReadText(os.ReadFile(path))
	if err != nil {
		return nil, false
	}
	value, err = LoadsString(text)
	return value, err == nil
}
