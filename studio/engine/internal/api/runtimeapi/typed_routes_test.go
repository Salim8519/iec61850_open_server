package runtimeapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

func firstIed(t *testing.T, project interface{ Current() *pyjson.Object }) string {
	t.Helper()
	return pyval.Str(pyval.List(project.Current(), "ieds")[0].(*pyjson.Object), "id")
}

func TestTypedRequestKeepsTheOldOperateBody(t *testing.T) {
	for body, typed := range map[string]bool{
		`{"ied_id":"a","ref":"r","value":true}`:                false,
		`{"ied_id":"a","ref":"r","value":false}`:               false,
		`{"ied_id":"a","ref":"r"}`:                             false,
		`{"ied_id":"a","ref":"r","value":3}`:                   true,
		`{"ied_id":"a","ref":"r","value":"higher"}`:            true,
		`{"ied_id":"a","ref":"r","value":true,"options":{}}`:   true,
		`{"ied_id":"a","ref":"r","value":null,"options":null}`: true,
	} {
		decoded, _ := pyjson.Decode([]byte(body))
		if typedRequest(decoded) != typed {
			t.Errorf("%s: typed=%v", body, !typed)
		}
	}
}

func TestTypedRoutesWhileStopped(t *testing.T) {
	server, _, project := newServer(t, 21320)
	id := firstIed(t, project)
	cases := []struct{ path, body, want string }{
		{"/controls/select", `{}`, "Send the IED (ied_id) and the control reference (ref)"},
		{"/controls/select", `{"ied_id":"nope","ref":"x"}`, "Unknown IED"},
		{"/controls/cancel", `{"ied_id":"` + id + `","ref":"BayACTRL/XCBR1.Pos"}`, "IED must be running to operate equipment"},
		{"/controls/operate", `{"ied_id":"` + id + `","ref":"BayACTRL/XCBR1.Pos","value":true,"options":{"orcat":9}}`,
			"Originator category must be between 0 and 8"},
		{"/controls/operate", `{"ied_id":"` + id + `","ref":"BayACTRL/XCBR1.Pos","value":true,"options":{"test":1}}`,
			"Command option test is true or false"},
		{"/controls/operate", `{"ied_id":"` + id + `","ref":"BayACTRL/XCBR1.Pos","value":true}`, "IED is not running"},
	}
	for _, c := range cases {
		status, body := call(t, server, "POST", c.path, "application/json", c.body)
		if status == 200 || !strings.Contains(body, c.want) {
			t.Errorf("%s %s: %d %s (want %q)", c.path, c.body, status, body, c.want)
		}
	}
	status, body := call(t, server, "GET", "/controls/typed?ied_id="+id, "", "")
	if status != 200 || !strings.Contains(body, `"extended_controls":[]`) || !strings.Contains(body, "BayACTRL/XCBR1.Pos") {
		t.Fatalf("typed list: %d %s", status, body)
	}
}

func TestTypedOperateOfABreakerOverHTTP(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "..", "..", "native", "build", "Release", "studio_control.exe")); err != nil {
		t.Skip("native build required")
	}
	server, _, project := newServer(t, 21330)
	id := firstIed(t, project)
	if status, body := call(t, server, "POST", "/runtime/start", "", ""); status != 200 {
		t.Fatalf("start: %d %s", status, body)
	}
	status, body := call(t, server, "POST", "/controls/select", "application/json",
		`{"ied_id":"`+id+`","ref":"BayACTRL/XCBR1.Pos","value":true}`)
	if status != 200 || !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"model":"direct-enhanced"`) ||
		!strings.Contains(body, `"expires_ms":0`) {
		t.Fatalf("select of a direct control: %d %s", status, body)
	}
	status, body = call(t, server, "POST", "/controls/operate", "application/json",
		`{"ied_id":"`+id+`","ref":"BayACTRL/XCBR1.Pos","value":true,"options":{"orident":"http-test","orcat":2}}`)
	if status != 200 || !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"terminated":true`) {
		t.Fatalf("typed operate: %d %s", status, body)
	}
	status, body = call(t, server, "POST", "/controls/operate", "application/json",
		`{"ied_id":"`+id+`","ref":"BayACTRL/XCBR1.Pos","value":2}`)
	if status != 400 || !strings.Contains(body, "Control value must be boolean") {
		t.Fatalf("a number for a breaker: %d %s", status, body)
	}
}
