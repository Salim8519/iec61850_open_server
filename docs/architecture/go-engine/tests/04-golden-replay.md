# 3. Golden / recorded comparison

Tests in section 2 check what someone thought to assert. The golden layer checks
everything else the frontend can see: every response body, header, status and event.

## Inputs allowed (privacy guard)

Only public material: the demo station `marketing/videos/demo/scadabench_demo.scd`
(OpenSCD-derived, FEED1/FEED2/BUS1/BUS2/TR1/TR2), the built-in sample IED
(`make_template`), `scd/model_substation.scd`, `scd/open_substation.scd`,
`scd/protection_relay.scd`, and the generated corpus of section 4.
`studio/tests/golden/public_manifest.json` lists their sha256. The recorder hashes every
uploaded body (`/scl/preview|validate|import`, `/project/open`, `/traces/preview`,
`/tools/*/points.csv`) and refuses, deleting partial output, if a hash is not listed, or if
`GET /project` shows an IED name outside the manifest. `test_cids/` can never be recorded.

## Scenario format

`studio/tests/golden/scenarios/<area>/<name>.json`:

```json
{"name": "demo-run-operate", "seams": {"token": "golden"}, "steps": [
  {"req": "POST /api/v1/scl/import", "multipart": {"file": "@demo"}, "form": {"selected_names": "[\"FEED1\",\"FEED2\"]"}},
  {"req": "PUT /api/v1/project/diagram", "json": {"diagram": "@vectors/demo_diagram.json", "revision": "$rev"}},
  {"req": "POST /api/v1/runtime/start"},
  {"wait_event": {"kind": "runtime", "match": {"state": "running"}}, "timeout": 30},
  {"req": "POST /api/v1/equipment/CBR_F1/operate", "json": {"value": false}},
  {"probe": "read", "ied": "FEED1", "ref": "FEED1LD1/XCBR1.Pos.stVal", "fc": "ST"},
  {"req": "GET /api/v1/equipment/states"}]}
```

`$name` binds a value from an earlier response (`$rev` = last `revision`); `<uuid:n>`
tokens in later requests are replaced with the engine's real id (id binding), so ids
never need to be deterministic. `probe` uses `studio_probe.exe` (independent client).
No `sleep` steps: only `wait_event`/`wait_until` with timeouts.

Two sources of scenarios:
1. **Hand-written per API family** - each of the 125 routes needs one success and one
   error scenario (coverage report fails otherwise).
2. **Recorded from the UI** - `tools/parity/record_proxy.py` sits between the browser and
   the backend while a scripted Playwright tour drives the demo station (five places,
   tool pages, SLD with both signal layers, run/stop, operate, test mode). The proxy writes
   every request/response pair and the raw SSE bytes; `scenario_from_recording.py` turns
   that into a scenario. This captures the real polling mix (`sharedPoll`) the frontend uses.

## Transcript

Running a scenario against one engine produces `transcript.jsonl`, one record per step:
`{step, req, status, headers{content-type, content-disposition, cache-control},
body_kind: json|text|bytes|zip, body (parsed, lexeme-preserving), body_sha256, events: [...]}`.
`events` holds every SSE event received between this step and the next, read from a real
`GET /api/v1/events` stream opened before step 0 (so SSE framing is exercised, not the
inspect API). With the IPC tap enabled (below) the transcript also holds
`ipc/<worker>.jsonl` frame sequences.

## Determinism before diffing

- `PYTHONHASHSEED=0` for Python. Then **Python self-diff**: record each scenario twice
  with seeds 0 and 1. Any difference is Python nondeterminism (the backend has 88 `set()`
  constructions; a list built from a set changes order with the hash seed). Fix Python
  (sort) in a normal change record; do not exempt it. Go comparisons count only for
  scenarios whose self-diff is clean.
- Generators: `simulation/generators.py` uses `random.Random(seed + int(elapsed*10))`.
  Go needs a CPython-compatible MT19937 (`init_by_array` seeding from the int's 32-bit
  words, 53-bit `random()`, `uniform = a + (b-a)*random()`); a vector file of
  `(seed, elapsed) -> value` with 10 000 rows is part of G1. Scenarios that run generators
  compare values via the IPC tap, not wall-clock sampled HTTP reads.
- Update check jitter (`random.uniform`) is disabled by `SCADABENCH_UPDATE_CHECK=off`.

## Normaliser (`studio/tests/golden/normalise.json`, versioned, one reason per rule)

Applied to both sides identically, on the parsed tree:

| Rule | Match | Replace |
| --- | --- | --- |
| ids | 32 lowercase hex (`uuid4().hex`: IED ids, session_id, run_id, pulse ids, IPC request ids) | `<uuid:n>`, n by first appearance in the transcript (keeps cross-references) |
| wall time | keys `timestamp`, `time`, `at`, `*_at`, `last_*` holding epoch floats in [1e9, 4e9] or ISO-8601 strings | `<ts>`; diff tool separately asserts per-kind monotonic order |
| durations | keys `elapsed*`, `*_ms`, `*_seconds`, `latency*`, `age*` | `<dur>`; asserts value >= 0 and same type |
| dynamic ports | server port, GOOSE inbox/deliver ports, `udp:<port>:` interfaces | `<port>` |
| paths | the temp data dir, `STUDIO_NATIVE_BIN` | `<data>`, `<bin>` (separators kept) |
| SSE id | `id:` values and `event.id` | relative to the first id of the scenario |
| cookie / token | `Set-Cookie` value, launch token | presence + attributes (`HttpOnly`, `SameSite=Strict`, `Path=/`) |

Anything not in the table is compared exactly. Adding a rule needs a change record.

## JSON diff rules

