package runtime

import (
	"errors"
	"fmt"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// fakeProject is the template project on 127.67.<net>.x with fake workers: IED A simulated,
// IED B a real device (no worker), no sources or scenarios unless keepAutomation.
func fakeProject(t testing.TB, net int, portBase int64, keepAutomation bool) (*pyjson.Object, string, string) {
	t.Helper()
	project := loadProject(t, portBase)
	ieds := projectIeds(project)
	for i, ied := range ieds {
		ied.Set("ip", fmt.Sprintf("127.67.%d.%d", net, 11+i))
	}
	if !keepAutomation {
		project.Set("sources", []any{})
		project.Set("scenarios", []any{})
	}
	return project, pyval.Str(ieds[0], "id"), pyval.Str(ieds[1], "id")
}

func TestHubReportsEveryChange(t *testing.T) {
	folder := fakeNativeDir(t)
	project, a, b := fakeProject(t, 1, 22100, false)
	projectIeds(project)[1].Set("role", "real")
	store := NewMemoryProject(project)
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: store})
	defer c.Close()
	r := &recorder{seen: map[Change]int{}}
	cancel := c.Subscribe(r.note)
	defer cancel()

	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	r.expect(t, "run", Change{}, Change{IEDID: a}, Change{IEDID: b})
	host, port, err := c.Endpoint(a)
	if err != nil || host != "127.67.1.11" || port != int64(22101) {
		t.Fatalf("endpoint %v %v %v", host, port, err)
	}

	r.reset()
	analog := "BayACTRL/GGIO1.AnIn2"
	if _, err := c.Update(a, analog+".mag.f", 42.5, int64(0), "manual"); err != nil {
		t.Fatal(err)
	}
	r.expect(t, "manual write (value, quality, timestamp)", Change{a, analog + ".mag.f"}, Change{a, analog + ".q"},
		Change{a, analog + ".t"})
	if record, ok := c.Record(a, analog+".mag.f"); !ok || record.Value != 42.5 || record.QualityRef != analog+".q" ||
		record.TimestampRef != analog+".t" || record.LiveState != "" {
		t.Fatalf("record %+v %v", record, ok)
	}

	r.reset()
	real := "BayBCTRL/GGIO1.AnIn2.mag.f"
	c.Values().SetLive(b, []RefValue{{real, 1.5}, {"not/a.ref", 1}})
	r.expect(t, "live value", Change{b, real})
	if value, live, ok := c.Values().Get(b, real); !ok || value != 1.5 || live != "fresh" {
		t.Fatalf("live get %v %v %v", value, live, ok)
	}
	r.reset()
	c.Values().MarkStale(b)
	r.expect(t, "stale", Change{IEDID: b})
	if _, live, _ := c.Values().Get(b, real); live != "stale" {
		t.Fatalf("live_state %v", live)
	}
	view, err := c.Signal(b, real)
	if err != nil || pyval.Get(view, "live_state") != "stale" || pyval.Get(view, "value") != 1.5 {
		t.Fatalf("signal view %v %v", view, err)
	}
	r.reset()
	c.SetIEDState(b, "real_live")
	r.expect(t, "IED state", Change{IEDID: b})

	r.reset()
	if err := c.Pause(); err != nil {
		t.Fatal(err)
	}
	r.expect(t, "pause", Change{})
	r.reset()
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	r.expect(t, "stop", Change{}, Change{IEDID: a})
	if _, _, err := c.Endpoint(a); err == nil {
		t.Fatal("endpoint after Stop")
	}
	// Python kept ied_indexes after Stop: /signals still shows the last values and live_state.
	if !c.Values().Has(b) || !c.Values().Has(a) {
		t.Fatal("index dropped at Stop")
	}
	if record, ok := c.Record(b, real); !ok || record.LiveState != "stale" {
		t.Fatalf("record after stop %+v", record)
	}
	if err := c.Stopped(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	r.reset()
	c.ResetValues(a)
	r.expect(t, "values reset", Change{IEDID: a})
}

func TestStoppedRefusesWhileRunning(t *testing.T) {
	folder := fakeNativeDir(t)
	project, a, _ := fakeProject(t, 2, 22200, false)
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	defer c.Close()
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	ran := false
	err := c.Stopped(func() error { ran = true; return nil })
	if err == nil || err.Error() != StoppedRefusal || ran {
		t.Fatalf("Stopped while running: %v ran=%v", err, ran)
	}
	if got := c.RunningIEDs(); len(got) != 1 || got[0] != a {
		t.Fatalf("running %v", got)
	}
}

// Stopping an IED that is not in the project names the real problem (change 0186).
func TestStoppingAnUnknownIEDSaysSo(t *testing.T) {
	folder := fakeNativeDir(t)
	project, a, _ := fakeProject(t, 3, 22300, false)
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	defer c.Close()
	for _, running := range []bool{false, true} {
		if running {
			if err := c.Start([]string{a}); err != nil {
				t.Fatal(err)
			}
		}
		err := c.Stop([]string{"no-such-ied"})
		var coded *apperr.Error
		if !errors.As(err, &coded) || coded.Message != "Unknown IED" || coded.Code != "not_found" {
			t.Fatalf("running=%v: %v", running, err)
		}
	}
	if got := c.RunningIEDs(); len(got) != 1 || got[0] != a {
		t.Fatalf("the running IED must keep running: %v", got)
	}
}
