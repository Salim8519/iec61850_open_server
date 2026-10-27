package api

import (
	"context"
	"time"

	"substation-license-server/internal/mail"
	"substation-license-server/internal/sales"
	"substation-license-server/internal/store"
)

// recordSale writes the permanent payment record and sends the owner a copy with
// the buyer's details. Both run once per order, when the key is first issued.
func (s *Server) recordSale(order store.Order, payment paid) {
	record := sales.Record{Time: order.CreatedAt, Event: "paid", OrderID: order.OrderID,
		CaptureID: order.CaptureID, Amount: order.Amount, Currency: order.Currency, Email: order.Email,
		Name: payment.Name, Country: payment.Country, PayerID: payment.PayerID,
		LicenseKey: order.LicenseKey, HWID: payment.HWID, Mode: payment.Mode}
	if err := s.Sales.Append(record); err != nil {
		s.Log.Event("sales_record_error", map[string]any{"order": order.OrderID, "error": err.Error()})
	}
	if s.Mailer == nil || s.Config.NotifyEmail == "" {
		return
	}
	subject, text := mail.SaleEmail(s.Config.ProductName, mail.Sale{OrderID: record.OrderID,
		CaptureID: record.CaptureID, Amount: record.Amount, Currency: record.Currency, Email: record.Email,
		Name: record.Name, Country: record.Country, PayerID: record.PayerID, LicenseKey: record.LicenseKey,
		HWID: record.HWID, Mode: record.Mode, Time: time.Unix(record.Time, 0)})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.Mailer.Send(ctx, s.Config.NotifyEmail, subject, text); err != nil {
			s.Log.Event("owner_mail_error", map[string]any{"order": record.OrderID, "error": err.Error()})
		}
	}()
}

func (s *Server) recordRefund(order store.Order, event string) {
	mode := s.Config.PayPalMode
	if orderIsLemon(order) {
		mode = "lemon"
	}
	record := sales.Record{Time: s.Now().Unix(), Event: event, OrderID: order.OrderID,
		CaptureID: order.CaptureID, Amount: order.Amount, Currency: order.Currency, Email: order.Email,
		LicenseKey: order.LicenseKey, Mode: mode}
	if err := s.Sales.Append(record); err != nil {
		s.Log.Event("sales_record_error", map[string]any{"order": order.OrderID, "error": err.Error()})
	}
}
