package licensing

import (
	"encoding/base64"

	"studio-engine/internal/pyjson"
)

// BindingClient is the part of the license server API added by ADR-0098: the chip challenge
// at activation and the lease token in verify requests. A Client that also implements it is
// used for those; any other Client (the Python-compatible fakes of the parity tests) simply
// never gets a chip offer or a lease token.
type BindingClient interface {
	// ActivateWithChip binds licenseKey to hwid offering this computer's security chip; the
	// answer is either a challenge ({"challenge": {session, credential, secret}}) or, when
	// the server cannot use the chip, a plain certificate.
	ActivateWithChip(licenseKey, hwid string, offer *pyjson.Object) (any, error)
	// ActivateChip answers the challenge with the decrypted secret (base64) and returns the
	// certificate tied to the chip.
	ActivateChip(session, secret string) (any, error)
	// VerifyLease is Verify with the lease token this computer holds.
	VerifyLease(licenseKey any, hwid, nonce, leaseToken string) (any, error)
}

// ActivateWithChip posts the key, machine ID and chip offer (with the proof of work).
func (c *HTTPClient) ActivateWithChip(licenseKey, hwid string, offer *pyjson.Object) (any, error) {
	body := pyjson.NewObject().Set("license_key", licenseKey).Set("hwid", hwid)
	if offer != nil {
		body.Set("tpm", offer)
	}
	return c.proofRequest("/api/v1/activate", body)
}

// ActivateChip is POST /api/v1/activate/tpm.
func (c *HTTPClient) ActivateChip(session, secret string) (any, error) {
	return c.call("POST", "/api/v1/activate/tpm",
		pyjson.NewObject().Set("session", session).Set("secret", secret))
}

// VerifyLease is POST /api/v1/verify with the lease token.
func (c *HTTPClient) VerifyLease(licenseKey any, hwid, nonce, leaseToken string) (any, error) {
	body := pyjson.NewObject().Set("license_key", licenseKey).Set("hwid", hwid).Set("nonce", nonce).
		Set("lease_token", leaseToken)
	return c.call("POST", "/api/v1/verify", body)
}

// chipChallenge is the server's credential challenge for a chip offer.
type chipChallenge struct {
	Session    string
	Credential []byte
	Secret     []byte
}

// challengeOf reads {"challenge": {"session", "credential", "secret"}} from an activation
// answer; ok is false for an answer without one (a plain certificate).
func challengeOf(answer any) (chipChallenge, bool) {
	object, isObject := answer.(*pyjson.Object)
	if !isObject {
		return chipChallenge{}, false
	}
	value, _ := object.Get("challenge")
	challenge, isObject := value.(*pyjson.Object)
	if !isObject {
		return chipChallenge{}, false
	}
	session, hasSession := lookupString(challenge, "session")
	credential, hasCredential := lookupString(challenge, "credential")
	secret, hasSecret := lookupString(challenge, "secret")
	if !hasSession || !hasCredential || !hasSecret || session == "" {
		return chipChallenge{}, false
	}
	credentialBytes, err1 := decodeAnyBase64(credential)
	secretBytes, err2 := decodeAnyBase64(secret)
	if err1 != nil || err2 != nil || len(credentialBytes) == 0 || len(secretBytes) == 0 {
		return chipChallenge{}, false
	}
	return chipChallenge{Session: session, Credential: credentialBytes, Secret: secretBytes}, true
}

// decodeAnyBase64 reads standard or URL-safe base64, with or without padding.
func decodeAnyBase64(text string) ([]byte, error) {
	if data, err := base64.StdEncoding.DecodeString(text); err == nil {
		return data, nil
	}
	if data, err := base64.RawStdEncoding.DecodeString(text); err == nil {
		return data, nil
	}
	if data, err := base64.URLEncoding.DecodeString(text); err == nil {
		return data, nil
	}
	return base64.RawURLEncoding.DecodeString(text)
}
