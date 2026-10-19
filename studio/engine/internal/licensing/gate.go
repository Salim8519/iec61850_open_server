package licensing

import (
	"strings"

	"studio-engine/internal/pyjson"
)

// Gate is what the HTTP server (stream A) consults for the 402 step of the security
// order 403 -> 401 -> 402 -> startup wait -> route. *Service implements it.
type Gate interface {
	// Enforced reports whether this build checks licenses (packaged builds only).
	Enforced() bool
	// Licensed is is_licensed(): the cached status (re-evaluated after 60 s). An error
	// stands for an exception Python's before_request would raise (HTTP 500).
	Licensed() (bool, error)
}

// Answer to an unlicensed request, byte-identical to api/security.py _require_license.
const (
	RequiredStatus = 402
	RequiredError  = "A license is required. Start the free trial or activate your key."
	RequiredCode   = "license_required"
	exemptPrefix   = "/api/v1/license"
)

// Exempt reports whether a request path is never gated: every path starting with
// "/api/v1/license" (status, trial, activation, config) stays reachable so users can
// activate. The prefix match is Python's str.startswith, so "/api/v1/licenses" is exempt too.
func Exempt(path string) bool { return strings.HasPrefix(path, exemptPrefix) }

// RequiredBody is the Flask jsonify body of the 402 answer:
// {"code":"license_required","error":"A license is required. ..."} plus "\n".
func RequiredBody() []byte {
	body, err := pyjson.Flask(pyjson.NewObject().Set("error", RequiredError).Set("code", RequiredCode))
	if err != nil {
		panic(err) // two fixed strings always encode
	}
	return body
}

// Require is _require_license for a request that already passed the 403 and 401 checks:
// blocked is true when the caller must answer RequiredStatus with RequiredBody(). A nil
// gate (no license service), a non-enforcing gate and exempt paths are never blocked.
func Require(gate Gate, path string) (blocked bool, err error) {
	if gate == nil || !gate.Enforced() || Exempt(path) {
		return false, nil
	}
	licensed, err := gate.Licensed()
	if err != nil {
		return false, err
	}
	return !licensed, nil
}
