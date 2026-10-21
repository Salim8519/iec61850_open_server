package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const movesText = "Moving a license to another computer is not possible. Each license works on one computer only."
const boundText = "This license key is already used on another computer. Each license works on one computer only."

func TestTransferRouteAnswers410MovesDisabledAndChangesNothing(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	// An old 0.1.x app sends exactly this; with proof of work, without it, or with junk.
	for _, body := range []map[string]any{{"hwid": b, "license_key": license.Key}, {}, {"hwid": "x"}} {
		code, result := call(t, s, "/api/v1/transfer", body, true)
		if code != http.StatusGone || result["code"] != "moves_disabled" || result["error"] != movesText {
			t.Fatalf("transfer answered %d %v", code, result)
		}
	}
	code, result := call(t, s, "/api/v1/transfer", map[string]any{"hwid": b, "license_key": license.Key}, false)
	if code != http.StatusGone || result["code"] != "moves_disabled" {
		t.Fatalf("transfer without proof of work answered %d %v", code, result)
	}
	if after, _ := s.Store.GetLicense(license.Key); after.HWID != a || len(after.Transfers) != 0 {
		t.Fatalf("a refused move changed the key: %+v", after)
	}
}

func TestKeyOnAnotherComputerIsRefusedWithPlainText(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": b, "license_key": license.Key}, true)
	if code != http.StatusConflict || result["code"] != "bound_elsewhere" || result["error"] != boundText {
		t.Fatalf("bound key answered %d %v", code, result)
	}
	for _, word := range []string{"move it", "transfer", "Move"} {
		if strings.Contains(result["error"].(string), word) {
			t.Fatalf("the refusal still hints at moving: %q", result["error"])
		}
	}
	if code, _ = call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true); code != http.StatusOK {
		t.Fatal("the same computer must still be able to activate again")
	}
}

func TestNoAdminActionCanUnbindOrRebindAKey(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	for _, action := range []string{"reset-hwid", "unbind", "rebind", "transfer", "free"} {
		if _, err := s.RunAdmin(action, AdminRequest{Key: license.Key, HWID: b}); err == nil ||
			!strings.Contains(err.Error(), "unknown admin action") {
			t.Fatalf("admin action %q still exists: %v", action, err)
		}
		body := `{"key":"` + license.Key + `","hwid":"` + b + `"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/"+action, strings.NewReader(body))
		request.RemoteAddr = "127.0.0.1:1234"
		request.Header.Set("Authorization", "Bearer "+s.Config.AdminToken)
		recorder := httptest.NewRecorder()
		s.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("HTTP admin action %q answered %d", action, recorder.Code)
		}
	}
	if after, _ := s.Store.GetLicense(license.Key); after.HWID != a {
		t.Fatal("the key left its computer")
	}
}

func TestCertificateActionNeverBindsOrChangesTheMachine(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	if _, err := s.RunAdmin("certificate", AdminRequest{Key: license.Key, HWID: b}); err == nil {
		t.Fatal("a certificate was made for a key that is not activated anywhere")
	}
	if after, _ := s.Store.GetLicense(license.Key); after.HWID != "" {
		t.Fatalf("certificate bound a key: %+v", after)
	}
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	// The request names another machine; the certificate must still be for the bound one.
	result, err := s.RunAdmin("certificate", AdminRequest{Key: license.Key, HWID: b})
	if err != nil {
		t.Fatal(err)
	}
	if payload := certificateOf(t, s, map[string]any{"certificate": result}); payload.HWID != a {
		t.Fatalf("certificate was made for %s, not the bound computer", payload.HWID)
	}
	if after, _ := s.Store.GetLicense(license.Key); after.HWID != a {
		t.Fatalf("certificate changed the machine: %+v", after)
	}
	s.Store.Revoke(license.Key, "refund")
	if _, err := s.RunAdmin("certificate", AdminRequest{Key: license.Key}); err == nil {
		t.Fatal("a revoked key got a certificate")
	}
}

func TestOwnerPageHasNoFreeForNewPCAction(t *testing.T) {
	h := newOwnerHarness(t)
	h.login()
	_, result, _ := h.do("POST", "/owner-test/api/licenses/issue", map[string]any{"email": "a@example.com"}, nil)
	key := result["license"].(map[string]any)["key"].(string)
	hwid := strings.Repeat("a", 64)
	h.s.Store.Activate(key, hwid, h.clock)
	status, _, _ := h.do("POST", "/owner-test/api/licenses/reset-hwid", map[string]string{"key": key},
		map[string]string{confirmHeader: h.code()})
	if status != http.StatusNotFound {
		t.Fatalf("the owner page still has a reset action: %d", status)
	}
	if license, _ := h.s.Store.GetLicense(key); license.HWID != hwid {
		t.Fatal("the owner page changed the bound computer")
	}
	_, _, recorder := h.do("GET", "/owner-test/static/admin-views.js", nil, nil)
	for _, text := range []string{"reset-hwid", "Free for a new PC"} {
		if strings.Contains(recorder.Body.String(), text) {
			t.Fatalf("owner page script still offers %q", text)
		}
	}
}
