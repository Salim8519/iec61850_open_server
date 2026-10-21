# 0119: Go stream B merged: SCL parser/importer/exporter, project service with journal, store, project routes

Date: 2026-10-20. Workstream B of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(phase P2). Branch `go-b-scl-project` (0f38eca..54fa91a, 11 commits) merged into `master` (4de65d7).
Stream log: [b-scl-project](../../architecture/go-engine/streams/b-scl-project.md) (proposed ADR text
B1 pinned libxml2 helper, B2 journal and background snapshot, B3 derived-model cache, B4 byte-exact
test data; numbers assigned when wired). Python stays the engine; the routes are not wired yet.

## Change

- `internal/scl` (+ `xmldoc`, `value`, `pyrules`): XML parser reproducing expat/defusedxml, SCL
  preview, schema verdicts, importer, template and verified exporter with lxml serialisation.
- `internal/project` (+ `contract`, `pyval`, `netset`, `gateway`, `csvtrace`, `service`): project
  document format 15, migrations 1-15, pydantic-identical validation, removal, settings,
  preferences; service saving the change (journal line, fsynced) with a background snapshot.
- `internal/store` (`archive`, `autosave`, `recovery`, `settings`, `jsonfile`, `modelcache`, `xsd`,
  `pytext`, `crosscheck`): `.subsim` archives read/written both ways with Python, startup
  recovery, derived-model cache, XSD verdict cache with a client for `studio_xsd.exe`.
- `internal/api/projectapi`: project, IED, SCL and preferences routes as `api.Route`.
- `studio/native/xsd`: C sources of the pinned libxml2 XSD helper (not built).
- Test data marked `-text` in five `testdata/.gitattributes` so line endings never change bytes.

## Evidence

- All against the frozen Python app, byte for byte: `parse_document` 3403/3403, `utf-8-sig` errors
  326/326, SCL corpus 1170 golden files (295 cases), template bytes identical, exports 2407/2407,
  project vectors 1629/1629, `.subsim` round trip Python->Go->Python same snapshot, 68/68 Flask
  requests identical in status, headers (except Content-Length) and body, XSD client 21/21.
- Open/save on the same 11-IED project: open Python 86.6 ms, Go 49.6 ms (22.5 ms from the model
  cache); save median Python 7.9 ms, Go 1.9 ms; small project open 45.7 vs 21.6 ms.
- Integrator on `master`: only owned paths; corpus inputs are public (`scd/`, demo station,
  template, unit-test samples, generated SCL); scan for the IED names and file names of the 20
  private `test_cids` models: no match; `go vet ./...` clean; all scl, project, store and
  projectapi packages pass with `go test -p 2`.
- Integrator fix: `TestFlaskParity` failed 1 in ~40 runs (68/68 answers identical, but the temp
  folder removal hit Windows delete-pending files). The patient cleanup moved from `autosave` to
  `internal/testutil.TempDir` and is used by `projectapi` and `project/service` tests; 40/40 runs
  pass. `verify.py`: see ledger line.

## Limitations

- `studio_xsd.exe` not built (needs unmodified libxml2 2.11.9 + static libiconv and zlib, see
  `studio/native/xsd/README.md`); until then Go serves only the 559 recorded verdicts and answers
  RuntimeError on a miss.
- Known differences: internal DTD ATTLIST defaults not applied; encodings beyond 27 tabled ones
  "unknown encoding"; huge report integers clamped; downloads ignore `Range` (UI never sends it);
  ASCII download-name fallback for Latin-1 letters only; some rare `selected_names` JSON error
  wordings; corrupt deflate/bzip2 answers "Invalid project archive"; lone surrogates become U+FFFD;
  snapshots not fsynced (journal lines are).
- Wiring left for the server: error mapping from `projectapi/respond.go` into `httpapi`,
  `projectapi.Deps`, startup gate, 100 MB limit, `autosave.Fold` before Python opens a data folder,
  `RecoverStartup`, runtime hooks (`project_changed`, model cache).
