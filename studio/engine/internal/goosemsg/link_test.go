package goosemsg

import (
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

const (
	pos  = "PUBLD0/XCBR1.Pos.stVal"
	ind1 = "PUBLD0/GGIO1.Ind1.stVal"
	ind2 = "PUBLD0/GGIO1.Ind2.stVal"
	totW = "PUBLD0/MMXU1.TotW.mag.f"
)

func message(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Check refuses with plain words what cannot be sent as saved.
func TestLinkCheckRefusals(t *testing.T) {
	catalog := Build(publisher())
	cases := []struct {
		kind string
		link Link
		want string
	}{
		{"CB", Link{GocbRef: "PUBLD0/LLN0$GO$Nope", Mode: "one", One: pos}, "PUB does not send the GOOSE message"},
		{"CT", Link{GocbRef: gocb1, Mode: "contacts", Open: ind1, Closed: ind2}, "Only a breaker, disconnector or earth switch"},
		{"CB", Link{GocbRef: gocb1, Mode: "contacts", Open: ind1, Closed: ind1}, "two different values"},
		{"CB", Link{GocbRef: gocb1, Mode: "contacts", Open: ind1}, "Choose the open value and the closed value"},
		{"CB", Link{GocbRef: gocb1, Mode: "contacts", Open: pos, Closed: ind2}, "must be an on/off (BOOLEAN) value"},
		{"CB", Link{GocbRef: gocb1, Mode: "one"}, "Choose the one value"},
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: "PUBLD0/XCBR1.Pos.ctlModel"}, "is not part of the GOOSE message GoCB1"},
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: "PUBLD0/XCBR1.PosX.stVal"}, "is not part of the GOOSE message"},
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: totW}, "a switch sends a position (Dbpos) or one on/off value"},
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: "PUBLD0/XCBR1.Pos.q"}, "cannot be sent from the drawing"},
		{"PUSH_BUTTON", Link{GocbRef: gocb1, Mode: "one", One: "PUBLD0/GGIO1.Ind3.stVal"}, "read-only in the simulated IED"},
	}
	for _, item := range cases {
		if got := message(item.link.Check(item.kind, catalog)); !strings.Contains(got, item.want) {
			t.Errorf("%s %+v: %q, want %q", item.kind, item.link, got, item.want)
		}
	}
	good := []struct {
		kind string
		link Link
	}{
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: pos}},
		{"DS", Link{GocbRef: gocb1, Mode: "one", One: ind1}},
		{"ES", Link{GocbRef: gocb1, Mode: "contacts", Open: ind1, Closed: ind2}},
		{"SOURCE", Link{GocbRef: gocb1, Mode: "one", One: totW}},
		{"PUSH_BUTTON", Link{GocbRef: gocb1, Mode: "one", One: ind2}},
		{"CB", Link{GocbRef: gocb1, Mode: "one", One: "BAYCTRL/CSWI1.Pos.stVal"}},
	}
	for _, item := range good {
		if err := item.link.Check(item.kind, catalog); err != nil {
			t.Errorf("%s %+v must pass: %v", item.kind, item.link, err)
		}
	}
}

