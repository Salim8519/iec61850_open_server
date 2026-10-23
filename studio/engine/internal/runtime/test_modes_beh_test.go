package runtime

import (
	"strconv"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

var modeEnums = pyval.Obj("on", int64(1), "on-blocked", int64(2), "test", int64(3), "test/blocked", int64(4), "off", int64(5))

func modeSignal(ref string, value int64) *pyjson.Object {
	return pyval.Obj("ref", ref, "fc", "ST", "kind", "Enum", "value", value, "description", "", "enums", modeEnums,
		"writable", true, "quality", int64(0), "timestamp", int64(0), "quality_ref", nil, "timestamp_ref", nil)
}

// behRun is IED A of the fake project with LLN0.Mod (on), XCBR1.Mod (test) and XCBR1.Beh
// (still on, as an imported model says before anything derives it) beside LLN0.Beh.
func behRun(t *testing.T, portBase int64, net int, settings *pyjson.Object) (*Coordinator, string) {
	t.Helper()
	folder := fakeNativeDir(t)
	project, a, _ := fakeProject(t, net, portBase, false)
	ied := projectIeds(project)[0]
	signals := append(pyval.List(ied, "signals"), modeSignal("BayACTRL/LLN0.Mod.stVal", 1),
		modeSignal("BayACTRL/XCBR1.Mod.stVal", 3), modeSignal("BayACTRL/XCBR1.Beh.stVal", 1))
	ied.Set("signals", signals)
	if settings != nil {
		ied.Set("test_mode", settings)
	}
	projectIeds(project)[1].Set("role", "off")
	t.Setenv(fakeValuesEnv, `{"BayACTRL/LLN0.Mod.stVal":1,"BayACTRL/XCBR1.Mod.stVal":3,"BayACTRL/XCBR1.Beh.stVal":1,`+
		`"BayACTRL/LLN0.Beh.stVal":1}`) // what the imported model holds at every worker start
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	return c, a
}

func expectBeh(t *testing.T, c *Coordinator, a, step string, lln0, xcbr int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		ld, _, _ := c.Values().Get(a, "BayACTRL/LLN0.Beh.stVal")
		ln, _, _ := c.Values().Get(a, "BayACTRL/XCBR1.Beh.stVal")
		if ld == lln0 && ln == xcbr {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: LLN0.Beh=%v XCBR1.Beh=%v, want %d and %d", step, ld, ln, lln0, xcbr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TMODE-05/TMODE-13: Beh follows IEC 61850-7-4 Annex A at Run, after a Mod write and after an
// MMS client's Mod command, from the LN's own Mod and its LD's LLN0.Mod.
func TestBehIsDerivedAtRunAndAfterEveryModChange(t *testing.T) {
	c, a := behRun(t, 24100, 21, nil)
	expectBeh(t, c, a, "at Run (LD on, LN test)", 1, 3)
	if _, err := c.Update(a, "BayACTRL/LLN0.Mod.stVal", int64(2), nil, "manual"); err != nil {
		t.Fatal(err)
	}
	expectBeh(t, c, a, "LD on-blocked, LN test", 2, 4)
	c.onWorkerEvent(a, pyval.Obj("event", "control_applied", "ref", "BayACTRL/LLN0.Mod", "status_ref",
		"BayACTRL/LLN0.Mod.stVal", "value", int64(5), "ctl_value", int64(5), "timestamp", time.Now().UnixMilli(),
		"typed", true, "cdc", "ENC"))
	expectBeh(t, c, a, "LD off by an MMS client's command", 5, 5)
	c.onWorkerEvent(a, pyval.Obj("event", "control_applied", "ref", "BayACTRL/LLN0.Mod", "status_ref",
		"BayACTRL/LLN0.Mod.stVal", "value", int64(1), "ctl_value", int64(1), "timestamp", time.Now().UnixMilli(),
		"typed", true, "cdc", "ENC"))
	expectBeh(t, c, a, "LD back on", 1, 3)
	if _, err := c.Update(a, "BayACTRL/XCBR1.Mod.stVal", int64(1), nil, "manual"); err != nil {
		t.Fatal(err)
	}
	expectBeh(t, c, a, "LN back on", 1, 1)
}

// TMODE-14: a mode change on a large LD (here 300 LNs: 600 Mod/Beh values with their times)
// is written in one native batch up to nativeBatchLimit; above it, in chunks that never split
// one point's value from its quality and time.
func TestModeChangeOnALargeLogicalDevice(t *testing.T) {
	for _, limit := range []int{0, 50} {
		if limit > 0 {
			original := nativeBatchLimit
			nativeBatchLimit = limit
			t.Cleanup(func() { nativeBatchLimit = original })
		}
		folder := fakeNativeDir(t)
		project, a, _ := fakeProject(t, 23+limit%7, int64(24300+limit*2), false)
		ied := projectIeds(project)[0]
		signals := append(pyval.List(ied, "signals"), modeSignal("BayACTRL/LLN0.Mod.stVal", 1))
		for i := 1; i <= 300; i++ {
			ln := "BayACTRL/GGIO" + strconv.Itoa(100+i)
			signals = append(signals, modeSignal(ln+".Mod.stVal", 1), modeSignal(ln+".Beh.stVal", 1),
				pyval.Obj("ref", ln+".Beh.t", "fc", "ST", "kind", "Timestamp", "value", int64(0), "writable", true))
		}
		ied.Set("signals", signals)
		ied.Set("test_mode", pyval.Obj("devices", pyval.Obj("BayACTRL", int64(3)), "simulation", false))
		projectIeds(project)[1].Set("role", "off")
		c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
		if err := c.Start(nil); err != nil {
			c.Close()
			t.Fatalf("limit %d: Run with a saved mode on a 300-LN device: %v", limit, err)
		}
		for _, ref := range []string{"BayACTRL/GGIO101.Beh.stVal", "BayACTRL/GGIO400.Beh.stVal", "BayACTRL/LLN0.Beh.stVal"} {
			if value, _, _ := c.Values().Get(a, ref); value != int64(3) {
				t.Errorf("limit %d: %s = %v, want 3", limit, ref, value)
			}
		}
		if stamp, _, _ := c.Values().Get(a, "BayACTRL/GGIO400.Beh.t"); !pyval.Truthy(stamp) {
			t.Errorf("limit %d: GGIO400.Beh.t not written with its value", limit)
		}
		c.Close()
	}
}

// Saved test-mode settings still win at Run; Beh of every LN then follows them.
func TestSavedTestModeIsKeptAtRun(t *testing.T) {
	settings := pyval.Obj("devices", pyval.Obj("BayACTRL", int64(4)), "simulation", false)
	c, a := behRun(t, 24200, 22, settings)
	expectBeh(t, c, a, "saved test/blocked", 4, 4)
	if mod, _, _ := c.Values().Get(a, "BayACTRL/LLN0.Mod.stVal"); mod != int64(4) {
		t.Fatalf("LLN0.Mod %v, want the saved 4", mod)
	}
}

// A worker restarts from its SCL values while the app keeps the last run's values, so every
// Beh is written again at the next Run, not only those the app thinks differ.
func TestBehIsWrittenAgainOnTheNextRun(t *testing.T) {
	c, a := behRun(t, 24400, 25, nil)
	expectBeh(t, c, a, "first Run", 1, 3)
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	bus := c.opt.Bus.(*MemoryBus)
	before := len(bus.Events())
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	for _, event := range bus.Events()[before:] {
		if pyval.Get(event, "kind") == "signal" && pyval.Get(event, "source") == behaviourSource &&
			pyval.Get(event, "ref") == "BayACTRL/XCBR1.Beh.stVal" && pyval.Get(event, "value") == int64(3) {
			return
		}
	}
	t.Fatal("the second Run did not write XCBR1.Beh to the restarted worker")
}
