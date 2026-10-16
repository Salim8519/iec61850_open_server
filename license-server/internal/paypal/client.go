// Package paypal talks to the PayPal Orders v2 and webhook-verification REST APIs.
package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client caches one OAuth token; BaseURL is overridable for tests.
type Client struct {
	BaseURL  string
	ID       string
	Secret   string
	HTTP     *http.Client
	mu       sync.Mutex
	token    string
	tokenEnd time.Time
}

// New creates a client for the sandbox or live base URL.
func New(base, id, secret string) *Client {
	return &Client{BaseURL: strings.TrimRight(base, "/"), ID: id, Secret: secret,
		HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// Capture is the part of a completed capture the server needs.
type Capture struct {
	OrderID   string
	CaptureID string
	Status    string
	Amount    string
	Currency  string
	Email     string
	CustomID  string
	InvoiceID string
	Name      string
	Country   string
	PayerID   string
}

// OrderRequest describes one digital item; Brand is what PayPal shows the buyer.
type OrderRequest struct {
	Product, Brand, Amount, Currency, CustomID, InvoiceID string
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenEnd) {
		return c.token, nil
	}
	body := strings.NewReader(url.Values{"grant_type": {"client_credentials"}}.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/oauth2/token", body)
	if err != nil {
		return "", err
	}
	request.SetBasicAuth(c.ID, c.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := c.do(request, &result); err != nil {
		return "", fmt.Errorf("paypal login failed: %w", err)
	}
	c.token = result.AccessToken
	c.tokenEnd = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return c.token, nil
}

func (c *Client) do(request *http.Request, target any) error {
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode >= 300 {
		return fmt.Errorf("paypal returned %d: %s", response.StatusCode, truncate(string(raw), 300))
	}
	return json.Unmarshal(raw, target)
}

func (c *Client) call(ctx context.Context, method, path string, payload, target any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	return c.do(request, target)
}

// CreateOrder starts a one-item order and returns PayPal's order id. NO_SHIPPING: a
// license is digital, so PayPal never asks the buyer for a delivery address.
func (c *Client) CreateOrder(ctx context.Context, order OrderRequest) (string, error) {
	payload := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"description": order.Product, "custom_id": order.CustomID, "invoice_id": order.InvoiceID,
			"amount": map[string]string{"currency_code": order.Currency, "value": order.Amount},
		}},
		"application_context": map[string]string{"brand_name": order.Brand,
			"shipping_preference": "NO_SHIPPING", "user_action": "PAY_NOW"},
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.call(ctx, http.MethodPost, "/v2/checkout/orders", payload, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

type orderResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Payer  struct {
		Email   string `json:"email_address"`
		PayerID string `json:"payer_id"`
		Name    struct {
			Given   string `json:"given_name"`
			Surname string `json:"surname"`
		} `json:"name"`
		Address struct {
			Country string `json:"country_code"`
		} `json:"address"`
	} `json:"payer"`
	PurchaseUnits []struct {
		CustomID  string `json:"custom_id"`
		InvoiceID string `json:"invoice_id"`
		Payments  struct {
			Captures []struct {
				ID        string `json:"id"`
				Status    string `json:"status"`
				InvoiceID string `json:"invoice_id"`
				Amount    struct {
					Currency string `json:"currency_code"`
					Value    string `json:"value"`
				} `json:"amount"`
			} `json:"captures"`
		} `json:"payments"`
	} `json:"purchase_units"`
}

func (r orderResult) capture() (Capture, error) {
	result := Capture{OrderID: r.ID, Status: r.Status, Email: r.Payer.Email, PayerID: r.Payer.PayerID,
		Name:    strings.TrimSpace(r.Payer.Name.Given + " " + r.Payer.Name.Surname),
		Country: r.Payer.Address.Country}
	if len(r.PurchaseUnits) == 0 || len(r.PurchaseUnits[0].Payments.Captures) == 0 {
		return result, errors.New("paypal order has no capture yet")
	}
	unit := r.PurchaseUnits[0]
	captured := unit.Payments.Captures[0]
	result.CaptureID, result.Status = captured.ID, captured.Status
	result.Amount, result.Currency, result.CustomID = captured.Amount.Value, captured.Amount.Currency, unit.CustomID
	result.InvoiceID = unit.InvoiceID
	if result.InvoiceID == "" {
		result.InvoiceID = captured.InvoiceID
	}
	return result, nil
}

// CaptureOrder captures an approved order. An already-captured order is read
// back instead, so retries and the webhook path converge on the same result.
func (c *Client) CaptureOrder(ctx context.Context, orderID string) (Capture, error) {
	var result orderResult
	path := "/v2/checkout/orders/" + url.PathEscape(orderID)
	err := c.call(ctx, http.MethodPost, path+"/capture", map[string]any{}, &result)
	if err != nil && strings.Contains(err.Error(), "ORDER_ALREADY_CAPTURED") {
		err = c.call(ctx, http.MethodGet, path, nil, &result)
	}
	if err != nil {
		return Capture{}, err
	}
	return result.capture()
}

// GetOrder reads an order (used by the webhook path).
func (c *Client) GetOrder(ctx context.Context, orderID string) (Capture, error) {
	var result orderResult
	if err := c.call(ctx, http.MethodGet, "/v2/checkout/orders/"+url.PathEscape(orderID), nil, &result); err != nil {
		return Capture{}, err
	}
	return result.capture()
}

// VerifyWebhook asks PayPal whether a webhook delivery is genuine.
func (c *Client) VerifyWebhook(ctx context.Context, webhookID string, header http.Header, body []byte) error {
	if webhookID == "" {
		return errors.New("paypal_webhook_id is not configured")
	}
	payload := map[string]any{
		"auth_algo": header.Get("Paypal-Auth-Algo"), "cert_url": header.Get("Paypal-Cert-Url"),
		"transmission_id": header.Get("Paypal-Transmission-Id"), "transmission_sig": header.Get("Paypal-Transmission-Sig"),
		"transmission_time": header.Get("Paypal-Transmission-Time"), "webhook_id": webhookID,
		"webhook_event": json.RawMessage(body),
	}
	var result struct {
		Status string `json:"verification_status"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", payload, &result); err != nil {
		return err
	}
	if result.Status != "SUCCESS" {
		return errors.New("paypal webhook signature rejected")
	}
	return nil
}

func truncate(text string, limit int) string {
	if len(text) > limit {
		return text[:limit] + "..."
	}
	return text
}
