package licensing

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGoSignedFixture reuses studio/tests/fixtures/licensing/go_vector.json: certificates
// signed by the license server's Go crypto/ed25519, which Python also verifies.
func TestGoSignedFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "licensing", "go_vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		HWID      string `json:"hwid"`
		Kid       string `json:"kid"`
		PublicKey string `json:"public_key"`
		Lifetime  struct{ Payload, Signature string }
		Trial     struct{ Payload, Signature string }
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	public, _ := hex.DecodeString(fixture.PublicKey)
	if KeyID(public) != fixture.Kid {
		t.Fatalf("kid %s, want %s", KeyID(public), fixture.Kid)
	}
	keys := map[string][]byte{fixture.Kid: public}
	lifetime, err := VerifyBundle(Bundle{fixture.Lifetime.Payload, fixture.Lifetime.Signature}, keys)
	if err != nil {
		t.Fatal(err)
	}
	status, _ := EvaluateCertificate(lifetime, fixture.HWID, 1_760_000_060, 0, false, ClockToleranceSeconds)
	if !status.Licensed || status.Mode != "lifetime" || status.Email != "buyer@example.com" {
		t.Fatalf("lifetime status %+v", status)
	}
	trial, err := VerifyBundle(Bundle{fixture.Trial.Payload, fixture.Trial.Signature}, keys)
	if err != nil {
		t.Fatal(err)
	}
	status, _ = EvaluateCertificate(trial, fixture.HWID, 1_760_000_000+86400+5, 0, false, TrialClockToleranceSeconds)
	if !status.Licensed || status.DaysRemaining != int64(6) || status.ExpiresAt != int64(1760604800) {
		t.Fatalf("trial status %+v", status)
	}
	tampered := Bundle{fixture.Lifetime.Payload + " ", fixture.Lifetime.Signature}
	if _, err := VerifyBundle(tampered, keys); err == nil || err.Error() != CertChanged {
		t.Fatalf("tampered payload: %v", err)
	}
	if _, err := VerifyBundle(Bundle{fixture.Trial.Payload, fixture.Lifetime.Signature}, keys); err == nil {
		t.Fatal("a signature of another payload must fail")
	}
}
