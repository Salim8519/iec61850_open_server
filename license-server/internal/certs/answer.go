package certs

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// AnswerPurpose marks a signed verify answer. Certificates and time stamps carry other
// purposes (or none), so none of the three can ever stand in for another.
const AnswerPurpose = "verify-answer"

// Verify statuses a signed answer can carry.
const (
	StatusValid   = "valid"
	StatusRevoked = "revoked"
	StatusMoved   = "moved"
	StatusUnknown = "unknown"
)

// Answer is the signed content of one POST /api/v1/verify reply. The struct field order
// below is the exact key order of the signed JSON string (it is also alphabetical), so
// every client can verify the raw string it receives. Nonce echoes the client's request
// nonce (empty when the client sent none) and binds the answer to that one request.
type Answer struct {
	HWID      string `json:"hwid"`
	KeyID     string `json:"kid"`
	LicenseID string `json:"license_id"`
	Nonce     string `json:"nonce"`
	Purpose   string `json:"purpose"`
	Status    string `json:"status"`
	Time      int64  `json:"time"`
	Version   int    `json:"version"`
}

// ValidNonce accepts exactly 32 lowercase hex characters.
func ValidNonce(nonce string) bool {
	if len(nonce) != 32 {
		return false
	}
	for _, c := range nonce {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// SignAnswer stamps key id, purpose and version, serialises the answer and signs the
// exact bytes with the certificate signing key, like certificates and server time.
func (s *Signer) SignAnswer(a Answer) (Bundle, error) {
	a.KeyID, a.Purpose, a.Version = s.KeyID, AnswerPurpose, 1
	raw, err := json.Marshal(a)
	if err != nil {
		return Bundle{}, err
	}
	signature := ed25519.Sign(s.private, raw)
	return Bundle{Payload: string(raw), Signature: base64.RawURLEncoding.EncodeToString(signature)}, nil
}

// VerifyAnswer checks a signed verify answer (tests and tooling).
func VerifyAnswer(public ed25519.PublicKey, b Bundle) (Answer, error) {
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(b.Signature, "="))
	if err != nil || !ed25519.Verify(public, []byte(b.Payload), signature) {
		return Answer{}, errors.New("signature does not match")
	}
	var answer Answer
	if err := json.Unmarshal([]byte(b.Payload), &answer); err != nil || answer.Purpose != AnswerPurpose {
		return Answer{}, errors.New("not a verify answer")
	}
	return answer, nil
}
