package modes

import (
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

func TestValidateTestModeNonValueErrors(t *testing.T) {
	cases := map[string]any{
		"'list' object has no attribute 'get'": []any{int64(1)},
		"'str' object has no attribute 'get'":  "x",
		"unhashable type: 'list'":              pyv.O("devices", pyv.O("LD1", []any{})),
		"unhashable type: 'dict'":              pyv.O("devices", pyv.O("LD1", pyjson.NewObject())),
	}
	for want, value := range cases {
		if _, err := ValidateTestMode(value); err == nil || err.Error() != want {
			t.Errorf("ValidateTestMode(%v) = %v, want %q", value, err, want)
		}
	}
}

func TestMissingInputsDoNotPanic(t *testing.T) {
	if got := ModeBadges(nil, nil, nil, true); got.Len() != 2 {
		t.Fatalf("ModeBadges(nil) = %v", got)
	}
	if got := SimulationSources(nil, nil); len(got) != 0 {
		t.Fatalf("SimulationSources(nil) = %v", got)
	}
	if got := ModeWrites(nil, nil, nil); len(got) != 0 {
		t.Fatalf("ModeWrites(nil) = %v", got)
	}
	if got := ModeSummary(nil, nil); pyv.Get(got, "worst") != nil {
		t.Fatalf("ModeSummary(nil) = %v", got)
	}
}

func TestReferencePatterns(t *testing.T) {
	sims := map[string]bool{"LD/LPHD1.Sim.stVal": true, "LD/LPHD.Sim.stVal\n": true, "LD/LPHD٣.Sim.stVal": true,
		"LD/LPHD1.Sim.stVal\n\n": false, "/LPHD1.Sim.stVal": false, "LD/LPHDx.Sim.stVal": false, "a/b/LPHD1.Sim.stVal": false}
	for ref, want := range sims {
		if isSimulationRef(ref) != want {
			t.Errorf("isSimulationRef(%q) != %v", ref, want)
		}
	}
	if domain, node, ok := behaviourRef("LD/A/B.Beh.stVal"); !ok || domain != "LD" || node != "A/B" {
		t.Errorf("behaviourRef = %q %q %v", domain, node, ok)
	}
}
