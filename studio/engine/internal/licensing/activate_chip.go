package licensing

import (
	"encoding/base64"
	"errors"
	"os"

	"studio-engine/internal/licensing/tpm"
	"studio-engine/internal/pyjson"
)

// Activation with the computer's security chip (ADR-0098). The app offers the chip's
// endorsement key, its maker's certificate and a new attestation key; the server checks the
// certificate against the TPM makers' root certificates, sends a credential challenge only
// the chip that holds both keys can open, and, once the app answers it, binds the license to
// that chip. Anything that goes wrong on the way is NOT an error for the user: the app
// activates again without the chip and gets a lease license.

// chipEnrollment is a new attestation key waiting to be bound to a license.
type chipEnrollment struct {
	enrolled tpm.Enrollment
	stored   bool // its blob is now ak.blob
}

// enrollChip creates the attestation key; nil when this computer has no chip the maker
// vouches for (no certificate, no TPM 2.0, a virtual TPM, any error).
func (s *Service) enrollChip() *chipEnrollment {
	enrolled, err := s.options.Chip.Enroll()
	if err != nil {
		s.options.Logf("No security chip is used for the license: %v", err)
		return nil
	}
	if len(enrolled.EKCert()) == 0 {
		s.options.Logf("The security chip has no certificate from its maker; the license will be a lease.")
		enrolled.Discard()
		return nil
	}
	return &chipEnrollment{enrolled: enrolled}
}

func b64(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

// offer is the "tpm" part of the activation request.
func (c *chipEnrollment) offer() *pyjson.Object {
	parameters := c.enrolled.AK()
	key := pyjson.NewObject().Set("public", b64(parameters.Public)).Set("create_data", b64(parameters.CreateData)).
		Set("create_attestation", b64(parameters.CreateAttestation)).Set("create_signature", b64(parameters.CreateSignature))
	return pyjson.NewObject().Set("ek_cert", b64(c.enrolled.EKCert())).Set("ek_public", b64(c.enrolled.EKPublic())).
		Set("ak", key)
}

// chipRefusal reports server answers that end the activation instead of falling back to a
// lease: the key is unknown or cancelled, bound to another computer, or the server cannot
// be reached at all.
func chipRefusal(err error) bool {
	var server *ServerError
	if !errors.As(err, &server) {
		return true
	}
	code, _ := server.Code.(string)
	switch code {
	case "offline", "no_server", "unknown_key", "revoked", "bound_elsewhere", "pow_invalid", "rate_limited":
		return true
	}
	return false
}

// activateOnline asks the server for this computer's certificate. With a chip the server
// can vouch for, the certificate is tied to the chip (the returned enrollment must then be
// kept, or Discarded when the install fails); otherwise it is a lease license. The returned
// enrollment is nil for a lease.
func (s *Service) activateOnline(client Client, key, hwid string) (any, *chipEnrollment, error) {
	binder, canBind := client.(BindingClient)
	plain := func() (any, *chipEnrollment, error) {
		answer, err := client.Activate(key, hwid)
		return answer, nil, err
	}
	if !canBind {
		return plain()
	}
	chip := s.enrollChip()
	if chip == nil {
		return plain()
	}
	answer, err := binder.ActivateWithChip(key, hwid, chip.offer())
	if err != nil {
		chip.enrolled.Discard()
		if chipRefusal(err) {
			return nil, nil, err
		}
		return plain()
	}
	challenge, found := challengeOf(answer)
	if !found { // the server could not use the chip and sent a plain certificate
		chip.enrolled.Discard()
		return answer, nil, nil
	}
	secret, err := chip.enrolled.ActivateCredential(challenge.Credential, challenge.Secret)
	if err != nil {
		s.options.Logf("The security chip could not open the server's challenge: %v", err)
		chip.enrolled.Discard()
		return plain()
	}
	answer, err = binder.ActivateChip(challenge.Session, b64(secret))
	if err != nil {
		chip.enrolled.Discard()
		if chipRefusal(err) {
			return nil, nil, err
		}
		return plain()
	}
	return answer, chip, nil
}

// storeChipKey keeps the new attestation key's blob (ak.blob) and returns the previous one,
// which the caller forgets once the new license is in place.
func (s *Service) storeChipKey(chip *chipEnrollment) (previous []byte, err error) {
	blob, err := chip.enrolled.Blob()
	if err != nil {
		return nil, err
	}
	previous, _ = os.ReadFile(s.akFile)
	if err := writeBinary(s.akFile, blob); err != nil {
		return nil, err
	}
	return previous, nil
}

// restoreChipKey puts the previous key blob back when storing the new certificate failed.
func (s *Service) restoreChipKey(previous []byte) {
	if len(previous) > 0 {
		_ = writeBinary(s.akFile, previous)
	} else {
		_ = os.Remove(s.akFile)
	}
}
