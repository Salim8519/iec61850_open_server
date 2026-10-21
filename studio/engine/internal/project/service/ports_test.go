package service

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// TestAssignIPKeepsRevision: Run-start addresses are stored without a revision change or
// project_changed, survive a reopen, and leave other IEDs and unknown ids alone.
func TestAssignIPKeepsRevision(t *testing.T) {
	s := newService(t)
	changes := 0
	if _, err := s.AddTemplate("BayController"); err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(worktree, "scd", "model_substation.scd"))
	if err != nil {
		t.Skip("public model_substation.scd missing")
	}
	if _, err := s.AddSCL(model, "model_substation.scd", scl.Selection{Names: []any{"FEED1"}}); err != nil {
		t.Fatal(err)
	}
	s.config.OnChange = func(int64, any) { changes++ }
	revision := s.Revision()
	ieds := s.IEDs()
	first, _ := ieds[0].Get("id")
	before, _ := ieds[1].Get("ip")
	s.AssignIP(first.(string), "127.50.0.9")
	s.AssignIP("nope", "127.50.0.10")
	if s.Revision() != revision || changes != 0 {
		t.Fatalf("revision %d -> %d, %d change events", revision, s.Revision(), changes)
	}
	if ip, _ := s.IEDs()[0].Get("ip"); ip != "127.50.0.9" {
		t.Fatalf("ip %v", ip)
	}
	if ip, _ := s.IEDs()[1].Get("ip"); ip != before {
		t.Fatalf("other IED changed: %v", ip)
	}
	if ip, _ := ieds[0].Get("ip"); ip == "127.50.0.9" {
		t.Fatal("the published IED object was mutated")
	}
	payload, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	reopened := newService(t)
	if err := reopened.Restore(payload, false); err != nil {
		t.Fatal(err)
	}
	if ip, _ := reopened.IEDs()[0].Get("ip"); ip != "127.50.0.9" {
		t.Fatalf("reopened ip %v", ip)
	}
}

// TestSaversKeepPreviousOnFailure: SaveGateway and SaveTestPanel announce the change
// (source "gateway" / None) and keep the previous value when the save fails.
func TestSaversKeepPreviousOnFailure(t *testing.T) {
	s := newService(t)
	sources := []any{}
	s.config.OnChange = func(_ int64, source any) { sources = append(sources, source) }
	section := pyjson.NewObject().Set("protocol", "modbus")
	if err := s.SaveGateway(section); err != nil || s.Gateway() != section {
		t.Fatalf("save gateway: %v", err)
	}
	bad := pyjson.NewObject().Set("x", math.NaN())
	if err := s.SaveGateway(bad); err == nil || s.Gateway() != section {
		t.Fatalf("failed gateway save: %v, kept %v", err, s.Gateway() == section)
	}
	panel := []any{pyjson.NewObject().Set("id", "a")}
	if err := s.SaveTestPanel(panel); err != nil || len(s.TestPanel()) != 1 {
		t.Fatalf("save panel: %v", err)
	}
	if err := s.SaveTestPanel([]any{math.Inf(1)}); err == nil || len(s.TestPanel()) != 1 {
		t.Fatalf("failed panel save: %v", err)
	}
	if len(sources) != 2 || sources[0] != "gateway" || sources[1] != nil {
		t.Fatalf("change sources %v", sources)
	}
}
