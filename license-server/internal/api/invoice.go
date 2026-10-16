package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"substation-license-server/internal/config"
)

// An invoice id carries the price quoted when the order was created, signed by the
// server: "SB-<nonce>-<mac>". When the owner changes the price, a buyer who was
// already inside PayPal still gets the key for the amount they were shown, and a
// forged cheap order (made with the public client id) is still refused.
func (s *Server) newInvoice(amount, currency string) string {
	nonce := config.RandomHex(6)
	return "SB-" + nonce + "-" + s.invoiceMAC(nonce, amount, currency)
}

func (s *Server) invoiceQuoted(invoice, amount, currency string) bool {
	parts := strings.Split(invoice, "-")
	if len(parts) != 3 || parts[0] != "SB" {
		return false
	}
	return hmac.Equal([]byte(parts[2]), []byte(s.invoiceMAC(parts[1], amount, currency)))
}

func (s *Server) invoiceMAC(nonce, amount, currency string) string {
	mac := hmac.New(sha256.New, []byte("invoice|"+s.Config.PowSecret))
	mac.Write([]byte(nonce + "|" + amount + "|" + currency))
	return hex.EncodeToString(mac.Sum(nil))[:20]
}

// paidAmountAccepted is true for the current price or a price this server quoted.
func (s *Server) paidAmountAccepted(invoice, amount, currency string) bool {
	if amount == s.Config.LicensePrice && currency == s.Config.LicenseCurrency {
		return true
	}
	return s.invoiceQuoted(invoice, amount, currency)
}
