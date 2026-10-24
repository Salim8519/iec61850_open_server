# 0190: NR/NARI access points and SCD deadbands

Date: 2026-10-24. Owner decisions in the future-work review: NR/NARI "S1 exact + auto-pick";
deadbands off by default with a per-IED switch, "Follow the SCD" for SCADA writes, no rangeC means
no deadband with a note; "be careful, test yourself, make sure the app almost stays unaffected".
Decisions: [ADR-0117](../../decisions/ADR-0117-station-access-point.md),
[ADR-0118](../../decisions/ADR-0118-scd-deadbands.md).

## Change

| File | Why |
| --- | --- |
| `engine/internal/scl/access_points.go` (new) | `stationOf`, `stationDevices`, preview note |
| `scl/importer.go`, `preview.go`, `communication.go`, `data_classes.go`, `controls_extended_*.go`, `flow_*.go`, `goose_*.go`, `substation.go`, `import_ied.go` | read the station server only |
| `scl/deadband_kinds.go` (new), `controls_extended_doc.go`, `controls_extended_read.go` | db/zeroDb references a client may write (valKind), from the same single SCL read |
| `scl/python_twin.go` | `AsPythonTwin` for the project golden test |
| `engine/internal/runtime/deadbands.go`, `deadband_switch.go` (new); `actor.go`, `sample_writer.go`, `typed_checks.go`, `worker_events.go` | switch flag, generator filter, `writable_deadbands` start key, `deadband_written` event |
| `engine/internal/deadband/` (new), `api/projectapi/deadbands_route.go` (new), `api.go`, `data_classes_route.go` | `GET/PUT /ieds/{id}/deadbands` |
| `engine/internal/project/document.go`, `restore.go` | `scd_deadbands` kept only while true |
| `native/worker/server/deadband_writes.cpp` (new), `server.cpp`, `server.hpp`, `native/CMakeLists.txt` | write handlers for the listed db/zeroDb only |
| `frontend/src/features/devices/DeadbandCard.tsx`, `useDeadbands.ts` (new), `DeviceOverview.tsx`, `DeviceSignals.tsx`, `SignalTable.tsx`; `features/ieds/ImportSelection.tsx`; `copy/hints.ts` | switch, per-measurement list, row tag, preview note |

Tests: `scl/station_deadbands_test.go`, `runtime/deadbands_test.go`, `deadband/points_test.go`;
`project/sld_flow_golden_test.go` now runs as the Python twin (an IED with two servers keeps every
LD there, as Python did).

## Evidence

- **NR/NARI, corpus.** 182 local SCL files, preview/import answers before and after: only the four
  files with NR/NARI IEDs differ, and in them only those IEDs (TR1M2, BMR_705, two NARI templates);
  the other 28 IEDs of the mixed NRH station are identical.
- **NR/NARI, wire (independent mmscli).** TR1M2 (NR PCS-9705S) and BMR_705 (FBMR-E02): exactly the
  six S1 LDs on the wire (no PIGO/PISV), reads, report enable and reports, a command, GOOSE gocb0 of
  S1; the preview note names G1 and M1.
- **Deadbands, wire (independent mmscli, C14 BCU with db=100000 and rangeC).** Generator on
  `GGIO4.AnIn09.mag.f`: switch off, 12 distinct mag values in 12 reads; switch on while running, 1;
  SCADA writes db=0 (valKind unmarked): accepted, read back 0, 10 distinct values in 10 reads with
  the switch on; db=200000 refused (object value invalid); `rangeC.min.f` refused (access denied).
  Switch off removes `scd_deadbands` from the project. NR TR2M1 has no rangeC: 115 measurements,
  none applied, each with the "no measuring range" note.
- **App unaffected (20-IED harness of change 0189, switch off).** Master vs this branch: reports/s
  76.0/76.0, GOOSE frames 2041/2037, server peak 762/762 MB, workers 507/507 MB, Run start 2556/2545
  ms, SLD command median 57/51 ms. Worker CPU in the activity window varies run to run on the same
  build (master 3.9 and 7.4 s; this branch 5.9, 8.1, and 7.8 s with master's native worker).
- `go test` of deadband, scl, api, project and runtime pass; `verify.py` pass; protocol suite with
  the new native worker: 90 passed, 9 skipped; frontend `tsc` clean; browser at 1366x768: switch,
  counts, per-measurement list, Data points tag.

## Limitations

- An IED whose several servers cannot be told apart (no unique IP, no unique reports) stays refused.
- The db list shows the SCD values; a db written by a SCADA during a Run is used and readable over
  MMS and in Data points, but the overview list keeps the SCD number.
- Re-importing an IED resets the switch to off (the safe default).
- The packaged EXE was not built (not requested).
