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
	return m.post(ctx, url.Values{"from": {m.From}, "to": {to}, "subject": {subject}, "text": {text}})
}

// SendNews posts one news message with one-click unsubscribe headers (RFC 8058),
// which mailbox providers expect from bulk senders.
func (m *Mailgun) SendNews(ctx context.Context, to, subject, text, unsubscribeURL string) error {
	return m.post(ctx, url.Values{"from": {m.From}, "to": {to}, "subject": {subject}, "text": {text},
		"h:List-Unsubscribe":      {"<" + unsubscribeURL + ">"},
		"h:List-Unsubscribe-Post": {"List-Unsubscribe=One-Click"}, "o:tag": {"news"}})
}

func (m *Mailgun) post(ctx context.Context, form url.Values) error {
	if m == nil || m.Domain == "" || m.APIKey == "" {
		return errors.New("mailgun is not configured")
	}
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
		"on any other computer, enter the key and the activation code shown in ScadaBench,",
		"then copy the downloaded license file back to ScadaBench ('Offline activation').",
		"",
		"Each license works on one computer only. It cannot be moved to another computer.",
		"Reinstalling Windows on the same computer is fine; keep this email, you need the key then.",
		"A new computer or a new motherboard needs a new license.",
		"No refunds after a license is activated.",
	}
	if support != "" {
		lines = append(lines, "", "Questions? Reply to "+support+".")
	}
	return subject, strings.Join(lines, "\n")
}
