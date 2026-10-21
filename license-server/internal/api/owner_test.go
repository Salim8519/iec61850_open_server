package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"substation-license-server/internal/adminauth"
)

const testGateway = "0123456789abcdef0123456789abcdef0123456789abcdef"
const testPassword = "a long owner password"

// ownerHarness drives the owner page like the Worker would: gateway header,
// browser origin, cookie and CSRF token.
type ownerHarness struct {
	t       *testing.T
	s       *Server
	handler http.Handler
	clock   time.Time
	cookie  *http.Cookie
	csrf    string
	secret  string
	ip      string
}

func newOwnerHarness(t *testing.T) *ownerHarness {
	t.Helper()
	s := newTestServer(t)
	s.Config.AdminPath, s.Config.AdminGatewaySecret = "/owner-test", testGateway
	h := &ownerHarness{t: t, s: s, clock: time.Unix(1_800_000_000, 0), ip: "203.0.113.7"}
	s.Now = func() time.Time { return h.clock }
	account, enrollment, err := adminauth.NewAccount(testPassword, 0, h.clock)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.owner.keeper.Replace(account); err != nil {
		t.Fatal(err)
	}
	h.secret, h.handler = enrollment.Secret, s.Handler()
	return h
}

func (h *ownerHarness) code() string {
	h.clock = h.clock.Add(30 * time.Second) // a fresh step, so codes are never replays
	code, _ := adminauth.Code(h.secret, h.clock)
	return code
}

func (h *ownerHarness) do(method, path string, body any, headers map[string]string) (int, map[string]any, *httptest.ResponseRecorder) {
	h.t.Helper()
	var reader *bytes.Reader
	raw, _ := json.Marshal(body)
	reader = bytes.NewReader(raw)
	request := httptest.NewRequest(method, path, reader)
	request.RemoteAddr = "127.0.0.1:5555"
	request.Header.Set(gatewayHeader, testGateway)
	request.Header.Set(clientIPHeader, h.ip)
	request.Header.Set("Origin", "https://scadabench.com")
	request.Header.Set("Content-Type", "application/json")
	if h.cookie != nil {
		request.AddCookie(h.cookie)
	}
	if h.csrf != "" {
		request.Header.Set(csrfHeader, h.csrf)
	}
	for name, value := range headers {
		if value == "" {
			request.Header.Del(name)
		} else {
			request.Header.Set(name, value)
		}
	}
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, request)
	var result map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &result)
	return recorder.Code, result, recorder
}

func (h *ownerHarness) login() {
	h.t.Helper()
	status, result, recorder := h.do("POST", "/owner-test/api/login", map[string]string{"password": testPassword, "code": h.code()}, nil)
	if status != 200 {
		h.t.Fatalf("login failed %d %v", status, result)
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == ownerCookie {
			h.cookie = cookie
		}
	}
	h.csrf, _ = result["csrf"].(string)
}

func TestOwnerPageIsInvisibleWithoutGateway(t *testing.T) {
	h := newOwnerHarness(t)
	for _, path := range []string{"/owner-test", "/owner-test/", "/owner-test/api/session", "/owner-test/static/admin-core.js"} {
		if status, _, _ := h.do("GET", path, nil, map[string]string{gatewayHeader: ""}); status != 404 {
			t.Fatalf("%s without gateway: %d", path, status)
		}
		if status, _, _ := h.do("GET", path, nil, map[string]string{gatewayHeader: strings.Repeat("x", 48)}); status != 404 {
			t.Fatalf("%s with wrong gateway: %d", path, status)
		}
	}
	status, _, recorder := h.do("GET", "/owner-test/", nil, nil)
	body := recorder.Body.String()
	if status != 200 || !strings.Contains(body, "/owner-test/static/admin-core.js") ||
		!strings.Contains(recorder.Header().Get("Content-Security-Policy"), "script-src 'self'") ||
		recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("page not served safely: %d %v", status, recorder.Header())
	}
	if status, _, recorder := h.do("GET", "/owner-test/static/admin-ops.js", nil, nil); status != 200 ||
		!strings.HasPrefix(recorder.Header().Get("Cache-Control"), "private") {
		t.Fatal("static files must be private so Cloudflare never caches them")
	}
	h.s.Config.AdminGatewaySecret = ""
	if status, _, _ := h.do("GET", "/owner-test/", nil, map[string]string{gatewayHeader: ""}); status != 404 ||
		len(newOwnerHandlerFor(h.s)) != 0 {
		t.Fatal("without a gateway secret the page must not exist")
	}
}