1. **API bodies are Flask `jsonify`**: Flask 3.1.2 provider = `sort_keys=True`,
   `ensure_ascii=True`, compact `(",", ":")` separators, trailing `\n`, `NaN`/`Infinity`
   allowed. Canonical compare: normalise the tree, re-serialise both sides with these
   exact rules, compare bytes. From G5 the **raw** bytes must also match (normaliser
   substitutions applied textually).
2. **Key order.** API responses: recursively sorted (Go `encoding/json` sorts map keys but
   emits struct fields in declaration order - Go must serialise API bodies through a
   sorted writer). SSE events (`json.dumps(event)`: `id, kind, timestamp, **payload`,
   separators `", "`/`": "`) and `project.json` (insertion order, `allow_nan=False`) keep
   Python insertion order: Go needs ordered maps there. The frontend iterates object order
   in `SignalValueEditor` and `ExportScl` (enum options), `TestModeCard`
   (`summary.devices`), `SignalOverlay` (`state.clients`), `useStudio` (`event.updates`),
   so object order is **never** relaxed.
3. **Numbers** are parsed keeping the lexeme. `1` vs `1.0` is a `numtype` diff: warning
   in API bodies before G5 (JavaScript cannot tell), error from G5, and always an error in
   persisted documents, CSV, cfg and SCL text, because Python writes `str(2.0) == "2.0"`
   there. Go floats must follow Python `repr`: shortest round-trip, exponent only when
   exp < -4 or >= 16 (`1e-05`, `1e+16`, `1234567.0` - Go `strconv 'g'` prints
   `1.234567e+06`), so Go needs a `pyfloat.Repr`. `true` vs `1` is a type diff.
   `round()` is banker's rounding in Python (9 call sites): Go must use `math.RoundToEven`.
4. **NaN/Infinity**: Python emits bare `NaN` in jsonify bodies and SSE (`allow_nan`
   default). `JSON.parse` rejects it, so any occurrence is reported as a **Python bug** to
   fix first (fixed in Python, then Go matches); never encoded into a golden.
5. **Strings** exact, including error text: `str(KeyError('x'))` is `'x'` with quotes;
   pydantic errors are `[{type, loc, msg, url}]` with `url` pinned to pydantic 2.10
   (`https://errors.pydantic.dev/2.10/v/missing`); werkzeug 404/405/413 descriptions;
   expat/defusedxml/lxml parser text. `ensure_ascii` escapes are lowercase hex with
   surrogate pairs for astral characters; `/` and `<>&` are **not** escaped (Go must turn
   HTML escaping off).
6. **Arrays** are ordered. No "unordered" exemption list: an order Python cannot keep
   stable is fixed in Python first (rule above).
7. **Headers/status**: status exact; `Content-Type` exact (`application/json`,
   `text/csv; charset=utf-8`, `text/event-stream; charset=utf-8`, `text/html; charset=utf-8`);
   `Content-Disposition` exact (download file names are user-visible); `Cache-Control` on
   SSE. Ignore `Date`, `Server`, `Content-Length`.
8. **SSE framing**: first chunk `retry: 1000\n\n`; heartbeat `: heartbeat\n\n` after 10 s
   idle; `id: N\ndata: <json>\n\n`; `resync` when the cursor fell out of the 5000-event
   window. Events are partitioned by `kind` (48 kinds published today).
   *Causal* kinds (`project_changed`, `equipment_*`, `signal`, `runtime`, `ied_runtime`,
   `scenario_*`, `pulse_*`, `gateway_*`, `master_*`, `explorer_*`, `ieds_removed`, ...)
   compare as exact ordered lists, and their position relative to the HTTP step that
   caused them must match. *Periodic* kinds (`goose_status`, `sv_status`, `live_status`,
   `energization` refreshes) compare shapes exactly and counts within +-1 per period;
   the split lives in `golden/event_kinds.json`, one reason per kind.
9. **Binary**: `.subsim` / project download = zip -> compare entry list, compression
   method and `project.json` bytes (zip timestamps ignored). CSV and HTML report bodies
   byte-compare after the timestamp rule. SCL export bytes: section 4.

## IPC tap (native boundary)

`studio_ipc_tap.exe` (test-only Go tool) is passed as `--native-exe` and copied under the
names `studio_live.exe`, `studio_control.exe`, `studio_master.exe`, ... It records every
frame in both directions to `<data>/ipc/<name>-<pid>.jsonl`, then proxies to the real
binary. Per-worker frame sequences are diffed after the id rule. This proves the Go engine
sends the natives the same cfg, start parameters, controls and pulses - the "native
protocol semantics" AGENTS.md requires to be identical - and supplies replay vectors for
the Go `internal/worker` tests.

## Replay modes

- **Dual-run** (development): `parity/replay.py --engines python,go scenarios/...` spawns
  both engines through the section 2 harness, runs the scenario on each and diffs the two
  transcripts directly.
- **Committed golden** (CI): `studio/tests/golden/transcripts/` holds Python transcripts
  recorded at the current commit. A Python change that alters a golden must update it in
  the same commit (reviewed diff, change record); Go is replayed against it. Goldens are
  regenerated only by `parity/record.py`, never hand-edited.

Diff output: one line per difference
`<scenario> step 7 GET /api/v1/health body.$.ieds[2].status: python "ok" != go "stale"`
plus a summary by diff class (`value`, `numtype`, `order`, `missing`, `extra`, `header`,
`event`, `ipc`). Exit code 0 only with zero differences.

## Coverage requirements

- every route in `docs/reference/API.md` / the 125 blueprint routes: >= 1 success and
  >= 1 error step;
- every published event kind appears at least once;
- every download (`*.csv`, `report.html`, `export-scl`, `project/download`) appears;
- the recorded UI tour covers the pages used in the screenshot gate (section 5).
