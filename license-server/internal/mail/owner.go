package mail

import (
	"strings"
	"time"
)

// Sale is what the owner needs to help a buyer later.
type Sale struct {
	OrderID, CaptureID, Amount, Currency, Email, Name, Country, PayerID, LicenseKey, HWID, Mode string
	Time                                                                                        time.Time
}

// SaleEmail builds the owner's copy of a successful payment.
func SaleEmail(product string, sale Sale) (string, string) {
	subject := "New sale: " + sale.Amount + " " + sale.Currency + " - " + product
	if sale.Mode != "live" {
		subject = "[TEST " + strings.ToUpper(sale.Mode) + "] " + subject
	}
	value := func(text string) string {
		if strings.TrimSpace(text) == "" {
			return "-"
		}
		return text
	}
	lines := []string{
		"A payment was completed and a license key was issued.",
		"",
		"Buyer name:     " + value(sale.Name),
		"Buyer email:    " + value(sale.Email),
		"Buyer country:  " + value(sale.Country),
		"PayPal payer:   " + value(sale.PayerID),
		"",
		"Amount:         " + sale.Amount + " " + sale.Currency,
		"PayPal order:   " + sale.OrderID,
		"PayPal capture: " + value(sale.CaptureID),
		"Time (UTC):     " + sale.Time.UTC().Format("2006-01-02 15:04:05"),
		"",
		"License key:    " + sale.LicenseKey,
		"Machine ID:     " + value(sale.HWID),
		"",
		"Keep this email. If the buyer loses the key, send it again from here. A key works on one",
		"computer only and cannot be moved or freed; a new computer needs a new license. No refunds",
		"after a license is activated. A refund in PayPal revokes the key automatically. Every sale",
		"is also recorded on the server (payments.cmd).",
	}
	return subject, strings.Join(lines, "\n")
}
