package certs

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// TimePurpose marks a signed time stamp; certificates never carry it, so a stamp
// can never be mistaken for a license and a license never for a stamp.
const TimePurpose = "server-time"

// TimeStamp is the server's signed "now", used by trial apps to anchor their clock
// (ADR-0046 addendum 2026-10-19). Field order is alphabetical, like Payload.
type TimeStamp struct {
	KeyID   string `json:"kid"`
	Purpose string `json:"purpose"`
	Time    int64  `json:"time"`
	Version int    `json:"version"`
}

// SignTime signs unix seconds now.
func (s *Signer) SignTime(now int64) (Bundle, error) {
	stamp := TimeStamp{KeyID: s.KeyID, Purpose: TimePurpose, Time: now, Version: 1}
	raw, err := json.Marshal(stamp)
	if err != nil {
		return Bundle{}, err
	}
	signature := ed25519.Sign(s.private, raw)
	return Bundle{Payload: string(raw), Signature: base64.RawURLEncoding.EncodeToString(signature)}, nil
}

// VerifyTime checks a signed time stamp (tests and tooling).
func VerifyTime(public ed25519.PublicKey, b Bundle) (TimeStamp, error) {
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(b.Signature, "="))
	if err != nil || !ed25519.Verify(public, []byte(b.Payload), signature) {
		return TimeStamp{}, errors.New("signature does not match")
	}
	var stamp TimeStamp
	if err := json.Unmarshal([]byte(b.Payload), &stamp); err != nil || stamp.Purpose != TimePurpose {
		return TimeStamp{}, errors.New("not a server time stamp")
	}
	return stamp, nil
}

// PartHash is the salted SHA-256 a client sends for one machine part, so the
// server can compare parts without ever learning the raw identifiers.
func PartHash(kind, value string) string {
	digest := sha256.Sum256([]byte("scadabench-trial-part-v1|" + kind + "|" + value))
	return hexString(digest[:])
}

func hexString(raw []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(raw)*2)
	for _, b := range raw {
		out = append(out, digits[b>>4], digits[b&15])
	}
	return string(out)
}
