# Go engine workstreams (parallel, ADR-0076)

Part of the [Go engine plan](README.md). Six streams build P0-P5 at the same time. Each stream works
in its own worktree on a branch cut from `go-engine`, touches **only the paths it owns**, and keeps
a log at `streams/<stream>.md` (decisions, evidence, open questions, proposed ADR text).

## Shared contracts (do not change without the integrator)

| Contract | Where | Rule |
|---|---|---|
| JSON | `studio/engine/internal/pyjson` | Only JSON reader/writer for Python-visible data; REST bodies `pyjson.Flask`, events/files `pyjson.Dumps` |
| Engineering model | ordered `*pyjson.Object` trees | Same keys/order as Python dicts (`scl/importer.py`, `projects/service.py`); immutable once published; typed indexes built once |
| Routes | `internal/api` | Each area package returns `[]api.Route` with Python's exact paths |
| Errors | `internal/apperr` | StudioError (`code`, default `validation`; 409 for `conflict`), ValueError, KeyError (repr text), Runtime (503) |
| Native IPC | 4-byte LE length + JSON ≤ 16 MiB | int/float/bool distinct (`pyjson.Decode`); events and replies share one reader |
| Module | `studio/engine/go.mod` | Go 1.24, stdlib only, no cgo, `CGO_ENABLED=0` |

Rules for every stream: files under 300 lines, one responsibility each; domain/scl/project import only
`pyjson`, `pycompat`, `apperr`; no lock across I/O; never touch `studio/backend` (frozen), `studio/data`,
a running app, adapters or firewall; never build the EXE; tests with every package; parity against
Python via the harness (stream F) or golden files produced by Python on public fixtures only
(`studio/tests/fixtures`, demo station, generated SCL), never `test_cids/` or customer files.

## Streams

| Stream | Phase | Owns (under `studio/engine` unless noted) | Delivers |
|---|---|---|---|
| A platform-core | P0 | `internal/pycompat`, `internal/httpapi`, `internal/app`, `internal/events`, `internal/launcher`, `cmd/studio_server`; `studio/tools/run_dev_engine.py` | Python number/text rules; net/http server on 127.0.0.1 with the security order 403 → 401 → 402 → startup → route, Flask session cookie compatibility, Flask error bodies, static `/` and `/assets`; SSE `/api/v1/events` (5000 ring, replay, `resync`, heartbeat, encode once); `studio.lock` byte lock and engine switch (`--engine`, `SCADABENCH_ENGINE`); readiness line `ScadaBench: http://...` |
| B scl-project | P3 | `internal/scl`, `internal/project`, `internal/store`, `internal/api/projectapi`; native `studio/native/xsd` | One-pass XML parse, type resolver, model builder, cfg text, GOOSE/SV/report extraction, validation, export; project document format 15 (compact/expand, migrations 1-15), diagram/source/scenario/panel validation; archive, journal + background snapshot, derived-model cache, preferences/tool settings; project and SCL routes; `studio_xsd.exe` with pinned libxml2 |
| C runtime | P1 | `internal/native` (framing + ied/live/goose/control clients), `internal/runtime`, `internal/api/runtimeapi`; native `studio/native/control` multiplexed mode | Per-IED actors, value hub, cadence (0.1 s), pulses, automation/generators (CPython `random` compatible), scenarios, equipment control and feedback, energization on switch change, equipment states, run log writer, Run/Stop/single IED in parallel; operate through persistent `studio_control` (PDU-identical to `studio_probe control`) |
| D live-acquisition | P2 | `internal/acquisition`, `internal/api/liveapi` | Real-IED sessions (persistent, discovery cache), live values, GOOSE/SV monitors and links, MMS access, station health, test mode, signal search index, `/live`, `/goose*`, `/sv`, `/health*`, `/signals` routes |
| E tools | P4 | `internal/tools`, `internal/gateway`, `internal/api/toolsapi` | Gateway IEC 104 / converter / master process clients and rules, explorer, find/scan, test panel, activity, CSV exports |
| F harness-platform | P0/P5 | `studio/tests/parity/`, `studio/tests/conftest.py` (engine switch only), `studio/tools/benchmark_engines.py`; `internal/netcfg`, `internal/licensing`, `internal/updates` | Parity harness `STUDIO_ENGINE=python|python-http|go`, golden recorder/normaliser on the demo station, SCL corpus generator, benchmarks S1-S10 for both engines; network helper (IP Helper via syscall, journal, firewall names identical), licensing and update check |

Dependencies are through the contracts above, not through each other's code. Until stream A's server
exists, streams test their services and handlers directly (`httptest`). Until stream B's builder
exists, streams C-E load models from the derived-model cache written by Python
(`--derive-models`, defined by B) or from Python-generated golden JSON of public fixtures.

## Integration

The integrating session merges streams into `go-engine` in this order when each stream's checks pass
and the owner agrees: F (harness) and A (server) first, then B, C, D, E. On each merge it runs
`python studio/tools/verify.py`, the parity suite available so far, writes the change record and the
ledger line, and numbers proposed ADRs (0077 onward). Conflicts are resolved in favour of the
shared contracts.
