package licensing

import "studio-engine/internal/pyjson"

// VerifyAnswerPurpose marks the license server's signed answer to POST /api/v1/verify
// (licensing/verify_answer.py, ADR-0097). The answer carries
// {"answer": {"payload": "<exact JSON text>", "signature": "<base64url>"}} signed with the
// same Ed25519 key as certificates; the payload is {"hwid", "kid", "license_id", "nonce",
// "purpose": "verify-answer", "status", "time", "version": 1}.
const VerifyAnswerPurpose = "verify-answer"

// verifyStatuses are the verdicts a genuine answer may carry.
var verifyStatuses = map[string]bool{"valid": true, "revoked": true, "moved": true, "unknown": true}

// VerifiedStatus returns the signed verdict only for a genuine answer to THIS request: the
// signature verifies with keys, purpose and version match, and the nonce this client just
// sent, the license ID and the machine ID come back unchanged. The unsigned top-level
// "status" is never used. ok is false for anything else, which the caller treats exactly
// like an unreachable server.
func VerifiedStatus(answer any, nonce string, licenseID any, hwid string, keys map[string][]byte) (status string, ok bool) {
	object, isObject := answer.(*pyjson.Object)
	if !isObject {
		return "", false
	}
	value, _ := object.Get("answer")
	signed, isObject := value.(*pyjson.Object)
	if !isObject {
		return "", false
	}
	text, hasPayload := lookupString(signed, "payload")
	signature, hasSignature := lookupString(signed, "signature")
	if !hasPayload || !hasSignature {
		return "", false
	}
	payload, err := VerifyBundle(Bundle{Payload: text, Signature: signature}, keys)
	if err != nil {
		return "", false // a CertificateError, or the unhashable "kid" Python also refuses
	}
	if purpose, _ := payload.Get("purpose"); purpose != any(VerifyAnswerPurpose) {
		return "", false
	}
	if version, _ := payload.Get("version"); version != any(int64(1)) {
		return "", false
	}
	id, isText := licenseID.(string)
	if got, _ := payload.Get("license_id"); !isText || got != any(id) {
		return "", false
	}
	if got, _ := payload.Get("nonce"); got != any(nonce) {
		return "", false
	}
	if got, _ := payload.Get("hwid"); got != any(hwid) {
		return "", false
	}
	verdict, _ := payload.Get("status")
	name, isText := verdict.(string)
	return name, isText && verifyStatuses[name]
}
