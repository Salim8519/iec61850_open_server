package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"math/big"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Decode is archive.decode(payload) WITHOUT the final expand(): it returns the raw
// project.json object (format 1 with inline sources, or a compact format 2..15 document);
// the caller (project layer) runs expand() on it. Errors, in Python's order:
//
//   - StudioError "Project archive exceeds 100 MB" (payload larger than 100 MiB)
//   - StudioError "Invalid project archive" (not a ZIP, unreadable member, bad CRC, invalid JSON)
//   - StudioError "Expected a single project.json entry"
//   - StudioError "Unpacked project exceeds the 256 MB legacy recovery limit"
//   - ValueError "'utf-8' codec can't decode ..." (json.loads UnicodeDecodeError, not caught by Python)
//   - StudioError "Compact project exceeds 100 MB" (format 2..15 with more than 100 MiB of JSON)
//   - StudioError "Unsupported project format" (JSON that is not an object)
//
// Known differences: Python also reads bzip2/lzma members (refused here as invalid) and
// raises zlib.error/EOFError (not "Invalid project archive") for corrupt deflate data.
func Decode(payload []byte) (*pyjson.Object, error) {
	if len(payload) > MaxUnpacked {
		return nil, apperr.New("Project archive exceeds 100 MB")
	}
	invalid := apperr.New("Invalid project archive")
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, invalid
	}
	if len(reader.File) != 1 || reader.File[0].Name != EntryName {
		return nil, apperr.New("Expected a single project.json entry")
	}
	entry := reader.File[0]
	if entry.UncompressedSize64 > MaxLegacyUnpacked {
		return nil, apperr.New("Unpacked project exceeds the 256 MB legacy recovery limit")
	}
	data, err := readMember(entry)
	if err != nil {
		return nil, invalid
	}
	text, err := pyText(data)
	if err != nil {
		return nil, err
	}
	value, err := pyjson.Decode(text)
	if err != nil {
		return nil, invalid
	}
	document, isObject := value.(*pyjson.Object)
	if isObject && entry.UncompressedSize64 > MaxUnpacked {
		if format, _ := document.Get("format"); isCompactFormat(format) {
			return nil, apperr.New("Compact project exceeds 100 MB")
		}
	}
	if !isObject {
		return nil, apperr.New("Unsupported project format")
	}
	return document, nil
}

// readMember reads the member and lets archive/zip verify its size and CRC-32.
func readMember(entry *zip.File) ([]byte, error) {
	if entry.Method != zip.Deflate && entry.Method != zip.Store {
		return nil, zip.ErrAlgorithm
	}
	stream, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	data := make([]byte, 0, entry.UncompressedSize64)
	buffer := bytes.NewBuffer(data)
	if _, err := io.Copy(buffer, io.LimitReader(stream, int64(entry.UncompressedSize64)+1)); err != nil {
		return nil, err
	}
	if uint64(buffer.Len()) != entry.UncompressedSize64 {
		return nil, zip.ErrFormat
	}
	return buffer.Bytes(), nil
}

// isCompactFormat is Python's `value in COMPACT_FORMATS` (set(range(2, 16))): equal numbers
// match across int, float and bool (2.0 is in the set, True == 1 is not).
func isCompactFormat(value any) bool {
	var number float64
	switch v := value.(type) {
	case int64:
		return v >= 2 && v <= 15
	case int:
		return v >= 2 && v <= 15
	case *big.Int:
		return v.IsInt64() && v.Int64() >= 2 && v.Int64() <= 15
	case float64:
		number = v
	default:
		return false
	}
	return number == math.Trunc(number) && number >= 2 && number <= 15
}
