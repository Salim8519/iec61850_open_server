package trace

import (
	"errors"
	"math"
	"math/big"
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ValidateTrace is validate_trace(signal, spec): the stored csv source spec with its points.
func ValidateTrace(signal, spec *pyjson.Object) (*pyjson.Object, error) {
	text, _ := spec.Get("csv_text")
	headers, rows, err := ReadRows(text)
	if err != nil {
		return nil, err
	}
	timeColumn, _ := spec.Get("time_column")
	valueColumn, _ := spec.Get("value_column")
	timeIndex, valueIndex := index(headers, timeColumn), index(headers, valueColumn)
	if timeIndex < 0 || valueIndex < 0 || pytext.Equal(timeColumn, valueColumn) {
		return nil, apperr.ValueError("Choose different existing time and value columns")
	}
	interpolation, found := spec.Get("interpolation")
	if !found {
		interpolation = "hold"
	}
	repeat, found := spec.Get("repeat")
	if !found {
		repeat = false
	}
	if _, unhashable := interpolation.([]any); unhashable {
		return nil, errors.New("TypeError: unhashable type: 'list'")
	}
	if _, unhashable := interpolation.(*pyjson.Object); unhashable {
		return nil, errors.New("TypeError: unhashable type: 'dict'")
	}
	repeatFlag, isBool := repeat.(bool)
	if (interpolation != "hold" && interpolation != "linear") || !isBool {
		return nil, apperr.ValueError("Choose hold/linear interpolation and a true/false repeat flag")
	}
	kindValue, _ := signal.Get("kind")
	kind := pytext.Str(kindValue)
	if interpolation == "linear" && !strings.HasPrefix(kind, "FLOAT") {
		return nil, apperr.ValueError("Linear playback requires a floating-point measurement")
	}
	points := make([]any, 0, len(rows))
	last := math.Inf(-1)
	for offset, row := range rows {
		line := itoa(offset + 2)
		at, err := pytext.ParseFloat(row[timeIndex])
		var value any
		if err == nil {
			var number float64
			number, err = pytext.ParseFloat(row[valueIndex])
			value = number
			if err == nil && strings.HasPrefix(kind, "INT") && number == math.Trunc(number) && !math.IsInf(number, 0) {
				value = integer(number)
			}
			if err == nil {
				value, err = ValidateValue(signal, value)
			}
		}
		if err != nil {
			return nil, apperr.ValueError("Invalid numeric CSV sample at row " + line + ": " + message(err))
		}
		if math.IsNaN(at) || math.IsInf(at, 0) || at < 0 || at > 86400 {
			return nil, apperr.ValueError("CSV time at row " + line + " must be 0 to 86400 seconds")
		}
		if len(points) > 0 && at <= last {
			return nil, apperr.ValueError("CSV times must strictly increase (row " + line + ")")
		}
		last = at
		points = append(points, []any{at, value})
	}
	if points[0].([]any)[0].(float64) != 0 {
		return nil, apperr.ValueError("CSV time must start at 0 seconds")
	}
	if repeatFlag && last == 0 {
		return nil, apperr.ValueError("A repeating trace needs a positive duration")
	}
	filename, found := spec.Get("filename")
	if !found {
		filename = "trace.csv"
	}
	if name, ok := filename.(string); !ok || pytext.Len(name) > 255 {
		return nil, apperr.ValueError("CSV filename must be at most 255 characters")
	}
	return pyjson.NewObject().Set("mode", "csv").Set("filename", filename).Set("csv_text", text).
		Set("time_column", timeColumn).Set("value_column", valueColumn).Set("interpolation", interpolation).
		Set("repeat", repeatFlag).Set("points", points).Set("row_count", int64(len(points))).
		Set("duration", last).Set("overridden", false), nil
}

func index(headers []string, column any) int {
	for position, header := range headers {
		if column == header {
			return position
		}
	}
	return -1
}

func message(err error) string {
	var studio *apperr.Error
	if errors.As(err, &studio) {
		return studio.Message
	}
	return err.Error()
}

// integer is int(value) of an integral float (int64 when it fits, else a big int).
func integer(value float64) any {
	if value >= -9.223372036854775808e18 && value < 9.223372036854775808e18 {
		return int64(value)
	}
	whole, _ := new(big.Float).SetFloat64(value).Int(nil)
	return whole
}

// TraceValue is trace_value(spec, elapsed): the held or interpolated value at elapsed seconds.
func TraceValue(points [][2]any, duration float64, repeat, linear bool, elapsed float64) any {
	if repeat {
		elapsed = math.Mod(elapsed, duration)
		if elapsed < 0 {
			elapsed += duration // Python's % takes the sign of the divisor
		}
	}
	position := sort.Search(len(points), func(i int) bool { return points[i][0].(float64) > elapsed }) - 1
	if position < 0 {
		position = 0
	}
	at, value := points[position][0].(float64), points[position][1]
	if !linear || position == len(points)-1 {
		return value
	}
	end, next := points[position+1][0].(float64), points[position+1][1]
	start, _ := pytext.Float(value)
	stop, _ := pytext.Float(next)
	return start + (stop-start)*math.Max(0, (elapsed-at)/(end-at))
}
