# ADR-0075: Python-compatible JSON in the Go engine, and how parity is checked

Accepted: 2026-10-19. Part of [ADR-0074](ADR-0074-go-engine.md) phase P0. Change: [0113](../changes/2026/0113-go-engine-p0-pyjson.md).

**Problem**

The frontend, project files, run logs and native workers read JSON written by Python. Go's
`encoding/json` differs in every visible detail: key order, separators, HTML escaping, non-ASCII
output, float form (`1` vs `1.0`, `1e+06` vs `1000000.0`), NaN (refused), big integers, and an
integer `1` decoded as float. Any of these would be a change the user or a stored file can notice.

**Decision**

1. A package `studio/engine/internal/pyjson` (stdlib only) is the only JSON writer and reader of
   the Go engine for anything Python wrote or reads.
2. Three encoders reproduce CPython exactly:
   - `Flask`: Flask 3.1 `jsonify` outside debug mode (the app never sets debug): keys sorted by
     code point, `,` and `:` separators, ASCII-only output, trailing newline. Used for every REST
     body.
   - `Dumps`: `json.dumps` defaults (insertion order, `", "` / `": "`), used for SSE events, run
     logs and other files Python writes with defaults.
   - `Compact`: `json.dumps(separators=(",", ":"))`.
   All three: `ensure_ascii` escapes (short escapes for `"` `\` and `\b \f \n \r \t`, lowercase
   `\uXXXX` otherwise, surrogate pairs above U+FFFF, DEL escaped); `repr(float)` (shortest
   round-trip digits, positional for decimal exponents -4..15 with `.0`, otherwise `e±XX`);
   bare `NaN`, `Infinity`, `-Infinity`; arbitrary-size integers.
3. The decoder follows `json.loads` (strict): insertion-ordered objects, a repeated key keeps its
   first place with its last value, integers stay integers (`int64` or `big.Int`), `1.0` stays a
   float, overflow gives ±Inf, NaN/Infinity literals accepted, control characters in strings and
   trailing data rejected.
4. **Parity check by differential testing against CPython itself**: a Python generator writes
   random documents (nested objects, all string classes incl. controls, non-BMP and quotes, random
   64-bit floats incl. NaN/Inf/subnormals, boundary floats, big integers) with the exact bytes of
   Flask's `jsonify` and `json.dumps`; the Go test decodes each and must reproduce all three forms
   byte for byte. `verify.py` runs 20,000 documents; the P0 gate runs 1,000,000 with a new seed.

**Known differences (documented, not reachable from ScadaBench data)**

- A lone UTF-16 surrogate escape (`"\ud800"`) becomes U+FFFD in Go; Python keeps the surrogate.
- Decode error texts are not yet Python's `JSONDecodeError` texts; request-body errors get their
  own parity work with the HTTP layer.
- CPython refuses integers above 4300 digits when decoding; Go accepts them.

**Compatibility impact**

None for users: Python remains the engine. The Go engine gains the encoding every later phase uses.
`verify.py` gains one check (the Go engine module tests).
