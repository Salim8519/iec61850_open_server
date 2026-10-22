package toolsapi

import (
	"strings"
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

func masterProject() *fakeProjects {
	section := gateway.EmptyGateway().Set("role", "master")
	point := func(kind string, address int64, name string) *pyjson.Object {
		return pyjson.NewObject().Set("id", "f1:FEED1LD0/XCBR1.Pos.stVal:"+kind).Set("kind", kind).Set("ied", "f1").
			Set("ref", "FEED1LD0/XCBR1.Pos.stVal").Set("address", address).Set("scale", 1.0).Set("description", name).
			Set("ied_name", "FEED1")
	}
	points, _ := section.Get("points")
	points.(*pyjson.Object).Set("iec104", []any{point("double", 20001, "Feeder 1 breaker"),
		point("command_double", 50001, "Feeder 1 breaker command")})
	return &fakeProjects{section: section, ieds: []*pyjson.Object{addIED()}}
}

// TestMasterListRoutes: the Go-only master point list (ADR-0110) is planned, saved, compared
// and exported; planning routes never save; Check against IED reads the gateway's mapping.
func TestMasterListRoutes(t *testing.T) {
	projects := masterProject()
	srv := server(t, newDeps(projects, &fakeRuntime{state: "stopped"}, map[string]string{}))
	status, text := call(t, srv, "GET", "/tools/master/list", "")
	if status != 200 || !strings.Contains(text, `"points":[],"protocol":"iec104"`) || !strings.Contains(text, `"kind":"single","most_used":true,"name":"Single point"`) ||
		!strings.Contains(text, `"check_ied":false`) {
		t.Fatalf("list %d %s", status, text)
	}
	before := projects.Gateway()
	status, text = call(t, srv, "POST", "/tools/master/list/copy-gateway", "{}")
	if status != 200 || !strings.Contains(text, `"ok":true`) || strings.Contains(text, `"ied"`) ||
		!strings.Contains(text, `"from":{"address":20001,"kind":"double","protocol":"iec104"}`) {
		t.Fatalf("copy %d %s", status, text)
	}
	if projects.Gateway() != before {
		t.Fatal("a planning route saved the project")
	}
	points, _ := decode(t, text).Get("points")
	saved, _ := pyjson.Compact(pyjson.NewObject().Set("points", points))
	status, text = call(t, srv, "PUT", "/tools/master/list", string(saved))
	if status != 200 || !strings.Contains(text, `"name":"FEED1 Feeder 1 breaker command","offset":0.0,"scale":1.0,"select":true,"type":"command_double"`) {
		t.Fatalf("save %d %s", status, text)
	}
	status, text = call(t, srv, "PUT", "/tools/master/list", `{"points":[{"type":"double","address":20001},{"type":"single","address":20001}]}`)
	if status != 400 || !strings.Contains(text, "Point 2: IOA 20001 is already used by point 1") {
		t.Fatalf("a clash is refused %d %s", status, text)
	}
	status, text = call(t, srv, "GET", "/tools/master/list/compare", "")
	if status != 200 || !strings.Contains(text, `"state":"waiting"`) || !strings.Contains(text, `"state":"command"`) ||
		!strings.Contains(text, `"command":{"address":50001,"kind":"double","select":true,"value":2}`) {
		t.Fatalf("compare %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/master/list/new", `{"type":"single","count":2,"from":20001,"name":"Alarm"}`)
	if status != 200 || !strings.Contains(text, `"ok":false`) || !strings.Contains(text, `IOA 20001 is already used by \"FEED1 Feeder 1 breaker\"`) {
		t.Fatalf("range %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/master/list/rows", `{"text":"Name\tAddress\tType\nTrip\t10001\tM_SP_NA_1\n"}`)
	if status != 200 || !strings.Contains(text, `"ok":true`) || !strings.Contains(text, `"columns":["name","address","type"]`) {
		t.Fatalf("rows %d %s", status, text)
	}
	if status, text = call(t, srv, "GET", "/tools/master/list.xlsx", ""); status != 200 || !strings.HasPrefix(text, "PK") {
		t.Fatalf("template %d", status)
	}
	if status, text = call(t, srv, "GET", "/tools/master/list.csv", ""); status != 200 ||
		!strings.Contains(text, "FEED1 Feeder 1 breaker,20001,Double point (M_DP 3/31)") {
		t.Fatalf("csv %d %s", status, text)
	}
	if status, text = call(t, srv, "PUT", "/tools/master/list/check-ied", `{"on":true}`); status != 200 {
		t.Fatalf("check-ied %d %s", status, text)
	}
	status, text = call(t, srv, "GET", "/tools/master/list/compare", "")
	if status != 200 || !strings.Contains(text, `"check":{"ok":null,"source":"FEED1 FEED1LD0/XCBR1.Pos.stVal","text":"The IED has no value now`) ||
		!strings.Contains(text, `"source":"FEED1 FEED1LD0/XCBR1.Pos.stVal"`) {
		t.Fatalf("check against IED %d %s", status, text)
	}
	if status, text = call(t, srv, "GET", "/tools/master/config", ""); status != 200 || strings.Contains(text, "point_list") {
		t.Fatalf("the frozen config answer is unchanged %d %s", status, text)
	}
}
