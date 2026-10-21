package tpm

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/google/go-attestation/attest"
)

// The tests below run against THIS computer's real TPM 2.0 and are skipped unless
// STUDIO_REAL_TPM=1 (CI and other computers never need a chip). They need no administrator
// rights. They create attestation keys in the Windows Platform Crypto Provider and delete
// them again.
func realChip(t *testing.T) Chip {
	t.Helper()
	if os.Getenv("STUDIO_REAL_TPM") != "1" {
		t.Skip("set STUDIO_REAL_TPM=1 to test this computer's real security chip")
	}
	return System()
}

// TestRealChipProvesPossessionOfItsKey is the part every start depends on: read the EK and
// its maker's certificate, create an AK, store its blob, sign fresh nonces with it through
// the stored blob and check them with software only.
func TestRealChipProvesPossessionOfItsKey(t *testing.T) {
	chip := realChip(t)
	enrolled, err := chip.Enroll()
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	defer enrolled.Discard()
	if len(enrolled.EKCert()) == 0 {
		t.Fatal("this computer's chip has no endorsement key certificate")
	}
	certificate, err := x509.ParseCertificate(enrolled.EKCert())
	if err != nil {
		t.Fatalf("the EK certificate does not parse: %v", err)
	}
	if _, err := x509.ParsePKIXPublicKey(enrolled.EKPublic()); err != nil {
		t.Fatalf("EK public key: %v", err)
	}
	t.Logf("EK certificate: issuer %q, valid until %s", certificate.Issuer.CommonName, certificate.NotAfter.Format("2006-01-02"))
	parameters := enrolled.AK()
	if _, err := attest.ParseAKPublic(parameters.Public); err != nil {
		t.Fatalf("AK public: %v", err)
	}
	blob, err := enrolled.Blob()
	if err != nil || len(blob) == 0 {
		t.Fatalf("blob: %v", err)
	}
	for round := 0; round < 3; round++ {
		nonce := make([]byte, 32)
		rand.Read(nonce)
		started := time.Now()
		proof, err := chip.Prove(blob, nonce)
		if err != nil {
			t.Fatalf("prove %d: %v", round, err)
		}
		t.Logf("proof %d: %d bytes in %v", round, len(proof), time.Since(started).Round(time.Millisecond))
		if err := chip.Verify(parameters.Public, nonce, proof); err != nil {
			t.Fatalf("verify %d: %v", round, err)
		}
		other := make([]byte, 32)
		rand.Read(other)
		if chip.Verify(parameters.Public, other, proof) == nil {
			t.Fatal("a proof of one nonce verified for another")
		}
		proof[len(proof)-1] ^= 0xff
		if chip.Verify(parameters.Public, nonce, proof) == nil {
			t.Fatal("a damaged signature verified")
		}
	}
	// After the key is forgotten the chip can no longer prove with the blob.
	chip.Forget(blob)
	nonce := make([]byte, 32)
	rand.Read(nonce)
	if _, err := chip.Prove(blob, nonce); err == nil {
		t.Fatal("the chip still proved with a deleted key")
	}
}

// TestRealChipActivatesACredential is the server's challenge: only the chip that holds the
// EK and the AK can open it. Windows builds and chips differ here; on this computer's
// build the Platform Crypto Provider answers 0x80280095 (see ADR-0098), in which case the
// license falls back to the online lease. The test reports the outcome and only fails when
// STUDIO_REAL_TPM_ACTIVATE=1 asks for a chip that supports it.
func TestRealChipActivatesACredential(t *testing.T) {
	chip := realChip(t)
	enrolled, err := chip.Enroll()
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	defer enrolled.Discard()
	public, err := x509.ParsePKIXPublicKey(enrolled.EKPublic())
	if err != nil {
		t.Fatal(err)
	}
	parameters := enrolled.AK()
	activation := attest.ActivationParameters{EK: public, AK: attest.AttestationParameters{
		Public: parameters.Public, CreateData: parameters.CreateData,
		CreateAttestation: parameters.CreateAttestation, CreateSignature: parameters.CreateSignature}}
	secret, challenge, err := activation.Generate()
	if err != nil {
		t.Fatalf("the server side rejects this AK: %v", err)
	}
	answer, err := enrolled.ActivateCredential(challenge.Credential, challenge.Secret)
	if err != nil {
		if os.Getenv("STUDIO_REAL_TPM_ACTIVATE") == "1" {
			t.Fatalf("activate credential: %v", err)
		}
		t.Logf("KNOWN LIMIT on this computer: activate credential failed (%v); the license would use the lease", err)
		return
	}
	if !bytes.Equal(answer, secret) {
		t.Fatal("the chip answered the credential challenge with a different secret")
	}
	t.Log("activate credential works on this computer: the license would be tied to the chip")
}
