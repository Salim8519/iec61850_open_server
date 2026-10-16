package api

import (
	"strings"
	"testing"
	"time"

	"substation-license-server/internal/paypal"
	"substation-license-server/internal/sales"
)

func TestQuotedPriceSurvivesPriceChangeButForgedInvoiceDoesNot(t *testing.T) {
	s := newTestServer(t)
	invoice := s.newInvoice("0.10", "USD")
	s.Config.LicensePrice = "149.00"
	if !s.paidAmountAccepted(invoice, "0.10", "USD") {
		t.Fatal("buyer who was quoted 0.10 before the price change is refused")
	}
	if s.paidAmountAccepted(invoice, "0.01", "USD") || s.paidAmountAccepted("SB-000000000000-forged", "0.10", "USD") {
		t.Fatal("forged or altered invoice accepted")
	}
	if !s.paidAmountAccepted("", "149.00", "USD") || s.paidAmountAccepted("", "1.00", "USD") {
		t.Fatal("current price check is wrong")
	}
}

func TestSaleIsRecordedAndOwnerGetsBuyerDetails(t *testing.T) {
	s := newTestServer(t)
	s.Config.NotifyEmail = "owner@example.com"
	mailer := &fakeMail{}
	s.Payments, s.Mailer = &fakePayments{result: paypal.Capture{OrderID: "ORDER-9", CaptureID: "CAP-9",
		Status: "COMPLETED", Amount: s.Config.LicensePrice, Currency: s.Config.LicenseCurrency,
		Email: "buyer@example.com", Name: "Ada Engineer", Country: "OM", PayerID: "PAYER9"}}, mailer
	_, result := call(t, s, "/api/v1/checkout/capture-order", map[string]any{"order_id": "ORDER-9"}, false)
	key, _ := result["license_key"].(string)
	records, err := sales.Read(s.Config.DataDir)
	if err != nil || len(records) != 1 || records[0].Name != "Ada Engineer" || records[0].LicenseKey != key {
		t.Fatalf("sale not recorded: %+v %v", records, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mailer.mu.Lock()
		owner := ""
		for _, sent := range mailer.sent {
			if strings.HasPrefix(sent, "owner@example.com|") {
				owner = sent
			}
		}
		mailer.mu.Unlock()
		if owner != "" {
			if !strings.Contains(owner, "Ada Engineer") || !strings.Contains(owner, "buyer@example.com") ||
				!strings.Contains(owner, key) || !strings.Contains(owner, "ORDER-9") {
				t.Fatalf("owner email misses buyer details: %s", owner)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("owner was not emailed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
