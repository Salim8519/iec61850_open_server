package goosedom

import (
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
)

func TestReachMatchesPython(t *testing.T) {
	run(t, "goose_reach.json", map[string]func(goldentest.Case) any{
		"is_npcap_loopback": func(c goldentest.Case) any { return IsNpcapLoopback(c.Obj(0)) },
		"pick_uplink":       func(c goldentest.Case) any { return PickUplink(objs(c.Args[0]), c.Obj(1)) },
		"advanced_choices":  func(c goldentest.Case) any { return AdvancedChoices(objs(c.Args[0])) },
		"share_risk": func(c goldentest.Case) any {
			return ShareRisk(c.Obj(0), c.Obj(1), objs(c.Args[2]), c.Args[3] == true)
		},
		"share_hold": func(c goldentest.Case) any { return ShareHold(c.Obj(0), c.Obj(1), c.Args[2] == true) },
		"reach_status": func(c goldentest.Case) any {
			state, message := ReachStatus(c.Obj(0), c.Obj(1), pyv.Str(c.Args[2]), c.Obj(3))
			return []any{state, message}
		},
	})
}

func TestAdaptersMatchPython(t *testing.T) {
	run(t, "goose_adapters.json", map[string]func(goldentest.Case) any{
		"normalize": func(c goldentest.Case) any { return objectList(NormalizeAdapters(pyv.List(c.Args[0]))) },
	})
}

func TestPyLower(t *testing.T) {
	cases := map[string]string{"ABC": "abc", "İX": "i̇x", "ΟΣ": "ος",
		"Σ": "σ", "ΟΣΟ": "οσο", "ΟΣ.": "ος."}
	for input, want := range cases {
		if got := pyLower(input); got != want {
			t.Errorf("pyLower(%q) = %q, want %q", input, got, want)
		}
	}
}
