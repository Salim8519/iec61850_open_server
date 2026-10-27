package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/lemon"
	"substation-license-server/internal/store"
)

// Lemon Squeezy orders are stored as "LS-<order UUID>", so the buyer's browser, which
// learns the UUID when the checkout succeeds, can fetch its own key and nobody can
// guess another buyer's (ADR-0126).
const lemonPrefix = "LS-"

// lemonStart returns the hosted checkout link for the buy page, carrying the app's
// computer id (when the app opened the page) and the "send me news" choice.
func (s *Server) lemonStart(r *http.Request, _ string) (any, error) {
	if !s.Config.LemonReady() {
		return nil, errCheckoutOff
	}
	var request struct {
		HWID string `json:"hwid"`
		News bool   `json:"news"`
	}
	decode(r, &request)
	link, err := url.Parse(s.Config.LemonCheckoutURL)
	if err != nil {
		return nil, errCheckoutOff
	}
	query := link.Query()
	query.Set("media", "0")
	if request.News {
		query.Set("checkout[custom][news]", "1")
	}
	if hwid := strings.TrimSpace(request.HWID); certs.ValidHWID(hwid) {
		query.Set("checkout[custom][hwid]", hwid)
	}
	link.RawQuery = query.Encode()
	return map[string]string{"url": link.String()}, nil
}

// lemonKey answers the buy page once the order's webhook has arrived; until then it
// says "pending" and the page asks again a few seconds later.
func (s *Server) lemonKey(r *http.Request, _ string) (any, error) {
	identifier := strings.TrimSpace(r.URL.Query().Get("order"))
	if len(identifier) < 32 || len(identifier) > 40 {
		return nil, badRequest("order is required")
	}
	order, found, err := s.Store.GetOrder(lemonPrefix + identifier)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]bool{"pending": true}, nil
	}
	return s.purchaseResult(order), nil
}

// lemonWebhook issues the key for a paid order and revokes it after a refund.
func (s *Server) lemonWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow(s.clientIP(r), 0, s.Now()) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || !s.Config.LemonReady() {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	if !lemon.Verify(s.Config.LemonSigningSecret, body, r.Header.Get("X-Signature")) {
		s.Log.Event("lemon_webhook_rejected", map[string]any{"ip": s.clientIP(r)})
		http.Error(w, "signature rejected", http.StatusBadRequest)
		return
	}
	order, err := lemon.Parse(body)
	if err != nil {
		http.Error(w, "bad event", http.StatusBadRequest)
		return
	}
	if err := s.handleLemon(order); err != nil {
		s.Log.Event("lemon_webhook_error", map[string]any{"event": order.Event, "order": order.ID, "error": err.Error()})
		http.Error(w, "retry later", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLemon(order lemon.Order) error {
	if order.Identifier == "" {
		return nil
	}
	if order.TestMode && !s.Config.LemonAcceptTestMode {
		s.Log.Event("lemon_test_order_ignored", map[string]any{"order": order.ID})
		return nil
	}
	if order.VariantID != s.Config.LemonVariantID {
		s.Log.Event("lemon_other_product", map[string]any{"order": order.ID, "variant": order.VariantID})
		return nil
	}
	switch order.Event {
	case "order_created":
		if order.Status != "paid" {
			s.Log.Event("lemon_unpaid_order", map[string]any{"order": order.ID, "status": order.Status})
			return nil
		}
		mode := "lemon"
		if order.TestMode {
			mode = "lemon-test"
		}
		_, err := s.issuePaid(paid{OrderID: lemonPrefix + order.Identifier, CaptureID: lemonPrefix + order.ID,
			Email: order.Email, Name: order.Name, Amount: order.Subtotal, Currency: order.Currency,
			HWID: order.Custom["hwid"], Mode: mode, News: order.Custom["news"] == "1"})
		return err
	case "order_refunded":
		if order.Status != "refunded" {
			return nil
		}
		issued, found, err := s.Store.GetOrder(lemonPrefix + order.Identifier)
		if err != nil || !found {
			return err
		}
		return s.revokeOrder(issued, "refunded")
	}
	return nil
}

// orderIsLemon tells the sales book which gateway an order came through.
func orderIsLemon(order store.Order) bool { return strings.HasPrefix(order.OrderID, lemonPrefix) }
