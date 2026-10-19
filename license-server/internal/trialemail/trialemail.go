// Package trialemail normalises and hashes the email address a suspicious trial
// request must confirm, and makes the one-time codes sent to it. Addresses are only
// ever stored as salted hashes and never written to logs.
package trialemail

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrInvalid means the text is not a usable email address.
var ErrInvalid = errors.New("enter a valid email address, for example name@company.com")

// Normalize lowercases, drops a +tag, and for Gmail drops dots and maps googlemail.com,
// so one mailbox always gives the same value.
func Normalize(address string) (string, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	local, domain, found := strings.Cut(address, "@")
	if !found || local == "" || !strings.Contains(domain, ".") || strings.ContainsAny(address, " \t\r\n,;<>\"") ||
		strings.Contains(domain, "@") || len(address) > 254 {
		return "", ErrInvalid
	}
	if plus := strings.Index(local, "+"); plus >= 0 {
		local = local[:plus]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	if local == "" {
		return "", ErrInvalid
	}
	return local + "@" + domain, nil
}

// Domain returns the part after "@" of a normalised address.
func Domain(normalized string) string {
	_, domain, _ := strings.Cut(normalized, "@")
	return domain
}

// Hash is the stored form of a normalised address.
func Hash(normalized string) string {
	digest := sha256.Sum256([]byte("scadabench-trial-email-v1|" + normalized))
	return hex.EncodeToString(digest[:])
}

// NewCode returns a random 6-digit code.
func NewCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic("system random source failed: " + err.Error())
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// MAC binds a code to one machine ID and one address; only the MAC is stored.
func MAC(secret, hwid, emailHash, code string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(hwid + "|" + emailHash + "|" + strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Same compares two MACs in constant time.
func Same(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }

// disposable is a short list of well-known throwaway-mailbox services. It is a speed
// bump, not a guarantee; keep it small and only add services that exist to dodge sign-ups.
var disposable = map[string]bool{
	"mailinator.com": true, "guerrillamail.com": true, "guerrillamail.net": true, "sharklasers.com": true,
	"grr.la": true, "10minutemail.com": true, "temp-mail.org": true, "tempmail.com": true, "yopmail.com": true,
	"trashmail.com": true, "getnada.com": true, "dispostable.com": true, "maildrop.cc": true,
	"throwawaymail.com": true, "fakeinbox.com": true, "mintemail.com": true, "tempail.com": true,
	"emailondeck.com": true, "mohmal.com": true, "mailnesia.com": true, "mytemp.email": true,
	"tempmailo.com": true, "burnermail.io": true, "moakt.com": true, "tempr.email": true,
}

// Disposable reports a known throwaway-mailbox domain.
func Disposable(domain string) bool { return disposable[strings.ToLower(domain)] }
