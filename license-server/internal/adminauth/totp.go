package adminauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Authenticator codes follow RFC 6238 with the settings every phone app uses by
// default: SHA-1, 6 digits, 30-second steps. One step of clock drift is accepted.
const (
	totpStep   = 30
	totpDigits = 6
)

var secretEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret returns a 160-bit secret as 32 base32 letters (the RFC 4226 size).
func NewSecret() string { return secretEncoding.EncodeToString(randomBytes(20)) }

// FormatSecret groups the secret in fours so it is easy to type into a phone.
func FormatSecret(secret string) string {
	var groups []string
	for start := 0; start < len(secret); start += 4 {
		groups = append(groups, secret[start:min(start+4, len(secret))])
	}
	return strings.Join(groups, " ")
}

// SetupURI is the otpauth:// link authenticator apps import.
func SetupURI(secret, account string) string {
	label := url.PathEscape("ScadaBench:" + account)
	return "otpauth://totp/" + label + "?secret=" + secret + "&issuer=ScadaBench&digits=6&period=30"
}

func decodeSecret(secret string) ([]byte, error) {
	clean := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	return secretEncoding.DecodeString(strings.TrimRight(clean, "="))
}

// codeAt computes the code for one 30-second counter value.
func codeAt(key []byte, counter uint64) string {
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, value%1_000_000)
}

// Code returns the current code (tests and the setup check use it).
func Code(secret string, now time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	return codeAt(key, uint64(now.Unix()/totpStep)), nil
}

// VerifyCode accepts the code for the current step or one step either side. It
// refuses any step at or before lastStep, so a code that was seen once cannot be
// replayed. It returns the step that matched.
func VerifyCode(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits {
		return 0, false
	}
	key, err := decodeSecret(secret)
	if err != nil || len(key) == 0 {
		return 0, false
	}
	current := now.Unix() / totpStep
	for step := current - 1; step <= current+1; step++ {
		if step <= lastStep || step < 0 {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(codeAt(key, uint64(step))), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}
