// Package updates is the Go port of studio_app/updates (ADR-0071): a quiet, signed,
// at most daily check of the license server's GET /api/v1/release. It sends nothing about
// the PC, verifies Ed25519 over "scadabench-release-v1\n" + payload with the release keys
// (licensing.PublicKeys), re-validates the record (scadabench.com https links with
// SHA-256 only), and remembers the answer in <data>/update_check.json, written
// byte-identically to Python's json.dumps(state, indent=2) in text mode. It never
// downloads or installs anything.
//
// Routes (api/updates.py): GET /api/v1/app/update -> Status(), POST
// /api/v1/app/update/check -> Check(true), PUT /api/v1/app/update -> Configure(body) with
// a *ConfigError answered 400 {"error", "code": "invalid"}. A *licensing.UncaughtError
// (or any other error) stands for an exception Python did not catch: HTTP 500.
//
// Seams: Options.Opener (the HTTP GET), Options.Clock, Options.Keys, Options.ServerURL
// and Schedule (sleep and jitter) let tests run without network or real waiting;
// SCADABENCH_UPDATE_CHECK=off disables the background loop.
package updates
