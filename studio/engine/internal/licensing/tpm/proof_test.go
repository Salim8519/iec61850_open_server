package tpm

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/google/go-tpm/legacy/tpm2"
	"github.com/google/go-tpm/tpmutil"
)

// softQuote builds what a TPM would sign: a TPMS_ATTEST quote carrying nonce, signed with
// RSASSA SHA-256 by key; the AK's public area is a TPMT_PUBLIC with a signing scheme.
func softQuote(t *testing.T, key *rsa.PrivateKey, nonce []byte, attestType tpmutil.Tag) (public, proof []byte) {
	t.Helper()
	area := tpm2.Public{Type: tpm2.AlgRSA, NameAlg: tpm2.AlgSHA256,
		Attributes: tpm2.FlagSign | tpm2.FlagRestricted | tpm2.FlagFixedTPM | tpm2.FlagFixedParent |
			tpm2.FlagSensitiveDataOrigin | tpm2.FlagUserWithAuth,
		RSAParameters: &tpm2.RSAParams{Sign: &tpm2.SigScheme{Alg: tpm2.AlgRSASSA, Hash: tpm2.AlgSHA256},
			KeyBits: 2048, ModulusRaw: key.N.Bytes()}}
	public, err := area.Encode()
	if err != nil {
		t.Fatal(err)
	}
	name := tpm2.Name{Digest: &tpm2.HashValue{Alg: tpm2.AlgSHA256, Value: make([]byte, 32)}}
	attested := tpm2.AttestationData{Magic: 0xff544347, Type: attestType, QualifiedSigner: name, ExtraData: nonce}
	if attestType == tpm2.TagAttestQuote {
		attested.AttestedQuoteInfo = &tpm2.QuoteInfo{
			PCRSelection: tpm2.PCRSelection{Hash: tpm2.AlgSHA256, PCRs: []int{0}}, PCRDigest: make([]byte, 32)}
	} else {
		attested.AttestedCertifyInfo = &tpm2.CertifyInfo{Name: name, QualifiedName: name}
	}
	quote, err := attested.Encode()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(quote)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tpm2.Signature{Alg: tpm2.AlgRSASSA, RSA: &tpm2.SignatureRSA{HashAlg: tpm2.AlgSHA256,
		Signature: signature}}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return public, packProof(quote, raw)
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestProofOfTheRightKeyAndNonceVerifies(t *testing.T) {
	key, nonce := newKey(t), make([]byte, 32)
	rand.Read(nonce)
	public, proof := softQuote(t, key, nonce, tpm2.TagAttestQuote)
	if err := VerifyProof(public, nonce, proof); err != nil {
		t.Fatalf("a genuine proof was refused: %v", err)
	}
}

func TestProofIsRefusedForAnotherNonceKeyOrDamage(t *testing.T) {
	key, nonce := newKey(t), make([]byte, 32)
	rand.Read(nonce)
	public, proof := softQuote(t, key, nonce, tpm2.TagAttestQuote)
	other := append([]byte(nil), nonce...)
	other[0] ^= 1
	if VerifyProof(public, other, proof) == nil {
		t.Error("a proof of one nonce verified for another")
	}
	otherPublic, _ := softQuote(t, newKey(t), nonce, tpm2.TagAttestQuote)
	if VerifyProof(otherPublic, nonce, proof) == nil {
		t.Error("a proof verified under another computer's attestation key")
	}
	damaged := append([]byte(nil), proof...)
	damaged[len(damaged)-1] ^= 0x55
	if VerifyProof(public, nonce, damaged) == nil {
		t.Error("a damaged signature verified")
	}
	flipped := append([]byte(nil), proof...)
	flipped[10] ^= 0x01 // inside the signed quote
	if VerifyProof(public, nonce, flipped) == nil {
		t.Error("a changed quote verified")
	}
	if VerifyProof(public, nil, proof) == nil || VerifyProof(public, nonce, nil) == nil ||
		VerifyProof(nil, nonce, proof) == nil || VerifyProof(public, nonce, proof[:6]) == nil {
		t.Error("an empty or truncated input verified")
	}
}

func TestOnlyAQuoteGeneratedByAChipCountsAsProof(t *testing.T) {
	key, nonce := newKey(t), make([]byte, 32)
	// A structure that is not a quote (a "certify" attestation) never proves possession.
	public, proof := softQuote(t, key, nonce, tpm2.TagAttestCertify)
	if VerifyProof(public, nonce, proof) == nil {
		t.Error("a non-quote attestation verified")
	}
}

func TestProofWireFormatRoundTripsAndRefusesBadLengths(t *testing.T) {
	quote, signature, err := unpackProof(packProof([]byte("quote"), []byte("signature")))
	if err != nil || string(quote) != "quote" || string(signature) != "signature" {
		t.Fatalf("round trip: %q %q %v", quote, signature, err)
	}
	bad := make([]byte, 4)
	binary.BigEndian.PutUint32(bad, 1<<30)
	for _, proof := range [][]byte{nil, {1, 2, 3}, bad, append(append([]byte{}, bad[:3]...), 0), {0, 0, 0, 5, 1, 2}} {
		if _, _, err := unpackProof(proof); err == nil {
			t.Errorf("proof %v was accepted", proof)
		}
	}
}

func TestStoredKeyBlobKeepsItsNameAndRefusesDamage(t *testing.T) {
	name, inner, err := unwrapBlob(wrapBlob("ScadaBench-0123", []byte("library-blob")))
	if err != nil || name != "ScadaBench-0123" || string(inner) != "library-blob" {
		t.Fatalf("unwrap: %q %q %v", name, inner, err)
	}
	for _, blob := range [][]byte{nil, []byte("SBAK1"), []byte("SBAK1\x05ab"), []byte("other-format"), []byte("SBAK1\x00")} {
		if _, _, err := unwrapBlob(blob); err == nil {
			t.Errorf("blob %q was accepted", blob)
		}
	}
}
