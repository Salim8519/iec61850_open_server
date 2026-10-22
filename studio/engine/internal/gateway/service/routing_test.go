package service

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// The routing table of ADR-0107: S/E and the cause of transmission decide the phase, the
// phase and the point decide the call, and every refusal is a negative answer with a reason.

func wire(pairs ...any) *pyjson.Object {
	object := pyjson.NewObject()
	for index := 0; index+1 < len(pairs); index += 2 {
		object.Set(pairs[index].(string), pairs[index+1])
	}
	return object
}

func TestPhaseOf(t *testing.T) {
	cases := []struct {
		protocol string
		wire     *pyjson.Object
		phase    Phase
		refused  bool
	}{
		{"iec104", nil, PhaseExecute, false},
		{"iec104", wire("cot", int64(6), "select", true), PhaseSelect, false},
		{"iec104", wire("cot", int64(6), "select", false), PhaseExecute, false},
		{"iec104", wire("cot", int64(8), "select", true), PhaseCancel, false},
		{"iec104", wire("cot", int64(3), "select", false), "", true},
		{"dnp3", wire("function", "select"), PhaseSelect, false},
		{"dnp3", wire("function", "operate"), PhaseExecute, false},
		{"dnp3", wire("function", "direct"), PhaseExecute, false},
		{"dnp3", wire("function", "direct_no_ack"), PhaseExecute, false},
		{"dnp3", wire("function", "freeze"), "", true},
		{"modbus", wire("function", int64(16)), PhaseExecute, false},
		{"manual", nil, PhaseExecute, false},
	}
	for _, item := range cases {
		phase, reason := PhaseOf(item.protocol, item.wire)
		if phase != item.phase || (reason != "") != item.refused {
			t.Errorf("%s %v: %q %q", item.protocol, item.wire, phase, reason)
		}
	}
}

func TestRoute(t *testing.T) {
	cases := []struct {
		kind     string
		phase    Phase
		selected bool
		action   Action
	}{
		{"command_double", PhaseSelect, false, ActSelect},
		{"command_double", PhaseExecute, false, ActLegacy},
		{"command_double", PhaseExecute, true, ActOperate},
		{"command_single", PhaseCancel, true, ActCancel},
		{gateway.KindStep, PhaseExecute, false, ActOperate},
		{gateway.KindSetpoint, PhaseSelect, false, ActSelect},
		{gateway.KindSetpoint, PhaseCancel, false, ActCancel},
	}
	for _, item := range cases {
		if got := Route(item.kind, item.phase, item.selected); got != item.action {
			t.Errorf("%s %s %v: %s", item.kind, item.phase, item.selected, got)
		}
	}
	unavailable := apperr.Coded("not yet", "not_available")
	if !SelectFallback("command_double", unavailable) || SelectFallback(gateway.KindStep, unavailable) ||
		SelectFallback("command_double", errors.New("refused")) {
		t.Fatal("select fallback")
	}
}

func TestCommandValue(t *testing.T) {
	cases := []struct {
		kind  string
		raw   any
		scale float64
		cdc   string
		value any
		bad   bool
	}{
		{gateway.KindStep, int64(1), 1, "BSC", "lower", false},
		{gateway.KindStep, int64(2), 1, "BSC", "higher", false},
		{gateway.KindStep, int64(3), 1, "BSC", nil, true},
		{gateway.KindStep, true, 1, "BSC", "higher", false},  // DNP3 close / ON = raise
		{gateway.KindStep, false, 1, "ISC", "lower", false},  // DNP3 trip / OFF = lower
		{gateway.KindStep, "higher", 1, "BSC", "higher", false},
		{gateway.KindSetpoint, 12.5, 1, "APC", 12.5, false},
		{gateway.KindSetpoint, 125.0, 10, "APC", 12.5, false}, // SCADA sees IED value x scale
		{gateway.KindSetpoint, 70000.0, 1, "INC", int64(70000), false},
		{gateway.KindSetpoint, 2.6, 1, "INC", int64(3), false},
		{gateway.KindSetpoint, 1e12, 1, "INC", nil, true},
		{gateway.KindSetpoint, true, 1, "APC", nil, true},
		{"command_double", true, 1, "DPC", true, false},
	}
	for _, item := range cases {
		value, _, reason := CommandValue(item.kind, item.raw, item.scale, item.cdc)
		if value != item.value || (reason != "") != item.bad {
			t.Errorf("%s %v: %v %q", item.kind, item.raw, value, reason)
		}
	}
}

// fakeTyped plays the typed IEC 61850 commands: it records calls and answers errors on request.
type fakeTyped struct {
	mu        sync.Mutex
	calls     []string
	selectErr error
	operate   error
}

func (f *fakeTyped) record(call string) {
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
}

func (f *fakeTyped) Select(request TypedRequest) (int, error) {
	f.record("select " + request.Ref + " " + pyStr(request.Value))
	return 5000, f.selectErr
}

func (f *fakeTyped) Operate(request TypedRequest) (*pyjson.Object, error) {
	f.record("operate " + request.Ref + " " + pyStr(request.Value))
	return pyjson.NewObject().Set("ok", true).Set("model", "sbo-with-normal-security"), f.operate
}

func (f *fakeTyped) Cancel(_, ref string) error {
	f.record("cancel " + ref)
	return nil
}

