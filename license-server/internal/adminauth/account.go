package adminauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"
)

// RecoveryCodeCount is how many one-time codes setup prints.
const RecoveryCodeCount = 10

// recoveryAlphabet leaves out 0/O, 1/I/L so codes can be read off paper.
const recoveryAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// Account is the single owner login. Only hashes of the password and recovery
// codes are kept; the authenticator secret must stay readable to check codes.
type Account struct {
	PasswordHash   string   `json:"password_hash"`
	TOTPSecret     string   `json:"totp_secret"`
	LastStep       int64    `json:"last_step"`
	RecoveryHashes []string `json:"recovery_hashes"`
	// Epoch rises on every reset; sessions from an older epoch are refused.
	Epoch     int64 `json:"epoch"`
	CreatedAt int64 `json:"created_at"`
}

// Enrollment is shown to the owner once, at setup, and never stored in plain text.
type Enrollment struct {
	Secret        string   `json:"secret"`
	SecretGrouped string   `json:"secret_grouped"`
	SetupURI      string   `json:"setup_uri"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// NewAccount builds a fresh login: new password hash, authenticator secret and
// recovery codes. previousEpoch is the old account's epoch (0 when none).
func NewAccount(password string, previousEpoch int64, now time.Time) (Account, Enrollment, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return Account{}, Enrollment{}, err
	}
	secret := NewSecret()
	enrollment := Enrollment{Secret: secret, SecretGrouped: FormatSecret(secret),
		SetupURI: SetupURI(secret, "owner")}
	account := Account{PasswordHash: hash, TOTPSecret: secret, Epoch: previousEpoch + 1, CreatedAt: now.Unix()}
	for range RecoveryCodeCount {
		code := newRecoveryCode()
		enrollment.RecoveryCodes = append(enrollment.RecoveryCodes, code)
		account.RecoveryHashes = append(account.RecoveryHashes, hashRecovery(code))
	}
	return account, enrollment, nil
}

func newRecoveryCode() string {
	raw := randomBytes(10)
	letters := make([]byte, len(raw))
	for i, b := range raw {
		letters[i] = recoveryAlphabet[int(b)%len(recoveryAlphabet)]
	}
	return string(letters[:5]) + "-" + string(letters[5:])
}

func normalizeRecovery(code string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
}

func hashRecovery(code string) string {
	sum := sha256.Sum256([]byte("scadabench-recovery|" + normalizeRecovery(code)))
	return hex.EncodeToString(sum[:])
}

// SecondFactor says which second step was used at sign-in.
type SecondFactor int

// Results of CheckSecondFactor.
const (
	FactorNone SecondFactor = iota
	FactorAuthenticator
	FactorRecovery
)

// CheckAuthenticator accepts a fresh 6-digit code and remembers its step so it
// cannot be used twice. The caller saves the account when it returns true.
func (a *Account) CheckAuthenticator(code string, now time.Time) bool {
	step, ok := VerifyCode(a.TOTPSecret, code, now, a.LastStep)
	if ok {
		a.LastStep = step
	}
	return ok
}

// CheckSecondFactor accepts an authenticator code, or else spends one recovery
// code. The caller saves the account whenever the result is not FactorNone.
func (a *Account) CheckSecondFactor(code string, now time.Time) SecondFactor {
	if a.CheckAuthenticator(code, now) {
		return FactorAuthenticator
	}
	if len(normalizeRecovery(code)) != 10 {
		return FactorNone
	}
	want := []byte(hashRecovery(code))
	for i, stored := range a.RecoveryHashes {
		if subtle.ConstantTimeCompare([]byte(stored), want) == 1 {
			a.RecoveryHashes = append(a.RecoveryHashes[:i:i], a.RecoveryHashes[i+1:]...)
			return FactorRecovery
		}
	}
	return FactorNone
}
