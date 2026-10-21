package hostkeyapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/pyjson"
)

type fakeLicense struct {
	info licensing.HostKeyInfo
	code string
	err  error
}

func (f *fakeLicense) HostKeyInfo() licensing.HostKeyInfo { return f.info }
func (f *fakeLicense) ActivationCode() (string, error)    { return f.code, f.err }

type fakeSharing struct {
	state             hostkey.State
	enabled, disabled int
	err               error
}

func (f *fakeSharing) State() hostkey.State { return f.state }
func (f *fakeSharing) Enable() error {
	f.enabled++
	f.state.Enabled = f.err == nil
	return f.err
}
func (f *fakeSharing) Disable() error { f.disabled++; f.state.Enabled = false; return nil }

type fakeFirewall struct{ confirmed []any }

func (f *fakeFirewall) Preview() (*pyjson.Object, error) {
	return pyjson.NewObject().Set("fingerprint", "fp1").Set("open", false), nil
}
func (f *fakeFirewall) Apply(fingerprint, confirm any) (*pyjson.Object, error) {
	f.confirmed = []any{fingerprint, confirm}
	return pyjson.NewObject().Set("applied", true), nil
}

type rig struct {
	license  *fakeLicense
	sharing  *fakeSharing
	firewall *fakeFirewall
	mux      *http.ServeMux
}

func newRig() *rig {
	r := &rig{license: &fakeLicense{info: licensing.HostKeyInfo{Available: true, Key: "AbCdEf...WxYz"}, code: strings.Repeat("a", 64) + ".key"},
		sharing: &fakeSharing{state: hostkey.State{HelperFound: true}}, firewall: &fakeFirewall{}, mux: http.NewServeMux()}
	for _, route := range New(Deps{License: r.license, Sharing: r.sharing, Firewall: r.firewall}).Routes() {
		r.mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	return r
}

func (r *rig) call(method, path, body string) (int, map[string]any) {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	r.mux.ServeHTTP(recorder, request)
	var decoded map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func TestStatusSaysWhetherSharingIsPossibleOnAndRunning(t *testing.T) {
	r := newRig()
	code, body := r.call("GET", Prefix, "")
	if code != 200 || body["available"] != true || body["reason"] != nil || body["key"] != "AbCdEf...WxYz" ||
		body["enabled"] != false || body["running"] != false || body["helper_found"] != true || body["port"] != float64(47862) {
		t.Fatalf("status %d %v", code, body)
	}
	r.license.info = licensing.HostKeyInfo{Reason: "Only a lifetime license can be shared with virtual machines."}
	_, body = r.call("GET", Prefix, "")
	if body["available"] != false || body["key"] != nil || body["reason"] != "Only a lifetime license can be shared with virtual machines." {
		t.Fatalf("status %v", body)
	}
}

func TestTurningSharingOnNeedsAShareableLicenseAndOffIsAlwaysAllowed(t *testing.T) {
	r := newRig()
	code, body := r.call("PUT", Prefix, `{"enabled": true}`)
	if code != 200 || r.sharing.enabled != 1 || body["enabled"] != true {
		t.Fatalf("turning on: %d %v", code, body)
	}
	r.license.info = licensing.HostKeyInfo{Reason: licensing.HostNoCertificateText}
	r2 := newRig()
	r2.license.info = r.license.info
	code, body = r2.call("PUT", Prefix, `{"enabled": true}`)
	if code != 400 || body["error"] != licensing.HostNoCertificateText || r2.sharing.enabled != 0 {
		t.Fatalf("turning on without a license: %d %v (enabled %d)", code, body, r2.sharing.enabled)
	}
	code, body = r2.call("PUT", Prefix, `{"enabled": false}`)
	if code != 200 || r2.sharing.disabled != 1 || body["enabled"] != false {
		t.Fatalf("turning off: %d %v", code, body)
	}
}

func TestAHelperThatCannotStartIsReportedInWords(t *testing.T) {
	r := newRig()
	r.sharing.err = errors.New("the virtual machine helper is not part of this copy of ScadaBench")
	code, body := r.call("PUT", Prefix, `{"enabled": true}`)
	if code != 400 || !strings.Contains(body["error"].(string), "not part of this copy") {
		t.Fatalf("%d %v", code, body)
	}
}

func TestOnlyATrueOrFalseIsAnAnswer(t *testing.T) {
	r := newRig()
	for _, body := range []string{``, `{}`, `{"enabled": "yes"}`, `{"enabled": 1}`, `[]`, `null`} {
		if code, _ := r.call("PUT", Prefix, body); code != 400 {
			t.Errorf("body %q answered %d", body, code)
		}
	}
	if r.sharing.enabled != 0 || r.sharing.disabled != 0 {
		t.Fatal("a request that was not understood changed something")
	}
}

func TestTheFirewallRoutesPassTheReviewedFingerprintAndConfirmation(t *testing.T) {
	r := newRig()
	if code, body := r.call("GET", Prefix+"/firewall", ""); code != 200 || body["fingerprint"] != "fp1" {
		t.Fatalf("preview %d %v", code, body)
	}
	if code, body := r.call("POST", Prefix+"/firewall", `{"fingerprint": "fp1", "confirm": true}`); code != 200 || body["applied"] != true ||
		r.firewall.confirmed[0] != "fp1" || r.firewall.confirmed[1] != true {
		t.Fatalf("apply %d %v %v", code, body, r.firewall.confirmed)
	}
	if code, _ := r.call("POST", Prefix+"/firewall", ``); code != 200 || r.firewall.confirmed[1] != nil {
		t.Fatalf("an empty body must reach Apply as no confirmation: %v", r.firewall.confirmed)
	}
}

func TestTheActivationCodeIsServedAndADevelopmentRefusalIsPlain(t *testing.T) {
	r := newRig()
	if code, body := r.call("GET", CodePath, ""); code != 200 || body["code"] != strings.Repeat("a", 64)+".key" {
		t.Fatalf("code %d %v", code, body)
	}
	r.license.err = &licensing.ActionError{Message: licensing.DevelopmentText, Code: "development"}
	if code, body := r.call("GET", CodePath, ""); code != 400 || body["error"] != licensing.DevelopmentText || body["code"] != "development" {
		t.Fatalf("development %d %v", code, body)
	}
}

func TestEveryRouteIsUnderTheLicensePrefixSoTheGateNeverBlocksIt(t *testing.T) {
	for _, route := range New(Deps{}).Routes() {
		if !strings.HasPrefix(route.Pattern, "/api/v1/license") {
			t.Errorf("%s %s is outside /api/v1/license: the 402 gate would block it", route.Method, route.Pattern)
		}
		if !GoOnly(route.Method + " " + route.Pattern) {
			t.Errorf("%s %s is not marked Go-only", route.Method, route.Pattern)
		}
	}
	if GoOnly("GET /api/v1/license/status") || GoOnly("POST /api/v1/license/activate") {
		t.Fatal("a Flask route was claimed as Go-only")
	}
}
