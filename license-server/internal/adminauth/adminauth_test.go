package adminauth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B, SHA-1 rows, last six digits.
func TestTOTPMatchesRFC6238(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{59: "287082", 1111111109: "081804", 1234567890: "005924", 2000000000: "279037"}
	for at, want := range cases {
		if got, _ := Code(secret, time.Unix(at, 0)); got != want {
			t.Fatalf("time %d: got %s want %s", at, got, want)
		}
	}
}

func TestVerifyCodeAllowsDriftAndRefusesReplay(t *testing.T) {
	secret := NewSecret()
	now := time.Unix(1_800_000_000, 0)
	previous, _ := Code(secret, now.Add(-30*time.Second))
	step, ok := VerifyCode(secret, previous, now, 0)
	if !ok {
		t.Fatal("one step of drift must be accepted")
	}
	if _, ok := VerifyCode(secret, previous, now, step); ok {
		t.Fatal("a used code must not work twice")
	}
	old, _ := Code(secret, now.Add(-2*time.Minute))
	if _, ok := VerifyCode(secret, old, now, 0); ok {
		t.Fatal("a two-minute-old code must be refused")
	}
	if len(FormatSecret(secret)) != 39 || len(secret) != 32 {
		t.Fatalf("secret %q must be 32 letters in 8 groups", secret)
	}
}

// RFC 7914 section 11 PBKDF2-HMAC-SHA256 vector, to pin the library we rely on.
func TestPasswordHashUsesStandardPBKDF2(t *testing.T) {
	key, _ := pbkdf2.Key(sha256.New, "passwd", []byte("salt"), 1, 64)
	if !strings.HasPrefix(hex.EncodeToString(key), "55ac046e56e3089fec1691c22544b605") {
		t.Fatalf("unexpected PBKDF2 output %x", key)
	}
	if _, err := HashPassword("short"); err != ErrWeakPassword {
		t.Fatal("short passwords must be refused")
	}
	hash, err := HashPassword("correct horse battery")
	if err != nil || !CheckPassword(hash, "correct horse battery") || CheckPassword(hash, "correct horse batterY") {
		t.Fatalf("hash check failed: %v", err)
	}
	if CheckPassword("plain", "plain") || CheckPassword("pbkdf2-sha256$1$00$00", "x") {
		t.Fatal("malformed or weak hashes must never match")
	}
}

func TestRecoveryCodesWorkOnceAndResetRaisesEpoch(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	account, enrollment, err := NewAccount("a long owner password", 4, now)
	if err != nil || account.Epoch != 5 || len(enrollment.RecoveryCodes) != RecoveryCodeCount {
		t.Fatalf("bad account %+v %v", account, err)
	}
	code := strings.ToLower(enrollment.RecoveryCodes[3])
	if account.CheckSecondFactor(code, now) != FactorRecovery {
		t.Fatal("recovery code must be accepted, case-insensitive")
	}
	if account.CheckSecondFactor(code, now) != FactorNone || len(account.RecoveryHashes) != RecoveryCodeCount-1 {
		t.Fatal("recovery code must be spent")
	}
	current, _ := Code(account.TOTPSecret, now)
	if account.CheckSecondFactor(current, now) != FactorAuthenticator || account.CheckAuthenticator(current, now) {
		t.Fatal("authenticator code must work once")
	}
}

func TestKeeperRoundTripAndUpdate(t *testing.T) {
	keeper := NewKeeper(t.TempDir())
	if _, ready, err := keeper.Load(); ready || err != nil {
		t.Fatal("no account before setup")
	}
	account, _, _ := NewAccount("a long owner password", 0, time.Now())
	if err := keeper.Replace(account); err != nil {
		t.Fatal(err)
	}
	updated, ready, err := keeper.Update(func(a *Account) bool { a.LastStep = 42; return true })
	if !ready || err != nil || updated.LastStep != 42 {
		t.Fatalf("update failed %v", err)
	}
	if loaded, _, _ := keeper.Load(); loaded.LastStep != 42 || loaded.PasswordHash != account.PasswordHash {
		t.Fatal("update was not saved")
	}
}

func TestSessionsExpireAndFollowEpoch(t *testing.T) {
	sessions := NewSessions()
	now := time.Unix(1_800_000_000, 0)
	token, session := sessions.Create(1, "1.2.3.4", now)
	if len(session.CSRF) != 64 {
		t.Fatal("csrf token missing")
	}
	if _, ok := sessions.Get(token, 1, now.Add(29*time.Minute)); !ok {
		t.Fatal("active session must stay")
	}
	if _, ok := sessions.Get(token, 2, now.Add(30*time.Minute)); ok {
		t.Fatal("a reset must end older sessions")
	}
	token, _ = sessions.Create(1, "", now)
	if _, ok := sessions.Get(token, 1, now.Add(31*time.Minute)); ok {
		t.Fatal("idle session must expire")
	}
	for i := 0; i < maxSessions+5; i++ {
		sessions.Create(1, "", now.Add(time.Duration(i)*time.Second))
	}
	if sessions.Count() != maxSessions || sessions.DeleteAll() != maxSessions || sessions.Count() != 0 {
		t.Fatal("session table must stay bounded")
	}
}

func TestLockoutClosesAfterLimit(t *testing.T) {
	lock := NewLockout(3, time.Hour, time.Hour)
	now := time.Unix(1_800_000_000, 0)
	if lock.Fail(now) || lock.Fail(now) {
		t.Fatal("not locked yet")
	}
	if !lock.Fail(now) {
		t.Fatal("third failure must lock")
	}
	if locked, _ := lock.Locked(now.Add(59 * time.Minute)); !locked {
		t.Fatal("must stay locked for the duration")
	}
	if locked, _ := lock.Locked(now.Add(61 * time.Minute)); locked {
		t.Fatal("lock must end")
	}
}
