package runtime

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// bandIED is one MV with db 1000 (1 % of 0..200 = 2), zeroDb 500 (1), plus one MV without rangeC.
func bandIED(t *testing.T, on bool) *running {
	t.Helper()
	text := `{"id":"x","name":"X","scd_deadbands":` + map[bool]string{true: "true", false: "false"}[on] + `,"signals":[
	{"ref":"XLD/MMXU1.TotW.mag.f","kind":"FLOAT32","fc":"MX","writable":true,"value":10.0},
	{"ref":"XLD/MMXU1.TotW.instMag.f","kind":"FLOAT32","fc":"MX","writable":true,"value":10.0},
	{"ref":"XLD/MMXU1.TotW.db","kind":"INT32U","fc":"CF","value":1000},
	{"ref":"XLD/MMXU1.TotW.zeroDb","kind":"INT32U","fc":"CF","value":500},
	{"ref":"XLD/MMXU1.TotW.rangeC.min.f","kind":"FLOAT32","fc":"CF","value":0.0},
	{"ref":"XLD/MMXU1.TotW.rangeC.max.f","kind":"FLOAT32","fc":"CF","value":200.0},
	{"ref":"XLD/MMXU1.Hz.mag.f","kind":"FLOAT32","fc":"MX","writable":true,"value":50.0},
	{"ref":"XLD/MMXU1.Hz.db","kind":"INT32U","fc":"CF","value":100000}]}`
	raw, err := pyjson.Decode([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	model := newIedModel(raw.(*pyjson.Object))
	return newRunning(model, newValueTable(model, nil), model.raw)
}

func TestDeadbandOffChangesNothing(t *testing.T) {
	if _, banded := deadbandOf(bandIED(t, false), "XLD/MMXU1.TotW.mag.f"); banded {
		t.Fatal("a deadband applied with the switch off")
	}
}

func TestDeadbandFilter(t *testing.T) {
	target := bandIED(t, true)
	band, banded := deadbandOf(target, "XLD/MMXU1.TotW.mag.f")
	if !banded || band.band != 2 || band.zero != 1 || band.instant != "XLD/MMXU1.TotW.instMag.f" {
		t.Fatalf("band %+v %v", band, banded)
	}
	cases := []struct {
		value  float64
		want   any
		report bool
	}{{11.5, 11.5, false}, {12.5, 12.5, true}, {11.0, 11.0, false}, {10.4, 10.4, true}, {0.6, 0.0, true}}
	for _, item := range cases {
		value, report := band.filter(target, "XLD/MMXU1.TotW.mag.f", item.value)
		if value != item.want || report != item.report {
			t.Fatalf("%v: got %v %v", item.value, value, report)
		}
		if report {
			target.table.set("XLD/MMXU1.TotW.mag.f", value)
		}
	}
}

func TestNoRangeNoDeadband(t *testing.T) {
	if _, banded := deadbandOf(bandIED(t, true), "XLD/MMXU1.Hz.mag.f"); banded {
		t.Fatal("a deadband applied without rangeC")
	}
}
