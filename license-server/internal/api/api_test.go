package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/paypal"
	"substation-license-server/internal/security"
	"substation-license-server/internal/store"
)

type fakePayments struct {
	captures int
	result   paypal.Capture
}

func (f *fakePayments) CreateOrder(context.Context, paypal.OrderRequest) (string, error) {
	return "ORDER-1", nil
}
func (f *fakePayments) CaptureOrder(context.Context, string) (paypal.Capture, error) {
	f.captures++
	return f.result, nil
}
func (f *fakePayments) GetOrder(context.Context, string) (paypal.Capture, error) {
	return f.result, nil
}
func (f *fakePayments) VerifyWebhook(context.Context, string, http.Header, []byte) error {
	return nil
}

type fakeMail struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeMail) Send(_ context.Context, to, _, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, to+"|"+text)
	return nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Defaults(dir)
	cfg.EnsureSecrets()
	cfg.PowDifficulty = 6
	cfg.PayPalClientID, cfg.PayPalClientSecret = "id", "secret"
	db, err := store.Open(filepath.Join(dir, "licenses.db"))
	if err != nil {
		t.Fatal(err)
	}
	signer, _ := certs.Generate(dir, false)
	logger, _ := logging.Open(cfg.LogDir, "server.log", 1, 1)
	t.Cleanup(func() { db.Close(); logger.Close() })
	server := New(cfg, db, signer, logger)
	server.Limiter = security.NewLimiter(security.LimiterConfig{GeneralPerMinute: 1000,
		SensitivePerMinute: 1000, MaxFailures: 3, BanDuration: time.Minute, MaxEntries: 100})
	return server
}

func call(t *testing.T, s *Server, path string, body map[string]any, solve bool) (int, map[string]any) {
	t.Helper()
	if solve {
		c := security.IssueChallenge(s.Config.PowSecret, s.Config.PowDifficulty, s.Now())
		body["pow_nonce"], body["pow_ts"], body["pow_difficulty"] = c.Nonce, c.Timestamp, c.Difficulty
		body["pow_mac"], body["pow_counter"] = c.MAC, security.Solve(c)
	}
	raw, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	request.RemoteAddr = "127.0.0.1:9999"
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	var result map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &result)
	return recorder.Code, result
}

func certificateOf(t *testing.T, s *Server, result map[string]any) certs.Payload {
	t.Helper()
	raw, _ := json.Marshal(result["certificate"])
	var bundle certs.Bundle
	json.Unmarshal(raw, &bundle)
	payload, err := certs.Verify(s.Signer.Public(), bundle)
	if err != nil {
		t.Fatalf("certificate does not verify: %v (%v)", err, result)
	}
	return payload
}

func TestTrialNeedsProofOfWorkAndIsOncePerMachine(t *testing.T) {
	s := newTestServer(t)
	hwid := strings.Repeat("a", 64)
	if code, _ := call(t, s, "/api/v1/trial", map[string]any{"hwid": hwid}, false); code != http.StatusBadRequest {
		t.Fatalf("trial without proof of work answered %d", code)
	}
	code, result := call(t, s, "/api/v1/trial", map[string]any{"hwid": hwid}, true)
	if code != http.StatusOK {
		t.Fatalf("trial failed: %d %v", code, result)
	}
	payload := certificateOf(t, s, result)
	if payload.Type != "trial" || payload.ExpiresAt-payload.IssuedAt != 7*86400 || payload.HWID != hwid {
		t.Fatalf("wrong trial certificate %+v", payload)
	}
	s.Now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	if code, result := call(t, s, "/api/v1/trial", map[string]any{"hwid": hwid}, true); code != http.StatusConflict || result["code"] != "trial_used" {
		t.Fatalf("second trial after expiry: %d %v", code, result)
	}
}

