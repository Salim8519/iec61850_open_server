package api

import (
	"context"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/mail"
	"substation-license-server/internal/store"
)

// paid is one completed payment from either gateway (PayPal or Lemon Squeezy,
// ADR-0126), already checked for status and amount by the gateway's own code.
type paid struct {
	OrderID, CaptureID, Email, Name, Country, PayerID string
	Amount, Currency, HWID, Mode                       string
	News                                               bool
}

// issuePaid issues exactly one key per order, records the sale once and emails the
// key until one email went out.
func (s *Server) issuePaid(payment paid) (store.Order, error) {
	order, created, err := s.Store.IssueForOrder(store.Order{OrderID: payment.OrderID, CaptureID: payment.CaptureID,
		Email: payment.Email, Amount: payment.Amount, Currency: payment.Currency}, certs.NewLicenseKey, s.Now())
	if err != nil {
		return order, err
	}
	if created {
		s.Log.Event("order_fulfilled", map[string]any{"order": order.OrderID, "mode": payment.Mode,
			"key": logging.MaskKey(order.LicenseKey)})
		s.recordSale(order, payment)
		if payment.News && order.Email != "" {
			if _, err := s.Store.Subscribe(order.Email, "checkout", s.Now()); err != nil {
				s.Log.Event("subscribe_error", map[string]any{"order": order.OrderID, "error": err.Error()})
			}
		}
	}
	if !order.Emailed && s.Mailer != nil && order.Email != "" {
		subject, text := mail.LicenseEmail(s.Config.ProductName, order.LicenseKey, s.Config.PublicURL, s.Config.SupportEmail)
		mailCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.Mailer.Send(mailCtx, order.Email, subject, text); err != nil {
			s.Log.Event("mail_error", map[string]any{"order": order.OrderID, "error": err.Error()})
		} else if s.Store.MarkEmailed(order.OrderID) == nil {
			order.Emailed = true
		}
	}
	return order, nil
}

// revokeOrder turns a refunded or charged-back order's key off and records why.
func (s *Server) revokeOrder(order store.Order, reason string) error {
	_, err := s.Store.Revoke(order.LicenseKey, reason)
	if err == nil {
		s.recordRefund(order, reason)
	}
	s.Log.Event("revoked", map[string]any{"order": order.OrderID, "reason": reason, "key": logging.MaskKey(order.LicenseKey)})
	return err
}
