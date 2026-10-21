package trace

import (
	"errors"
	"unicode/utf8"

	"studio-engine/internal/apperr"
	"studio-engine/internal/tools/pycsv"
	"studio-engine/internal/tools/pytext"
)

// Trace limits (csv_trace.py).
const (
	MaxBytes = 5 * 1024 * 1024
	MaxRows  = 100000
)

// ReadRows is read_rows(text): the header row and the non-blank data rows.
func ReadRows(text any) ([]string, [][]string, error) {
	value, ok := text.(string)
	if !ok || len(value) > MaxBytes {
		return nil, nil, apperr.ValueError("CSV trace must be UTF-8 text up to 5 MB")
	}
	if r, size := utf8.DecodeRuneInString(value); r == 0xfeff {
		value = value[size:]
		for {
			if r, size = utf8.DecodeRuneInString(value); r != 0xfeff {
				break
			}
			value = value[size:] // str.lstrip(BOM) removes every leading BOM
		}
	}
	malformed := apperr.ValueError("CSV is empty or malformed")
	reader := pycsv.NewReader(value, true)
	headers, found, err := reader.Next()
	if err != nil || !found {
		return nil, nil, malformed
	}
	if len(headers) < 2 || len(headers) > 256 {
		return nil, nil, apperr.ValueError("CSV needs 2 to 256 named columns")
	}
	unique := map[string]bool{}
	for _, header := range headers {
		if pytext.Strip(header) == "" {
			return nil, nil, apperr.ValueError("CSV needs 2 to 256 named columns")
		}
		unique[header] = true
	}
	if len(unique) != len(headers) {
		return nil, nil, apperr.ValueError("CSV column names must be unique")
	}
	var rows [][]string
	for line := 2; ; line++ {
		row, more, err := reader.Next()
		var csvError *pycsv.Error
		if errors.As(err, &csvError) {
			return nil, nil, malformed
		}
		if !more {
			break
		}
		if len(row) == 0 {
			continue
		}
		if len(row) != len(headers) {
			return nil, nil, apperr.ValueError("CSV row " + itoa(line) + " has the wrong number of columns")
		}
		rows = append(rows, row)
		if len(rows) > MaxRows {
			return nil, nil, apperr.ValueError("CSV exceeds 100,000 data rows")
		}
	}
	if len(rows) == 0 {
		return nil, nil, apperr.ValueError("CSV has no data rows")
	}
	return headers, rows, nil
}

func itoa(number int) string { return pytext.Str(int64(number)) }
