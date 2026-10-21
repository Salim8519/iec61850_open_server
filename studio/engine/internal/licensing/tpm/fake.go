package tpm

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

// Fake is a software chip for tests. Each Fake is one "computer": it only proves with keys
// it created itself, so a second Fake stands for a cloned license on another PC. Its
// attestation keys are Ed25519 keys, its "credential activation" returns the credential
// unchanged (the test server expects the same bytes back), and its proofs are plain
// signatures, so nothing here depends on hardware.
type Fake struct {
	// EKCertDER and EKPublicDER are what Enroll reports; EKCertDER may be nil (no certificate).
	EKCertDER, EKPublicDER []byte
	// EnrollError and ActivateError make the matching step fail (nil: works).
	EnrollError, ActivateError error
	// ProveError makes Prove fail, as a chip that is switched off would.
	ProveError error

	mu      sync.Mutex
	keys    map[string]ed25519.PrivateKey
	Enrolls int
	Proofs  int
	Gone    []string // names of keys deleted by Discard or Forget
}

// NewFake returns a working fake chip with an endorsement key certificate.
func NewFake() *Fake {
	identity := make([]byte, 8)
	rand.Read(identity) // every fake computer has its own endorsement key
	return &Fake{EKCertDER: []byte("fake-ek-certificate"), EKPublicDER: append([]byte("fake-ek-public-key-"), identity...),
		keys: map[string]ed25519.PrivateKey{}}
}

type fakeEnrollment struct {
	chip   *Fake
	name   string
	public ed25519.PublicKey
}

// Enroll creates a new fake attestation key.
func (f *Fake) Enroll() (Enrollment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Enrolls++
	if f.EnrollError != nil {
		return nil, f.EnrollError
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	random := make([]byte, 6)
	rand.Read(random)
	name := "fake-" + hex.EncodeToString(random)
	f.keys[name] = private
	return &fakeEnrollment{chip: f, name: name, public: public}, nil
}

func (e *fakeEnrollment) EKCert() []byte   { return e.chip.EKCertDER }
func (e *fakeEnrollment) EKPublic() []byte { return e.chip.EKPublicDER }

func (e *fakeEnrollment) AK() AKParams {
	return AKParams{Public: e.public, CreateData: []byte("create-data"),
		CreateAttestation: []byte("create-attestation"), CreateSignature: []byte("create-signature")}
}

func (e *fakeEnrollment) ActivateCredential(credential, _ []byte) ([]byte, error) {
	if e.chip.ActivateError != nil {
		return nil, e.chip.ActivateError
	}
	return credential, nil
}

func (e *fakeEnrollment) Blob() ([]byte, error) { return []byte(e.name), nil }
func (e *fakeEnrollment) Close()                {}

func (e *fakeEnrollment) Discard() {
	e.chip.mu.Lock()
	defer e.chip.mu.Unlock()
	delete(e.chip.keys, e.name)
	e.chip.Gone = append(e.chip.Gone, e.name)
}

// Prove signs nonce with the key the blob names, if this fake computer created it.
func (f *Fake) Prove(blob, nonce []byte) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Proofs++
	if f.ProveError != nil {
		return nil, f.ProveError
	}
	private, found := f.keys[string(blob)]
	if !found {
		return nil, errors.New("this chip does not hold that attestation key")
	}
	return ed25519.Sign(private, nonce), nil
}

// Forget deletes the key a blob names.
func (f *Fake) Forget(blob []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.keys, string(blob))
	f.Gone = append(f.Gone, string(blob))
}

// Verify checks a fake proof with the fake attestation public key.
func (f *Fake) Verify(akPublic, nonce, proof []byte) error {
	if len(akPublic) != ed25519.PublicKeySize || !ed25519.Verify(akPublic, nonce, proof) {
		return errors.New("the proof does not match the attestation key")
	}
	return nil
}

// Keys is how many attestation keys this fake computer holds.
func (f *Fake) Keys() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.keys)
}
