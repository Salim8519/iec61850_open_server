// Package adminauth holds the owner page's sign-in rules: password hashing, the
// authenticator-app code, recovery codes, sessions and lockout. It has no HTTP,
// database or mail code; the api package wires it to the web page.
package adminauth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MinPasswordLength is the shortest owner password accepted.
const MinPasswordLength = 8

// passwordIterations follows the OWASP minimum for PBKDF2-HMAC-SHA256. PBKDF2 is in
// the Go standard library and needs almost no memory, which matters under the
// service's 64 MB cap; a hash takes about a quarter of a second, so guessing is slow.
const passwordIterations = 600_000

// ErrWeakPassword is returned for passwords shorter than MinPasswordLength.
var ErrWeakPassword = fmt.Errorf("the password must be at least %d characters", MinPasswordLength)

// HashPassword returns "pbkdf2-sha256$<iterations>$<salt hex>$<hash hex>".
func HashPassword(password string) (string, error) {
	if len([]rune(password)) < MinPasswordLength {
		return "", ErrWeakPassword
	}
	salt := randomBytes(16)
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations, hex.EncodeToString(salt),
		hex.EncodeToString(key)), nil
}

// CheckPassword compares in constant time; a malformed hash never matches.
func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	salt, saltErr := hex.DecodeString(parts[2])
	want, wantErr := hex.DecodeString(parts[3])
	if err != nil || saltErr != nil || wantErr != nil || iterations < 100_000 || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// randomBytes reads the system random source; failure stops the program because
// every secret here depends on it.
func randomBytes(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(errors.New("system random source failed: " + err.Error()))
	}
	return buf
}

// RandomToken returns n random bytes as lowercase hex.
func RandomToken(n int) string { return hex.EncodeToString(randomBytes(n)) }
