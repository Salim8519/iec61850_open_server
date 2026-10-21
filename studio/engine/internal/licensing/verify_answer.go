package licensing

import "studio-engine/internal/pyjson"

// VerifyAnswerPurpose marks the license server's signed answer to POST /api/v1/verify
// (licensing/verify_answer.py, ADR-0097). The answer carries
// {"answer": {"payload": "<exact JSON text>", "signature": "<base64url>"}} signed with the
// same Ed25519 key as certificates; the payload is {"hwid", "kid", "lease_token",
// "license_id", "nonce", "purpose": "verify-answer", "status", "time", "version": 1}
// (lease_token is new in ADR-0098 and "" for licenses without a lease; Python answers
// have no such key).
const VerifyAnswerPurpose = "verify-answer"

// verifyStatuses are the verdicts a genuine answer may carry.
var verifyStatuses = map[string]bool{"valid": true, "revoked": true, "moved": true, "unknown": true}

// VerifiedAnswer is a genuine signed answer to one verify request.
type VerifiedAnswer struct {
	Status     string // valid, revoked, moved or unknown
	LeaseToken string // the new lease token, "" when the answer carries none
	Time       int64  // the server's signed time
	Bundle     Bundle // the signed answer exactly as received
}

// VerifyAnswer returns the signed verdict only for a genuine answer to THIS request: the
// signature verifies with keys, purpose and version match, and the nonce this client just
// sent, the license ID and the machine ID come back unchanged. The unsigned top-level
// "status" is never used. ok is false for anything else, which the caller treats exactly
// like an unreachable server.
func VerifyAnswer(answer any, nonce string, licenseID any, hwid string, keys map[string][]byte) (VerifiedAnswer, bool) {
	object, isObject := answer.(*pyjson.Object)
	if !isObject {
		return VerifiedAnswer{}, false
	}
	value, _ := object.Get("answer")
	signed, isObject := value.(*pyjson.Object)
	if !isObject {
		return VerifiedAnswer{}, false
	}
	text, hasPayload := lookupString(signed, "payload")
	signature, hasSignature := lookupString(signed, "signature")
	if !hasPayload || !hasSignature {
		return VerifiedAnswer{}, false
	}
	bundle := Bundle{Payload: text, Signature: signature}
	payload, err := VerifyBundle(bundle, keys)
	if err != nil {
		return VerifiedAnswer{}, false // a CertificateError, or the unhashable "kid" Python also refuses
	}
	if purpose, _ := payload.Get("purpose"); purpose != any(VerifyAnswerPurpose) {
		return VerifiedAnswer{}, false
	}
	if version, _ := payload.Get("version"); version != any(int64(1)) {
		return VerifiedAnswer{}, false
	}
	id, isText := licenseID.(string)
	if got, _ := payload.Get("license_id"); !isText || got != any(id) {
		return VerifiedAnswer{}, false
	}
	if got, _ := payload.Get("nonce"); got != any(nonce) {
		return VerifiedAnswer{}, false
	}
	if got, _ := payload.Get("hwid"); got != any(hwid) {
		return VerifiedAnswer{}, false
	}
	verdict, _ := payload.Get("status")
	name, isText := verdict.(string)
	if !isText || !verifyStatuses[name] {
		return VerifiedAnswer{}, false
	}
	result := VerifiedAnswer{Status: name, Bundle: bundle}
	if token, ok := payload.Get("lease_token"); ok {
		if text, isText := token.(string); isText && ValidLeaseToken(text) {
			result.LeaseToken = text
		}
	}
	if raw, ok := payload.Get("time"); ok {
		result.Time, _ = pyInt(raw)
	}
	return result, true
}

// VerifiedStatus is VerifyAnswer's verdict alone (the Python-compatible check).
func VerifiedStatus(answer any, nonce string, licenseID any, hwid string, keys map[string][]byte) (status string, ok bool) {
	verified, ok := VerifyAnswer(answer, nonce, licenseID, hwid, keys)
	return verified.Status, ok
}
