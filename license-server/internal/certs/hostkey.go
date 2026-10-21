package certs

import "encoding/base64"

// HostKeyBytes is the size of a VM host key: an Ed25519 public key (ADR-0099).
const HostKeyBytes = 32

// ValidHostKey accepts exactly the text the app sends: 32 bytes in canonical base64url without
// padding (43 characters). Padding, the standard alphabet, spaces and any other length are refused,
// so every certificate carries one spelling of one key.
func ValidHostKey(text string) bool {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(text)
	// Go's decoder skips CR and LF, so the canonical spelling is checked by encoding it again.
	return err == nil && len(raw) == HostKeyBytes && base64.RawURLEncoding.EncodeToString(raw) == text
}
