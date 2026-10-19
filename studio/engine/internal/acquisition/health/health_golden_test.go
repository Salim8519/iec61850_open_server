package health

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// dumps compares json.dumps(result) (insertion order) recorded by gen_health.py.
func dumps(t *testing.T, c goldentest.Case, got any) {
	t.Helper()
	body, err := pyjson.Dumps(got)
	if err != nil {
		t.Fatalf("%s: encode: %v", c.Name, err)
	}
	goldentest.Text(t, c, string(body))
}

// parseNow reads the recorded `now` (an aware ISO timestamp, offsets may carry seconds).
func parseNow(t *testing.T, value any) time.Time {
	t.Helper()
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999-07:00:00"} {
		if now, err := time.Parse(layout, pyv.Str(value)); err == nil {
			return now
		}
	}
	t.Fatalf("bad now %v", value)
	return time.Time{}
}

func sessionMap(object *pyjson.Object) map[string]*pyjson.Object {
	sessions := map[string]*pyjson.Object{}
	if object != nil {
		for _, key := range object.Keys() {
			value, _ := object.Get(key)
			sessions[key] = pyv.Obj(value)
		}
	}
	return sessions
}

func texts(value any) []string {
	var result []string
	for _, item := range pyv.List(value) {
		result = append(result, pyv.Str(item))
	}
	return result
}

// optional returns argument index, or fallback when Python used the default.
func optional(c goldentest.Case, index int, fallback any) any {
	if index < len(c.Args) {
		return c.Args[index]
	}
	return fallback
}

// ageValue is _age's result: seconds as Python's int/int true division of microseconds.
func ageValue(observed any, now time.Time) any {
	micros, ok := ageMicros(observed, now)
	if !ok {
		return nil
	}
	seconds, _ := new(big.Rat).SetFrac64(micros, 1_000_000).Float64()
	return seconds
}

func run(t *testing.T, c goldentest.Case) {
	t.Helper()
	function, _, _ := strings.Cut(c.Name, "/")
	switch function {
	case "build_health":
		running, _ := c.Args[1].(bool)
		dumps(t, c, BuildHealth(pyv.Objects(pyv.List(c.Args[0])), running, sessionMap(c.Obj(2)), c.Obj(3),
			parseNow(t, c.Args[4])))
	case "result":
		dumps(t, c, Result(pyv.Str(c.Args[0]), c.Args[1], pyv.Str(c.Args[2]), pyv.Str(c.Args[3]),
			optional(c, 4, nil), pyv.Str(optional(c, 5, "scd")), pyv.Obj(optional(c, 6, nil))))
	case "sentence":
		dumps(t, c, Sentence(texts(c.Args[0])))
	case "device_gocb_checks":
		dumps(t, c, DeviceGocbChecks(c.Obj(0), c.Obj(1)))
	case "wire_check":
		dumps(t, c, WireCheck(c.Obj(0), c.Obj(1)))
	case "unknown_publishers":
		rows, note := UnknownPublishers(c.Obj(0))
		dumps(t, c, []any{rows, note})
	case "model_rows":
		dumps(t, c, ModelRows(c.Obj(0), c.Obj(1), c.Args[2]))
	case "report_setting_rows":
		dumps(t, c, ReportSettingRows(c.Obj(0), c.Obj(1)))
	case "mode_rows":
		dumps(t, c, ModeRows(c.Obj(0), c.Obj(1)))
	case "worst":
		dumps(t, c, Worst(texts(c.Args[0])))
	case "age":
		dumps(t, c, ageValue(c.Args[0], parseNow(t, c.Args[1])))
	case "csv_text":
		goldentest.Text(t, c, CSVText(c.Obj(0)))
	case "html_report":
		goldentest.Text(t, c, HTMLReport(c.Obj(0), pyv.Str(c.Args[1]), pyv.Str(c.Args[2])))
	default:
		t.Fatalf("%s: no Go function for this golden case", c.Name)
	}
}

func TestGoldenBuildHealth(t *testing.T) {
	for _, c := range goldentest.Load(t, "health_build.json") {
		run(t, c)
	}
}

func TestGoldenChecks(t *testing.T) {
	for _, c := range goldentest.Load(t, "health_checks.json") {
		run(t, c)
	}
}

func TestGoldenExport(t *testing.T) {
	for _, c := range goldentest.Load(t, "health_export.json") {
		run(t, c)
	}
}
