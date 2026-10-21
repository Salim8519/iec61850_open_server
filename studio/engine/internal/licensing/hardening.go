package licensing

import (
	"crypto/rand"
	"encoding/hex"
)

// Texts of ADR-0097 (no moves, one built-in server, signed server answers), byte-identical
// to domain/license_policy.py.
const (
	MovesDisabledText = "Moving a license to another computer is not possible. Each license works on one computer only."
	ServerFixedText   = "This ScadaBench always uses its own license server. The server address cannot be changed."
	BadAnswerText     = "The license server's answer could not be verified, so nothing was changed. Try again later. Help: " + SupportURL
)

// RandomNonce is the default Options.Nonce: 16 random bytes as 32 lowercase hex characters.
func RandomNonce() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		panic("no random numbers available: " + err.Error())
	}
	return hex.EncodeToString(data)
}