func TestActivationVerifyRevokeAndBan(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": strings.ToLower(license.Key)}, true)
	if code != http.StatusOK || certificateOf(t, s, result).ExpiresAt != 0 {
		t.Fatalf("activation failed: %d %v", code, result)
	}
	if code, result = call(t, s, "/api/v1/activate", map[string]any{"hwid": b, "license_key": license.Key}, true); result["code"] != "bound_elsewhere" {
		t.Fatalf("second PC activation: %d %v", code, result)
	}
	if _, result = call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": license.Key}, false); result["status"] != "valid" {
		t.Fatalf("the first PC lost its license: %v", result)
	}
	if _, result = call(t, s, "/api/v1/verify", map[string]any{"hwid": b, "license_key": license.Key}, false); result["status"] != "moved" {
		t.Fatalf("another PC must not be told its license is valid: %v", result)
	}
	s.Store.Revoke(license.Key, "refunded")
	if _, result = call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": license.Key}, false); result["status"] != "revoked" {
		t.Fatalf("revocation not reported: %v", result)
	}
	for i := 0; i < 3; i++ {
		call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": "SUBS-NOPE-NOPE-NOPE-NOP" + string(rune('A'+i))}, true)
	}
	if code, _ = call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true); code != http.StatusTooManyRequests {
		t.Fatalf("guessing IP not banned: %d", code)
	}
}

func TestPayPalCaptureIsIdempotentAndRefundRevokes(t *testing.T) {
	s := newTestServer(t)
	payments := &fakePayments{result: paypal.Capture{OrderID: "ORDER-1", CaptureID: "CAP-1", Status: "COMPLETED",
		Amount: s.Config.LicensePrice, Currency: s.Config.LicenseCurrency, Email: "buyer@example.com"}}
	mailer := &fakeMail{}
	s.Payments, s.Mailer = payments, mailer
	_, first := call(t, s, "/api/v1/checkout/capture-order", map[string]any{"order_id": "ORDER-1"}, false)
	_, second := call(t, s, "/api/v1/checkout/capture-order", map[string]any{"order_id": "ORDER-1"}, false)
	key, _ := first["license_key"].(string)
	if !strings.HasPrefix(key, "SUBS-") || second["license_key"] != key || payments.captures != 1 {
		t.Fatalf("capture not idempotent: %v %v captures=%d", first, second, payments.captures)
	}
	if err := s.handleWebhook(context.Background(), webhookEvent{Type: "CHECKOUT.ORDER.APPROVED"}); err != nil {
		t.Fatal(err)
	}
	if len(mailer.sent) != 1 || !strings.Contains(mailer.sent[0], key) {
		t.Fatalf("expected exactly one delivery email, got %v", mailer.sent)
	}
	refund := webhookEvent{Type: "PAYMENT.CAPTURE.REFUNDED"}
	refund.Resource.Links = append(refund.Resource.Links, struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	}{Href: "https://api-m.paypal.com/v2/payments/captures/CAP-1", Rel: "up"})
	if err := s.handleWebhook(context.Background(), refund); err != nil {
		t.Fatal(err)
	}
	if license, _ := s.Store.GetLicense(key); !license.Revoked {
		t.Fatal("refund did not revoke the key")
	}
}

func TestPayPalWrongAmountIsRejected(t *testing.T) {
	s := newTestServer(t)
	s.Payments = &fakePayments{result: paypal.Capture{OrderID: "ORDER-2", CaptureID: "CAP-2", Status: "COMPLETED",
		Amount: "1.00", Currency: s.Config.LicenseCurrency}}
	if code, result := call(t, s, "/api/v1/checkout/capture-order", map[string]any{"order_id": "ORDER-2"}, false); result["code"] != "amount_mismatch" {
		t.Fatalf("underpayment accepted: %d %v", code, result)
	}
}

func TestAdminRefusesTunnelAndWrongToken(t *testing.T) {
	s := newTestServer(t)
	send := func(token, cf string) int {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/list", strings.NewReader("{}"))
		request.RemoteAddr = "127.0.0.1:1234"
		request.Header.Set("Authorization", "Bearer "+token)
		if cf != "" {
			request.Header.Set("CF-Connecting-IP", cf)
		}
		recorder := httptest.NewRecorder()
		s.Handler().ServeHTTP(recorder, request)
		return recorder.Code
	}
	if send(s.Config.AdminToken, "") != http.StatusOK {
		t.Fatal("local admin with token refused")
	}
	if send(s.Config.AdminToken, "203.0.113.1") != http.StatusForbidden || send("wrong", "") != http.StatusForbidden {
		t.Fatal("admin reachable through the tunnel or with a wrong token")
	}
}

func TestPagesRender(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/buy", "/activate", "/static/common.js", "/static/style.css", "/static/geist.woff2"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "127.0.0.1:1"
		s.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || recorder.Body.Len() < 100 {
			t.Fatalf("%s: %d", path, recorder.Code)
		}
	}
}
