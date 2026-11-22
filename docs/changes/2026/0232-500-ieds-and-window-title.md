# 0232 A project holds up to 500 IEDs; the window title names the project (0.1.36)

Date: 2026-11-22. Owner: "raise to 500 ieds allwoed, build instaeller, same veriosn 0136, dont
increase verison ... show project naemon windows header of windows. not the app insadie, but the
window header". Decision: [ADR-0144](../../decisions/ADR-0144-500-ieds-per-project.md).
Layers: Go engine (`internal/project`, `internal/project/service`, `internal/api/projectapi`,
`internal/scl`), frozen Python backend (same limit, for parity and golden vectors), frontend.

## What changed

- **Limit.** New `project/limits.go` (`MaxIEDs = 500` and the four messages) and
  `studio_app/contracts/limits.py` (`MAX_IEDS = 500`). Used by import (`service/scl.go`,
  `projects/service.py`), the import selection and `available_slots` (`scl_routes.go`,
  `api/scl.py`), open/restore (`document.go`, `stored_import.go`, `document.py`, `service.py`)
  and bulk delete (`bulk_removal.go`, `bulk_removal.py`, frontend `MAX_BULK`).
- **Caches.** Go `maxDigests` 64 → `2 * MaxIEDs`, `sourceCacheLimit` 64 → 1024; Python source
  digest `lru_cache` 20 → 1024.
- **Window title.** New `app/useWindowTitle.ts`, called from `App.tsx`: `<project> - ScadaBench`.
- **Golden vectors.** `cases_removal.py` / `cases_restore.py` send `MAX_IEDS + 1` items for the
  "too many" cases; `vectors_removal.json`, `vectors_restore.json` and `http_golden.json`
  regenerated from the Python backend (only the limit messages, the larger inputs and the
  per-run temp folder / zip time noise differ).

## Evidence

- `go test ./internal/project/... ./internal/api/projectapi/... ./internal/scl/...`: all pass,
  incl. `TestFlaskParity` 70/70 after the Python `available_slots` fix.
- Whole engine `go test ./...`: all pass except `internal/tools/control` and
  `internal/tools/explorer` ("MMS bind failed" on 127.67.0.41 / 127.65.0.32); the same tests fail
  identically with this change stashed, so they are an existing problem on this PC.
  `internal/api/runtimeapi` failed once in the parallel run and passes on its own.
- `verify.py` checks: file lengths, architecture, documentation, history, specs, offline UI and
  GOOSE engine pass; the Go step fails only on the two packages above.
- Python `pytest studio/tests/unit`: 554 passed, 7 failed, all native license-enforcement tests
  ("The license belongs to another computer"), unrelated to this change.
- Frontend `tsc --noEmit` clean.
- Release (owner asked, same version): `compile_go_release.py --version 0.1.36` with `studio\.venv`:
  exit 0, `ScadaBench-Setup-0.1.36.exe` 20.2 MB and `ScadaBench-Portable-0.1.36.exe` in
  `studio/dist-go`; antivirus gate 19 programs, 0 findings.

## Limitations

- No live test with a 500-IED project yet. The first real use is Al Kamil (28 bay controllers).
  Runtime cost of very large projects (workers, polls, diagram) is not measured here.
- Earlier releases refuse projects with more than 20 IEDs.
