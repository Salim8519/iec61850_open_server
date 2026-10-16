package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func get(s *Server, path, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = "127.0.0.1:1"
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestProductPriceIsSharedOnlyWithTheWebsite(t *testing.T) {
	s := newTestServer(t)
	s.Config.LicensePrice = "99.00"
	recorder := get(s, "/api/v1/product", "https://scadabench.com")
	var body map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &body)
	if body["amount"] != "$99.00" || recorder.Header().Get("Access-Control-Allow-Origin") != "https://scadabench.com" {
		t.Fatalf("product: %v %v", body, recorder.Header())
	}
	if get(s, "/api/v1/product", "https://evil.example").Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("price readable by any website")
	}
}

func TestCheckoutPageLocksScriptsToPayPal(t *testing.T) {
	policy := get(newTestServer(t), "/buy", "").Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "frame-ancestors 'none'") || !strings.Contains(policy, "https://*.paypal.com") {
		t.Fatalf("policy: %s", policy)
	}
}

func TestTrialsPauseWhenDataBudgetIsFull(t *testing.T) {
	s := newTestServer(t)
	s.Config.MaxDataMB = 1
	os.WriteFile(filepath.Join(s.Config.DataDir, "filler.bin"), make([]byte, 2<<20), 0o600)
	code, result := call(t, s, "/api/v1/trial", map[string]any{"hwid": strings.Repeat("c", 64)}, true)
	if code != http.StatusServiceUnavailable || result["code"] != "trials_paused" {
		t.Fatalf("trial with a full disk budget: %d %v", code, result)
	}
}

func TestCapacityRefusesInsteadOfQueueing(t *testing.T) {
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(maxInFlight)
	handler := (&Server{}).capacity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started.Done()
		<-release
	}))
	for i := 0; i < maxInFlight; i++ {
		go handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}
	started.Wait()
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil)); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("extra request waited instead of being refused")
	}
	close(release)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("extra request answered %d", recorder.Code)
	}
}

func TestForgedWebhookNeverReachesPayPal(t *testing.T) {
	s := newTestServer(t)
	s.Payments = &fakePayments{}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/paypal", strings.NewReader(`{"event_type":"X"}`))
	request.RemoteAddr = "127.0.0.1:1"
	request.Header.Set("Paypal-Cert-Url", "https://evil.example/cert.pem")
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("forged webhook answered %d", recorder.Code)
	}
	header := http.Header{}
	for _, name := range []string{"Paypal-Transmission-Id", "Paypal-Transmission-Sig", "Paypal-Transmission-Time", "Paypal-Auth-Algo"} {
		header.Set(name, "x")
	}
	header.Set("Paypal-Cert-Url", "https://api.paypal.com/v1/notifications/certs/CERT-1")
	if !looksLikePayPal(header) {
		t.Fatal("genuine PayPal headers refused")
	}
}
