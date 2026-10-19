package certs

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSignVerifyRoundTripAndTamper(t *testing.T) {
	dir := t.TempDir()
	signer, err := Generate(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(dir, false); err == nil {
		t.Fatal("existing key overwritten without --force")
	}
	loaded, err := Load(dir)
	if err != nil || loaded.PublicHex() != signer.PublicHex() {
		t.Fatal("reloaded key differs")
	}
	bundle, err := signer.Sign(Payload{Email: "a@b.c", HWID: strings.Repeat("a", 64), IssuedAt: 1, LicenseID: "SUBS-X",
		MaxMachines: 1, Type: "lifetime"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := Verify(signer.Public(), bundle)
	if err != nil || payload.KeyID != signer.KeyID || payload.Version != 1 {
		t.Fatalf("verify failed: %v %+v", err, payload)
	}
	tampered := bundle
	tampered.Payload = strings.Replace(bundle.Payload, `"lifetime"`, `"lifetimf"`, 1)
	if _, err := Verify(signer.Public(), tampered); err == nil {
		t.Fatal("tampered payload verified")
	}
}

func TestPayloadIsCanonicalAlphabeticalJSON(t *testing.T) {
	signer, _ := Generate(t.TempDir(), false)
	bundle, _ := signer.Sign(Payload{HWID: "h", Type: "trial"})
	want := `{"email":"","expires_at":0,"hwid":"h","issued_at":0,"kid":"` + signer.KeyID +
		`","license_id":"","max_machines":0,"type":"trial","version":1}`
	if bundle.Payload != want {
		t.Fatalf("payload not canonical:\n%s\n%s", bundle.Payload, want)
	}
	if strings.Contains(bundle.Signature, "=") {
		t.Fatal("signature must be unpadded base64url")
	}
}

func TestLicenseKeyFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		key := NewLicenseKey()
		if len(key) != 24 || !strings.HasPrefix(key, "SUBS-") || strings.ContainsAny(key[5:], "01ILO") {
			t.Fatalf("bad key %s", key)
		}
		seen[key] = true
	}
	if len(seen) != 200 {
		t.Fatal("duplicate keys generated")
	}
	if !ValidHWID(strings.Repeat("0f", 32)) || ValidHWID(strings.Repeat("0F", 32)) || ValidHWID("abc") {
		t.Fatal("hwid validation wrong")
	}
}

// TestWriteCrossLanguageVector regenerates the fixture the Python and C++
// verifiers check: STUDIO_WRITE_VECTOR=path go test ./internal/certs -run Vector
func TestWriteCrossLanguageVector(t *testing.T) {
	target := os.Getenv("STUDIO_WRITE_VECTOR")
	if target == "" {
		t.Skip("set STUDIO_WRITE_VECTOR to regenerate the cross-language fixture")
	}
	dir := t.TempDir()
	seed := strings.Repeat("42", 32)
	os.WriteFile(KeyPath(dir), []byte(seed), 0o600)
	signer, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	hwid := strings.Repeat("ab", 32)
	lifetime, _ := signer.Sign(Payload{Email: "buyer@example.com", HWID: hwid, IssuedAt: 1760000000,
		LicenseID: "SUBS-TEST-TEST-TEST-TEST", MaxMachines: 1, Type: "lifetime"})
	trial, _ := signer.Sign(Payload{ExpiresAt: 1760000000 + 7*86400, HWID: hwid, IssuedAt: 1760000000,
		LicenseID: "TRIAL-abababababab", MaxMachines: 1, Type: "trial"})
	stamp, _ := signer.SignTime(1760000000)
	parts := map[string]string{"guid": PartHash("guid", "1a2b3c4d-aaaa-bbbb-cccc-0123456789ef"),
		"volume": PartHash("volume", "A1B2C3D4"), "firmware": PartHash("firmware", "00112233445566778899aabbccddeeff")}
	raw, _ := json.MarshalIndent(map[string]any{"public_key": signer.PublicHex(), "kid": signer.KeyID,
		"hwid": hwid, "lifetime": lifetime, "trial": trial, "server_time": stamp, "part_hashes": parts}, "", "  ")
	if err := os.WriteFile(target, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
