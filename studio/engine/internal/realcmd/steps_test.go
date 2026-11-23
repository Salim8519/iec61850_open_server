package realcmd

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/typedcmd"
)

type typedFake struct {
	answer *pyjson.Object
	ops    []string
	sent   []*pyjson.Object
}

func (f *typedFake) Operate(string, any, string, any, any) (*pyjson.Object, error) { return f.answer, nil }
func (f *typedFake) Typed(op string, fields *pyjson.Object) (*pyjson.Object, error) {
	f.ops = append(f.ops, op)
	f.sent = append(f.sent, fields)
	return f.answer, nil
}

// readings answers ReadOne per reference: a feedback point and the node's Beh.stVal.
type readings map[string]any

func (r readings) State() string { return "live" }
func (r readings) ReadOne(ref, fc string) (any, bool, error) {
	value, found := r[ref]
	return value, found, nil
}

func relay(local string) *pyjson.Object {
	live := pyjson.NewObject().Set("ip", "172.26.0.40").Set("port", int64(102)).Set("allow_commands", true)
	if local != "" {
		live.Set("local", local)
	}
	pos := pyjson.NewObject().Set("ref", "E10/DiscCSWI3.Pos").Set("status", "E10/DiscCSWI3.Pos.stVal").
		Set("cdc", "DPC").Set("model", int64(4))
	return pyjson.NewObject().Set("id", "r1").Set("name", "E10_BCU").Set("role", "real").Set("live", live).
		Set("controls", []any{pos})
}

func typedList(*pyjson.Object) []any {
	tap := pyjson.NewObject().Set("ref", "E10/TapISC1.TapPos").Set("status", "E10/TapISC1.TapPos.valWTr.posVal").
		Set("cdc", "ISC").Set("value_kind", "int").Set("model", int64(1)).Set("min", int64(1)).Set("max", int64(17))
	mode := pyjson.NewObject().Set("ref", "E10/LLN0.Mod").Set("status", "E10/LLN0.Mod.stVal").Set("cdc", "ENC").
		Set("value_kind", "enum").Set("model", int64(1)).Set("labels", pyjson.NewObject().Set("on", int64(1)).Set("test", int64(3)))
	return []any{tap, mode}
}

func stepService(ied *pyjson.Object, values readings, fake *typedFake, events *[]string) *Service {
	return &Service{FindIED: func(string) (*pyjson.Object, error) { return ied, nil },
		Sessions: func(string) (Session, error) { return values, nil }, Control: fake, Extended: typedList,
		Emit:    func(kind string, _ *pyjson.Object) { *events = append(*events, kind) },
		Observe: 20 * time.Millisecond, Poll: time.Millisecond}
}

