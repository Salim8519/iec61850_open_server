package deadband

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

func TestPoints(t *testing.T) {
	raw, err := pyjson.Decode([]byte(`{"signals":[
	{"ref":"L/M.A.phsA.cVal.mag.f","value":1.0},{"ref":"L/M.A.phsA.db","value":500},
	{"ref":"L/M.A.phsA.rangeC.min.f","value":0.0},{"ref":"L/M.A.phsA.rangeC.max.f","value":1000.0},
	{"ref":"L/M.Hz.mag.f","value":50.0},{"ref":"L/M.Hz.db","value":0},
	{"ref":"L/M.Cnt.mag.i","value":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	result := Points(raw.(*pyjson.Object), []string{"L/M.A.phsA.db"}, true)
	points := pyval.List(result, "points")
	if len(points) != 2 || pyval.Get(result, "applied") != int64(1) || pyval.Get(result, "without_range") != int64(1) {
		t.Fatalf("%v", result)
	}
	first := points[0].(*pyjson.Object)
	if pyval.Get(first, "band") != 5.0 || pyval.Get(first, "db_writable") != true || pyval.Get(first, "applies") != true {
		t.Fatalf("%v", first)
	}
	if note := pyval.Str(points[1].(*pyjson.Object), "note"); !strings.Contains(note, "No measuring range") {
		t.Fatal(note)
	}
	off := Points(raw.(*pyjson.Object), nil, false)
	if pyval.Get(off, "applied") != int64(0) || !strings.HasPrefix(pyval.Str(pyval.List(off, "points")[0].(*pyjson.Object), "note"), "Reported when") {
		t.Fatalf("%v", off)
	}
}