// newOwnerHandlerFor reports owner routes mounted for an unconfigured server.
func newOwnerHandlerFor(s *Server) string {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/owner-test/", nil)
	request.Header.Set(gatewayHeader, "")
	s.Handler().ServeHTTP(recorder, request)
	if recorder.Code == 404 {
		return ""
	}
	return "mounted"
}

func TestOwnerLoginNeedsPasswordAndCodeAndSetsStrictCookie(t *testing.T) {
	h := newOwnerHarness(t)
	if status, _, _ := h.do("POST", "/owner-test/api/login", map[string]string{"password": testPassword, "code": "000000"}, nil); status != 401 {
		t.Fatal("wrong code must fail")
	}
	if status, _, _ := h.do("POST", "/owner-test/api/login", map[string]string{"password": "wrong password!!", "code": h.code()}, nil); status != 401 {
		t.Fatal("wrong password must fail")
	}
	if status, _, _ := h.do("GET", "/owner-test/api/overview", nil, nil); status != 401 {
		t.Fatal("overview needs a session")
	}
	h.login()
	if h.cookie == nil || !h.cookie.HttpOnly || !h.cookie.Secure || h.cookie.SameSite != http.SameSiteStrictMode ||
		h.cookie.Path != "/owner-test" || len(h.csrf) != 64 {
		t.Fatalf("unsafe cookie %+v", h.cookie)
	}
	if status, result, _ := h.do("GET", "/owner-test/api/overview", nil, nil); status != 200 || result["trials"] == nil {
		t.Fatalf("overview failed %d %v", status, result)
	}
	if status, _, _ := h.do("POST", "/owner-test/api/licenses/issue", map[string]any{}, map[string]string{csrfHeader: ""}); status != 403 {
		t.Fatal("missing CSRF token must be refused")
	}
	if status, _, _ := h.do("POST", "/owner-test/api/licenses/issue", map[string]any{}, map[string]string{"Origin": "https://evil.example"}); status != 403 {
		t.Fatal("foreign origin must be refused")
	}
	entries, _ := h.s.owner.audit.Recent(10)
	if len(entries) < 3 || entries[0].Action != "login" || !entries[0].OK {
		t.Fatalf("sign-ins must be recorded %+v", entries)
	}
}

func TestOwnerLockoutStopsSpreadGuessesAndResetEndsSessions(t *testing.T) {
	h := newOwnerHarness(t)
	h.login()
	for i := 0; i < 10; i++ {
		h.ip = "198.51.100." + string(rune('1'+i%9)) + string(rune('0'+i))
		h.do("POST", "/owner-test/api/login", map[string]string{"password": "guess guess guess", "code": "123456"}, nil)
	}
	status, result, _ := h.do("POST", "/owner-test/api/login", map[string]string{"password": testPassword, "code": h.code()}, nil)
	if status != http.StatusLocked || !strings.Contains(result["error"].(string), "reset-admin.cmd") {
		t.Fatalf("ten wrong tries must lock sign-in, got %d %v", status, result)
	}
	h.ip = "203.0.113.7"
	if status, _, _ := h.do("GET", "/owner-test/api/overview", nil, nil); status != 200 {
		t.Fatal("an existing session keeps working during a lock")
	}
	account, _, _ := h.s.owner.keeper.Load()
	fresh, _, _ := adminauth.NewAccount("another long password", account.Epoch, h.clock)
	h.s.owner.keeper.Replace(fresh)
	if status, _, _ := h.do("GET", "/owner-test/api/overview", nil, nil); status != 401 {
		t.Fatal("admin-setup must sign every browser out")
	}
}

func TestOwnerAddressBanAfterFiveWrongTries(t *testing.T) {
	h := newOwnerHarness(t)
	for i := 0; i < 5; i++ {
		h.do("POST", "/owner-test/api/login", map[string]string{"password": "guess guess guess", "code": "123456"}, nil)
	}
	if status, _, _ := h.do("GET", "/owner-test/api/session", nil, nil); status != http.StatusTooManyRequests {
		t.Fatalf("the guessing address must be banned, got %d", status)
	}
}
