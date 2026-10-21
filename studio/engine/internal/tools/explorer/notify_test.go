package explorer

import (
	"sync/atomic"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestReportsChanges: the page-visible changes call OnChange("explorer").
func TestReportsChanges(t *testing.T) {
	service := NewService("missing.exe", nil, nil, nil, nil)
	var reported atomic.Int64
	service.OnChange(func(name string) {
		if name == Change {
			reported.Add(1)
		}
	})
	watch := []*pyjson.Object{pyjson.NewObject().Set("ref", "LD0/LLN0.Mod.stVal").Set("fc", "ST")}
	if _, err := service.SetWatch(watch, nil); err != nil {
		t.Fatal(err)
	}
	service.mu.Lock()
	service.state = "connected"
	service.mu.Unlock()
	service.event(pyjson.NewObject().Set("event", "live_connection").Set("state", "closed"))
	service.StopPolling()
	service.Disconnect()
	if count := reported.Load(); count < 4 {
		t.Fatalf("OnChange reported %d of 4 changes", count)
	}
}
