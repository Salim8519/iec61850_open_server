package runtime

import (
	"testing"

	"studio-engine/internal/testutil"
)

// TIME-01 (ADR-0115): at Run every timestamp "t" of a simulated IED reads the worker's start
// time, not 1970; a value written later still gets its own time.
func TestTimestampsStartAtTheRunStart(t *testing.T) {
	folder := fakeNativeDir(t)
	project, a, _ := fakeProject(t, 24, 24500, false)
	projectIeds(project)[1].Set("role", "off")
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	defer c.Close()
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"BayACTRL/LLN0.Health.t", "BayACTRL/LPHD1.PhyHealth.t", "BayACTRL/GGIO1.AnIn2.t"} {
		if stamp, _, _ := c.Values().Get(a, ref); stamp != fakeStartedAt {
			t.Errorf("%s = %v at Run, want the start time %d", ref, stamp, fakeStartedAt)
		}
	}
	if stamp, _, _ := c.Values().Get(a, "BayACTRL/XCBR1.Pos.Oper.T"); stamp == fakeStartedAt {
		t.Error("a control parameter (Oper.T) is no status timestamp and must stay as imported")
	}
	if _, err := c.Update(a, "BayACTRL/GGIO1.AnIn2.mag.f", 1.5, nil, "manual"); err != nil {
		t.Fatal(err)
	}
	if stamp, _, _ := c.Values().Get(a, "BayACTRL/GGIO1.AnIn2.t"); stamp == fakeStartedAt {
		t.Error("a later write must carry its own time")
	}
}
