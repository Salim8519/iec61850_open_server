package tpm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/google/go-attestation/attest"
)

// system is the real security chip, reached through the Windows Platform Crypto Provider
// of github.com/google/go-attestation. It needs no administrator rights. Every call opens
// the chip, does one job and closes it again; calls never overlap.
type system struct{}

var systemMutex sync.Mutex

// System returns the real chip of this computer. Nothing touches the hardware until a
// method is called.
func System() Chip { return system{} }

func openChip() (*attest.TPM, error) {
	opened, err := attest.OpenTPM(&attest.OpenConfig{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoChip, err)
	}
	return opened, nil
}

// Verify is the software check; it needs no chip.
func (system) Verify(akPublic, nonce, proof []byte) error { return VerifyProof(akPublic, nonce, proof) }

// Forget deletes the key a stored blob names.
func (system) Forget(blob []byte) {
	if name, _, err := unwrapBlob(blob); err == nil {
		deleteKey(name)
	}
}

type enrollment struct {
	chip   *attest.TPM
	key    *attest.AK
	name   string
	cert   []byte
	public []byte
	closed bool
}

// pickEK chooses the endorsement key to enroll with: an RSA key with the maker's
// certificate first (what the server and the Windows credential activation support).
func pickEK(keys []attest.EK) (certificate, public []byte, err error) {
	var best *attest.EK
	rank := -1
	for index := range keys {
		score := 0
		if keys[index].Certificate != nil {
			score += 2
		}
		if _, isRSA := keys[index].Public.(*rsa.PublicKey); isRSA {
			score++
		}
		if score > rank {
			best, rank = &keys[index], score
		}
	}
	if best == nil {
		return nil, nil, errors.New("the chip reports no endorsement key")
	}
	public, err = x509.MarshalPKIXPublicKey(best.Public)
	if err != nil {
		return nil, nil, fmt.Errorf("the endorsement key is unreadable: %w", err)
	}
	if best.Certificate != nil {
		certificate = best.Certificate.Raw
	}
	return certificate, public, nil
}

// Enroll reads the endorsement key and creates a new attestation key. The caller must
// Close (keep the key) or Discard (delete it) the enrollment.
func (system) Enroll() (Enrollment, error) {
	systemMutex.Lock()
	defer systemMutex.Unlock()
	chip, err := openChip()
	if err != nil {
		return nil, err
	}
	keys, err := chip.EKs()
	if err != nil {
		chip.Close()
		return nil, fmt.Errorf("reading the endorsement key: %w", err)
	}
	certificate, public, err := pickEK(keys)
	if err != nil {
		chip.Close()
		return nil, err
	}
	if certificate == nil {
		certificate = registryEKCertificate(public)
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		chip.Close()
		return nil, err
	}
	name := "ScadaBench-" + hex.EncodeToString(random)
	key, err := chip.NewAK(&attest.AKConfig{Name: name})
	if err != nil {
		chip.Close()
		return nil, fmt.Errorf("creating the attestation key: %w", err)
	}
	return &enrollment{chip: chip, key: key, name: name, cert: certificate, public: public}, nil
}

func (e *enrollment) EKCert() []byte   { return e.cert }
func (e *enrollment) EKPublic() []byte { return e.public }

func (e *enrollment) AK() AKParams {
	parameters := e.key.AttestationParameters()
	return AKParams{Public: parameters.Public, CreateData: parameters.CreateData,
		CreateAttestation: parameters.CreateAttestation, CreateSignature: parameters.CreateSignature}
}

func (e *enrollment) ActivateCredential(credential, secret []byte) ([]byte, error) {
	systemMutex.Lock()
	defer systemMutex.Unlock()
	return e.key.ActivateCredential(e.chip, attest.EncryptedCredential{Credential: credential, Secret: secret})
}

func (e *enrollment) Blob() ([]byte, error) {
	inner, err := e.key.Marshal()
	if err != nil {
		return nil, err
	}
	return wrapBlob(e.name, inner), nil
}

func (e *enrollment) release() {
	systemMutex.Lock()
	defer systemMutex.Unlock()
	if e.closed {
		return
	}
	e.closed = true
	_ = e.key.Close(e.chip)
	_ = e.chip.Close()
}

func (e *enrollment) Close() { e.release() }

func (e *enrollment) Discard() {
	e.release()
	deleteKey(e.name)
}

// quoteBank picks the PCR bank the quote covers: SHA-256 when the chip has it.
func quoteBank(chip *attest.TPM) attest.HashAlg {
	banks, err := chip.PCRBanks()
	if err != nil || len(banks) == 0 {
		return attest.HashSHA1
	}
	for _, bank := range banks {
		if bank == attest.HashSHA256 {
			return bank
		}
	}
	return banks[0]
}

// Prove signs nonce with the stored attestation key (a quote over PCR 0; the PCR values
// themselves do not matter, only that the key signed this nonce).
func (system) Prove(blob, nonce []byte) ([]byte, error) {
	_, inner, err := unwrapBlob(blob)
	if err != nil {
		return nil, err
	}
	systemMutex.Lock()
	defer systemMutex.Unlock()
	chip, err := openChip()
	if err != nil {
		return nil, err
	}
	defer chip.Close()
	key, err := chip.LoadAK(inner)
	if err != nil {
		return nil, fmt.Errorf("loading the attestation key: %w", err)
	}
	defer key.Close(chip)
	quote, err := key.QuotePCRs(chip, nonce, quoteBank(chip), []int{0})
	if err != nil {
		return nil, fmt.Errorf("asking the chip to sign: %w", err)
	}
	return packProof(quote.Quote, quote.Signature), nil
}
