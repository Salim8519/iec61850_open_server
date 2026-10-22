package gateway

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// The master point list (ADR-0110) survives a project load only when it was stored, so a
// project without one normalises exactly as the frozen Python engine does.
func TestNormalizeMasterKeepsAStoredPointList(t *testing.T) {
	plain, err := NormalizeMaster(pyjson.NewObject())
	if err != nil {
		t.Fatal(err)
	}
	if _, found := plain.Get("point_list"); found {
		t.Fatal("no point list is added to a master block without one")
	}
	point := pyjson.NewObject().Set("type", "double").Set("address", int64(20001)).Set("name", "Q0")
	stored := pyjson.NewObject().Set("point_list", pyjson.NewObject().Set("iec104", []any{point})).Set("check_ied", true)
	kept, err := NormalizeMaster(stored)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := pyjson.Compact(kept)
	if !strings.Contains(string(text), `"point_list":{"iec104":[{"id":"m1","name":"Q0","type":"double","address":20001,`) ||
		!strings.HasSuffix(string(text), `"check_ied":true}`) {
		t.Fatalf("kept: %s", text)
	}
	bad := pyjson.NewObject().Set("point_list", pyjson.NewObject().Set("dnp3", []any{point}))
	if _, err := NormalizeMaster(bad); err == nil || !strings.Contains(err.Error(), "The saved DNP3 master point list") {
		t.Fatalf("a wrong saved list stops the load with the reason: %v", err)
	}
}
