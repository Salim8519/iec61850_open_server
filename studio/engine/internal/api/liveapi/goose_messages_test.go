package liveapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// GET /ieds/{id}/goose/messages and GET /goose/members (ADR-0112) on the live area's fakes:
// receivers come from the declared link map, values only while running, refusals in words.

func goosePublisher(id, name, prefix string) *pyjson.Object {
	members := []any{"XCBR1$ST$Pos", "GGIO1$ST$Ind1$stVal"}
	signal := func(ref, kind, fc string, value any) any {
		return pyv.O("ref", ref, "fc", fc, "kind", kind, "value", value, "description", "", "writable", fc != "CF")
	}
	return pyv.O("id", id, "name", name, "role", "simulated", "signals", []any{
		signal(prefix+"/XCBR1.Pos.stVal", "Dbpos", "ST", int64(1)), signal(prefix+"/XCBR1.Pos.q", "Quality", "ST", int64(0)),
		signal(prefix+"/XCBR1.Pos.ctlModel", "Enum", "CF", int64(1)), signal(prefix+"/GGIO1.Ind1.stVal", "BOOLEAN", "ST", false)},
		"datasets", []any{pyv.O("name", "DS", "members", members, "ref", prefix+"/LLN0$DS",
			"fcda_refs", []any{prefix + "/XCBR1.Pos", prefix + "/GGIO1.Ind1.stVal"})},
		"goose", pyv.O("publishes", []any{pyv.O("name", "G1", "gocb_ref", prefix+"/LLN0$GO$G1", "dataset", "DS",
			"dataset_ref", prefix+"/LLN0$DS", "go_id", "G1", "app_id", int64(1), "conf_rev", int64(2), "members", members)},
			"subscribes", []any{}))
}

func gooseEnv(t *testing.T) *env {
	e := newEnv(t, pyv.O("now", "2026-10-23T00:00:00Z", "monotonic", 1.0, "script", []any{}, "adapters", []any{},
		"npcap", pyv.O(), "version", "0.1.0"))
	t.Cleanup(e.service.Live.Stop)
	e.projects.project = pyv.O("ieds", []any{goosePublisher("p", "PUB", "PUBLD0"), goosePublisher("s", "SUB", "SUBLD0")},
		"diagram", pyv.O("nodes", []any{}, "edges", []any{}))
	e.decls = goosedom.DeclarationsFromJSON(pyv.O("subscriptions", pyv.O("SUB", []any{pyv.O("publisher", "PUB",
		"gocb_ref", "PUBLD0/LLN0$GO$G1", "name", "G1", "status", "sendable", "inputs", []any{})})))
	return e
}

func get(t *testing.T, e *env, path string) (int, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	e.mux.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
	return recorder.Code, recorder.Body.String()
}

func TestGooseMessagesRoute(t *testing.T) {
	e := gooseEnv(t)
	status, body := get(t, e, "/api/v1/ieds/p/goose/messages")
	// Answers are jsonify'd: compact, keys sorted.
	for _, want := range []string{`"can_send":false`, `"why_not":"Run the IEDs to send GOOSE. Press Run, then try again."`,
		`"subscribers":["SUB"]`, `"text":"XCBR1.Pos [ST]"`,
		`"kind":"Dbpos","ref":"PUBLD0/XCBR1.Pos.stVal","sendable":true,"value":null,"value_type":"position"`} {
		if status != 200 || !strings.Contains(body, want) {
			t.Fatalf("stopped answer lacks %s: %d %s", want, status, body)
		}
	}
	if strings.Contains(body, "ctlModel") {
		t.Fatal("Pos.ctlModel [CF] is not a member of the ST object")
	}
	if _, body := get(t, e, "/api/v1/ieds/s/goose/messages"); !strings.Contains(body,
		`"receives":[{"block":"G1","gocb_ref":"PUBLD0/LLN0$GO$G1","publisher":"PUB","status":"declared"}]`) {
		t.Fatalf("the subscriber lists what it receives: %s", body)
	}
	e.runtime.state = "running"
	e.values.seed(e.projects.project)
	if _, body := get(t, e, "/api/v1/ieds/p/goose/messages"); !strings.Contains(body, `"can_send":true`) ||
		!strings.Contains(body, `"why_not":""`) || !strings.Contains(body, `"sendable":true,"value":1,`) {
		t.Fatalf("running: values and can send: %s", body)
	}
	real := pyv.Shallow(e.projects.project)
	ieds := pyv.GetList(real, "ieds")
	ieds[0] = pyv.Shallow(pyv.Obj(ieds[0])).Set("role", "real")
	e.projects.project = real.Set("ieds", ieds)
	if _, body := get(t, e, "/api/v1/ieds/p/goose/messages"); !strings.Contains(body, `"why_not":"PUB is a real device.`) {
		t.Fatalf("a real device is watched only: %s", body)
	}
	if status, body := get(t, e, "/api/v1/ieds/nope/goose/messages"); status != 400 || !strings.Contains(body, `"code":"not_found"`) {
		t.Fatalf("unknown IED: %d %s", status, body)
	}
}

func TestGooseMembersRoute(t *testing.T) {
	e := gooseEnv(t)
	status, body := get(t, e, "/api/v1/goose/members")
	want := `{"ieds":{"p":{"PUBLD0/LLN0$GO$G1":["PUBLD0/XCBR1.Pos.q","PUBLD0/XCBR1.Pos.stVal","PUBLD0/GGIO1.Ind1.stVal"]},` +
		`"s":{"SUBLD0/LLN0$GO$G1":["SUBLD0/XCBR1.Pos.q","SUBLD0/XCBR1.Pos.stVal","SUBLD0/GGIO1.Ind1.stVal"]}}}`
	if status != 200 || strings.TrimSpace(body) != want {
		t.Fatalf("members: %d %s", status, body)
	}
}
