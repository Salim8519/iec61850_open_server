package csvtrace

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/value"
)

const (
	maxBytes = 5 * 1024 * 1024
	maxRows  = 100000
	bom      = string(rune(0xfeff)) // text.lstrip(BOM)
)

func malformed() error { return apperr.ValueError("CSV is empty or malformed") }

// ReadRows is read_rows: the header and the non-empty data records of a CSV trace.
func ReadRows(text any) ([]string, [][]string, error) {
	source, isText := text.(string)
	if !isText || len(source) > maxBytes {
		return nil, nil, apperr.ValueError("CSV trace must be UTF-8 text up to 5 MB")
	}
	r := newReader(strings.TrimLeft(source, bom))
	headers, done, err := r.next()
	if err != nil || done {
		return nil, nil, malformed()
	}
	if len(headers) < 2 || len(headers) > 256 || blankHeader(headers) {
		return nil, nil, apperr.ValueError("CSV needs 2 to 256 named columns")
	}
	seen := map[string]bool{}
	for _, header := range headers {
		if seen[header] {
			return nil, nil, apperr.ValueError("CSV column names must be unique")
		}
		seen[header] = true
	}
	rows := [][]string{}
	for line := 2; ; line++ {
		row, done, err := r.next()
		if errors.Is(err, errMalformed) {
			return nil, nil, malformed()
		}
		if done {
			break
		}
		if len(row) == 0 {
			continue
		}
		if len(row) != len(headers) {
			return nil, nil, apperr.ValueError("CSV row " + strconv.Itoa(line) + " has the wrong number of columns")
		}
		rows = append(rows, row)
		if len(rows) > maxRows {
			return nil, nil, apperr.ValueError("CSV exceeds 100,000 data rows")
		}
	}
	if len(rows) == 0 {
		return nil, nil, apperr.ValueError("CSV has no data rows")
	}
	return headers, rows, nil
}

func blankHeader(headers []string) bool {
	for _, header := range headers {
		if pyrules.Strip(header) == "" {
			return true
		}
	}
	return false
}

func indexOf(headers []string, column any) int {
	for index, header := range headers {
		if column == header {
			return index
		}
	}
	return -1
}

// ValidateTrace is validate_trace: a CSV source spec checked against its signal.
func ValidateTrace(signal, spec *pyjson.Object) (*pyjson.Object, error) {
	text := pyval.Lookup(spec, "csv_text", nil)
	headers, rows, err := ReadRows(text)
	if err != nil {
		return nil, err
	}
	timeColumn, valueColumn := pyval.Lookup(spec, "time_column", nil), pyval.Lookup(spec, "value_column", nil)
	timeIndex, valueIndex := indexOf(headers, timeColumn), indexOf(headers, valueColumn)
	if timeIndex < 0 || valueIndex < 0 || pyval.Equal(timeColumn, valueColumn) {
		return nil, apperr.ValueError("Choose different existing time and value columns")
	}
	interpolation := pyval.Lookup(spec, "interpolation", "hold")
	repeat := pyval.Lookup(spec, "repeat", false)
	known, err := pyval.InStrings(interpolation, "hold", "linear")
	if err != nil {
		return nil, err
	}
	if _, isBool := repeat.(bool); !known || !isBool {
		return nil, apperr.ValueError("Choose hold/linear interpolation and a true/false repeat flag")
	}
	kind, _ := signal.Get("kind")
	kindText, _ := kind.(string)
	if interpolation == "linear" && !strings.HasPrefix(kindText, "FLOAT") {
		return nil, apperr.ValueError("Linear playback requires a floating-point measurement")
	}
	points := []any{}
	last := math.Inf(-1)
	for index, row := range rows {
		line := strconv.Itoa(index + 2)
		at, sample, err := parseSample(kindText, row[timeIndex], row[valueIndex])
		if err != nil {
			return nil, apperr.ValueError("Invalid numeric CSV sample at row " + line + ": " + err.Error())
		}
		if math.IsInf(at, 0) || math.IsNaN(at) || at < 0 || at > 86400 {
			return nil, apperr.ValueError("CSV time at row " + line + " must be 0 to 86400 seconds")
		}
		if len(points) > 0 && at <= last {
			return nil, apperr.ValueError("CSV times must strictly increase (row " + line + ")")
		}
		last = at
		points = append(points, []any{at, sample})
	}
	first := points[0].([]any)[0].(float64)
	if first != 0 {
		return nil, apperr.ValueError("CSV time must start at 0 seconds")
	}
	if repeat == true && last == 0 {
		return nil, apperr.ValueError("A repeating trace needs a positive duration")
	}
	filename := pyval.Lookup(spec, "filename", "trace.csv")
	if name, ok := filename.(string); !ok || pyval.Len(name) > 255 {
		return nil, apperr.ValueError("CSV filename must be at most 255 characters")
	}
	return pyjson.NewObject().Set("mode", "csv").Set("filename", filename).Set("csv_text", text).
		Set("time_column", timeColumn).Set("value_column", valueColumn).Set("interpolation", interpolation).
		Set("repeat", repeat).Set("points", points).Set("row_count", int64(len(points))).
		Set("duration", last).Set("overridden", false), nil
}

// parseSample converts one row: float(time), float(value), int(value) for whole INT
// values, then validate_value; the error text is Python's.
func parseSample(kind, timeText, valueText string) (float64, any, error) {
	at, err := pyrules.Float(timeText)
	if err != nil {
		return 0, nil, errors.New(pyrules.FloatMessage(timeText))
	}
	number, err := pyrules.Float(valueText)
	if err != nil {
		return 0, nil, errors.New(pyrules.FloatMessage(valueText))
	}
	var sample any = number
	if strings.HasPrefix(kind, "INT") && !math.IsInf(number, 0) && number == math.Trunc(number) {
		whole, _ := new(big.Float).SetFloat64(number).Int(nil)
		sample = pyval.NormInt(whole)
	}
	checked, err := value.ValidateKind(kind, sample)
	if err != nil {
		return 0, nil, err
	}
	return at, checked, nil
}
