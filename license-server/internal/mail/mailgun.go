// Package mail delivers license keys through the Mailgun HTTP API.
package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Mailgun sends transactional email; BaseURL is overridable for tests.
type Mailgun struct {
	Domain  string
	APIKey  string
	From    string
	BaseURL string
	Client  *http.Client
}

// New picks the US or EU API host.
func New(domain, apiKey, from, region string) *Mailgun {
	base := "https://api.mailgun.net"
	if region == "eu" {
		base = "https://api.eu.mailgun.net"
	}
	return &Mailgun{Domain: domain, APIKey: apiKey, From: from, BaseURL: base,
		Client: &http.Client{Timeout: 10 * time.Second}}
}

// Send posts one plain-text message.
func (m *Mailgun) Send(ctx context.Context, to, subject, text string) error {
	if m == nil || m.Domain == "" || m.APIKey == "" {
		return errors.New("mailgun is not configured")
	}
	form := url.Values{"from": {m.From}, "to": {to}, "subject": {subject}, "text": {text}}
	endpoint := fmt.Sprintf("%s/v3/%s/messages", m.BaseURL, url.PathEscape(m.Domain))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.SetBasicAuth("api", m.APIKey)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := m.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("mailgun returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// LicenseEmail builds the plain-English delivery text for a new key.
func LicenseEmail(product, key, publicURL, support string) (string, string) {
	subject := "Your " + product + " key"
	lines := []string{
		"Thank you for your purchase.",
		"",
		"Your license key: " + key,
		"",
		"How to activate (takes one minute):",
		"1. Open ScadaBench on the computer you want to use.",
		"2. Paste the key into the 'License key' box and press 'Activate this computer'.",
		"",
		"Computer without internet? Open " + strings.TrimRight(publicURL, "/") + "/activate",
		"on any other computer, enter the key and the Machine ID shown in ScadaBench,",
		"then copy the downloaded license file back to ScadaBench ('Offline activation').",
		"",
		"The key works on one computer. Keep this email; you need the key to reinstall.",
	}
	if support != "" {
		lines = append(lines, "", "Questions? Reply to "+support+".")
	}
	return subject, strings.Join(lines, "\n")
}
