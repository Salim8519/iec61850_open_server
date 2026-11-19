package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// An IED added to a running station whose later start step fails is stopped and forgotten,
// not left running beside the others (fix plan A19).
func TestFailedAddLeavesNoIedRunning(t *testing.T) {
	folder := fakeNativeDir(t)
	_ = os.Chmod(filepath.Join(folder, "studio_ied.exe"), 0o755) // a plain copy has no exec bit on Linux
	project, a, b := fakeProject(t, 8, 22800, false)
	projectIeds(project)[1].Set("role", "simulated")
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{},
		Project: NewMemoryProject(project)})
	defer c.Close()
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	defer c.Stop(nil)
	// A breaker with a mapping the initial-position step refuses: the error comes after B's
	// worker started.
	diagram := pyval.Map(project, "diagram")
	bad := pyjson.NewObject().Set("id", "bad").Set("data", pyjson.NewObject().Set("kind", "CB").
		Set("state_mapping", pyjson.NewObject().Set("mode", "weird")))
	diagram.Set("nodes", append(pyval.List(diagram, "nodes"), bad))
	if err := c.Start([]string{b}); err == nil || err.Error() != "Unsupported equipment state mapping" {
		t.Fatalf("adding B: %v, want the mapping error", err)
	}
	c.mu.Lock()
	item, active := c.runs[b], c.active[b]
	c.mu.Unlock()
	if item != nil || active {
		t.Fatal("B is still listed in the run after its start failed")
	}
	if state := iedStateOf(c, b); state == "running" {
		t.Fatalf("B is %q after a failed add", state)
	}
	if _, _, err := c.Endpoint(b); err == nil {
		t.Fatal("B still has an endpoint, so its worker was not stopped")
	}
	if iedStateOf(c, a) != "running" {
		t.Fatalf("A is %q; the failed add must not stop it", iedStateOf(c, a))
	}
}
