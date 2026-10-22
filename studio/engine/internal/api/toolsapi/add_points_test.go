package toolsapi

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func addIED() *pyjson.Object {
	signal := func(ref, fc, kind, description string) *pyjson.Object {
		return pyjson.NewObject().Set("ref", ref).Set("fc", fc).Set("kind", kind).Set("description", description)
	}
	return pyjson.NewObject().Set("id", "f1").Set("name", "FEED1").Set("role", "real").
		Set("signals", []any{signal("FEED1LD0/XCBR1.Pos.stVal", "ST", "Dbpos", "Breaker position"),
			signal("FEED1LD0/MMXU1.TotW.mag.f", "MX", "FLOAT32", "Active power")}).
		Set("controls", []any{pyjson.NewObject().Set("ref", "FEED1LD0/XCBR1.Pos").Set("cdc", "DPC").
			Set("status", "FEED1LD0/XCBR1.Pos.stVal")}).Set("datasets", []any{})
}

// TestAddPointRoutes: the Go-only Add points routes (ADR-0105) list the types, the fitting
// signals, plan IED points with their status, check pasted rows and add ranges; nothing is
// saved by them.
func TestAddPointRoutes(t *testing.T) {
	projects := &fakeProjects{ieds: []*pyjson.Object{addIED()}}
	srv := server(t, newDeps(projects, &fakeRuntime{state: "stopped"}, map[string]string{}))
	before := projects.Gateway()
	status, text := call(t, srv, "GET", "/tools/gateway/types?protocol=dnp3", "")
	if status != 200 || !strings.HasPrefix(text, `{"types":[{"kind":"single","most_used":true,"name":"Binary input","type_id":"g1v2 / g2v2"}`) {
		t.Fatalf("types %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/candidates", `{"protocol":"iec104","ied":"f1","kind":"command_double","points":[]}`)
	if status != 200 || !strings.Contains(text, `"name":"Breaker position"`) || !strings.Contains(text, `"role":"real"`) ||
		!strings.Contains(text, `"total":1`) {
		t.Fatalf("candidates %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/points/from-ied",
		`{"protocol":"iec104","ied":"f1","kind":"command_double","picks":[{"ref":"FEED1LD0/XCBR1.Pos"}],"with_status":true,"points":[]}`)
	if status != 200 || !strings.Contains(text, `"ok":true`) || !strings.Contains(text, `"address":50001`) ||
		!strings.Contains(text, `"address":20001`) || !strings.Contains(text, `"status_for":"f1:FEED1LD0/XCBR1.Pos:command_double"`) ||
		!strings.Contains(text, `"monitor_only":true`) {
		t.Fatalf("from-ied %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/points/rows",
		`{"protocol":"iec104","text":"Address\tType\tIED\tPoint\n\tM_ME_NC\tFEED1\tFEED1LD0/MMXU1.TotW.mag.f\n1\tnope\tFEED1\tx\n","points":[]}`)
	if status != 200 || !strings.Contains(text, `"ok":false`) || !strings.Contains(text, `"points":[]`) ||
		!strings.Contains(text, `"problem":"Row 3: \"nope\" is not a type here`) || !strings.Contains(text, `"columns":["address","type","ied","point"]`) {
		t.Fatalf("rows %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/points/new", `{"protocol":"iec104","kind":"float","count":2,"from":30010,"name":"Level","points":[]}`)
	if status != 200 || !strings.Contains(text, `"ok":true`) || !strings.Contains(text, `"address":30011`) ||
		!strings.Contains(text, `"description":"Level 2"`) {
		t.Fatalf("range %d %s", status, text)
	}
	if projects.Gateway() != before {
		t.Fatal("a planning route saved the gateway")
	}
}

// TestExcelTemplateRoutes: the template downloads as .xlsx and an upload is checked row by row.
func TestExcelTemplateRoutes(t *testing.T) {
	projects := &fakeProjects{ieds: []*pyjson.Object{addIED()}}
	srv := server(t, newDeps(projects, &fakeRuntime{state: "stopped"}, map[string]string{}))
	response, err := http.Get(srv.URL + "/api/v1/tools/gateway/points.xlsx?protocol=iec104")
	if err != nil {
		t.Fatal(err)
	}
	book, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("Content-Disposition") != "attachment; filename=gateway-iec104-template.xlsx" ||
		len(book) < 1000 || string(book[:2]) != "PK" {
		t.Fatalf("template %d %v %d", response.StatusCode, response.Header, len(book))
	}
	body := `{"protocol":"iec104","points":[],"data":"` + base64.StdEncoding.EncodeToString(book) + `"}`
	status, text := call(t, srv, "POST", "/tools/gateway/points.xlsx", body)
	if status != 400 || !strings.Contains(text, "No rows were found") { // the empty template has only its title row
		t.Fatalf("empty upload %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/points.xlsx", `{"protocol":"iec104","data":"bm90IGEgYm9vaw=="}`)
	if status != 400 || !strings.Contains(text, "not an Excel workbook") {
		t.Fatalf("bad upload %d %s", status, text)
	}
}

// TestSmartProposeRoute: without "mode" Propose answers as before; with it the preview adds a
// per-IED summary and "add_new" keeps the current list.
func TestSmartProposeRoute(t *testing.T) {
	projects := &fakeProjects{ieds: []*pyjson.Object{addIED()}}
	srv := server(t, newDeps(projects, &fakeRuntime{state: "stopped"}, map[string]string{}))
	status, text := call(t, srv, "POST", "/tools/gateway/propose", `{"protocol":"iec104"}`)
	if status != 200 || strings.Contains(text, "summary") || !strings.Contains(text, `"description":""`) {
		t.Fatalf("plain %d %s", status, text)
	}
	current := `[{"id":"f1:FEED1LD0/XCBR1.Pos.stVal:double","kind":"double","ied":"f1","ref":"FEED1LD0/XCBR1.Pos.stVal","address":20009,"scale":1.0,"description":"Mine"}]`
	status, text = call(t, srv, "POST", "/tools/gateway/propose", `{"protocol":"iec104","mode":"add_new","current":`+current+`}`)
	if status != 200 || !strings.Contains(text, `"description":"Mine"`) || !strings.Contains(text, `"address":50009`) ||
		!strings.Contains(text, `"summary":[{"id":"f1","name":"FEED1","new":2,"offered":3,"role":"real","ticked":true}]`) {
		t.Fatalf("add_new %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/propose", `{"protocol":"iec104","mode":"other"}`)
	if status != 400 {
		t.Fatalf("bad mode %d %s", status, text)
	}
}
