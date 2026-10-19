package signal

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

const (
	traceMaxBytes = 5 * 1024 * 1024
	traceMaxRows  = 100000
)

// ReadRows is csv_trace.read_rows: header and non-empty data rows.
func ReadRows(text any) ([]string, [][]string, error) {
	source, ok := text.(string)
	if !ok || len(source) > traceMaxBytes {
		return nil, nil, apperr.ValueError("CSV trace must be UTF-8 text up to 5 MB")
	}
	parsed, err := pyCSVRows(strings.TrimLeft(source, string(rune(0xFEFF))))
	if err != nil || len(parsed) == 0 {
		return nil, nil, apperr.ValueError("CSV is empty or malformed")
	}
	headers := parsed[0]
	if len(headers) < 2 || len(headers) > 256 {
		return nil, nil, apperr.ValueError("CSV needs 2 to 256 named columns")
	}
	seen := map[string]bool{}
	for _, header := range headers {
		if strings.TrimFunc(header, isStripSpace) == "" {
			return nil, nil, apperr.ValueError("CSV needs 2 to 256 named columns")
		}
		seen[header] = true
	}
	if len(seen) != len(headers) {
		return nil, nil, apperr.ValueError("CSV column names must be unique")
	}
	var rows [][]string
	for index, row := range parsed[1:] {
		if len(row) == 0 {
			continue
		}
		if len(row) != len(headers) {
			return nil, nil, apperr.ValueError(fmt.Sprintf("CSV row %d has the wrong number of columns", index+2))
		}
		rows = append(rows, row)
		if len(rows) > traceMaxRows {
			return nil, nil, apperr.ValueError("CSV exceeds 100,000 data rows")
		}
	}
	if len(rows) == 0 {
		return nil, nil, apperr.ValueError("CSV has no data rows")
	}
	return headers, rows, nil
}

func isStripSpace(r rune) bool {
	return r == ' ' || (r >= '\t' && r <= '\r') || (r >= 0x1c && r <= 0x1f) || r == 0x85 || r == 0xa0 ||
		r == 0x1680 || (r >= 0x2000 && r <= 0x200a) || r == 0x2028 || r == 0x2029 || r == 0x202f ||
		r == 0x205f || r == 0x3000
}

func indexOf(items []string, wanted string) int {
	for i, item := range items {
		if item == wanted {
			return i
		}
	}
	return -1
}

// ValidateTrace is csv_trace.validate_trace.
func ValidateTrace(item Signal, spec *pyjson.Object) (*pyjson.Object, error) {
	text := pyval.Get(spec, "csv_text")
	headers, rows, err := ReadRows(text)
	if err != nil {
		return nil, err
	}
	timeColumn, _ := pyval.Get(spec, "time_column").(string)
	valueColumn, _ := pyval.Get(spec, "value_column").(string)
	_, timeIsText := pyval.Get(spec, "time_column").(string)
	_, valueIsText := pyval.Get(spec, "value_column").(string)
	timeIndex, valueIndex := indexOf(headers, timeColumn), indexOf(headers, valueColumn)
	if !timeIsText || !valueIsText || timeIndex < 0 || valueIndex < 0 || timeColumn == valueColumn {
		return nil, apperr.ValueError("Choose different existing time and value columns")
	}
	interpolation, present := spec.Get("interpolation")
	if !present {
		interpolation = "hold"
	}
	repeat, present := spec.Get("repeat")
	if !present {
		repeat = false
	}
	if (interpolation != "hold" && interpolation != "linear") || !pyval.IsBool(repeat) {
		return nil, apperr.ValueError("Choose hold/linear interpolation and a true/false repeat flag")
	}
	if interpolation == "linear" && !strings.HasPrefix(item.Kind, "FLOAT") {
		return nil, apperr.ValueError("Linear playback requires a floating-point measurement")
	}
	points := make([]any, 0, len(rows))
	last := math.Inf(-1)
	for index, row := range rows {
		line := index + 2
		at, err := pyval.ToFloat(row[timeIndex])
		var value any
		if err == nil {
			var number float64
			number, err = pyval.ToFloat(row[valueIndex])
			value = number
			if err == nil && strings.HasPrefix(item.Kind, "INT") && number == math.Trunc(number) && !math.IsInf(number, 0) {
				value, _ = pyval.ToInt(number)
			}
			if err == nil {
				value, err = Validate(item.Kind, value)
			}
		}
		if err != nil {
			return nil, apperr.ValueError(fmt.Sprintf("Invalid numeric CSV sample at row %d: %s", line, err.Error()))
		}
		if math.IsNaN(at) || math.IsInf(at, 0) || at < 0 || at > 86400 {
			return nil, apperr.ValueError(fmt.Sprintf("CSV time at row %d must be 0 to 86400 seconds", line))
		}
		if len(points) > 0 && at <= last {
			return nil, apperr.ValueError(fmt.Sprintf("CSV times must strictly increase (row %d)", line))
		}
		last = at
		points = append(points, []any{at, value})
	}
	if points[0].([]any)[0].(float64) != 0 {
		return nil, apperr.ValueError("CSV time must start at 0 seconds")
	}
	if repeat == true && last == 0 {
		return nil, apperr.ValueError("A repeating trace needs a positive duration")
	}
	filename, present := spec.Get("filename")
	if !present {
		filename = "trace.csv"
	}
	if name, ok := filename.(string); !ok || len([]rune(name)) > 255 {
		return nil, apperr.ValueError("CSV filename must be at most 255 characters")
	}
	return pyval.Obj("mode", "csv", "filename", filename, "csv_text", text, "time_column", timeColumn,
		"value_column", valueColumn, "interpolation", interpolation, "repeat", repeat, "points", points,
		"row_count", int64(len(points)), "duration", last, "overridden", false), nil
}

// TraceValue is csv_trace.trace_value.
func TraceValue(spec *pyjson.Object, elapsed float64) any {
	points := pyval.List(spec, "points")
	if pyval.Get(spec, "repeat") == true {
		duration, _ := pyval.Float(pyval.Get(spec, "duration"))
		elapsed = pyMod(elapsed, duration)
	}
	at := func(i int) float64 { f, _ := pyval.Float(points[i].([]any)[0]); return f }
	index := sort.Search(len(points), func(i int) bool { return at(i) > elapsed }) - 1
	if index < 0 {
		index = 0
	}
	value := points[index].([]any)[1]
	if pyval.Get(spec, "interpolation") == "hold" || index == len(points)-1 {
		return value
	}
	start, end := at(index), at(index+1)
	current, _ := pyval.Float(value)
	next, _ := pyval.Float(points[index+1].([]any)[1])
	return current + (next-current)*math.Max(0, (elapsed-start)/(end-start))
}

// pyMod is Python's float %: the result takes the divisor's sign.
func pyMod(a, b float64) float64 {
	result := math.Mod(a, b)
	if result != 0 && (result < 0) != (b < 0) {
		result += b
	}
	return result
}
