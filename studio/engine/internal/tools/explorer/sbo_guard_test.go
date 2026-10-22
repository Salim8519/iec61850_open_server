package explorer

import (
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// fakeLive answers every read with one value per ref and records what was asked.
type fakeLive struct{ asked [][]string }

func (f *fakeLive) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	refs, _ := get(fields, "refs").([]any)
	var names []string
	var values []any
	for _, raw := range refs {
		ref := get(raw.(*pyjson.Object), "ref").(string)
		names = append(names, ref)
		values = append(values, pyjson.NewObject().Set("ref", ref).Set("value", true))
	}
	f.asked = append(f.asked, names)
	return pyjson.NewObject().Set("values", values), nil
}
func (f *fakeLive) Alive() bool { return true }
func (f *fakeLive) Close()      {}

func liveService(fake *fakeLive) *Service {
	service := NewService("missing.exe", nil, nil, nil, nil)
	service.process, service.state = fake, "connected"
	return service
}

func attribute(ref, fc string) *pyjson.Object {
	return pyjson.NewObject().Set("ref", ref).Set("fc", fc)
}

// TestReadNeverSendsSBO: reading <DO>.SBO of an sbo-normal control IS a Select (IEC 61850-8-1), so
// opening a data object or pressing "Read again" must not send it; the row says why instead.
func TestReadNeverSendsSBO(t *testing.T) {
	fake := &fakeLive{}
	service := liveService(fake)
	items := []*pyjson.Object{attribute("CTRL/GGIO2.AnOut1.SBO", "CO"), attribute("CTRL/GGIO2.AnOut1.SBOw.ctlVal", "CO"),
		attribute("CTRL/GGIO2.AnOut1.ctlModel", "CF")}
	values, err := service.Read(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.asked) != 1 || len(fake.asked[0]) != 2 {
		t.Fatalf("worker was asked %v; SBO must never be read", fake.asked)
	}
	for _, ref := range fake.asked[0] {
		if ref == "CTRL/GGIO2.AnOut1.SBO" {
			t.Fatal("SBO was sent to the device")
		}
	}
	entry := get(values, "CTRL/GGIO2.AnOut1.SBO").(*pyjson.Object)
	if get(entry, "text") != SBONotRead || get(entry, "status") != "warning" {
		t.Fatalf("SBO entry %v", entry)
	}
	if service.reads != 1 {
		t.Fatalf("reads %d", service.reads)
	}
	// Only SBO: nothing is sent at all and the read counter stays.
	if _, err := service.Read(items[:1]); err != nil || len(fake.asked) != 1 || service.reads != 1 {
		t.Fatalf("SBO-only read asked the worker: %v %v", fake.asked, err)
	}
}

// TestWatchShowsWhySBOIsNotPolled: a watched SBO keeps its row (stored list unchanged) and polling skips it.
func TestWatchShowsWhySBOIsNotPolled(t *testing.T) {
	fake := &fakeLive{}
	service := liveService(fake)
	watch := []*pyjson.Object{attribute("CTRL/GGIO2$CO$AnOut1$SBO", "co"), attribute("CTRL/LLN0.Mod.stVal", "ST")}
	status, err := service.SetWatch(watch, nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := get(status, "watch").([]any)
	if len(rows) != 2 || get(rows[0].(*pyjson.Object), "text") != SBONotRead {
		t.Fatalf("watch rows %v", rows)
	}
	if _, err := service.Read(service.watch); err != nil {
		t.Fatal(err)
	}
	if len(fake.asked) != 1 || len(fake.asked[0]) != 1 || fake.asked[0][0] != "CTRL/LLN0.Mod.stVal" {
		t.Fatalf("poll asked %v", fake.asked)
	}
}

func TestSelectsOnReadOnlyForCOSBO(t *testing.T) {
	cases := map[[2]string]bool{{"A/B.Pos.SBO", "CO"}: true, {"A/B$CO$Pos$SBO", "CO"}: true,
		{"A/B.Pos.SBOw", "CO"}: false, {"A/B.Pos.SBO", "ST"}: false, {"A/B.Pos.Oper.ctlVal", "CO"}: false,
		{"A/B.Pos", "CO"}: false, {"A/B.SBO", " co "}: true}
	for input, want := range cases {
		if got := SelectsOnRead(input[0], input[1]); got != want {
			t.Errorf("SelectsOnRead%v = %v", input, got)
		}
	}
}
