package service

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// TestOwnPointsSurviveSaveAndReopen: gateway own points (lane G) keep their source, value,
// id and target through a save, an export and a reopen of the project.
func TestOwnPointsSurviveSaveAndReopen(t *testing.T) {
	s := newService(t)
	raw := []any{
		pyjson.NewObject().Set("kind", "double").Set("source", "manual").Set("address", int64(20001)).
			Set("description", "Spare breaker").Set("value", int64(2)),
		pyjson.NewObject().Set("kind", "float").Set("source", "manual").Set("address", int64(30001)).
			Set("value", 49.95).Set("scale", 0.1),
		pyjson.NewObject().Set("kind", "command_double").Set("source", "manual").Set("address", int64(50001)).
			Set("target", "own:1"),
	}
	clean, problems, err := gateway.ValidatePoints("iec104", raw, nil)
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	section := gateway.EmptyGateway()
	section.Set("points", pyjson.NewObject().Set("iec104", clean).Set("modbus", []any{}).Set("dnp3", []any{}))
	if err := s.SaveGateway(section); err != nil {
		t.Fatal(err)
	}
	payload, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	reopened := newService(t)
	if err := reopened.Restore(payload, false); err != nil {
		t.Fatal(err)
	}
	points, _ := reopened.Gateway().Get("points")
	stored, _ := points.(*pyjson.Object).Get("iec104")
	want, _ := pyjson.Dumps(clean)
	got, _ := pyjson.Dumps(stored)
	if string(got) != string(want) {
		t.Fatalf("reopened points\n got %s\nwant %s", got, want)
	}
	again, problems, err := gateway.ValidatePoints("iec104", stored, nil)
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	if text, _ := pyjson.Dumps(again); string(text) != string(want) {
		t.Fatalf("ids or values changed on the next validation: %s", text)
	}
}
