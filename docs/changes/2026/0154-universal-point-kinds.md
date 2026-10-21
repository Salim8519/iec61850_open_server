# 0154: Universal point types, edit pin, one-line rows and full view

Date: 2026-10-21. Owner requests: one colour and type for every point in every page and tool
("double points apply to all"), more filters, an edit hint on changeable values, more vertical
space for points, aligned rows, and a full view where only the points show.
Decision: [ADR-0095](../../decisions/ADR-0095-universal-point-kinds.md). The design history
(ADR-0094, layout.md, changes 0097, 0151, 0153) was read first, and its rules are kept.

## Change

- Shared parts: `points/pointKind.ts` (one classifier for IEC 61850, IEC 104, DNP3, Modbus,
  gateway points and SNMP), `points/dataClasses.ts`, `ui/KindBadge.tsx`, `ui/kind.css`,
  `ui/kindFilter.ts`, `ui/EditableText.tsx`, `copy/pointKinds.ts`; FilterBar dropdown groups.
- Lists that use them: Devices data points, Compare with SCD, real-IED live values, Test
  points, the gateway (points, served, commands, traffic), the master and converter, SNMP
  simulate and read, IED Explorer tree and watch list, and the signal pickers.
- Full view: `ui/FullView.tsx`, `state/useFullView.ts`, `copy/fullView.ts`, `TabBar end`. In
  Devices the dark bar holds the name, facts and *Run this IED*, so the device header and role
  line are left out while in full view.
- One line per row: scoped `td.type-cell`, `is-empty` for the empty edit pin, and the gateway
  path shortened (no IED name, … at the end, full path on hover).
- Go engine (Go-only, ADR-0076 rules): `GET /api/v1/ieds/{ied_id}/data-classes`
  (`scl/data_classes.go`), and `point_kind`, `helpers=hide` and `kind_counts=1` on the signals
  query (`signalindex/point_kind.go`). Requests without them are answered as before.
- Spec `docs/specs/tool-layout.md`: filter table and LAYOUT-06 test names updated.

## Evidence

- Browser, dev app (Go engine), 1366x768, rows visible normal → full view:
  Devices data points 5 → 13, gateway points 5 → 10 (rows 86 px → 43 px), SNMP simulate
  7 → 12. 1920x1080: Devices 16 → 21, gateway 14 → 17, SNMP 17 → 20. Misaligned cells 0 and
  clipped labels 0 on every measured list.
- Real data (C03_Y1105, 36,021 points): SP 2878, DP 242, MV 1351, CT 18, AT 9190, UN 69.
- Frontend tests 296/296 (new: pointKind, deviceKinds, scadaKinds, toolKinds, fullView);
  `tsc` clean; Go tests pass for scl, signalindex, liveapi, projectapi and the route parity test.
- `python studio/tools/verify.py`: all sections PASS.

## Limitations

- Master, converter, SNMP read and IED Explorer had no points in the dev project, so their full
  view was checked for the button and layout only, not for rows.
- GOOSE blocks and Test points keep taller rows on purpose (few rows, details under each).
- UN stays for SR service-tracking attributes; no type is guessed.
- On this PC Windows blocks the test programs of `internal/gateway` and
  `internal/store/modelcache` ("Access is denied"); neither package was changed.
- Owner visual acceptance is pending. The packaged EXE was not built (not requested).