func pyStr(value any) string {
	text, _ := pyjson.Dumps(value)
	return string(text)
}

func typedManager(t *testing.T) (*Manager, *fakeTyped, *fakeOperator) {
	t.Helper()
	hub := newHub()
	ied := bay()
	ied.Set("controls", append(get(ied, "controls").([]any),
		pyjson.NewObject().Set("ref", "BayCTRL/ATCC1.TapChg").Set("cdc", "BSC").Set("status", "BayCTRL/ATCC1.TapChg.valWTr"),
		pyjson.NewObject().Set("ref", "BayCTRL/ATCC1.BndCtr").Set("cdc", "APC").Set("status", "BayCTRL/ATCC1.BndCtr.mxVal")))
	section := gateway.EmptyGateway()
	get(section, "points").(*pyjson.Object).Set("iec104", []any{
		point("command_double", 50001, "BayCTRL/XCBR1.Pos").Set("id", "cb"),
		point(gateway.KindStep, 50002, "BayCTRL/ATCC1.TapChg").Set("id", "tap"),
		point(gateway.KindSetpoint, 55001, "BayCTRL/ATCC1.BndCtr").Set("id", "sp").Set("scale", 10.0)})
	operator := &fakeOperator{hub: hub, status: "BayCTRL/XCBR1.Pos.stVal"}
	manager := NewManager(&fakeProjects{ieds: []*pyjson.Object{ied}, section: section}, hub, operator, "", nil)
	typed := &fakeTyped{}
	manager.Typed = typed
	return manager, typed, operator
}

func step(m *Manager, id string, value any, cot int64, selected bool) (bool, string) {
	return m.executor.run(pyjson.NewObject().Set("point_id", id).Set("value", value).Set("protocol", "iec104").
		Set("peer", "127.0.0.1").Set("wire", wire("type_id", int64(47), "select", selected, "cot", cot)))
}

func TestTypedRoutesReachTheIED(t *testing.T) {
	m, typed, operator := typedManager(t)
	results := []string{}
	for _, run := range []struct {
		id    string
		value any
		cot   int64
		sel   bool
	}{
		{"tap", int64(2), 6, true}, {"tap", int64(2), 6, false}, // select, then execute completes it
		{"sp", 125.0, 6, true}, {"sp", 125.0, 8, true}, // select, then cancel (COT 8)
		{"sp", 50.0, 6, false},                         // direct execute of a set point
		{"tap", int64(0), 6, false},                    // invalid step state: refused, nothing sent
		{"cb", true, 6, false},                         // a direct double command keeps the old path
		{"nothing", true, 6, false},
	} {
		ok, reason := step(m, run.id, run.value, run.cot, run.sel)
		results = append(results, map[bool]string{true: "+", false: "-"}[ok]+reason)
	}
	want := []string{"+", "+", "+", "+", "+", "-the step must be lower (1) or higher / raise (2); 0 and 3 are not allowed",
		"+", "-this point is not in the gateway point list"}
	if strings.Join(results, "|") != strings.Join(want, "|") {
		t.Fatalf("results\n%s", strings.Join(results, "\n"))
	}
	calls := []string{`select BayCTRL/ATCC1.TapChg "higher"`, `operate BayCTRL/ATCC1.TapChg "higher"`,
		`select BayCTRL/ATCC1.BndCtr 12.5`, `cancel BayCTRL/ATCC1.BndCtr`, `operate BayCTRL/ATCC1.BndCtr 5.0`}
	if strings.Join(typed.calls, "|") != strings.Join(calls, "|") || len(operator.calls) != 1 {
		t.Fatalf("calls %v legacy %v", typed.calls, operator.calls)
	}
}

func TestTypedRefusalsAndFallback(t *testing.T) {
	m, typed, operator := typedManager(t)
	typed.selectErr = apperr.Coded("Select is not available yet", "not_available")
	if ok, _ := step(m, "cb", true, 6, true); !ok { // single/double: confirmed by ScadaBench, as before
		t.Fatal("double select fallback refused")
	}
	if ok, _ := step(m, "cb", true, 6, false); !ok || len(operator.calls) != 1 {
		t.Fatal("double execute after a fallback select must take the old path")
	}
	if ok, reason := step(m, "tap", int64(1), 6, true); ok || !strings.Contains(reason, "the IED refused the select") {
		t.Fatalf("typed select must be refused: %v %q", ok, reason)
	}
	typed.operate = errors.New("blocked by interlocking")
	if ok, reason := step(m, "sp", 10.0, 6, false); ok || reason != "the IED did not operate: blocked by interlocking" {
		t.Fatalf("operate refusal %v %q", ok, reason)
	}
	if ok, reason := step(m, "tap", int64(1), 3, false); ok || !strings.Contains(reason, "cause of transmission") {
		t.Fatalf("wrong cause %v %q", ok, reason)
	}
	entries := get(m.Commands.After(0, 200), "entries").([]any)
	last := entries[len(entries)-1].(*pyjson.Object)
	if !strings.HasPrefix(get(last, "applied").(string), "refused: the cause of transmission") ||
		get(last, "label") != "BayCTRL/ATCC1.TapChg lower" {
		t.Fatalf("log %v", last)
	}
}
