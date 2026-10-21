# Stream B — scl-project (P3)

Branch `go-b-scl-project` (from `master` c983a8f). Owner of `internal/scl`, `internal/project`,
`internal/store`, `internal/api/projectapi` and the native design in `studio/native/xsd`.
The Python backend (`studio/backend/studio_app`) was read only; nothing in it changed.
Public inputs only: `scd/*.scd`, the demo station
(`marketing/videos/demo/scadabench_demo.scd`), `make_template`, generated SCL and the
fixtures named in the brief. No `test_cids/`, `studio/data` or customer SCL was read.

## What exists

| Package | Responsibility |
|---|---|
| `scl/xmldoc` | One-pass XML parser that reproduces pyexpat + defusedxml: error texts and positions (line, character column, BOM counted), DOCTYPE entity refusal (`EntitiesForbidden(...)`), declared single-byte charsets through tables generated from CPython codecs (`tools/gen_charsets.py`), UTF-16/UTF-8 BOM detection, ElementTree view (text/tail without comments) and an lxml view (nodes, prefixes, namespace declarations) for the exporter; lxml serialisation |
| `scl/pyrules`, `scl/value` | CPython rules the builder depends on: `int()`/`float()` parsing, `repr`, `str.strip`, round-half-even, `datetime.fromisoformat`, `utf-8-sig` decode errors; `validate_value` texts |
| `scl` | Document limits and identifiers, type resolver, model builder (exact ordered `pyjson` trees and cfg text), reports and catalogue, communication, GOOSE/SV controls and links, preview, importer with selection rules, schema-version choice and validation front end, template generator, verified exporter |
| `project` (+ `contract`, `pyval`, `netset`, `gateway`, `csvtrace`) | Format 15 compact/expand, migrations 1-15, restore preparation, diagram/source/scenario/test-panel validation with pydantic-identical error lists, removal and bulk removal (fingerprint), IED settings and network, loopback endpoints, snapshot, preferences, list filters |
| `project/service` | The project actor: copy-on-write project behind an atomic pointer, one mutation lock, revision and `project_changed` source, journal + coalesced background snapshot, restore, startup recovery, removal with a recovery backup, derived-model cache on restore |
| `store/*` | Archive encode/decode (Python reads Go archives and the other way round; unchanged SCL segments are never deflated again), journal and snapshot writer, recovery backups and startup recovery, JSON files with Python's byte format (indent 2, CRLF on Windows), preferences and tool memory, derived-model cache, XSD verdict cache + `studio_xsd` client |
| `api/projectapi` | `/project`, `/project/diagram`, `/project/save`, `/project/download`, `/project/open`, `/scl/validate`, `/scl/preview`, `/scl/import`, `/ieds/template`, `/ieds/{id}/source`, `/ieds/{id}/export-scl`, IED removal (single and bulk), `/preferences` as `[]api.Route` |
| `studio/native/xsd` | C sources + CMake for `studio_xsd.exe` (libxml2 2.11.9, lxml 6.0.2's parse options), README with the build and parity plan. Not built (see Not done) |

## Evidence

All counts are byte-for-byte comparisons with the frozen Python app (CPython 3.12.6,
lxml 6.0.2 / libxml2 2.11.9, expat 2.6.3, defusedxml 0.7.1, pydantic 2.10.6).

| Check | Result |
|---|---|
| `parse_document` errors and accepted documents (`scl/testdata/golden/xml_errors.json`) | 3403 / 3403 |
| SCL corpus: 295 synthetic + public cases, preview / validate / import / per-IED import / cfg | 1170 golden files identical |
| `utf-8-sig` decode errors | 326 / 326 |
| `make_template` bytes | identical |
| Exporter (`export_ied`) over corpus IEDs and override sets | 2407 / 2407 documents identical |
| Project vectors (compact/expand, migrations 1-15, validation, removal, settings, preferences) | 1629 / 1629 |
| Store crosscheck: Python decodes Go archives (including cached segments and a folded journal); Go decodes Python archives | pass |
| `.subsim` round trip: Python make → Go restore (same snapshot) → Go export → Python open (same snapshot) | pass (`TestPythonArchiveRoundTrip`, needs `STUDIO_PYTHON`) |
| HTTP: 68 requests recorded from the Flask test client (`projectapi/testdata/http_gen.py`), replayed against the Go routes: status, every header except Content-Length, exact body | 68 / 68, including Werkzeug 400/415 texts, pydantic details, 409 conflicts, 500 bodies, `send_file` headers, `ieds_removed` event |
| Derived-model cache: cached restore gives the same snapshot and stored project as a full rebuild | pass (`TestModelCacheRestoreIsIdentical`) |
| XSD client against a fake helper: framing, cache file bytes = Python `json.dumps`, crash/timeout restart, concurrency, 559 seeded verdicts served with no helper start | 21 / 21 |
| `go vet ./...`, `python studio/tools/check_file_lengths.py` | clean / PASS |

Checks were run with `GOMAXPROCS=4`, `go test -p 2`, one heavy command at a time.

### Timings (same project, same machine)

`TestOpenSaveTimings` (opt-in: `STUDIO_TIMING=1`, `STUDIO_PYTHON`): template plus the public
SCD IEDs, 11 IEDs, archive 22.9 kB, built by the Python service.

| Operation | Python | Go |
|---|---|---|
| Open (restore, median of 5) | 86.6 ms | 49.6 ms full rebuild; 22.5 ms from the model cache |
| First save | 7.4 ms | 4.0 ms |
| Save (median of 5) | 7.9 ms | 1.9 ms |
| Open, small project (template + 2 IEDs) | 45.7 ms | 21.6 ms |
| Archive encode of a 22 MB SCL: full / after a diagram-only edit | — | 273 ms / 1 ms (unchanged SCL segment reused) |

Small project numbers are dominated by fixed costs; the gains grow with SCL size because the
journal makes a save a single appended line and the snapshot never re-deflates unchanged SCL.

## Decisions

1. **Parser.** A hand-written parser instead of `encoding/xml`: `encoding/xml` cannot give
   expat's error texts or positions, accepts input expat refuses and has no prefix/node view
   for an lxml-identical export. The parser is one pass and builds both views on demand.
2. **Charsets.** pyexpat accepts a declared single-byte encoding through Python codecs.
   The 24 tables in use plus `latin_1`, `ascii`, `utf_8` are generated from CPython; any
   other name answers `unknown encoding: X` (see Deviations).
3. **Project actor.** One mutation lock; readers take an immutable snapshot from an atomic
   pointer, so `GET /project` never waits behind a save. Edits copy only the touched path.
4. **Durability.** Each change appends one journal line (`seq` strictly increasing, separate
   from Python's revision). A background writer coalesces snapshots to `autosave.subsim`;
   on close the last snapshot is written and the journal removed. Startup folds a leftover
   journal before recovery, so Python can still open the data folder.
5. **Derived-model cache.** `<data>/cache/models/<key>.json`, key and file format in the
   `store/modelcache` package comment, versioned by `scl.ModelVersion`. The cache is
   untrusted: any damage is a miss. Ids are still drawn in import order, so results are
   identical to a rebuild.
6. **XSD.** No Go XSD engine can reproduce libxml2's verdicts and messages. Validation goes
   to `studio_xsd.exe` (libxml2 2.11.9, the version lxml 6.0.2 bundles) behind a verdict
   cache keyed by `sha256(SCL)`, schema, libxml2 version and a digest of the schema files.
   There is no Python fallback. A missing helper is a `RuntimeError` with an actionable text.
7. **Test data.** `testdata` folders carry `.gitattributes` `* -text` so goldens stay
   byte-exact with `core.autocrlf=true`; large goldens live under `generated/` folders.

## Deviations (known, documented)

- Internal DTD subset: ATTLIST default attributes are not applied (the DOCTYPE is refused
  before the model anyway; only the error position of malformed subsets is approximate).
- Declared encodings outside the tabled set answer `unknown encoding` where CPython might
  know the codec; multi-byte codecs answer as pyexpat does.
- Absurd integers in report attributes (count, confRev, ix beyond int64) are clamped.
- `fromisoformat` with a UTC offset of 24 h or more: message text approximate.
- Download names: the ASCII fallback approximates NFKD for Latin-1 letters only.
- Downloads answer `Range` requests in full (Werkzeug answers 206); the UI never sends Range.
- `selected_names` JSON error texts follow CPython's scanner for the common errors; rare
  error kinds may differ in wording.
- Store: a corrupt deflate/bzip2 stream answers `Invalid project archive`; lone surrogates
  in stored text become U+FFFD; snapshots are not fsynced (the journal line is).
- `Snapshot()` deep-copies the project (a later speed-up, not a behaviour change).

## Not done

- `studio_xsd.exe` is not built: the libxml2 2.11.9 and GNU libiconv sources are not on this
  machine and nothing was downloaded. The C sources compile cleanly at `/W4` against the
  headers shipped in the lxml wheel. Until the helper exists, Go serves cached/seeded
  verdicts and answers `RuntimeError` on a miss. Next: build per `studio/native/xsd/README.md`
  and compare every verdict behind `xsd_seed.json` with Python.
- `Service.Snapshot` copy cost and an `--derive-models` Python writer were not needed:
  streams C-E can use the Go importer directly.
- No packaged EXE was built (not requested).

## What other streams need from B

- **A (server).** Render `project.ContractError` as `{"details": [...], "error": "Invalid request"}`
  (400) and Werkzeug `HTTPException` bodies `{"error": description}`; Flask outside debug
  replaces JSON decode errors with the generic BadRequest text. `projectapi/respond.go`
  carries a mirror of the error mapping only so the routes test alone; A's version wins.
  Wire `Deps{Projects, Runtime, Startup, Preferences}`, the startup gate (excluding
  `/api/v1/preferences`) and the 100 MB request limit.
- **A / launcher.** Call `autosave.Fold(dataDir, encoder)` before handing a data folder to Python;
  run `Service.RecoverStartup` through `recovery.StartBackground` for `startup_recovery`.
- **C (runtime).** Implement `projectapi.Runtime` (`State`, `RunningIEDs`, `CycleLock`,
  `Emit`); `ieds_removed {ied_ids, count}` is emitted after a bulk removal commits.
  `service.Config.OnChange(revision, source)` feeds `project_changed`.
- **F (harness).** Reuse `projectapi/testdata/http_gen.py`'s deterministic-uuid recorder for
  the parity suite; `scl/testdata` holds the corpus generator and goldens.
- **All.** `scl/pyrules` and `project/pyval` overlap with a future shared `pycompat`.
  Windows Defender occasionally refuses to run fresh test binaries ("Access is denied") and
  delays deleting just-written files; `go test -c -o <scratch>` and the patient temp-dir
  cleanup in `store/autosave` tests work around it.

## Proposed ADR text

**ADR-B1: Pinned libxml2 helper for SCL schema validation.** Need: Python validates with
lxml 6.0.2 (libxml2 2.11.9); verdicts and messages must stay identical without Python.
Decision: `studio_xsd.exe`, a C helper statically linked with unmodified libxml2 2.11.9,
using lxml's parse options (`NOCDATA|NONET|COMPACT|BIG_LINES`), framed JSON over stdio,
documents passed as work files; the engine caches verdicts by SCL hash, schema, libxml2
version and schema digest. Compatibility: verdict cache is new state under `<data>/cache/xsd`;
Python ignores it. Upgrading libxml2 is an ADR change and invalidates the cache by key.

**ADR-B2: Project journal and background snapshot.** Need: saving re-deflated every SCL
source on each change. Decision: append-only journal per change, coalesced background
snapshot, unchanged SCL segments reused, fold on start and close. Compatibility: the
archive format is unchanged; Python opens any folder once the journal is folded.

**ADR-B3: Derived-model cache.** Need: reopening rebuilt every IED model from SCL.
Decision: `<data>/cache/models` keyed by builder version, source digest, filename and
selected names; untrusted, LRU of 64. Compatibility: new optional state; results are
identical with or without it.

**ADR-B4: Byte-exact test data.** `* -text` in Go `testdata`, generated goldens under
`generated/`, so goldens survive `core.autocrlf=true` and the file-length rule.

## Log

- Parser, charsets, SCL builder and goldens; importer, preview, exporter and template.
- Project document, validation, migrations, removal, settings; vectors from Python.
- Store (archive, journal, snapshots, recovery, settings, model cache) with Python crosscheck.
- Project service with journal, background snapshot, restore and `.subsim` round trip.
- `projectapi` routes with 68/68 Flask-recorded parity; `studio_xsd` sources and Go verdict
  cache; derived-model cache on restore; open/save timings; patient temp-dir cleanup in
  `store/autosave` tests (Windows delete-pending flake under load).
