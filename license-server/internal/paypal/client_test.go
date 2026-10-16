package paypal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const capturedOrder = `{"id":"O-1","status":"COMPLETED","payer":{"email_address":"buyer@example.com"},
"purchase_units":[{"custom_id":"hw","payments":{"captures":[{"id":"CAP-1","status":"COMPLETED",
"amount":{"currency_code":"USD","value":"149.00"}}]}}]}`

func TestCaptureFallsBackToReadWhenAlreadyCaptured(t *testing.T) {
	tokens := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/oauth2/token":
			tokens++
			w.Write([]byte(`{"access_token":"T","expires_in":3600}`))
		case strings.HasSuffix(r.URL.Path, "/capture"):
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"name":"UNPROCESSABLE_ENTITY","details":[{"issue":"ORDER_ALREADY_CAPTURED"}]}`))
		default:
			if r.Header.Get("Authorization") != "Bearer T" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(capturedOrder))
		}
	}))
	defer server.Close()
	client := New(server.URL, "id", "secret")
	capture, err := client.CaptureOrder(context.Background(), "O-1")
	if err != nil {
		t.Fatal(err)
	}
	if capture.CaptureID != "CAP-1" || capture.Amount != "149.00" || capture.Email != "buyer@example.com" {
		t.Fatalf("unexpected capture %+v", capture)
	}
	client.GetOrder(context.Background(), "O-1")
	if tokens != 1 {
		t.Fatalf("token fetched %d times; it must be cached", tokens)
	}
}

func TestWebhookVerificationRequiresSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth2/token" {
			w.Write([]byte(`{"access_token":"T","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"verification_status":"FAILURE"}`))
	}))
	defer server.Close()
	client := New(server.URL, "id", "secret")
	if err := client.VerifyWebhook(context.Background(), "WH", http.Header{}, []byte(`{}`)); err == nil {
		t.Fatal("failed verification accepted")
	}
	if err := client.VerifyWebhook(context.Background(), "", http.Header{}, []byte(`{}`)); err == nil {
		t.Fatal("missing webhook id accepted")
	}
}
