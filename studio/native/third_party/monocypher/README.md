# Monocypher (vendored, unmodified)

Monocypher 4.0.2 from https://github.com/LoupVaillant/Monocypher, tag `4.0.2`
(release archive SHA-256 `bc1ca30b1b2654e4e7daf2492c0d204200e55137f23fda6b7142fd7d523bd6b4`).
Dual-licensed CC0-1.0 / BSD-2-Clause; see `LICENCE.md`.

Studio uses only `crypto_ed25519_check` (RFC 8032 Ed25519 with SHA-512, from the
optional `monocypher-ed25519.c`) to verify license certificates in the native workers.
Files are copied byte-for-byte; do not edit them. Update by replacing all files and
the hash above, then run `studio/tests/unit/test_native_license.py`.
