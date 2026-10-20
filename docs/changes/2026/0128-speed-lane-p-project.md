# 0128: Speed lane P merged: click-path save 227 -> 2.3 ms, project ports, GOOSE declarations, design routes

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 1, lane P. Branch
`speed-p` (229e8a9..c931174, 7 commits) merged into `master` (6c649ca). Stream log:
[speed-p](../../architecture/go-engine/streams/speed-p.md). Not wired into the Go server yet.

## Change

- Click path (`project/service`): signal/control lookups indexed, only the edited node re-validated
  (differential test over 3000 valid/invalid changes = full validation), SCL hashed once per file,
  diagram encoding cached per node, background snapshot at most once per second; every edit still
  journaled and fsynced before the call returns. Saved project format unchanged.
- Ports: the service satisfies `runtime.Project` (`AssignIP` without revision bump; written to the
  journal at once, Python saved it with the next save), `gateway/service.Projects`,
  `toolsapi.Projects` (`SaveGateway`, `SaveTestPanel` keep the old value on failure), `State()`.
- GOOSE declarations: `scl.ReadGooseSources`, `scl.SubscriptionMembers`,
  `project.GooseDeclarations.For` (cached), `StoredSources`, `RequireSource`, `SourceID`.
- Routes: `PUT /ieds/{key}/settings`, `PUT /scenarios`, `GET /project/substation-sources`,
  `POST /project/diagram/from-scl/preview`, `GET /signal-flow`, `/signal-flow/sources`,
  `/signal-flow/edges/{edge_id}.csv` (substation, signal flow, report clients, SLD layout and
  auto-layout, bindings ported with Python's float behaviour).
- Store tests use `testutil.TempDir`; `TestFoldBeforePython`.

## Evidence

- 6 IEDs, 90k points, 22 MB SCL, 300 bound switches: operate save p50 204 -> 1.2-1.6 ms, p95
  227 -> 2.3-2.8 ms (4.3 ms in one loaded run); benchmark p95 268 -> 2.6-3.0 ms. Copies taken
  between operates fold to exactly the in-memory project.
- Parity: GOOSE declarations 2581/2581; new routes 64/64 Flask (existing 68/68); SLD/flow domain
  goldens 2558/2558; a Go data folder with 7 journal records, folded, opens in Python's recovery
  as the same project.
- Integrator on `master`: only owned paths; private-data scan of 72 files: no match; `verify.py`:
  see ledger line.

## Limitations

- First operate after open/import ~24 ms (index build); each journal line carries the whole
  changed diagram between snapshots (~0.5 MB at 300 nodes). Journal JSON spacing differs from the
  older Go writer (only Go reads it; Fold accepts both). A preview body that is a non-empty list or
  number answers 400 instead of Python's 500 (UI never sends it).
