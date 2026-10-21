package licensing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/hostkey"
)

var vmRequest = hostkey.Request{Nonce: strings.Repeat("12", 16), VMHWID: strings.Repeat("bb", 32)}

func refusal(t *testing.T, host *hkHost) string {
	t.Helper()
	_, err := host.responder().Answer(vmRequest)
	if err == nil {
		t.Fatal("the host answered although it must refuse")
	}
	return err.Error()
}

func TestALicensedHostAnswersWithItsCertificateAndAProofForThatVM(t *testing.T) {
	signer := newHKSigner(t)
	host := newHKHost(t, signer)
	bundle := host.install(nil)
	answer, err := host.responder().Answer(vmRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !hostkey.Verify(mustKey(t, host.publicKey()), vmRequest, answer.Signature) {
		t.Fatal("the proof does not verify with the host key")
	}
	parsed, err := ParseBundle(string(answer.Certificate))
	if err != nil || parsed != bundle {
		t.Fatalf("the answer carries another certificate: %v", err)
	}
	other := hostkey.Request{Nonce: vmRequest.Nonce, VMHWID: strings.Repeat("cc", 32)}
	if hostkey.Verify(mustKey(t, host.publicKey()), other, answer.Signature) {
		t.Fatal("a proof for one VM verifies for another")
	}
}

func mustKey(t *testing.T, text string) []byte {
	t.Helper()
	key, err := hostkey.DecodeKey(text)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestAHostWhoseMachineIDIsNotTheCertificatesRefusesToServe(t *testing.T) {
	host := newHKHost(t, newHKSigner(t))
	host.install(map[string]any{"hwid": strings.Repeat("9", 64)}) // a license file copied from another PC
	if text := refusal(t, host); text != WrongMachine {
		t.Fatalf("refusal %q", text)
	}
}

func TestACopiedDPAPIBlobCannotServe(t *testing.T) {
	signer := newHKSigner(t)
	original := newHKHost(t, signer)
	original.install(nil)
	// Everything is copied to another PC (or Windows user): the certificate, the blob, the files.
	thief := newHKHost(t, signer)
	thief.protector = hostkey.FakeProtector{Owner: "bob@OTHER"}
	for _, name := range []string{"license.cert", hostkey.KeyFile} {
		data, _ := os.ReadFile(filepath.Join(original.home, name))
		os.WriteFile(filepath.Join(thief.home, name), data, 0o666)
	}
	if text := refusal(t, thief); text != HostKeyUnusableText {
		t.Fatalf("refusal %q", text)
	}
	if _, err := original.responder().Answer(vmRequest); err != nil {
		t.Fatalf("the genuine host stopped working: %v", err)
	}
}

func TestACancelledLicenseStopsServingEvenWithTheCertificateStillThere(t *testing.T) {
	const id = "SUBS-HOST-HOST-HOST-HOST"
	signer := newHKSigner(t)
	healthy := newHKHost(t, signer)
	healthy.install(nil)
	if _, err := healthy.responder().Answer(vmRequest); err != nil {
		t.Fatal(err)
	}
	places := map[string]func(*hkHost){
		"the registry copy": func(h *hkHost) { h.registry.Set("RevokedLicenses", id) },
		"the dot-file":      func(h *hkHost) { os.WriteFile(filepath.Join(h.home, ".revoked_ids"), []byte(id), 0o666) },
		"the state file": func(h *hkHost) {
			os.WriteFile(filepath.Join(h.home, "license_state.json"), []byte(`{"revoked_ids":"`+id+`"}`), 0o666)
		},
	}
	for name, cancel := range places {
		host := newHKHost(t, signer)
		host.install(nil)
		cancel(host) // one copy of the cancelled ID is enough
		if text := refusal(t, host); text != RevokedText {
			t.Errorf("%s: refusal %q", name, text)
		}
	}
}

func TestOnlyALifetimeLicenseWithAHostKeyCanBeShared(t *testing.T) {
	signer := newHKSigner(t)
	trial := newHKHost(t, signer)
	trial.install(map[string]any{"type": "trial", "expires_at": 1_760_000_000 + 86400})
	if text := refusal(t, trial); text != HostNotLifetimeText {
		t.Fatalf("a trial: %q", text)
	}
	old := newHKHost(t, signer)
	old.install(map[string]any{"host_key": nil}) // activated before this feature existed
	if text := refusal(t, old); text != HostNoKeyInCertText {
		t.Fatalf("no host key in the certificate: %q", text)
	}
	garbled := newHKHost(t, signer)
	garbled.install(map[string]any{"host_key": strings.Repeat("A", 42)})
	if text := refusal(t, garbled); text != HostNoKeyInCertText {
		t.Fatalf("a host key of the wrong size: %q", text)
	}
	mismatch := newHKHost(t, signer)
	other := newHKHost(t, signer)
	mismatch.install(map[string]any{"host_key": other.publicKey()}) // the certificate names someone else's key
	if text := refusal(t, mismatch); text != HostKeyMismatchText {
		t.Fatalf("a key that is not the certificate's: %q", text)
	}
	nokey := newHKHost(t, signer)
	nokey.install(map[string]any{"host_key": mismatch.publicKey()})
	if text := refusal(t, nokey); text != HostKeyMismatchText {
		t.Fatalf("a PC whose key differs: %q", text)
	}
}

func TestACertificateNotSignedByTheReleaseKeysOrChangedIsRefused(t *testing.T) {
	host := newHKHost(t, newHKSigner(t))
	host.install(nil)
	foreign := NewHostResponder(HostOptions{Home: host.home, Keys: newHKSigner(t).keys(),
		HWID: func() (string, error) { return host.hwid, nil }, Protector: host.protector, Registry: host.registry})
	if _, err := foreign.Answer(vmRequest); err == nil || err.Error() != CertChanged {
		t.Fatalf("a certificate signed by a foreign key: %v", err)
	}
	path := filepath.Join(host.home, "license.cert")
	data, _ := os.ReadFile(path)
	os.WriteFile(path, []byte(strings.Replace(string(data), `\"lifetime\"`, `\"lifetimf\"`, 1)), 0o666)
	if text := refusal(t, host); text != CertChanged {
		t.Fatalf("a changed certificate: %q", text)
	}
	os.Remove(path)
	if text := refusal(t, host); text != HostNoCertificateText {
		t.Fatalf("no certificate: %q", text)
	}
}

func TestAClockBeforeTheCertificateWasIssuedRefusesToServe(t *testing.T) {
	host := newHKHost(t, newHKSigner(t))
	host.install(map[string]any{"issued_at": 1_760_000_000 + 10*86400})
	if text := refusal(t, host); text != ClockRollback {
		t.Fatalf("refusal %q", text)
	}
}

func TestAPCThatCannotReadItsMachineIDRefusesToServe(t *testing.T) {
	host := newHKHost(t, newHKSigner(t))
	host.install(nil)
	responder := NewHostResponder(HostOptions{Home: host.home, Keys: host.signer.keys(), Protector: host.protector,
		Registry: host.registry, HWID: func() (string, error) { return "", os.ErrPermission }})
	if _, err := responder.Check(); err == nil {
		t.Fatal("a host without a machine ID served")
	}
}
