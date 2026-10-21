package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

// newHostKey is a fresh public VM host key as the app sends it: 32 bytes, base64url, no padding.
func newHostKey(t *testing.T) string {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(public)
}

func TestActivationCarriesTheHostKeyIntoTheCertificateInAlphabeticalOrder(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	hwid, hostKey := strings.Repeat("a", 64), newHostKey(t)
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key,
		"host_key": hostKey}, true)
	if code != http.StatusOK {
		t.Fatalf("activate answered %d %v", code, result)
	}
	payload := certificateOf(t, s, result)
	if payload.HostKey != hostKey || payload.Version != 1 || payload.Type != "lifetime" {
		t.Fatalf("certificate: %+v", payload)
	}
	text := result["certificate"].(map[string]any)["payload"].(string)
	expires, host, machine := strings.Index(text, `"expires_at"`), strings.Index(text, `"host_key"`), strings.Index(text, `"hwid"`)
	if !(expires < host && host < machine) {
		t.Fatalf("host_key is not between expires_at and hwid: %s", text)
	}
}

func TestActivationWithoutAHostKeyGivesTheOldCertificateWithNoHostKeyField(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": strings.Repeat("a", 64),
		"license_key": license.Key}, true)
	if code != http.StatusOK {
		t.Fatalf("activate answered %d %v", code, result)
	}
	if text := result["certificate"].(map[string]any)["payload"].(string); strings.Contains(text, "host_key") {
		t.Fatalf("a certificate without a host key still names one: %s", text)
	}
}

func TestOnlyAStrictHostKeyIsAccepted(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	hwid := strings.Repeat("a", 64)
	good := newHostKey(t)
	wrong := map[string]string{
		"too short":          good[:42],
		"too long":           good + "A",
		"padded":             good + "=",
		"standard alphabet":  strings.NewReplacer("-", "+", "_", "/").Replace(base64.RawURLEncoding.EncodeToString(append([]byte{0xfb, 0xff}, make([]byte, 30)...))),
		"spaces":             " " + good,
		"not base64":         strings.Repeat("!", 43),
		"sixteen bytes":      base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
		"sixty-four bytes":   base64.RawURLEncoding.EncodeToString(make([]byte, 64)),
		"non-canonical tail": good[:42] + "B", // a last character with bits that are not zero
	}
	for name, hostKey := range wrong {
		code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key,
			"host_key": hostKey}, true)
		if code != http.StatusBadRequest || result["code"] != "bad_request" {
			t.Errorf("%s: answered %d %v", name, code, result)
		}
	}
	if after, _ := s.Store.GetLicense(license.Key); after.HWID != "" || after.HostKey != "" {
		t.Fatalf("a refused host key still bound the license: %+v", after)
	}
}

func TestActivatingAgainOnTheSameComputerReplacesTheHostKeyAndTheOwnerToolKeepsIt(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	hwid, first, second := strings.Repeat("a", 64), newHostKey(t), newHostKey(t)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key, "host_key": first}, true)
	// A Windows reinstall loses the DPAPI key: the app makes a new pair and activates again.
	_, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key, "host_key": second}, true)
	if payload := certificateOf(t, s, result); payload.HostKey != second {
		t.Fatalf("the new host key was not used: %+v", payload)
	}
	// An activation without a key (an older app) must not wipe the stored one.
	call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key}, true)
	certificate, err := s.RunAdmin("certificate", AdminRequest{Key: license.Key})
	if err != nil {
		t.Fatal(err)
	}
	if payload := certificateOf(t, s, map[string]any{"certificate": certificate}); payload.HostKey != second || payload.HWID != hwid {
		t.Fatalf("the owner tool's certificate lost the host key: %+v", payload)
	}
}

func TestAnotherComputerCannotTakeOverAKeyByOfferingAHostKey(t *testing.T) {
	s := newTestServer(t)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	call(t, s, "/api/v1/activate", map[string]any{"hwid": a, "license_key": license.Key, "host_key": newHostKey(t)}, true)
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": b, "license_key": license.Key,
		"host_key": newHostKey(t)}, true)
	if code != http.StatusConflict || result["code"] != "bound_elsewhere" {
		t.Fatalf("a second computer answered %d %v", code, result)
	}
}

func TestTrialKeysAndTrialCertificatesNeverCarryAHostKey(t *testing.T) {
	s := newTestServer(t)
	trialKey, _, _ := s.IssueKey("friend@example.com", "trial", 3, "")
	code, result := call(t, s, "/api/v1/activate", map[string]any{"hwid": strings.Repeat("c", 64),
		"license_key": trialKey.Key, "host_key": newHostKey(t)}, true)
	if code != http.StatusOK {
		t.Fatalf("activate answered %d %v", code, result)
	}
	if payload := certificateOf(t, s, result); payload.HostKey != "" || payload.Type != "trial" {
		t.Fatalf("a trial certificate carries a host key: %+v", payload)
	}
	code, result = call(t, s, "/api/v1/trial", map[string]any{"hwid": strings.Repeat("d", 64), "host_key": newHostKey(t)}, true)
	if code != http.StatusOK {
		t.Fatalf("trial answered %d %v", code, result)
	}
	if payload := certificateOf(t, s, result); payload.HostKey != "" {
		t.Fatalf("the free trial carries a host key: %+v", payload)
	}
}

func TestTheOfflineActivationPageTakesTheActivationCodeAndKeepsTheMachineIDWorking(t *testing.T) {
	s := newTestServer(t)
	text := pageText(t, s, "/activate")
	for _, want := range []string{"Activation code", "Copy activation code", "A plain 64-character Machine ID also works",
		`body.host_key = code.slice(dot + 1)`, `code.indexOf(".")`} {
		if !strings.Contains(text, want) {
			t.Errorf("the activation page misses %q", want)
		}
	}
	if strings.Contains(text, "Copy Machine ID") {
		t.Error("the page still tells people to copy the Machine ID")
	}
}
