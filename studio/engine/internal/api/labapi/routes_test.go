package labapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/labnet"
	"studio-engine/internal/labsetup"
)

type fakeLab struct{ chosen map[string]string }

func (f *fakeLab) Status(chosen map[string]string) (labsetup.Status, error) {
	f.chosen = chosen
	return labsetup.Status{CardName: "ScadaBench Lab", Rows: []labsetup.Row{{IEDID: "a", IP: "10.0.0.1", Status: "ok"}},
		Reals: []labsetup.Real{}, CardIPs: []string{}}, nil
}
func (f *fakeLab) Install() (labsetup.Status, error) { return f.Status(nil) }
func (f *fakeLab) Use(chosen map[string]string) (labsetup.Status, error) {
	return f.Status(chosen)
}
func (f *fakeLab) SetAddress(id, address string) (labsetup.Status, error) {
	if address == "bad" {
		return labsetup.Status{}, apperr.New("Not an address ScadaBench can use")
	}
	return f.Status(map[string]string{id: address})
}
func (f *fakeLab) Ping(string) (labnet.Pinged, error)                 { return labnet.Pinged{OK: true, TTL: 128}, nil }
func (f *fakeLab) Remove(leave func() error) (labsetup.Status, error) { return f.Status(nil) }

func serve(t *testing.T, lab *fakeLab, method, path, body string) (int, string) {
	t.Helper()
	for _, route := range New(Deps{Lab: lab, Leave: func() error { return nil }}).Routes() {
		if route.Method == method && route.Pattern == path {
			request := httptest.NewRequest(method, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			route.Handler.ServeHTTP(recorder, request)
			return recorder.Code, recorder.Body.String()
		}
	}
	t.Fatalf("no route %s %s", method, path)
	return 0, ""
}

func TestStatusAndTypedCheck(t *testing.T) {
	lab := &fakeLab{}
	code, body := serve(t, lab, "GET", Prefix, "")
	if code != http.StatusOK || !strings.Contains(body, `"card_name":"ScadaBench Lab"`) || !strings.Contains(body, `"rows":[`) {
		t.Fatalf("%d %s", code, body)
	}
	code, _ = serve(t, lab, "POST", Prefix+"/check", `{"addresses":{"a":"10.0.0.9"}}`)
	if code != http.StatusOK || lab.chosen["a"] != "10.0.0.9" {
		t.Fatalf("%d %v", code, lab.chosen)
	}
}

func TestAddressNeedsAnIEDAndReportsRefusals(t *testing.T) {
	lab := &fakeLab{}
	if code, body := serve(t, lab, "PUT", Prefix+"/address", `{"ip":"10.0.0.2"}`); code != 400 || !strings.Contains(body, "ied_id") {
		t.Fatalf("%d %s", code, body)
	}
	if code, body := serve(t, lab, "PUT", Prefix+"/address", `{"ied_id":"a","ip":"bad"}`); code != 400 || !strings.Contains(body, "Not an address") {
		t.Fatalf("%d %s", code, body)
	}
	if code, _ := serve(t, lab, "PUT", Prefix+"/address", `{"ied_id":"a","ip":"10.0.0.2"}`); code != 200 || lab.chosen["a"] != "10.0.0.2" {
		t.Fatalf("%d %v", code, lab.chosen)
	}
	if code, body := serve(t, lab, "POST", Prefix+"/use", `not json`); code != 400 {
		t.Fatalf("%d %s", code, body)
	}
}

func TestPingAndGoOnly(t *testing.T) {
	if code, body := serve(t, &fakeLab{}, "POST", Prefix+"/ping", `{"ip":"10.0.0.1"}`); code != 200 || body != `{"ms":0,"ok":true,"ttl":128}`+"\n" {
		t.Fatalf("%d %q", code, body)
	}
	if !GoOnly("GET /api/v1/network/lab") || !GoOnly("PUT /api/v1/network/lab/address") || GoOnly("GET /api/v1/network/labs") {
		t.Fatal("GoOnly is wrong")
	}
}
