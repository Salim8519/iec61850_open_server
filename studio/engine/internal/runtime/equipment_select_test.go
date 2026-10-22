package runtime

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// Select before operate for CB/DS/ES (ADR-0108) on the fake workers: BayA's breaker cb1 is made
// sbo-enhanced (ctlModel 4, sboTimeout 10 s) unless direct is set.

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (f *fakeClock) Now() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.now }
func (f *fakeClock) add(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func sboRun(t *testing.T, portBase int64, direct bool) (*Coordinator, *MemoryBus, *MemoryProject, *fakeClock) {
	t.Helper()
	project, a, _ := fakeProject(t, 12, portBase, false)
	if !direct {
		for _, item := range pyval.List(projectIeds(project)[0], "controls") {
			if control := item.(*pyjson.Object); pyval.Str(control, "ref") == "BayACTRL/XCBR1.Pos" {
				control.Set("model", int64(4))
			}
		}
	}
	bus, store := &MemoryBus{}, NewMemoryProject(project)
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: store})
	t.Cleanup(c.Close)
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c.selections.now = clock.Now
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	return c, bus, store, clock
}

func eventKinds(bus *MemoryBus, prefix string) []string {
	var found []string
	for _, event := range bus.Events() {
		if kind := pyval.Str(event, "kind"); strings.HasPrefix(kind, prefix) {
			found = append(found, kind)
		}
	}
	return found
}

