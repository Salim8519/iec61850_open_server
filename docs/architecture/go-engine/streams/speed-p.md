# Speed lane P — project (click-path save, ports, declarations, design routes)

Branch `speed-p` (rebased on master `62dbfb7`). Owner of `internal/project/**`, `internal/scl/**`,
`internal/store/**`, `internal/api/projectapi/**`. The frozen Python backend was only run
(goldens, cross-checks), never changed. Public inputs only: `scd/*.scd`, the demo station,
the SCL corpus, `studio/tests/unit/sld_fixtures.py` and generated SCL. No EXE, no owner data.
Laptop rules kept: `GOMAXPROCS=4`, `go test -p 2 -trimpath`, one heavy command at a time.

## 1. Click-path cost (speed plan S2): canvas operate save

`BenchmarkNodeDataLargeProject` / `TestNodeDataLatencyLargeProject` (`STUDIO_TIMING=1`):
6 IEDs x 15 000 signals (90k points), six distinct SCL texts of ~3.7 MB (22 MB), a diagram
of 300 bound switches; each operation is `Service.NodeData` (runtime switch position save,
journal line fsynced before returning).

| Measure | Before | After |
|---|---|---|
| NodeData p50 (200 operates) | 203.5 ms | 1.2-1.6 ms |
| NodeData p95 | 227.3 ms | 2.3-2.8 ms (one loaded run 4.3 ms) |
| Benchmark p95 (300-500 back-to-back) | 268.4 ms | 2.6-3.0 ms |
| Allocation per operate | (not measured) | 1.8 MB, ~510 allocs |
| First operate after open/import | — | ~24 ms (builds the ref index once) |

Target p95 <= 5 ms: met. Where the time went and what changed:

1. **Diagram re-validation scanned 15k signals per bound node** (190 ms). `project.Index`
   keeps one ref map per immutable engineering list (identity = first element + length,
   the entry pins the list); `ValidateDiagramWith(value, lookup)`. Same first-match answers
   as the scan (`TestIndexLookupMatchesLinear`).
2. **node_data re-ran the pydantic contract over every node** (7 ms for 300 nodes).
   `project.NodeDataDiagram` re-validates only the edited node (`contract.ValidateNodeAt`,
   same error locations) because the model dump is a fixed point of the contract; every
   cross-check still runs in Python's order, and when the IED list is the very list the
   diagram was validated against the unchanged nodes' binding checks are skipped. The
   service takes this path only for a diagram it validated itself. Differential test:
   3000 random valid/invalid changes, result and every error identical to a full
   validation, published objects never written (`TestNodeDataDiagramMatchesFullValidation`).
3. **Compact re-hashed 22 MB of SCL per save** (17 ms). `project.Compactor` hashes each text
   once (identity cache) and reuses the stored `ieds`/`scl_sources` values while the IED list
   is unchanged (`TestCompactorMatchesCompact`).
4. **The diagram was encoded four times per save** (checksum, two journal comparisons, the
   journal line). `archive.Encoder` memoises `json.dumps` bytes by identity for top-level
   values and two levels below (a canvas edit encodes one node); the journal compares and
   writes those bytes (`changeLine`), checks only changed values for non-finite floats
   (`TestMemoDumpsMatchesPyjson`). Journal lines now carry `json.dumps` spacing inside
   values (Go-only file; Fold decodes either).
5. **A background snapshot per click competed for the journal lock and the CPU.**
   `SnapshotWriter.SetInterval`; the service uses `SnapshotInterval = 1s`. Every edit is
   still durable in the journal before the call returns; Save/Flush/Close never wait.
   `TestNodeDataCrashFolds`: copies taken between operates (autosave + journal, no Close)
   fold to exactly the in-memory project (40/80/120 change records).

Files on disk are unchanged in format: the archive bytes are still `json.dumps(compact)`
(`TestPythonCrossCheck`, `TestPythonArchiveRoundTrip`, `TestFoldBeforePython` pass).

## 2. Ports for the composition (tasks 1-2)

`project/service` now satisfies, checked at compile time (`ports_iface_test.go`):

| Port | Methods |
|---|---|
| `runtime.Project` | `Current`, `NodeData`, `Source(iedID, ref, *pyjson.Object)`, `AssignIP` |
| `gateway/service.Projects` | `IEDs`, `Gateway()` (nil when not an object), `SaveGateway(*pyjson.Object)` |
| `toolsapi.Projects` | the above + `TestPanel() []any`, `SaveTestPanel([]any)` (stores the validated list) |

- `AssignIP(iedID, ip)`: copy-on-write IED, committed without revision bump or
  `project_changed` (Python mutated `ied["ip"]`, saved with the next save; Go journals at
  once). Unknown IED or same address: no-op; a failed save goes to `OnSnapshotError`.
- Old setters `Service.Gateway(config)` / `TestPanel(entries)` had no callers and were
  replaced by `SaveGateway` / `SaveTestPanel`; savers keep the previous value on failure
  (`TestSaversKeepPreviousOnFailure`, `TestAssignIPKeepsRevision`).
- `Service.State()` returns project and revision from one atomic read.

## 3. GOOSE declarations (task 4)

