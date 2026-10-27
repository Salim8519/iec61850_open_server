// Package lemon reads Lemon Squeezy order webhooks (ADR-0126). Lemon Squeezy is the
// merchant of record: it takes the card, adds tax and sends the receipt; this server
// only issues the ScadaBench key for a paid order and revokes it on a refund.
package lemon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// Verify checks the X-Signature header: the hex HMAC-SHA256 of the raw body keyed
// with the signing secret this server chose when the webhook was created.
func Verify(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature))))
}

// Order is the part of an order event the license server uses.
type Order struct {
	Event      string // order_created, order_refunded
	ID         string // numeric order id, shown in the Lemon Squeezy dashboard
	Identifier string // unguessable order UUID, also known to the buyer's browser
	Status     string // paid, pending, failed, refunded, partial_refund
	Email      string
	Name       string
	Currency   string
	Subtotal   string // price before tax and discount, "99.00"
	VariantID  string // the product variant bought
	TestMode   bool
	Custom     map[string]string // checkout[custom] values: invoice, hwid
}

type payload struct {
	Meta struct {
		EventName  string         `json:"event_name"`
		TestMode   bool           `json:"test_mode"`
		CustomData map[string]any `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Identifier string `json:"identifier"`
			Status     string `json:"status"`
			UserEmail  string `json:"user_email"`
			UserName   string `json:"user_name"`
			Currency   string `json:"currency"`
			Subtotal   int64  `json:"subtotal"`
			TestMode   bool   `json:"test_mode"`
			FirstItem  struct {
				VariantID int64 `json:"variant_id"`
			} `json:"first_order_item"`
		} `json:"attributes"`
	} `json:"data"`
}

// Parse reads an order event. Events about other objects return an Order with only
// Event set, so the caller can acknowledge and ignore them.
func Parse(body []byte) (Order, error) {
	var raw payload
	if err := json.Unmarshal(body, &raw); err != nil {
		return Order{}, errors.New("webhook body is not JSON")
	}
	order := Order{Event: raw.Meta.EventName, Custom: map[string]string{}}
	for key, value := range raw.Meta.CustomData {
		if text, ok := value.(string); ok {
			order.Custom[key] = strings.TrimSpace(text)
		}
	}
	if raw.Data.Type != "orders" {
		return order, nil
	}
	attributes := raw.Data.Attributes
	if raw.Data.ID == "" || attributes.Identifier == "" {
		return order, errors.New("order event without an order id")
	}
	order.ID, order.Identifier = raw.Data.ID, attributes.Identifier
	order.Status, order.Email, order.Name = attributes.Status, strings.TrimSpace(attributes.UserEmail), attributes.UserName
	order.Currency = strings.ToUpper(attributes.Currency)
	order.Subtotal = cents(attributes.Subtotal)
	order.TestMode = raw.Meta.TestMode || attributes.TestMode
	if attributes.FirstItem.VariantID > 0 {
		order.VariantID = strconv.FormatInt(attributes.FirstItem.VariantID, 10)
	}
	return order, nil
}

// cents turns Lemon Squeezy's integer cents into the two-decimal text the server
// compares prices with ("9900" -> "99.00").
func cents(value int64) string {
	sign := ""
	if value < 0 {
		sign, value = "-", -value
	}
	fraction := strconv.FormatInt(value%100, 10)
	if len(fraction) == 1 {
		fraction = "0" + fraction
	}
	return sign + strconv.FormatInt(value/100, 10) + "." + fraction
}
