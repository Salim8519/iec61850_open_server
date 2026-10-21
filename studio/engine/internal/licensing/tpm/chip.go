// Package tpm is the small seam between the license service and the computer's security
// chip (TPM 2.0, ADR-0098). The service only knows the Chip interface: the real chip talks
// to Windows through github.com/google/go-attestation, tests use the software Fake.
//
// What the chip is used for:
//
//   - Enroll: read the chip's endorsement key (EK) and its maker's certificate, and create
//     a fresh attestation key (AK) that never leaves the chip. The license server checks
//     the certificate against the TPM makers' root certificates and proves with a
//     credential challenge that the AK lives in the chip that owns the EK.
//   - Prove: at every start the app asks the chip to sign a fresh random nonce with the AK
//     (a TPM quote) and checks the signature offline with the AK public key written into
//     the signed license certificate. A copy of the license on another computer cannot
//     answer, so the license cannot be cloned.
//
// A chip that cannot be proven (no certificate, virtual TPM, TPM 1.2, any error) is not an
// error for the user: the license then uses the online lease instead (ADR-0098).
package tpm

import "errors"

// ErrNoChip means this computer has no usable TPM 2.0 (or it was turned off).
var ErrNoChip = errors.New("this computer has no usable security chip (TPM 2.0)")

// AKParams are the public facts about a new attestation key that the server needs to run
// its credential challenge (go-attestation AttestationParameters, TPM 2.0 only).
type AKParams struct {
	Public            []byte // TPMT_PUBLIC; the license certificate repeats it as "ak"
	CreateData        []byte
	CreateAttestation []byte
	CreateSignature   []byte
}

// Enrollment is one activation in progress: a new attestation key plus the chip's EK.
// Close releases the key; the blob from Blob stays valid for Prove afterwards.
type Enrollment interface {
	// EKCert is the maker's DER certificate of the endorsement key, nil when there is none.
	EKCert() []byte
	// EKPublic is the endorsement public key as PKIX DER.
	EKPublic() []byte
	// AK is the new attestation key's public parameters.
	AK() AKParams
	// ActivateCredential decrypts the server's challenge; only the chip that holds both
	// the EK and the AK can.
	ActivateCredential(credential, secret []byte) ([]byte, error)
	// Blob is the opaque text Prove needs to find the key again (kept as ak.blob).
	Blob() ([]byte, error)
	// Close lets go of the chip; the key stays stored for Prove.
	Close()
	// Discard lets go of the chip and deletes the new key (the activation did not finish).
	Discard()
}

// Chip is the security chip as the license service uses it.
type Chip interface {
	// Enroll creates a fresh attestation key and reads the endorsement key.
	Enroll() (Enrollment, error)
	// Prove loads the stored attestation key and signs nonce with it.
	Prove(blob, nonce []byte) ([]byte, error)
	// Forget deletes the stored attestation key a blob names (best effort, no error).
	Forget(blob []byte)
	// Verify checks, with software only, that proof is a signature of nonce made by the
	// attestation key whose public part is akPublic.
	Verify(akPublic, nonce, proof []byte) error
}

// None is the chip of a computer that has no security chip or must not touch one (tests,
// the parity build): enrolling always fails, so the license falls back to the lease.
type None struct{}

// Enroll always fails with ErrNoChip.
func (None) Enroll() (Enrollment, error) { return nil, ErrNoChip }

// Forget does nothing: there is no key.
func (None) Forget([]byte) {}

// Prove always fails with ErrNoChip.
func (None) Prove([]byte, []byte) ([]byte, error) { return nil, ErrNoChip }

// Verify is the software check; it needs no chip.
func (None) Verify(akPublic, nonce, proof []byte) error { return VerifyProof(akPublic, nonce, proof) }