// Select of an SBO-enhanced DPC sends its value, the options and the chosen card; the answer
// offers Operate for the selection time minus the safety margin.
func TestSelectSendsValueOptionsAndCard(t *testing.T) {
	fake := &typedFake{answer: pyjson.NewObject().Set("ok", true).Set("selected", true).Set("expires_ms", int64(30000))}
	var events []string
	options := typedcmd.Options{InterlockCheck: true, OrCat: 2}
	answer, err := stepService(relay("172.26.0.251"), readings{}, fake, &events).Command("r1",
		Step{Action: "select", Ref: "E10/DiscCSWI3.Pos", Value: true, Options: options, Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	sent := fake.sent[0]
	for key, want := range map[string]any{"kind": "bool", "value": true, "interlock_check": true, "or_cat": int64(2),
		"local": "172.26.0.251", "ip": "172.26.0.40", "test": false} {
		if got, _ := sent.Get(key); got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}
	if fake.ops[0] != "typed_select" || get(answer, "expires_ms") != int64(27000) {
		t.Fatalf("op %v, answer %v", fake.ops, answer)
	}
	if strings.Join(events, ",") != "real_command_requested,real_command_selected" {
		t.Fatalf("events %v", events)
	}
}

// Operate after Select needs that selection; one-click Operate selects by itself.
func TestOperateUsesTheSelectionOnlyStepByStep(t *testing.T) {
	for _, selected := range []bool{true, false} {
		fake := &typedFake{answer: pyjson.NewObject().Set("ok", true).Set("model", "sbo-enhanced")}
		var events []string
		values := readings{"E10/DiscCSWI3.Pos.stVal": int64(2)}
		answer, err := stepService(relay(""), values, fake, &events).Command("r1",
			Step{Action: "operate", Ref: "E10/DiscCSWI3.Pos", Value: true, Confirmed: true, Selected: selected})
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := fake.sent[0].Get("require_select"); got != selected {
			t.Errorf("require_select %v, want %v", got, selected)
		}
		if _, has := fake.sent[0].Get("local"); has {
			t.Error("no card was chosen, so none is sent")
		}
		if get(answer, "changed") != true || get(answer, "observed") != "E10/DiscCSWI3.Pos.stVal is closed" {
			t.Errorf("observed %v", answer)
		}
	}
}

// Typed classes: an ISC Raise goes one position up from the present one; an ENC label is its number.
func TestTypedClassesSendTheirWireValue(t *testing.T) {
	cases := []struct {
		ref        string
		value      any
		kind, want any
	}{{"E10/TapISC1.TapPos", "higher", "int", int64(6)}, {"E10/LLN0.Mod", "test", "enum", int64(3)}}
	for _, c := range cases {
		fake := &typedFake{answer: pyjson.NewObject().Set("ok", true)}
		var events []string
		values := readings{"E10/TapISC1.TapPos.valWTr.posVal": int64(5)}
		if _, err := stepService(relay(""), values, fake, &events).Command("r1",
			Step{Action: "operate", Ref: c.ref, Value: c.value, Confirmed: true}); err != nil {
			t.Fatal(c.ref, err)
		}
		if kind, _ := fake.sent[0].Get("kind"); kind != c.kind {
			t.Errorf("%s kind %v", c.ref, kind)
		}
		if value, _ := fake.sent[0].Get("value"); value != c.want {
			t.Errorf("%s value %v, want %v", c.ref, value, c.want)
		}
	}
}

// A refusal says what it means: interlocking, Test mode, Local, switched off.
func TestRefusalsAreExplained(t *testing.T) {
	cases := []struct {
		cause, code int64
		beh         any
		test        bool
		want        string
	}{
		{10, 25, int64(1), false, "interlocking does not allow"},
		{0, 32, int64(3), false, "is in Test mode"},
		{2, 25, int64(1), false, "is in Local"},
		{8, 25, int64(5), false, "switched off"},
		{0, 32, int64(1), true, "not in Test mode"},
	}
	for _, c := range cases {
		fake := &typedFake{answer: pyjson.NewObject().Set("ok", false).Set("add_cause", c.cause).
			Set("client_error", c.code).Set("error", "Select refused by the device: raw")}
		var events []string
		values := readings{"E10/DiscCSWI3.Beh.stVal": c.beh}
		answer, err := stepService(relay(""), values, fake, &events).Command("r1", Step{Action: "select",
			Ref: "E10/DiscCSWI3.Pos", Value: false, Confirmed: true, Options: typedcmd.Options{Test: c.test}})
		if err != nil {
			t.Fatal(err)
		}
		summary, _ := get(answer, "summary").(string)
		if !strings.Contains(summary, c.want) || !strings.HasSuffix(summary, "Device answer: Select refused by the device: raw.") {
			t.Errorf("cause %d: %q", c.cause, summary)
		}
		if events[len(events)-1] != "real_command_failed" {
			t.Errorf("events %v", events)
		}
	}
}

// Every reason not to send is in words, and nothing reaches the device.
func TestStepRefusals(t *testing.T) {
	base := Step{Action: "operate", Ref: "E10/DiscCSWI3.Pos", Value: true, Confirmed: true}
	cases := map[string]func(*Step, *StepRequest){
		"A command step is select, operate or cancel": func(s *Step, _ *StepRequest) { s.Action = "write" },
		"Confirm the command first":                   func(s *Step, _ *StepRequest) { s.Confirmed = false },
		"E10_BCU has no command X in its CID":         func(s *Step, r *StepRequest) { s.Ref = "X"; r.Known = false },
		"Choose the value to send first":              func(s *Step, _ *StepRequest) { s.Value = nil },
		"E10_BCU is not connected":                    func(_ *Step, r *StepRequest) { r.State = "retrying" },
	}
	for want, change := range cases {
		step := base
		request := StepRequest{IED: relay(""), Known: true, State: "live"}
		change(&step, &request)
		request.Step = step
		if reason := StepRefusal(request); !strings.HasPrefix(reason, want) {
			t.Errorf("refusal %q, want %q", reason, want)
		}
	}
	cancel := StepRequest{IED: relay(""), Step: Step{Action: "cancel", Ref: "E10/DiscCSWI3.Pos"}, Known: true, State: "live"}
	if reason := StepRefusal(cancel); reason != "" {
		t.Errorf("cancel needs no confirmation: %q", reason)
	}
}

func TestBehaviourRef(t *testing.T) {
	if got := behaviourRef("E10_BCUSystem/DiscCSWI3.Pos"); got != "E10_BCUSystem/DiscCSWI3.Beh.stVal" {
		t.Fatal(got)
	}
}
