package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

const lemonUUID = "8f0a7d3e-1b2c-4d5e-9f60-718293a4b5c6"

func lemonServer(t *testing.T) (*Server, *fakeMail) {
	t.Helper()
	s := newTestServer(t)
	s.Config.LemonCheckoutURL = "https://scadabench.lemonsqueezy.com/buy/abc"
	s.Config.LemonVariantID = "555"
	s.Config.LemonSigningSecret = "a-signing-secret-of-enough-length"
	mail := &fakeMail{}
	s.Mailer = mail
	s.Payments = &fakePayments{}
	return s, mail
}

func lemonEvent(event, status string, variant int, test bool) []byte {
	return []byte(fmt.Sprintf(`{"meta":{"event_name":%q,"test_mode":%t,"custom_data":{"hwid":"","news":"1"}},
		"data":{"type":"orders","id":"42","attributes":{"identifier":%q,"status":%q,"user_email":"buyer@example.com",
		"user_name":"Buyer","currency":"USD","subtotal":9900,"total":10395,"test_mode":%t,
		"first_order_item":{"variant_id":%d}}}}`, event, test, lemonUUID, status, test, variant))
}

func deliver(s *Server, body []byte, secret string) int {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	request := httptest.NewRequest("POST", "/api/v1/webhooks/lemonsqueezy", bytes.NewReader(body))
	request.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	return recorder.Code
}

func lemonKeyOf(s *Server) map[string]any {
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/api/v1/checkout/lemon-key?order="+lemonUUID, nil))
	var answer map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &answer)
	return answer
}

func TestLemonPaidOrderIssuesOneKeyShownAndEmailed(t *testing.T) {
	s, mail := lemonServer(t)
	if answer := lemonKeyOf(s); answer["pending"] != true {
		t.Fatalf("before the webhook the page must hear pending: %v", answer)
	}
	body := lemonEvent("order_created", "paid", 555, false)
	if code := deliver(s, body, s.Config.LemonSigningSecret); code != 200 {
		t.Fatalf("webhook answered %d", code)
	}
	deliver(s, body, s.Config.LemonSigningSecret) // Lemon Squeezy retries must not issue a second key
	answer := lemonKeyOf(s)
	key, _ := answer["license_key"].(string)
	if key == "" || answer["emailed"] != true || len(mail.sent) != 1 || !strings.Contains(mail.sent[0], key) {
		t.Fatalf("one key, shown and emailed once: %v %v", answer, mail.sent)
	}
	if order, _, _ := s.Store.GetOrder(lemonPrefix + lemonUUID); order.Amount != "99.00" || order.CaptureID != "LS-42" {
		t.Fatalf("order record %+v", order)
	}
	if subscribers, _ := s.Store.ListSubscribers(); len(subscribers) != 1 {
		t.Fatal("the news opt-in must subscribe the buyer")
	}
	deliver(s, lemonEvent("order_refunded", "refunded", 555, false), s.Config.LemonSigningSecret)
	if license, err := s.Store.GetLicense(key); err != nil || !license.Revoked {
		t.Fatalf("a refund must revoke the key: %+v %v", license, err)
	}
}

func TestLemonWebhookRefusesForgedAndIgnoresOthers(t *testing.T) {
	s, _ := lemonServer(t)
	if code := deliver(s, lemonEvent("order_created", "paid", 555, false), "wrong-secret"); code != 400 {
		t.Fatalf("a forged signature must be refused, got %d", code)
	}
	for _, body := range [][]byte{lemonEvent("order_created", "paid", 556, false),
		lemonEvent("order_created", "pending", 555, false), lemonEvent("order_created", "paid", 555, true)} {
		if code := deliver(s, body, s.Config.LemonSigningSecret); code != 200 {
			t.Fatalf("ignored events are still acknowledged, got %d", code)
		}
	}
	if answer := lemonKeyOf(s); answer["pending"] != true {
		t.Fatal("other products, unpaid and test orders must not issue a key")
	}
	s.Config.LemonAcceptTestMode = true
	deliver(s, lemonEvent("order_created", "paid", 555, true), s.Config.LemonSigningSecret)
	if answer := lemonKeyOf(s); answer["license_key"] == nil {
		t.Fatal("with test orders accepted a test purchase issues a key")
	}
}

func TestLemonStartCarriesComputerAndNews(t *testing.T) {
	s, _ := lemonServer(t)
	hwid := strings.Repeat("a", 64)
	request := httptest.NewRequest("POST", "/api/v1/checkout/lemon-start",
		strings.NewReader(`{"hwid":"`+hwid+`","news":true}`))
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	var answer map[string]string
	json.Unmarshal(recorder.Body.Bytes(), &answer)
	link := answer["url"]
	if !strings.HasPrefix(link, s.Config.LemonCheckoutURL+"?") || !strings.Contains(link, "news%5D=1") {
		t.Fatalf("checkout link %q", link)
	}
	page := httptest.NewRecorder()
	s.Handler().ServeHTTP(page, httptest.NewRequest("GET", "/buy", nil))
	html := page.Body.String()
	if !strings.Contains(html, `id="lemon-pay"`) || !strings.Contains(html, "or pay with PayPal") {
		t.Fatal("the buy page must offer the card checkout first and PayPal second")
	}
}