- `scl.ReadGooseSources([]scl.GooseDocument)` = `read_goose_sources`, in the JSON layout
  `goosedom.DeclarationsFromJSON` reads (`blocks` as `[[key], block]`). Ports
  `goose_inventory.block_inventory`, `goose_remote.remote_inputs`, `goose_vendor` (packed
  intAddr hints with Python's `$`-before-newline rule, vendor attributes, GE label).
- `scl.SubscriptionMembers(ied, subscription)` = `goose_inputs._with_members`
  (fits `acquisition.SubscriptionMembers`).
- `project.GooseDeclarations{}.For(project)` = `goose_links._declarations` (cached for the
  last set of files); `project.StoredSources` / `RequireSource` / `SourceID` (cached hash).
- Goldens: `scl/testdata/goose_sources_gen.py` -> `golden/goose_sources.json`, 776 inputs
  (every UTF-8 corpus case alone with all/first IED loaded, pairs of GOOSE cases,
  synthetic multi-file stations with remote, packed and copied IEDs) and 1805 member
  lookups: **2581 / 2581 identical**.

## 4. Route gaps (task 5)

New `projectapi` routes: `PUT /ieds/{key}/settings` (cycle lock, stopped), `PUT /scenarios`,
`GET /project/substation-sources`, `POST /project/diagram/from-scl/preview`,
`GET /signal-flow/sources`, `GET /signal-flow`, `GET /signal-flow/edges/{edge_id}.csv`.
Ports: `scl/substation.go` (`ReadSubstation`), `scl/flow_sources.go` (`ReadFlow`,
report clients), `project/sld` (layout, automatic layout, wires/junctions, bindings with
CPython float semantics: half-even round, float floor division, 3.12 compensated `sum`),
`project/flow` (graph, lookaround-free word match), `project/sld_import.go`,
`project/signal_flow.go` (CSV as `csv.writer(lineterminator="\n")`).

| Check | Result |
|---|---|
| Flask recording `projectapi/testdata/http_gen_routes.py` (demo station + open_substation: previews replace/beside/bogus, apply, signal-flow graph/filters/CSV, IED settings incl. conflicts, scenarios incl. 415/400) | **64 / 64** status, headers, body |
| Existing Flask recording `http_gen.py` | 68 / 68 (unchanged) |
| Domain goldens `scl/testdata/sld_flow_gen.py` (414 inputs: corpus, 120 seeded random stations, fixture feeders, open_substation variants: topology, diagram, bindings, graphs, CSV) | **2558 / 2558** |

Test-only: `projectapi/setup_test.go` adapts the Werkzeug segment `{edge_id}.csv` to
`http.ServeMux`; `httpapi`'s router matches it natively.
Deviation: a preview body that is a non-empty list or a number answers the ValueError text
(Python raised TypeError, a 500); the UI never sends one.

## 5. Fold before Python (task 6) — integrator notes

- **Before Python opens a data folder**, call `autosave.Fold(dataDir, nil)` (or the
  engine's encoder) once, with no Go service running on that folder. It returns the number
  of records applied, removes the journal only after the new `autosave.subsim` is renamed
  in, keeps both files and returns `ErrJournalMismatch` when the autosave is not the
  journal's base or a marked snapshot (another program saved it): then do not start
  Python silently, report it.
- The Go service folds itself on start (`RecoverStartup`) and folds on `Close`; a crash
  leaves a journal, which is why the launcher must fold before choosing Python.
- `TestFoldBeforePython` (needs `STUDIO_PYTHON`): a Go folder with a leftover journal
  (diagram, operates, AssignIP, gateway; 7 records) folds and the frozen Python
  `recover_startup` restores exactly the Go project snapshot. Pass.

## Evidence summary

`go vet ./...` clean; `go test -p 2 -trimpath ./...` green except `internal/launcher`
`TestLeaseSharedWithPython` in a worktree without `studio/.venv` (the test skips while
holding the lease, so its TempDir cleanup fails; green with `STUDIO_PYTHON` set; not this
lane's package). Python cross-checks run with `STUDIO_PYTHON`: crosscheck, archive round
trip, fold before Python: pass. `check_file_lengths.py`: PASS. Store tests now use
`testutil.TempDir` (integrator request).

## Open items

- First operate after an open/import builds the ref index (~24 ms at 90k points); a
  background warm-up after an IED-list change would hide it.
- Journal lines carry the whole changed top-level value (a 300-node diagram is ~0.5 MB per
  operate between snapshots); node-level journal records would shrink them further.
- `GooseDeclarations` is per process; the integrator should keep one instance.

## Needs from other lanes

- **I (integrator):** wire `liveapi.Service.Declarations` as
  `func(p) { v, err := decl.For(p); if err != nil { return goosedom.Declarations{}, err }; return goosedom.DeclarationsFromJSON(v), nil }`;
  `acquisition.Deps.Members = scl.SubscriptionMembers`; pass the service itself
  as `runtime.Project`, `gateway/service.Projects` and `toolsapi.Projects`; call
  `autosave.Fold` in the launcher before Python; serve the new projectapi routes through
  `httpapi` (they use `{edge_id}.csv`).
- **L/R/T:** none.