// One action is one batch: both contacts together, a position as 1/2, a switch's one on/off value
// as false/true, numbers whole for integer kinds.
func TestLinkPlansOneBatchPerAction(t *testing.T) {
	block := Build(publisher()).Block(gocb1)
	contacts := Link{GocbRef: gocb1, Mode: "contacts", Open: ind1, Closed: ind2}
	cases := []struct {
		kind   string
		link   Link
		action string
		value  any
		want   []Write
	}{
		{"CB", contacts, "open", nil, []Write{{ind1, true}, {ind2, false}}},
		{"CB", contacts, "closed", nil, []Write{{ind1, false}, {ind2, true}}},
		{"CB", Link{Mode: "one", One: pos}, "open", nil, []Write{{pos, int64(1)}}},
		{"DS", Link{Mode: "one", One: pos}, "closed", nil, []Write{{pos, int64(2)}}},
		{"ES", Link{Mode: "one", One: ind1}, "closed", nil, []Write{{ind1, true}}},
		{"ES", Link{Mode: "one", One: ind1}, "open", nil, []Write{{ind1, false}}},
		{"PUSH_BUTTON", Link{Mode: "one", One: ind1}, "on", nil, []Write{{ind1, true}}},
		{"SOURCE", Link{Mode: "one", One: totW}, "value", 49.5, []Write{{totW, 49.5}}},
	}
	for _, item := range cases {
		got, err := item.link.Plan(item.kind, block, item.action, item.value)
		if err != nil || !reflect.DeepEqual(got, item.want) {
			t.Errorf("%s %s: %v %v, want %v", item.kind, item.action, got, err, item.want)
		}
	}
	refused := []struct {
		kind   string
		link   Link
		action string
		value  any
		want   string
	}{
		{"CB", contacts, "on", nil, "sends Open or Closed"},
		{"CB", Link{Mode: "one", One: pos}, "on", nil, "is a position: send Open or Closed"},
		{"PUSH_BUTTON", Link{Mode: "one", One: ind1}, "open", nil, "send On or Off"},
		{"SOURCE", Link{Mode: "one", One: totW}, "on", nil, "type a number"},
		{"SOURCE", Link{Mode: "one", One: totW}, "value", "49", "enter a number"},
	}
	for _, item := range refused {
		if _, err := item.link.Plan(item.kind, block, item.action, item.value); !strings.Contains(message(err), item.want) {
			t.Errorf("%s %s: %q, want %q", item.kind, item.action, message(err), item.want)
		}
	}
}

// The IED component sends one value of its own message, typed by the leaf.
func TestPlanValueFromTheIEDComponent(t *testing.T) {
	block := Build(publisher()).Block(gocb1)
	for _, item := range []struct {
		ref   string
		value any
		want  any
	}{{pos, int64(2), int64(2)}, {pos, 3.0, int64(3)}, {ind2, true, true}, {totW, int64(12), 12.0}} {
		got, err := PlanValue(block, item.ref, item.value)
		if err != nil || !reflect.DeepEqual(got, []Write{{item.ref, item.want}}) {
			t.Errorf("%s %v: %v %v", item.ref, item.value, got, err)
		}
	}
	for _, item := range []struct {
		ref   string
		value any
		want  string
	}{{pos, int64(4), "0 (intermediate), 1 (open), 2 (closed) or 3 (bad)"}, {ind2, int64(1), "send ON or OFF"},
		{"PUBLD0/XCBR1.Pos.t", int64(1), "cannot be sent from the drawing"}, {"PUBLD0/GGIO1.Cnt.stVal", int64(1), "not part of"}} {
		if _, err := PlanValue(block, item.ref, item.value); !strings.Contains(message(err), item.want) {
			t.Errorf("%s: %q, want %q", item.ref, message(err), item.want)
		}
	}
	integer := Leaf{Ref: "LD/GGIO1.Cnt.stVal", Kind: "INT32", FC: "ST", Writable: true}
	if _, err := wireValue(integer, 1.5); !strings.Contains(message(err), "whole numbers") {
		t.Fatalf("integer leaf: %v", err)
	}
}

func TestParseLinkRoundTrip(t *testing.T) {
	stored := o("ied_id", "p", "gocb_ref", gocb1, "mode", "contacts", "one", nil, "open", ind1, "closed", ind2)
	link := ParseLink(stored)
	if link == nil || !reflect.DeepEqual(link.Refs(), []string{ind1, ind2}) {
		t.Fatalf("parsed %+v", link)
	}
	got, _ := pyjson.Dumps(link.Object())
	want, _ := pyjson.Dumps(stored)
	if string(got) != string(want) {
		t.Fatalf("round trip %s, want %s", got, want)
	}
	if ParseLink(nil) != nil {
		t.Fatal("no link")
	}
}
