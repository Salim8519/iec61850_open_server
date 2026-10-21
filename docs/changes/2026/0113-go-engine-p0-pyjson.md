# 0113: Go engine P0 starts: module `studio/engine` and Python-compatible JSON (`pyjson`)

Date: 2026-10-19. Owner: "start the work asap" after approving the plan ([ADR-0074](../../decisions/ADR-0074-go-engine.md)).
Decision: [ADR-0075](../../decisions/ADR-0075-python-compatible-json.md). First step of phase P0 in the
[Go engine plan](../../architecture/go-engine/README.md). No product behaviour changed; Python stays the engine.

## Change

- New Go module `studio/engine` (`studio-engine`, Go 1.24, stdlib only, no cgo). The GOOSE engine
  moves into it in a later P0 step.
- `internal/pyjson`: ordered `Object` (Python dict order rules), `Repr`/`AppendFloat`
  (`repr(float)`, NaN/Infinity), `AppendString` (`ensure_ascii`), encoders `Flask`, `Dumps`,
  `Compact`, and `Decode` (`json.loads` rules: int vs float, big integers, repeated keys, strict
  strings, NaN/Infinity, trailing data).
- Tests: fixed cases recorded from Flask 3.1.2 and Python, rejection cases, ordering after delete,
  and `TestAgainstCPython`, which runs `testdata/pyjson_vectors.py` (random documents with the exact
  bytes of Flask `jsonify` and `json.dumps`) and compares all three encodings byte for byte.
- `studio/tools/engine_tests.py` runs the module's tests with the venv Python (Flask needed);
  `verify.py` gains the check "Go engine (go test, pyjson vs CPython)".

## Evidence

- Flask 3.1.2 output recorded in this venv:
  `{"a":[1,1e-05,1e+16,"\u00e9\u007f<&>",null,true],"b":1.0,"c":{"y":NaN,"z":2}}` plus newline
  (sorted, compact, ASCII, NaN bare); the app never sets debug mode.
- `go vet` clean; `go test ./...` pass; differential test 20,000 documents (seed 61850): 0 differences.
- **P0 gate**: `PYJSON_CASES=1000000 PYJSON_SEED=20261019`: 1,000,000 documents, 0 differences,
  160.5 s.
- `python studio/tools/verify.py` (system Python): all 7 checks PASS.

## Limitations

- Lone UTF-16 surrogate escapes become U+FFFD; decode error texts are not yet Python's
  `JSONDecodeError` texts; integers above 4300 digits are accepted (CPython refuses). None occur in
  ScadaBench data; listed in ADR-0075.
- Only the JSON layer exists; no route, engine switch or harness yet (next P0 steps: `pycompat`,
  HTTP skeleton with the security order, SSE ring, engine switch and lock, parity harness).
