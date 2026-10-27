package lemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client creates checkouts through the Lemon Squeezy API, so the price set on the
// owner page is the price the buyer pays by card (ADR-0126).
type Client struct {
	Base    string // https://api.lemonsqueezy.com/v1
	APIKey  string
	StoreID string
	HTTP    *http.Client
}

// CheckoutRequest is one buyer's checkout: price in "99.00" form plus custom data
// that comes back in the order webhook.
type CheckoutRequest struct {
	VariantID string
	Price     string
	Custom    map[string]string
}

// CreateCheckout returns the checkout URL for one buyer at the given price.
func (c Client) CreateCheckout(ctx context.Context, request CheckoutRequest) (string, error) {
	cents, err := toCents(request.Price)
	if err != nil {
		return "", err
	}
	body := map[string]any{"data": map[string]any{
		"type": "checkouts",
		"attributes": map[string]any{
			"custom_price":     cents,
			"product_options":  map[string]any{"enabled_variants": []string{request.VariantID}},
			"checkout_options": map[string]any{"embed": true, "media": false},
			"checkout_data":    map[string]any{"custom": request.Custom},
			"expires_at":       time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		},
		"relationships": map[string]any{
			"store":   map[string]any{"data": map[string]string{"type": "stores", "id": c.StoreID}},
			"variant": map[string]any{"data": map[string]string{"type": "variants", "id": request.VariantID}},
		},
	}}
	raw, _ := json.Marshal(body)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.Base, "/")+"/checkouts", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpRequest.Header.Set("Accept", "application/vnd.api+json")
	httpRequest.Header.Set("Content-Type", "application/vnd.api+json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var answer struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil || response.StatusCode >= 300 {
		return "", fmt.Errorf("lemon squeezy checkout failed with status %d", response.StatusCode)
	}
	if !strings.HasPrefix(answer.Data.Attributes.URL, "https://") {
		return "", errors.New("lemon squeezy returned no checkout url")
	}
	return answer.Data.Attributes.URL, nil
}

// toCents turns "99.00" into 9900.
func toCents(price string) (int64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
	if err != nil || value <= 0 {
		return 0, errors.New("price must be a positive number")
	}
	return int64(value*100 + 0.5), nil
}