func TestSelectThenOperate(t *testing.T) {
	c, bus, _, _ := sboRun(t, 23300, false)
	answer, err := c.EquipmentSelect("cb1", "close", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pyval.Get(answer, "model") != "sbo-enhanced" || pyval.Get(answer, "expires_ms") != int64(7000) ||
		pyval.Get(answer, "action") != "close" || pyval.Get(answer, "selected") != true {
		t.Fatalf("select answer %v (10 s timeout keeps a 3 s margin: 7 s)", answer)
	}
	if _, err := c.Operate("cb1", nil, "open"); err == nil || !strings.Contains(err.Error(), "is selected to close") {
		t.Fatalf("another action must be refused while selected, got %v", err)
	}
	result, err := c.Operate("cb1", nil, "close")
	if err != nil || pyval.Get(result, "action") != "close" || pyval.Get(result, "wire_value") != true {
		t.Fatalf("operate within the window: %v %v", result, err)
	}
	if _, status := c.selections.peek(selectionKey{node: "cb1"}); status != selectionNone {
		t.Fatal("an operate uses up the selection")
	}
	if got := eventKinds(bus, "equipment_select"); !reflect.DeepEqual(got, []string{"equipment_selected"}) {
		t.Fatalf("events %v", got)
	}
}

func TestSelectExpiresInWords(t *testing.T) {
	c, bus, _, clock := sboRun(t, 23310, false)
	if _, err := c.EquipmentSelect("cb1", "open", nil); err != nil {
		t.Fatal(err)
	}
	clock.add(7 * time.Second)
	_, err := c.Operate("cb1", nil, "open")
	if err == nil || !strings.Contains(err.Error(), "ran out after 7 s, so nothing was sent. Press Open again") {
		t.Fatalf("an expired selection must be refused in words, got %v", err)
	}
	if got := eventKinds(bus, "equipment_select"); !reflect.DeepEqual(got, []string{"equipment_selected", "equipment_select_expired"}) {
		t.Fatalf("events %v", got)
	}
	if _, err := c.Operate("cb1", nil, "open"); err != nil {
		t.Fatalf("without a selection the canvas path works as before: %v", err)
	}
}

func TestCancelEndsTheSelection(t *testing.T) {
	c, _, _, _ := sboRun(t, 23320, false)
	if _, err := c.EquipmentSelect("cb1", "close", nil); err != nil {
		t.Fatal(err)
	}
	answer, err := c.EquipmentCancel("cb1")
	if err != nil || pyval.Get(answer, "cancelled") != true || pyval.Get(answer, "action") != "close" {
		t.Fatalf("cancel %v %v", answer, err)
	}
	if answer, _ = c.EquipmentCancel("cb1"); pyval.Get(answer, "cancelled") != false {
		t.Fatalf("a second cancel has nothing to cancel: %v", answer)
	}
	if _, err := c.EquipmentSelect("cb1", "close", nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	if _, status := c.selections.peek(selectionKey{node: "cb1"}); status != selectionNone {
		t.Fatal("Stop ends every selection")
	}
}

func TestSelectRefusals(t *testing.T) {
	c, _, _, _ := sboRun(t, 23330, true)
	if _, err := c.EquipmentSelect("cb1", "close", nil); err == nil || !strings.Contains(err.Error(), "needs no Select") {
		t.Fatalf("a direct model is not selected, got %v", err)
	}
	if _, err := c.EquipmentSelect("pb1", "press", nil); err == nil || !strings.Contains(err.Error(), "Only a breaker") {
		t.Fatalf("a push button is not selected, got %v", err)
	}
	if _, err := c.EquipmentSelect("cb1", "close", pyval.Obj("orcat", int64(9))); err == nil {
		t.Fatal("orCat 9 must be refused")
	}
	if _, err := c.Operate("cb1", nil, "sideways"); err == nil {
		t.Fatal("an unknown action stays refused")
	}
}

// A direct model without options keeps today's request and answer; options take the typed path
// (ADR-0104) and nothing is remembered without options.
func TestDirectModelRequestsUnchanged(t *testing.T) {
	c, _, _, _ := sboRun(t, 23340, true)
	result, err := c.Operate("cb1", nil, "close")
	if err != nil {
		t.Fatal(err)
	}
	if keys := result.Keys(); !reflect.DeepEqual(keys, []string{"command", "feedback", "node_id", "action", "wire_value"}) {
		t.Fatalf("answer keys %v", keys)
	}
	if command := pyval.Map(result, "command"); !reflect.DeepEqual(command.Keys(), []string{"ok"}) {
		t.Fatalf("the canvas client answered %v", command)
	}
	node, _ := c.findNode("cb1")
	if _, present := pyval.Map(node, "data").Get("command_options"); present {
		t.Fatal("an operate without options must not add command_options")
	}
	if _, err = c.OperateWith("cb1", nil, "open", pyval.Obj("test", true)); err != nil {
		t.Fatalf("options go through the typed operate: %v", err)
	}
	if _, err = c.OperateWith("cb1", nil, "open", pyval.Obj()); err != nil {
		t.Fatalf("default options keep the canvas path: %v", err)
	}
}

func TestOptionChanges(t *testing.T) {
	data := pyval.Obj("kind", "CB")
	if optionChanges(data, CommandOptions{}, true) != nil || optionChanges(data, CommandOptions{Test: true}, false) != nil {
		t.Fatal("defaults on a node without options, or nothing sent, change nothing")
	}
	changes := optionChanges(data, CommandOptions{Test: true, OrCat: 3, OrIdent: "bench"}, true)
	stored := pyval.Map(changes, "command_options")
	if !reflect.DeepEqual(stored.Keys(), []string{"test", "interlock_check", "synchro_check", "orcat", "orident"}) ||
		pyval.Get(stored, "test") != true || pyval.Get(stored, "orcat") != int64(3) || pyval.Get(stored, "orident") != "bench" {
		t.Fatalf("changes %v", pyval.Repr(changes))
	}
	data.Set("command_options", pyval.Get(changes, "command_options"))
	if optionChanges(data, CommandOptions{Test: true, OrCat: 3, OrIdent: "bench"}, true) != nil {
		t.Fatal("the same options are not written again")
	}
	if _, _, err := commandOptionsFrom(pyval.Obj("orident", strings.Repeat("x", 65))); err == nil {
		t.Fatal("orIdent is at most 64 characters")
	}
	if _, _, err := commandOptionsFrom(pyval.Obj("test", "yes")); err == nil {
		t.Fatal("test must be true or false")
	}
}

func TestOperateWindow(t *testing.T) {
	for in, want := range map[int]int{30000: 27000, 10000: 7000, 5000: 4500, 0: 0} {
		if got := operateWindow(in); got != want {
			t.Errorf("operateWindow(%d) = %d, want %d", in, got, want)
		}
	}
	for model, want := range map[any]string{int64(4): "sbo-enhanced", int64(1): "direct-normal", "sbo-normal": "", int64(7): "", nil: ""} {
		if got := controlModelName(pyval.Obj("model", model)); got != want {
			t.Errorf("model %v = %q, want %q", model, got, want)
		}
	}
}
