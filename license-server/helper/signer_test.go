package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testSigner(t *testing.T) *Signer {
	t.Helper()
	secrets, _, err := newOwnerKey()
	if err != nil {
		t.Fatal(err)
	}
	signer, err := newSigner(secrets)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func TestHelperCertificateMatchesServerFormat(t *testing.T) {
	signer, hwid, now := testSigner(t), strings.Repeat("ab", 32), time.Unix(1_800_000_000, 0)
	bundle, payload, err := signer.licenseFor(hwid, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	prefix := `{"email":"owner@scadabench.com","expires_at":0,"hwid":"` + hwid + `","issued_at":1799996400,"kid":"` + signer.keyID + `","license_id":"OWNER-`
	if !strings.HasPrefix(bundle.Payload, prefix) || !strings.HasSuffix(bundle.Payload, `"max_machines":1,"type":"lifetime","version":1}`) {
		t.Fatalf("payload is not in the server's canonical form: %s", bundle.Payload)
	}
	if payload.Type != "lifetime" || strings.ContainsAny(bundle.Signature, "=+/") {
		t.Fatalf("unexpected payload %+v or signature encoding %q", payload, bundle.Signature)
	}
	_, trial, err := signer.licenseFor(hwid, 7, now)
	if err != nil || trial.Type != "trial" || trial.ExpiresAt != now.Unix()+7*86400 {
		t.Fatalf("trial: %+v, %v", trial, err)
	}
	for _, bad := range []string{"", "ABC", strings.Repeat("AB", 32), strings.Repeat("ab", 31)} {
		if _, _, err := signer.licenseFor(bad, 0, now); err == nil {
			t.Errorf("Machine ID %q was accepted", bad)
		}
	}
}

// The Studio backend (Python) and, when built, the native guard must accept the
// helper's certificate for its machine only.
func TestHelperCertificateIsAcceptedByStudio(t *testing.T) {
	root, _ := filepath.Abs("../..")
	python := filepath.Join(root, "studio", ".venv", "Scripts", "python.exe")
	if _, err := os.Stat(python); err != nil {
		t.Skip("Studio venv not available")
	}
	signer, hwid, now := testSigner(t), strings.Repeat("cd", 32), time.Now()
	bundle, _, err := signer.licenseFor(hwid, 3, now)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "license.cert")
	text, _ := json.Marshal(bundle)
	if err := os.WriteFile(file, text, 0o600); err != nil {
		t.Fatal(err)
	}
	check := `import sys, json, time; sys.path.insert(0, sys.argv[1])
from studio_app.licensing import certificate
from studio_app.domain import license_policy
key = bytes.fromhex(sys.argv[3]); keys = {sys.argv[4]: key}
payload = certificate.verify_bundle(certificate.parse_bundle(open(sys.argv[2]).read()), keys)
ok = license_policy.evaluate_certificate(payload, sys.argv[5], int(time.time()), 0, False)
other = license_policy.evaluate_certificate(payload, "0" * 64, int(time.time()), 0, False)
print(ok.licensed, ok.plan, ok.days_remaining, other.licensed)`
	out, err := exec.Command(python, "-c", check, filepath.Join(root, "studio", "backend"), file,
		signer.publicHex(), signer.keyID, hwid).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "True trial 3 False" {
		t.Fatalf("Studio verdict %q (%v)", out, err)
	}
	native := filepath.Join(root, "studio", "native", "build", "Release", "studio_license_check.exe")
	if _, err := os.Stat(native); err != nil {
		t.Log("native license check tool not built; Python check only")
		return
	}
	stamp := []string{"verify", file, signer.publicHex()}
	if out, err := exec.Command(native, append(stamp, hwid, strconv.FormatInt(now.Unix(), 10))...).CombinedOutput(); err != nil {
		t.Fatalf("native guard refused the helper certificate: %s", out)
	}
	if _, err := exec.Command(native, append(stamp, strings.Repeat("0", 64), strconv.FormatInt(now.Unix(), 10))...).CombinedOutput(); err == nil {
		t.Fatal("native guard accepted the certificate on another machine")
	}
}

// "Works offline" is a property of the code, not a promise: no networking package may
// be linked into the helper at all.
func TestHelperHasNoNetworkCode(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Skip("go list unavailable:", err)
	}
	for _, dependency := range strings.Fields(string(out)) {
		if dependency == "net" || strings.HasPrefix(dependency, "net/") || dependency == "crypto/tls" ||
			strings.HasPrefix(dependency, "golang.org/x/crypto/ssh") {
			t.Errorf("the helper links networking code: %s", dependency)
		}
	}
}
