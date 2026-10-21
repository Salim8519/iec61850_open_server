package licensing

import (
	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Certificate texts, byte-identical to licensing/certificate.py.
const (
	CertUnreadableSignature = "The license file signature is unreadable."
	CertNotLicenseFile      = "This is not a license file. Paste the whole license.cert text."
	CertIncomplete          = "The license file is incomplete. Paste the whole license.cert text."
	CertNoKeys              = "This ScadaBench build has no license key configured. Contact the seller."
	CertDamaged             = "The license file is damaged."
	CertChanged             = "The license file is damaged or was changed. Activate again."
)

// CertificateError is certificate.CertificateError: the bundle is unreadable, unsigned by
// a release key, or altered. The license API answers it with code "bad_certificate".
type CertificateError struct{ Message string }

func (e *CertificateError) Error() string { return e.Message }

// errUnhashableKid stands for the TypeError Python raises for a list or object "kid";
// the caller does not catch it (HTTP 500), so it is not a CertificateError.
var errUnhashableKid = uncaught("unhashable type: certificate kid")

// Bundle is the {"payload", "signature"} pair of a license.cert file.
type Bundle struct {
	Payload   string
	Signature string
}

// Object returns the bundle as Python's dict, in its key order.
func (b Bundle) Object() *pyjson.Object {
	return pyjson.NewObject().Set("payload", b.Payload).Set("signature", b.Signature)
}

// ParseBundle accepts the license.cert text (a Go string, as saved or pasted) or an
// already decoded JSON value, optionally wrapped as {"certificate": {...}}.
func ParseBundle(input any) (Bundle, error) {
	bundle := input
	if text, isText := input.(string); isText {
		decoded, err := pytext.LoadsString(text)
		if err != nil {
			return Bundle{}, &CertificateError{CertNotLicenseFile}
		}
		bundle = decoded
	}
	object, isObject := bundle.(*pyjson.Object)
	if inner, found := lookup(object, "certificate"); found {
		if wrapped, ok := inner.(*pyjson.Object); ok {
			object = wrapped
		}
	}
	payload, okPayload := lookupString(object, "payload")
	signature, okSignature := lookupString(object, "signature")
	if !isObject || !okPayload || !okSignature {
		return Bundle{}, &CertificateError{CertIncomplete}
	}
	return Bundle{Payload: payload, Signature: signature}, nil
}

func lookupString(object *pyjson.Object, key string) (string, bool) {
	value, _ := lookup(object, key)
	text, ok := value.(string)
	return text, ok
}

// decodeSignature is certificate._decode_signature: strip, pad, urlsafe base64.
func decodeSignature(text string) ([]byte, error) {
	stripped := pytext.Strip(text)
	signature, err := URLSafeB64Decode(PadBase64(stripped))
	if err != nil {
		return nil, &CertificateError{CertUnreadableSignature}
	}
	return signature, nil
}

// VerifyBundle returns the payload object when the signature matches an accepted key.
// keys nil means this release's PublicKeys(); an empty map means no key is configured.
func VerifyBundle(bundle Bundle, keys map[string][]byte) (*pyjson.Object, error) {
	if keys == nil {
		keys = PublicKeys()
	}
	if len(keys) == 0 {
		return nil, &CertificateError{CertNoKeys}
	}
	message := []byte(bundle.Payload)
	decoded, err := pytext.LoadsBytes(message)
	if err != nil {
		return nil, &CertificateError{CertDamaged}
	}
	payload, ok := decoded.(*pyjson.Object)
	if !ok {
		return nil, &CertificateError{CertDamaged}
	}
	signature, err := decodeSignature(bundle.Signature)
	if err != nil {
		return nil, err
	}
	candidates, err := candidateKeys(payload, keys)
	if err != nil {
		return nil, err
	}
	for _, key := range candidates {
		if Verify(key, message, signature) {
			return payload, nil
		}
	}
	return nil, &CertificateError{CertChanged}
}

// candidateKeys is `[keys[kid]] if payload.get("kid") in keys else list(keys.values())`.
func candidateKeys(payload *pyjson.Object, keys map[string][]byte) ([][]byte, error) {
	kid, _ := payload.Get("kid")
	switch value := kid.(type) {
	case []any, *pyjson.Object:
		return nil, errUnhashableKid
	case string:
		if key, found := keys[value]; found {
			return [][]byte{key}, nil
		}
	}
	all := make([][]byte, 0, len(keys))
	for _, key := range keys {
		all = append(all, key)
	}
	return all, nil
}
