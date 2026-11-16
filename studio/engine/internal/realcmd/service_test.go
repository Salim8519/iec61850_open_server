package realcmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func breaker(allow any) *pyjson.Object {
	live := pyjson.NewObject().Set("ip", "127.50.0.2").Set("port", int64(102))
	if allow != nil {
		live.Set("allow_commands", allow)
	}
	control := pyjson.NewObject().Set("ref", "LD/CSWI1.Pos").Set("status", "LD/CSWI1.Pos.stVal").Set("cdc", "DPC")
	typed := pyjson.NewObject().Set("ref", "LD/ATCC1.TapChg").Set("cdc", "BSC")
	return pyjson.NewObject().Set("id", "r1").Set("name", "Bay1").Set("role", "real").Set("live", live).
		Set("controls", []any{control, typed})
}

// Every reason not to send is said in words, and nothing reaches the device (ADR-0138).
func TestRefusalsAreInWords(t *testing.T) {
	plain := Request{IED: breaker(true), Ref: "LD/CSWI1.Pos", Value: true, Confirmed: true, State: "live"}
	if reason := Refusal(plain); reason != "" {
		t.Fatalf("a confirmed command with the switch on is sent: %q", reason)
	}
	cases := map[string]func(*Request){
		"Confirm the command first":                                     func(r *Request) { r.Confirmed = false },
		"Bay1 is not set to Monitor real device, so nothing was sent":   func(r *Request) { r.IED = breaker(true).Set("role", "simulated") },
		"Bay1 has Allow commands off; turn it on in Devices first":      func(r *Request) { r.IED = breaker(nil) },
		"A command here is on/off or close/open":                        func(r *Request) { r.Value = int64(1) },
		"Bay1 has no on/off or open/close command LD/ATCC1.TapChg":      func(r *Request) { r.Ref = "LD/ATCC1.TapChg" },
		"Bay1 is not connected; press Run and wait until it shows Live": func(r *Request) { r.State = "retrying" },
	}
	for want, change := range cases {
		request := plain
		change(&request)
		if reason := Refusal(request); !strings.HasPrefix(reason, want) {
			t.Errorf("refusal %q, want %q", reason, want)
		}
	}
}

type fakeSession struct {
	state  string
	values []any
	reads  int
}

func (f *fakeSession) State() string { return f.state }
func (f *fakeSession) ReadOne(ref, fc string) (any, bool, error) {
	value := f.values[min(f.reads, len(f.values)-1)]
	f.reads++
	return value, true, nil
}

type fakeControl struct {
	answer *pyjson.Object
	asked  []string
}

func (f *fakeControl) Operate(host string, port any, ref string, value, options any) (*pyjson.Object, error) {
	f.asked = append(f.asked, host+" "+ref)
	if f.answer == nil {
		return nil, errors.New("no association")
	}
	return f.answer, nil
}

func service(ied *pyjson.Object, session *fakeSession, operator *fakeControl, events *[]string) *Service {
	return &Service{FindIED: func(string) (*pyjson.Object, error) { return ied, nil },
		Sessions: func(string) (Session, error) { return session, nil }, Control: operator,
		Emit:    func(kind string, payload *pyjson.Object) { *events = append(*events, kind) },
		Observe: 50 * time.Millisecond, Poll: time.Millisecond}
}

// A confirmed command goes to the device's address, and request, applied and observed are
// three separate events with one correlation id.
func TestCommandIsSentAndObserved(t *testing.T) {
	events := []string{}
	operator := &fakeControl{answer: pyjson.NewObject().Set("ok", true).Set("model", "sbo-with-enhanced-security").Set("selected", true)}
	session := &fakeSession{state: "live", values: []any{int64(1), int64(2)}}
	answer, err := service(breaker(true), session, operator, &events).Operate("r1", "LD/CSWI1.Pos", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(operator.asked, ","); got != "127.50.0.2 LD/CSWI1.Pos" {
		t.Fatalf("sent to %q", got)
	}
	if observed, _ := answer.Get("observed"); observed != "LD/CSWI1.Pos.stVal is closed" {
		t.Fatalf("observed %v", observed)
	}
	if got := strings.Join(events, ","); got != "real_command_requested,real_command_applied,real_command_observed" {
		t.Fatalf("events %s", got)
	}
	// The device refuses: no observation, ok false with its reason.
	events = nil
	operator.answer = pyjson.NewObject().Set("ok", false).Set("error", "blocked by switching hierarchy (local/remote)")
	answer, err = service(breaker(true), session, operator, &events).Operate("r1", "LD/CSWI1.Pos", false, true)
	if ok, _ := answer.Get("ok"); err != nil || ok != false || strings.Join(events, ",") != "real_command_requested,real_command_failed" {
		t.Fatalf("refused by the device: %v %v %v", err, answer, events)
	}
	// Refused here: the device is never asked.
	operator.asked = nil
	if _, err := service(breaker(false), session, operator, &events).Operate("r1", "LD/CSWI1.Pos", true, true); err == nil || len(operator.asked) != 0 {
		t.Fatalf("the switch off sends nothing: %v %v", err, operator.asked)
	}
	// A position that does not move is said as such.
	operator.answer = pyjson.NewObject().Set("ok", true)
	answer, _ = service(breaker(true), &fakeSession{state: "live", values: []any{int64(0)}}, operator, &events).
		Operate("r1", "LD/CSWI1.Pos", true, true)
	if observed, _ := answer.Get("observed"); !strings.HasPrefix(observed.(string), "LD/CSWI1.Pos.stVal still intermediate after") {
		t.Fatalf("not moved: %v", observed)
	}
}
