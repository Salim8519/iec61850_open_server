package api

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"substation-license-server/internal/paypal"
	"substation-license-server/internal/sales"
)

// lemonOrder is a paid order event with its own subtotal (cents) and tax settings.
func lemonOrder(subtotal, tax int, inclusive bool, invoice string) []byte {
	body := string(lemonEvent("order_created", "paid", 555, false))
	body = strings.Replace(body, `"subtotal":9900`, `"subtotal":`+strconv.Itoa(subtotal)+`,"tax":`+strconv.Itoa(tax)+
		`,"tax_inclusive":`+map[bool]string{true: "true", false: "false"}[inclusive], 1)
	return []byte(strings.Replace(body, `"news":"1"`, `"news":"1","invoice":"`+invoice+`"`, 1))
}

func TestLemonUnderpaidOrderIssuesNoKey(t *testing.T) {
	s, _ := lemonServer(t)
	if deliver(s, lemonOrder(100, 0, false, ""), s.Config.LemonSigningSecret) != 200 {
		t.Fatal("a refused order is still acknowledged")
	}
	if answer := lemonKeyOf(s); answer["pending"] != true {
		t.Fatalf("a 1.00 order for a 99.00 product must issue no key: %v", answer)
	}
}

func TestLemonAcceptsQuotedAndTaxInclusivePrices(t *testing.T) {
	s, _ := lemonServer(t)
	quoted := s.newInvoice("79.00", "USD", false) // quoted before the owner raised the price
	if deliver(s, lemonOrder(7900, 0, false, quoted), s.Config.LemonSigningSecret); lemonKeyOf(s)["license_key"] == nil {
		t.Fatal("a price this server quoted must be accepted")
	}
	s2, _ := lemonServer(t)
	if deliver(s2, lemonOrder(8250, 1650, true, ""), s2.Config.LemonSigningSecret); lemonKeyOf(s2)["license_key"] == nil {
		t.Fatal("a tax-inclusive full-price order must be accepted")
	}
}

func TestTestPaymentsGiveTrialKeys(t *testing.T) {
	s, _ := lemonServer(t)
	s.Config.LemonAcceptTestMode = true
	deliver(s, lemonEvent("order_created", "paid", 555, true), s.Config.LemonSigningSecret)
	key, _ := lemonKeyOf(s)["license_key"].(string)
	if license, err := s.Store.GetLicense(key); err != nil || license.Type != "trial" || license.Days != s.Config.TrialDays {
		t.Fatalf("a Lemon test-mode order must be a trial: %+v %v", license, err)
	}
	p := newTestServer(t) // PayPal mode defaults to sandbox
	p.Payments = &fakePayments{result: paypal.Capture{OrderID: "ORDER-S", CaptureID: "CAP-S", Status: "COMPLETED",
		Amount: p.Config.LicensePrice, Currency: p.Config.LicenseCurrency}}
	order, err := p.Fulfil(context.Background(), p.Payments.(*fakePayments).result)
	if license, _ := p.Store.GetLicense(order.LicenseKey); err != nil || license.Type != "trial" {
		t.Fatalf("a PayPal sandbox order must be a trial: %+v %v", license, err)
	}
	p.Config.PayPalMode = "live"
	live, _ := p.Fulfil(context.Background(), paypal.Capture{OrderID: "ORDER-L", CaptureID: "CAP-L",
		Status: "COMPLETED", Amount: p.Config.LicensePrice, Currency: p.Config.LicenseCurrency})
	if license, _ := p.Store.GetLicense(live.LicenseKey); license.Type != "lifetime" {
		t.Fatalf("a live order stays lifetime: %+v", license)
	}
}

func TestRefundBeforeOrderIssuesNothingAndIsRecordedOnce(t *testing.T) {
	s, mail := lemonServer(t)
	refund := lemonEvent("order_refunded", "refunded", 555, false)
	for i := 0; i < 2; i++ {
		if code := deliver(s, refund, s.Config.LemonSigningSecret); code != 200 {
			t.Fatalf("early refund answered %d", code)
		}
	}
	if code := deliver(s, lemonEvent("order_created", "paid", 555, false), s.Config.LemonSigningSecret); code != 200 {
		t.Fatalf("late order answered %d", code)
	}
	if answer := lemonKeyOf(s); answer["pending"] != true || len(mail.sent) != 0 {
		t.Fatalf("an order refunded first must issue nothing: %v %v", answer, mail.sent)
	}
	s2, _ := lemonServer(t)
	deliver(s2, lemonEvent("order_created", "paid", 555, false), s2.Config.LemonSigningSecret)
	for i := 0; i < 3; i++ {
		deliver(s2, refund, s2.Config.LemonSigningSecret)
	}
	records, _ := sales.Read(s2.Config.DataDir)
	refunds := 0
	for _, record := range records {
		if record.Event == "refunded" {
			refunds++
		}
	}
	if refunds != 1 {
		t.Fatalf("repeated refund webhooks must add one refund line, got %d", refunds)
	}
}
