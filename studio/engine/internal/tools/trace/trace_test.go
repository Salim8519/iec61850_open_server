package trace

import (
	"errors"
	"path/filepath"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

func errorText(err error) string {
	var studio *apperr.Error
	if errors.As(err, &studio) && studio.Kind == apperr.Value {
		return "ValueError: " + studio.Message
	}
	return err.Error()
}

// TestAgainstPython replays read_rows, validate_trace, validate_value, utf-8-sig and trace_value.
func TestAgainstPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "trace_oracle.py"), nil, "61850", "2500")
	if len(cases) < 10000 {
		t.Fatalf("only %d cases", len(cases))
	}
	points := [][2]any{{0.0, 1.0}, {2.0, 5.0}, {3.5, -1.0}}
	for _, item := range cases {
		var got any
		var err error
		switch item.Fn {
		case "read_rows":
			var headers []string
			var rows [][]string
			headers, rows, err = ReadRows(item.Args[0])
			got = []any{texts(headers), rowsOf(rows)}
		case "validate_trace":
			got, err = ValidateTrace(item.Args[0].(*pyjson.Object), item.Args[1].(*pyjson.Object))
		case "validate_value":
			got, err = ValidateValue(item.Args[0].(*pyjson.Object), item.Args[1])
		case "decode":
			var raw []byte
			for _, b := range item.Args[0].([]any) {
				raw = append(raw, byte(b.(int64)))
			}
			got, err = DecodeUTF8Sig(raw)
			if item.Error != "" {
				item.Error = "ValueError" + item.Error[len("UnicodeDecodeError"):]
			}
		case "trace_value":
			elapsed, _ := item.Args[2].(float64)
			if number, ok := item.Args[2].(int64); ok {
				elapsed = float64(number)
			}
			got = TraceValue(points, 3.5, item.Args[0].(bool), item.Args[1] == "linear", elapsed)
		}
		if item.Error != "" {
			if err == nil || errorText(err) != item.Error {
				t.Errorf("%s%s: got %v want %s", item.Fn, pyoracle.Dumps(t, item.Args), err, item.Error)
			}
			continue
		}
		if err != nil || pyoracle.Dumps(t, got) != pyoracle.Dumps(t, item.Result) {
			t.Errorf("%s%s:\n got %s %v\nwant %s", item.Fn, pyoracle.Dumps(t, item.Args), pyoracle.Dumps(t, got), err,
				pyoracle.Dumps(t, item.Result))
		}
	}
}

func texts(items []string) []any {
	result := []any{}
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func rowsOf(rows [][]string) []any {
	result := []any{}
	for _, row := range rows {
		result = append(result, texts(row))
	}
	return result
}
