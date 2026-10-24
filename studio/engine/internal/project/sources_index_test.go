package project

import (
	"fmt"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestSignalIndexOfMatchesTheFullIndex: looking up only the named keys gives the same signal
// as indexing the whole project, including a repeated ref (the last one wins) (ADR-0116).
func TestSignalIndexOfMatchesTheFullIndex(t *testing.T) {
	ieds := []*pyjson.Object{}
	for i := 0; i < 4; i++ {
		signals := []any{}
		for j := 0; j < 50; j++ {
			signals = append(signals, pyjson.NewObject().Set("ref", fmt.Sprintf("LD/GGIO1.AnIn%d.mag.f", j%40)).Set("n", int64(j)))
		}
		ieds = append(ieds, pyjson.NewObject().Set("id", fmt.Sprintf("ied%d", i%3)).Set("signals", signals))
	}
	full := signalIndex(ieds)
	wanted := map[signalKey]bool{}
	for key := range full {
		if len(wanted) < 30 {
			wanted[key] = true
		}
	}
	wanted[signalKey{"ied9", "LD/GGIO1.AnIn1.mag.f"}] = true
	wanted[signalKey{"ied0", "missing"}] = true
	limited := signalIndexOf(ieds, wanted)
	for key := range wanted {
		if limited[key] != full[key] {
			t.Fatalf("%v: %v, want %v", key, limited[key], full[key])
		}
	}
}

func TestSourceAndStepKeysNameOnlyTextPairs(t *testing.T) {
	list := []any{pyjson.NewObject().Set("ied_id", "a").Set("ref", "r"), "x",
		pyjson.NewObject().Set("ied_id", int64(1)).Set("ref", "r")}
	if keys := sourceKeys(list); len(keys) != 1 || !keys[signalKey{"a", "r"}] {
		t.Fatalf("source keys %v", keys)
	}
	scenarios := []any{pyjson.NewObject().Set("steps", []any{pyjson.NewObject().Set("ied_id", "b").Set("ref", "q"), int64(3)})}
	if keys := stepKeys(scenarios); len(keys) != 1 || !keys[signalKey{"b", "q"}] {
		t.Fatalf("step keys %v", keys)
	}
}
