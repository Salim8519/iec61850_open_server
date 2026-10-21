package licensing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/hostkey"
)

func TestAVMWithoutALicenseIsLicensedByItsHostAndTheNativeWorkersGetTheirProof(t *testing.T) {
	rig := newVMRig(t)
	status := rig.untilLicensed()
	if !status.Licensed || status.Mode != ModeVMHost || status.Plan != "lifetime" || status.LicenseID != "SUBS-HOST-HOST-HOST-HOST" ||
		status.OnlineCheckRequired || status.Email != nil {
		t.Fatalf("status of the VM: %+v", status)
	}
	data, err := os.ReadFile(filepath.Join(rig.vmHome, VMProofFile))
	if err != nil {
		t.Fatalf("no vm_proof.json for the native workers: %v", err)
	}
	var proof struct {
		Certificate struct{ Payload, Signature string }
		Nonce       string
		VMHWID      string `json:"vm_hwid"`
		Signature   string
		Time        int64
	}
	if err := json.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	request := hostkey.Request{Nonce: proof.Nonce, VMHWID: proof.VMHWID}
	if request.Validate() != nil || proof.VMHWID != rig.vmHWID || proof.Time != int64(*rig.now) {
		t.Fatalf("proof file: %+v", proof)
	}
	if !hostkey.Verify(mustKey(t, rig.host.publicKey()), request, proof.Time, proof.Signature) {
		t.Fatal("the proof file's signature does not verify with the host key")
	}
	if _, err := VerifyBundle(Bundle{proof.Certificate.Payload, proof.Certificate.Signature}, rig.host.signer.keys()); err != nil {
		t.Fatalf("the proof file's certificate does not verify: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rig.vmHome, VMProofFile+".tmp")); err == nil {
		t.Fatal("a temporary proof file was left behind")
	}
}

func TestTheVMStatusNamesTheModeAndMasksBothKeys(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	status, err := rig.vm.Status()
	if err != nil {
		t.Fatal(err)
	}
	get := func(key string) any { value, _ := status.Get(key); return value }
	if get("licensed") != true || get("mode") != ModeVMHost || get("plan") != "lifetime" ||
		get("masked_key") != "SUBS-****-****-****-HOST" {
		t.Fatalf("status %v", status.Keys())
	}
	masked, _ := get("host_key").(string)
	if full := rig.host.publicKey(); masked != hostkey.MaskKey(full) || strings.Contains(masked, full[6:len(full)-4]) {
		t.Fatalf("host key shown as %q", masked)
	}
	if _, has := status.Get("binding"); has {
		t.Fatal("a VM's status names a binding")
	}
}

func TestAVMWithAValidLicenseOfItsOwnNeverAsksAHost(t *testing.T) {
	rig := newVMRig(t)
	own := rig.host.signer.sign(t, map[string]any{"hwid": rig.vmHWID, "license_id": "SUBS-OWN1-OWN1-OWN1-OWN1"})
	data, _ := json.Marshal(map[string]string{"payload": own.Payload, "signature": own.Signature})
	os.WriteFile(filepath.Join(rig.vmHome, "license.cert"), data, 0o666)
	status := rig.status()
	if !status.Licensed || status.Mode != "lifetime" || status.LicenseID != "SUBS-OWN1-OWN1-OWN1-OWN1" {
		t.Fatalf("status %+v", status)
	}
	if rig.dials.Load() != 0 {
		t.Fatalf("a licensed computer asked a host %d times", rig.dials.Load())
	}
	if _, err := os.Stat(filepath.Join(rig.vmHome, VMProofFile)); err == nil {
		t.Fatal("a proof was written for a computer with its own license")
	}
}

func TestWhenTheHostIsNotLicensedTheVMStaysUnlicensedWithItsOwnReason(t *testing.T) {
	rig := newVMRig(t)
	os.Remove(filepath.Join(rig.host.home, "license.cert"))
	err := rig.vm.RefreshVMHost()
	if err == nil || !strings.Contains(err.Error(), HostNoCertificateText) {
		t.Fatalf("the refusal of the host was not passed on in words: %v", err)
	}
	status := rig.status()
	if status.Licensed || status.Reason != NoLicense {
		t.Fatalf("status %+v", status)
	}
	if _, err := os.Stat(filepath.Join(rig.vmHome, VMProofFile)); err == nil {
		t.Fatal("a proof file exists without a host")
	}
}

func TestNoHostAtAllMeansUnlicensedAndNeverAHang(t *testing.T) {
	rig := newVMRig(t)
	rig.stop()
	rig.address = "127.0.0.1:1" // nothing listens
	if err := rig.vm.RefreshVMHost(); err == nil {
		t.Fatal("a missing host confirmed the VM")
	}
	if status := rig.status(); status.Licensed {
		t.Fatalf("status %+v", status)
	}
}

func TestAnAnswerIsAcceptedOnlyForThisNonceAndThisMachine(t *testing.T) {
	rig := newVMRig(t)
	foreign := newHKSigner(t)
	other := newHKHost(t, foreign)
	other.install(nil)
	answers := map[string]hostkey.Answer{
		"a replay of an earlier proof (another nonce)": rig.answerOf(hostkey.Request{Nonce: strings.Repeat("01", 16), VMHWID: rig.vmHWID}),
		"a proof made for another VM":                  rig.answerOf(hostkey.Request{Nonce: strings.Repeat("02", 16), VMHWID: strings.Repeat("dd", 32)}),
		"a certificate signed by a key that is not ours": func() hostkey.Answer {
			answer, _ := other.responder().Answer(hostkey.Request{Nonce: strings.Repeat("03", 16), VMHWID: rig.vmHWID})
			return answer
		}(),
		"a certificate with changed text": func() hostkey.Answer {
			answer := rig.answerOf(hostkey.Request{Nonce: strings.Repeat("04", 16), VMHWID: rig.vmHWID})
			answer.Certificate = json.RawMessage(strings.Replace(string(answer.Certificate), "SUBS-HOST", "SUBS-EVIL", 1))
			return answer
		}(),
		"nothing but a signature": {Signature: "AAAA"},
		"an empty answer":         {},
	}
	for name, answer := range answers {
		rig.dial = canned(answer)
		if err := rig.vm.RefreshVMHost(); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if status := rig.status(); status.Licensed {
		t.Fatalf("the VM was licensed by a bad answer: %+v", status)
	}
}

func TestOnlyAGenuineLifetimeLicenseThatNamesTheSigningHostKeyIsAccepted(t *testing.T) {
	signer := newHKSigner(t)
	rig := newVMRig(t)
	rig.host.signer = signer
	rig.vm = rig.newVM() // trusts this signer too
	key := newAttacker(t)
	otherKey := newAttacker(t)
	cases := map[string]func(){
		"a trial certificate": func() {
			key.serves(rig, signer, map[string]any{"type": "trial", "expires_at": 1_760_000_000 + 86400})
		},
		"a lifetime certificate without a host key": func() { key.serves(rig, signer, map[string]any{"host_key": nil}) },
		"a host key of the wrong size":              func() { key.serves(rig, signer, map[string]any{"host_key": key.public[:40]}) },
		"a host key padded with ==":                 func() { key.serves(rig, signer, map[string]any{"host_key": key.public + "="}) },
		"a signature by another key than the certificate names": func() {
			otherKey.serves(rig, signer, map[string]any{"host_key": key.public})
		},
		"a lifetime license that has expired":       func() { key.serves(rig, signer, map[string]any{"expires_at": 1_700_000_100}) },
		"a certificate issued in the future":        func() { key.serves(rig, signer, map[string]any{"issued_at": 1_760_000_000 + 7*86400}) },
		"a certificate of a version we do not know": func() { key.serves(rig, signer, map[string]any{"version": 2}) },
		"a certificate signed by a foreign key":     func() { key.serves(rig, newHKSigner(t), nil) },
		"a license this VM learned was cancelled": func() {
			key.serves(rig, signer, nil)
			rig.vm.revoked.Add("SUBS-HOST-HOST-HOST-HOST")
		},
	}
	for name, setUp := range cases {
		rig.vm = rig.newVM()
		setUp()
		if err := rig.vm.RefreshVMHost(); err == nil {
			t.Errorf("%s was accepted", name)
		}
		if status := rig.status(); status.Licensed {
			t.Errorf("%s licensed the VM: %+v", name, status)
		}
	}
	rig.vm = rig.newVM() // the same attacker with a genuine, matching certificate is the happy path
	key.serves(rig, signer, nil)
	if err := rig.vm.RefreshVMHost(); err != nil {
		t.Fatalf("a genuine lifetime license with its own host key was refused: %v", err)
	}
}
