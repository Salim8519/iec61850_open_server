// Package licensing is the Go port of studio_app/licensing and domain/license_policy.py:
// license.cert parsing and Ed25519 verification against the release keys, the license
// home files (license.cert, client_config.json, license_state.json, .time_anchor, .revoked_ids) written
// byte-identically to Python, the clock-rollback anchor fed by <data>/logs/runs.jsonl,
// trial and key activation (online through Client, offline from a pasted certificate),
// the weekly revocation check (a cancelled license ID is remembered for good, and only a
// server answer signed for the request counts), the read-only Windows machine ID and the
// tamper checks. A packaged build asks only its built-in server (ADR-0097).
//
// # License gate (consumed by the HTTP server)
//
// The server runs the security order 403 -> 401 -> 402 -> startup wait -> route. For the
// 402 step it calls
//
//	blocked, err := licensing.Require(gate, request.URL.Path)
//
// with gate a licensing.Gate (normally the *Service; nil when no license service exists).
// When blocked, it answers licensing.RequiredStatus (402) with licensing.RequiredBody()
// (Content-Type application/json). When err is not nil, Python would have raised inside
// before_request, so the server answers its generic 500. Paths for which
// licensing.Exempt(path) is true (prefix "/api/v1/license") and non-enforcing gates are
// never blocked; "/api/v1/session" is handled by the 401 step before the gate.
//
// # Routes
//
// The license routes (api/license.py) map errors with ErrorResponse and wrap the status
// with StatusBody (adds "app_version"). Error texts and codes are byte-identical to
// Python's LicenseActionError, CertificateError and LicenseServerError.
//
// # Seams
//
// Options injects the clock, machine-ID providers, release keys, enforce flag, home and
// data folders, the license-server client factory, the debugger probe and the expected
// native file digests, so the parity harness can drive Go and Python identically.
// Options.AllowTestServer and Options.Nonce are test-only seams the release never sets.
package licensing
