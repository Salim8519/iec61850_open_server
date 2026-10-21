package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"substation-license-server/internal/certs"
)

const testNonce = "00112233445566778899aabbccddeeff"

func answerBundle(t *testing.T, result map[string]any) certs.Bundle {
	t.Helper()
	raw, _ := json.Marshal(result["answer"])
	var bundle certs.Bundle
	if err := json.Unmarshal(raw, &bundle); err != nil || bundle.Payload == "" || bundle.Signature == "" {
		t.Fatalf("answer missing or malformed: %v (%v)", err, result)
	}
	return bundle
}

// wantPayload is the exact byte string every client verifies: fixed key order, no spaces.
func wantPayload(s *Server, hwid, licenseID, nonce, status string) string {
	raw, _ := json.Marshal(certs.Answer{HWID: hwid, KeyID: s.Signer.KeyID, LicenseID: licenseID, Nonce: nonce,
		Purpose: "verify-answer", Status: status, Time: s.Now().Unix(), Version: 1})
	return string(raw)
}

func TestVerifyAnswerIsSignedExactlyForEveryStatus(t *testing.T) {
	s := newTestServer(t)
	s.Now = func() time.Time { return time.Unix(1_800_000_123, 0) }
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	good, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	revoked, _, _ := s.IssueKey("gone@example.com", "lifetime", 0, "")
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": good.Key}, true)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": revoked.Key}, true)
	s.Store.Revoke(revoked.Key, "refund")
	cases := []struct{ name, hwid, key, status string }{
		{"valid", a, good.Key, "valid"},
		{"moved", b, good.Key, "moved"},
		{"revoked", a, revoked.Key, "revoked"},
		{"unknown", a, "SUBS-NOPE-NOPE-NOPE-NOPE", "unknown"},
		{"trial", a, "TRIAL-" + a[:12], "valid"},
		{"lower-case key", a, strings.ToLower(good.Key), "valid"},
		{"no key", a, "", "valid"},
	}
	for _, nonce := range []string{testNonce, ""} {
		for _, c := range cases {
			body := map[string]any{"hwid": c.hwid, "license_key": c.key}
			if nonce != "" {
				body["nonce"] = nonce
			}
			code, result := call(t, s, "/api/v1/verify", body, false)
			if code != http.StatusOK || result["status"] != c.status {
				t.Fatalf("%s: %d %v", c.name, code, result)
			}
			bundle := answerBundle(t, result)
			if want := wantPayload(s, c.hwid, c.key, nonce, c.status); bundle.Payload != want {
				t.Fatalf("%s: payload bytes\n got %s\nwant %s", c.name, bundle.Payload, want)
			}
			answer, err := certs.VerifyAnswer(s.Signer.Public(), bundle)
			if err != nil || answer.Status != c.status || answer.Nonce != nonce || answer.LicenseID != c.key ||
				answer.HWID != c.hwid || answer.Time != 1_800_000_123 || answer.KeyID != s.Signer.KeyID {
				t.Fatalf("%s: %+v %v", c.name, answer, err)
			}
			stampOf(t, s, result) // old apps still get the signed server time
		}
	}
}

func TestVerifyAnswerKeepsEveryOldFieldForOldApps(t *testing.T) {
	s := newTestServer(t)
	a := strings.Repeat("a", 64)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	_, result := call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": license.Key}, false)
	if result["status"] != "valid" || result["time"] == nil || result["server_time"] == nil {
		t.Fatalf("old fields missing: %v", result)
	}
	if payload := certificateOf(t, s, result); payload.HWID != a || payload.LicenseID != license.Key {
		t.Fatalf("certificate changed: %+v", payload)
	}
}

func TestVerifyNonceMustBe32LowercaseHexWhenPresent(t *testing.T) {
	s := newTestServer(t)
	a := strings.Repeat("a", 64)
	for _, bad := range []string{"short", strings.ToUpper(testNonce), testNonce + "0", testNonce[:31], strings.Repeat("g", 32),
		" " + testNonce[1:], strings.Repeat("0", 31) + "-"} {
		code, result := call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": "", "nonce": bad}, false)
		if code != http.StatusBadRequest || result["code"] != "bad_request" || result["answer"] != nil {
			t.Fatalf("nonce %q answered %d %v", bad, code, result)
		}
	}
	code, result := call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": "", "nonce": testNonce}, false)
	if code != http.StatusOK || answerBundle(t, result).Payload == "" {
		t.Fatalf("good nonce refused: %d %v", code, result)
	}
}

func TestVerifyAnswerCannotBeForgedOrReusedAsAnotherSignedThing(t *testing.T) {
	s := newTestServer(t)
	a := strings.Repeat("a", 64)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key}, true)
	_, result := call(t, s, "/api/v1/verify", map[string]any{"hwid": a, "license_key": license.Key, "nonce": testNonce}, false)
	bundle := answerBundle(t, result)
	// A different request nonce, or a flipped status, must not verify.
	for _, tampered := range []string{strings.Replace(bundle.Payload, testNonce, strings.Repeat("0", 32), 1),
		strings.Replace(bundle.Payload, `"valid"`, `"unknown"`, 1)} {
		if _, err := certs.VerifyAnswer(s.Signer.Public(), certs.Bundle{Payload: tampered, Signature: bundle.Signature}); err == nil {
			t.Fatalf("tampered answer verified: %s", tampered)
		}
	}
	// An answer is not a time stamp, a time stamp is not an answer, a certificate is neither.
	if _, err := certs.VerifyTime(s.Signer.Public(), bundle); err == nil {
		t.Fatal("an answer passed as a server time stamp")
	}
	stamp := certs.Bundle{}
	raw, _ := json.Marshal(result["server_time"])
	json.Unmarshal(raw, &stamp)
	if _, err := certs.VerifyAnswer(s.Signer.Public(), stamp); err == nil {
		t.Fatal("a server time stamp passed as an answer")
	}
	raw, _ = json.Marshal(result["certificate"])
	certificate := certs.Bundle{}
	json.Unmarshal(raw, &certificate)
	if _, err := certs.VerifyAnswer(s.Signer.Public(), certificate); err == nil {
		t.Fatal("a license certificate passed as an answer")
	}
	// Another server key never verifies it.
	other, _ := certs.Generate(t.TempDir(), false)
	if _, err := certs.VerifyAnswer(other.Public(), bundle); err == nil {
		t.Fatal("a foreign key verified the answer")
	}
	if strings.Contains(bundle.Signature, "=") {
		t.Fatal("signature must be unpadded base64url")
	}
}
