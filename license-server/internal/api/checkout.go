package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/mail"
	"substation-license-server/internal/paypal"
	"substation-license-server/internal/store"
)

var errCheckoutOff = apiError{http.StatusServiceUnavailable, "checkout_unavailable",
	"Online purchase is not set up yet. Contact the seller."}

func (s *Server) createOrder(r *http.Request, ip string) (any, error) {
	if s.Payments == nil {
		return nil, errCheckoutOff
	}
	var request struct {
		HWID string `json:"hwid"`
		News bool   `json:"news"`
	}
	decode(r, &request)
	custom := ""
	if certs.ValidHWID(strings.TrimSpace(request.HWID)) {
		custom = strings.TrimSpace(request.HWID)
	}
	amount, currency := s.price()
	id, err := s.Payments.CreateOrder(r.Context(), paypal.OrderRequest{Product: s.Config.ProductName,
		Brand: "ScadaBench", Amount: amount, Currency: currency, CustomID: custom,
		InvoiceID: s.newInvoice(amount, currency, request.News)})
	if err != nil {
		s.Log.Event("paypal_error", map[string]any{"step": "create", "error": err.Error()})
		return nil, apiError{http.StatusBadGateway, "paypal_error", "PayPal could not start the payment. Try again."}
	}
	return map[string]string{"id": id}, nil
}

func (s *Server) captureOrder(r *http.Request, ip string) (any, error) {
	if s.Payments == nil {
		return nil, errCheckoutOff
	}
	var request struct {
		OrderID string `json:"order_id"`
	}
	if err := decode(r, &request); err != nil || strings.TrimSpace(request.OrderID) == "" {
		return nil, badRequest("order_id is required")
	}
	if order, found, err := s.Store.GetOrder(request.OrderID); err == nil && found {
		return s.purchaseResult(order), nil
	}
	captured, err := s.Payments.CaptureOrder(r.Context(), request.OrderID)
	if err != nil {
		s.Log.Event("paypal_error", map[string]any{"step": "capture", "error": err.Error()})
		return nil, apiError{http.StatusBadGateway, "paypal_error", "PayPal did not confirm the payment. You were not charged twice; try again."}
	}
	order, err := s.Fulfil(r.Context(), captured)
	if err != nil {
		return nil, err
	}
	return s.purchaseResult(order), nil
}

func (s *Server) purchaseResult(order store.Order) map[string]any {
	return map[string]any{"license_key": order.LicenseKey, "email": order.Email, "emailed": order.Emailed}
}

