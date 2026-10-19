package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/modelcache"
)

// TestModelCacheRestoreIsIdentical: a restore served from the derived-model cache gives
// the same project as a full SCL rebuild, and skips the rebuild.
func TestModelCacheRestoreIsIdentical(t *testing.T) {
	builder := newService(t)
	if _, err := builder.AddTemplate("BayController"); err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(worktree, "scd", "model_substation.scd"))
	if err != nil {
		t.Skip("public model_substation.scd missing")
	}
	if _, err := builder.AddSCL(model, "model_substation.scd", scl.Selection{Names: []any{"FEED1", "BUS1", "TR1"}}); err != nil {
		t.Fatal(err)
	}
	payload, err := builder.Export()
	if err != nil {
		t.Fatal(err)
	}
	plain := newService(t)
	if err := plain.Restore(payload, false); err != nil {
		t.Fatal(err)
	}
	want := publicSnapshot(t, plain)

	dir := t.TempDir()
	cached, err := New(Config{DataDir: dir, ModelCache: modelcache.New(dir),
		Importer: &scl.Importer{Validator: loadSeed(t)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cached.Close() })
	durations := []time.Duration{}
	for round := 0; round < 3; round++ {
		started := time.Now()
		if err := cached.Restore(payload, false); err != nil {
			t.Fatal(err)
		}
		durations = append(durations, time.Since(started))
		if got := publicSnapshot(t, cached); got != want {
			t.Fatalf("round %d differs from the rebuilt project:\n got  %.300s\n want %.300s", round, got, want)
		}
	}
	gotStored, _ := pyjson.Dumps(cached.current.Load().compact)
	wantStored, _ := pyjson.Dumps(plain.current.Load().compact)
	if string(gotStored) != string(wantStored) {
		t.Fatal("stored (compact) project differs after a cached restore")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "cache", "models"))
	if len(entries) != 2 {
		t.Fatalf("want 2 cache entries (template, model_substation), got %d", len(entries))
	}
	t.Logf("restore: rebuild %v, cached %v, %v", durations[0], durations[1], durations[2])
}
