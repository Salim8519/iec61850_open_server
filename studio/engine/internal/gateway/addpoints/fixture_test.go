package addpoints

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// Synthetic IEDs (no customer data): a simulated feeder with a breaker (DPC), a disconnector
// position without a command, an alarm, a measurement and a single-point command (SPC), and
// a real IED with one breaker.
func signal(ref, fc, kind, description string) *pyjson.Object {
	return pyjson.NewObject().Set("ref", ref).Set("fc", fc).Set("kind", kind).Set("description", description)
}

func control(ref, cdc, status string) *pyjson.Object {
	return pyjson.NewObject().Set("ref", ref).Set("cdc", cdc).Set("status", status)
}

func feeder() *pyjson.Object {
	return pyjson.NewObject().Set("id", "f1").Set("name", "FEED1").Set("role", "simulated").
		Set("signals", []any{
			signal("FEED1LD0/XCBR1.Pos.stVal", "ST", "Dbpos", "Breaker position"),
			signal("FEED1LD0/XCBR1.Pos.q", "ST", "Quality", ""),
			signal("FEED1LD0/XSWI1.Pos.stVal", "ST", "Dbpos", ""),
			signal("FEED1LD0/PTOC1.Op.general", "ST", "BOOLEAN", "Overcurrent trip"),
			signal("FEED1LD0/GGIO1.SPCSO1.stVal", "ST", "BOOLEAN", "Spare output"),
			signal("FEED1LD0/MMXU1.TotW.mag.f", "MX", "FLOAT32", "Active power"),
			signal("FEED1LD0/XCBR1.Pos.ctlModel", "CF", "Enum", ""),
		}).
		Set("controls", []any{control("FEED1LD0/XCBR1.Pos", "DPC", "FEED1LD0/XCBR1.Pos.stVal"),
			control("FEED1LD0/GGIO1.SPCSO1", "SPC", "FEED1LD0/GGIO1.SPCSO1.stVal")}).
		Set("datasets", []any{})
}

func realIED() *pyjson.Object {
	return pyjson.NewObject().Set("id", "r1").Set("name", "REAL1").Set("role", "real").
		Set("signals", []any{signal("REAL1LD0/XCBR1.Pos.stVal", "ST", "Dbpos", "")}).
		Set("controls", []any{control("REAL1LD0/XCBR1.Pos", "DPC", "REAL1LD0/XCBR1.Pos.stVal")}).
		Set("datasets", []any{})
}

func project() []*pyjson.Object { return []*pyjson.Object{feeder(), realIED()} }

func pointJSON(t *testing.T, value any) string {
	t.Helper()
	text, err := pyjson.Compact(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func field(point any, key string) any {
	value, _ := point.(*pyjson.Object).Get(key)
	return value
}

func problems(results []Result) []string {
	list := []string{}
	for _, result := range results {
		if result.Problem != "" {
			list = append(list, result.Problem)
		}
	}
	return list
}
