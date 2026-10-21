package hostkey

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
)

// Request is the one JSON line a VM sends: a fresh nonce and the VM's own machine ID.
type Request struct {
	Nonce  string `json:"nonce"`
	VMHWID string `json:"vm_hwid"`
}

// Answer is the one JSON line the host returns: its license.cert object, the host's own clock in
// Unix seconds (Time) and the host key's signature over ProofMessage, which covers that exact time.
// Error is set instead (and the rest empty) when the host refuses.
type Answer struct {
	Certificate json.RawMessage `json:"certificate,omitempty"`
	Signature   string          `json:"signature,omitempty"`
	Time        int64           `json:"time,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// ProofMessage is the exact byte string the host key signs: the prefix, the VM's nonce, the VM's
// machine ID and the host's time as decimal Unix seconds, joined by "|".
func ProofMessage(nonce, vmHWID string, at int64) []byte {
	return []byte(ProofPrefix + nonce + "|" + vmHWID + "|" + strconv.FormatInt(at, 10))
}

func lowerHex(text string, length int) bool {
	if len(text) != length {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// ValidNonce reports whether text is 32 lowercase hex characters.
func ValidNonce(text string) bool { return lowerHex(text, 32) }

// ValidHWID reports whether text is a machine ID: 64 lowercase hex characters.
func ValidHWID(text string) bool { return lowerHex(text, 64) }

// Validate checks the shape of a request.
func (r Request) Validate() error {
	if !ValidNonce(r.Nonce) {
		return errors.New("the nonce must be 32 lowercase hex characters")
	}
	if !ValidHWID(r.VMHWID) {
		return errors.New("the machine ID must be 64 lowercase hex characters")
	}
	return nil
}

// Sign signs the proof for request at the host's time at with the host's private key (base64url,
// no padding).
func Sign(private ed25519.PrivateKey, request Request, at int64) string {
	return base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, ProofMessage(request.Nonce, request.VMHWID, at)))
}

// Verify checks a proof signature (base64url) made for request and exactly the time at with the
// host's public key. A different time, nonce or machine never verifies.
func Verify(public ed25519.PublicKey, request Request, at int64, signature string) bool {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(signature)
	if err != nil || base64.RawURLEncoding.EncodeToString(raw) != signature || len(raw) != ed25519.SignatureSize || len(public) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(public, ProofMessage(request.Nonce, request.VMHWID, at), raw)
}

// EncodeKey is the text of a host public key: base64url without padding (43 characters).
func EncodeKey(public ed25519.PublicKey) string { return base64.RawURLEncoding.EncodeToString(public) }

// DecodeKey reads a host public key strictly: exactly 32 bytes in canonical unpadded base64url.
func DecodeKey(text string) (ed25519.PublicKey, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(text)
	// Go's decoder skips CR and LF, so the canonical spelling is checked by encoding it again.
	if err != nil || len(raw) != ed25519.PublicKeySize || base64.RawURLEncoding.EncodeToString(raw) != text {
		return nil, errors.New("a host key is 32 bytes in base64url without padding")
	}
	return ed25519.PublicKey(raw), nil
}

// MaskKey shortens a host key for display: the first six and the last four characters.
func MaskKey(text string) string {
	if len(text) < 14 {
		return ""
	}
	return text[:6] + "..." + text[len(text)-4:]
}
