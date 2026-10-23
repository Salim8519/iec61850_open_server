package goosemsg

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func dumps(t *testing.T, value any) string {
	t.Helper()
	data, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func TestMessagesNameReceiversMembersAndValues(t *testing.T) {
	partners := []Partner{
		{GocbRef: gocb1, Publisher: "PUB", Subscriber: "SUB1", Block: "GoCB1", Status: "declared"},
		{GocbRef: gocb1, Publisher: "PUB", Subscriber: "SUB2", Block: "GoCB1", Status: "declared"},
		{GocbRef: gocb1, Publisher: "PUB", Subscriber: "SUB1", Block: "GoCB1", Status: "declared"},
		{GocbRef: "FAR/LLN0$GO$X", Publisher: "FAR", Subscriber: "PUB", Block: "X", Status: "remote"},
		{GocbRef: "SUB1LD0/LLN0$GO$Back", Publisher: "SUB1", Subscriber: "PUB", Block: "Back", Status: "declared"},
	}
	values := map[string]any{pos: int64(2)}
	value := func(ref string) (any, bool) { found, ok := values[ref]; return found, ok }
	answer := Messages(Build(publisher()), "simulated", partners, value, SendState{Running: true})
	text := dumps(t, answer)
	for _, want := range []string{
		`"can_send": true, "why_not": ""`,
		`"subscribers": ["SUB1", "SUB2"], "complete": true`,
		`{"position": 1, "text": "XCBR1.Pos [ST]", "fc": "ST", "refs": [{"ref": "PUBLD0/XCBR1.Pos.q", "kind": "Quality", "fc": "ST", "value_type": null, "writable": true, "sendable": false`,
		`{"ref": "PUBLD0/XCBR1.Pos.stVal", "kind": "Dbpos", "fc": "ST", "value_type": "position", "writable": true, "sendable": true, "description": "desc of PUBLD0/XCBR1.Pos.stVal", "value": 2}`,
		`"name": "GoCB2", "dataset": "DS2", "go_id": "PUB/GoCB2", "app_id": 4097, "conf_rev": 1, "subscribers": [], "complete": false, "members": []`,
		`"receives": [{"gocb_ref": "FAR/LLN0$GO$X", "publisher": "FAR", "block": "X", "status": "remote"}, {"gocb_ref": "SUB1LD0/LLN0$GO$Back", "publisher": "SUB1", "block": "Back", "status": "declared"}]`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("answer lacks %s\n%s", want, text)
		}
	}
	stopped := dumps(t, Messages(Build(publisher()), "simulated", nil, value, SendState{Refusal: NotRunning}))
	if !strings.Contains(stopped, `"value": null`) || strings.Contains(stopped, `"value": 2`) ||
		!strings.Contains(stopped, `"can_send": false, "why_not": "Run the IEDs to send GOOSE.`) {
		t.Fatalf("stopped: no values and a reason\n%s", stopped)
	}
}

func TestRefusalWords(t *testing.T) {
	cases := []struct {
		change func(*pyjson.Object)
		inRun  bool
		want   string
	}{
		{func(*pyjson.Object) {}, true, ""},
		{func(*pyjson.Object) {}, false, "PUB is not running. Run this IED first."},
		{func(ied *pyjson.Object) { ied.Set("role", "real") }, true, "PUB is a real device. ScadaBench only watches its GOOSE"},
		{func(ied *pyjson.Object) { ied.Set("role", "off") }, true, "PUB is skipped"},
		{func(ied *pyjson.Object) { ied.Set("goose_publish", false) }, true, "'Send its GOOSE messages' is off in Devices"},
	}
	for _, item := range cases {
		copy := publisher()
		item.change(copy)
		if got := Refusal(copy, item.inRun); !strings.Contains(got, item.want) || (item.want == "" && got != "") {
			t.Errorf("%q, want %q", got, item.want)
		}
	}
}

func TestLeafRefsListEveryMessageOfEveryIED(t *testing.T) {
	plain := o("id", "q", "name", "PLAIN", "signals", []any{}, "goose", o("publishes", []any{}, "subscribes", []any{}))
	text := dumps(t, LeafRefs([]*pyjson.Object{publisher(), plain}))
	if !strings.HasPrefix(text, `{"ieds": {"p": {"PUBLD0/LLN0$GO$GoCB1": ["PUBLD0/XCBR1.Pos.q", "PUBLD0/XCBR1.Pos.stVal", "PUBLD0/XCBR1.Pos.t", "PUBLD0/QA1XSWI1.Pos.stVal"`) ||
		!strings.Contains(text, `"PUBLD0/LLN0$GO$GoCB2": []`) || strings.Contains(text, `"q"`) {
		t.Fatalf("leaf refs %s", text)
	}
}