// Fulfil issues exactly one key per completed PayPal order and emails it.
func (s *Server) Fulfil(_ context.Context, captured paypal.Capture) (store.Order, error) {
	if captured.Status != "COMPLETED" {
		return store.Order{}, apiError{http.StatusPaymentRequired, "payment_incomplete",
			"PayPal reports the payment as " + strings.ToLower(captured.Status) + ". The key is sent when it completes."}
	}
	if !s.paidAmountAccepted(captured.InvoiceID, captured.Amount, captured.Currency) {
		s.Log.Event("paypal_amount_mismatch", map[string]any{"order": captured.OrderID,
			"amount": captured.Amount, "currency": captured.Currency})
		return store.Order{}, apiError{http.StatusBadRequest, "amount_mismatch", "The paid amount does not match the price. Contact support."}
	}
	order, created, err := s.Store.IssueForOrder(store.Order{OrderID: captured.OrderID, CaptureID: captured.CaptureID,
		Email: captured.Email, Amount: captured.Amount, Currency: captured.Currency}, certs.NewLicenseKey, s.Now())
	if err != nil {
		return order, err
	}
	if created {
		s.Log.Event("order_fulfilled", map[string]any{"order": order.OrderID, "key": logging.MaskKey(order.LicenseKey)})
		s.recordSale(order, captured)
		if wantsNews(captured.InvoiceID) && order.Email != "" {
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

type webhookEvent struct {
	Type     string `json:"event_type"`
	Resource struct {
		ID                string `json:"id"`
		SupplementaryData struct {
			RelatedIDs struct {
				OrderID string `json:"order_id"`
			} `json:"related_ids"`
		} `json:"supplementary_data"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	} `json:"resource"`
}

// paypalWebhook guarantees delivery when the buyer closed the tab, and revokes
// keys on refunds and chargebacks.
func (s *Server) paypalWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow(s.clientIP(r), 0, s.Now()) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || s.Payments == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	if !looksLikePayPal(r.Header) {
		http.Error(w, "not a PayPal delivery", http.StatusBadRequest)
		return
	}
	if err := s.Payments.VerifyWebhook(r.Context(), s.Config.PayPalWebhookID, r.Header, body); err != nil {
		s.Log.Event("webhook_rejected", map[string]any{"error": err.Error(), "ip": s.clientIP(r)})
		http.Error(w, "signature rejected", http.StatusBadRequest)
		return
	}
	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "bad event", http.StatusBadRequest)
		return
	}
	if err := s.handleWebhook(r.Context(), event); err != nil {
		s.Log.Event("webhook_error", map[string]any{"type": event.Type, "error": err.Error()})
		http.Error(w, "retry later", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWebhook(ctx context.Context, event webhookEvent) error {
	switch event.Type {
	case "CHECKOUT.ORDER.APPROVED":
		captured, err := s.Payments.CaptureOrder(ctx, event.Resource.ID)
		if err == nil {
			_, err = s.Fulfil(ctx, captured)
		}
		return ignorePending(err)
	case "PAYMENT.CAPTURE.COMPLETED":
		orderID := event.Resource.SupplementaryData.RelatedIDs.OrderID
		if orderID == "" {
			return errors.New("capture event without order id")
		}
		captured, err := s.Payments.GetOrder(ctx, orderID)
		if err == nil {
			_, err = s.Fulfil(ctx, captured)
		}
		return ignorePending(err)
	case "PAYMENT.CAPTURE.REFUNDED", "PAYMENT.CAPTURE.REVERSED":
		return s.revokeForCapture(event)
	}
	return nil
}

func (s *Server) revokeForCapture(event webhookEvent) error {
	captureID := event.Resource.ID
	for _, link := range event.Resource.Links {
		if link.Rel == "up" && strings.Contains(link.Href, "/captures/") {
			captureID = link.Href[strings.LastIndex(link.Href, "/")+1:]
		}
	}
	order, found, err := s.Store.FindOrderByCapture(captureID)
	if err != nil || !found {
		return err
	}
	reason := strings.ToLower(strings.TrimPrefix(event.Type, "PAYMENT.CAPTURE."))
	_, err = s.Store.Revoke(order.LicenseKey, reason)
	if err == nil {
		s.recordRefund(order, reason)
	}
	s.Log.Event("revoked", map[string]any{"order": order.OrderID, "reason": reason, "key": logging.MaskKey(order.LicenseKey)})
	return err
}

// looksLikePayPal drops forged deliveries before the server spends a PayPal API call
// on them: every genuine webhook carries these headers and a PayPal certificate URL.
func looksLikePayPal(header http.Header) bool {
	for _, name := range []string{"Paypal-Transmission-Id", "Paypal-Transmission-Sig", "Paypal-Transmission-Time", "Paypal-Auth-Algo"} {
		if header.Get(name) == "" {
			return false
		}
	}
	certURL, err := url.Parse(header.Get("Paypal-Cert-Url"))
	return err == nil && certURL.Scheme == "https" &&
		(certURL.Hostname() == "paypal.com" || strings.HasSuffix(certURL.Hostname(), ".paypal.com"))
}

func ignorePending(err error) error {
	var known apiError
	if errors.As(err, &known) && known.code == "payment_incomplete" {
		return nil
	}
	return err
}
