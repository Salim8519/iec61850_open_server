# 0165: SLD point panel

Date: 2026-10-22. Owner request: replace hundreds of push buttons on one bay with one small
component that opens a searchable list of the bay's points and sets or operates each in one step.
Decision: [ADR-0102](../../decisions/ADR-0102-sld-point-panel.md). The design history
([ADR-0090](../../decisions/ADR-0090-sld-measurement-list.md), [ADR-0095](../../decisions/ADR-0095-universal-point-kinds.md),
layout.md, interactions.md) was read first, and its rules are kept.

## Change

- New diagram kind `POINT_PANEL` (Add > Protection & control > Point panel): one fixed-width
  card per bay (name, IED, counts per type, pinned values, "N not healthy" badge); click or
  Enter opens the popup; the sheet edits the name and the IED. One IED and at most 300 points
  per panel.
- Popup: filter chips All / Status / Double / Commands / Values with counts, search (name, group,
  reference), points grouped and scrollable, a row per type (ON/OFF buttons, open/closed/
  intermediate/bad buttons, a number field with unit and scale, a command with a one-step
  confirmation), "All healthy (N)". While stopped the same rows edit: rename, group, pin a value
  to the card, healthy value, scale, offset, unit, decimals, remove; "Add points" picks signals
  and controls of the IED by type with search, several ticks and "Add N points".
- Go engine, all new code in new files: `project/contract/point_panel.go`,
  `project/validation_point_panel.go`, `acquisition/watched_point_panel.go`,
  `runtime/point_panel.go`, `point_panel_write.go`, `point_panel_healthy.go`.
- Existing Go files only got an addition: the panel kind accepted (`contract/diagram.go`), two
  calls (`project/validation.go`), IED removal (`project/removal.go`), one watched-point line
  (`acquisition/watched.go`), one live-row case (`runtime/plant_states.go`) and one dispatch case
  (`runtime/test_buttons.go`).
- Requests reuse `POST /equipment/{node}/test` (`action` `set`, `send`, `healthy`), so the route
  table and its parity test are unchanged. See ADR-0102 for the shapes.
- Frontend (new files): `equipment/pointPanel.ts`, `pointDisplay.ts`, `pointFilter.ts`,
  `pointRequests.ts`, `pointPanelContext.ts`, `PointPanelCard.tsx`, `point-panel.css`,
  `copy/pointPanel.ts`, `features/diagram/points/*` (dialog, rows, adder, sheet, IED select).
  Shared look: `KindBadge`/`kindRow` (SP, DP, MV, SC, DC), `FilterBar`, `EditableText`,
  `SegmentedControl`. All text is plain English with hints.
- Stopped rows stay one compact line; a settings button on each row opens its group, healthy
  value, value settings, pin and remove fields (owner feedback: rows were too tall to scan).

## Evidence

- Go: new tests in `project` (contract, cross-checks, bytes of documents without a panel, IED
  removal), `runtime` (live rows, set, refusals, healthy batch, send, real device refused),
  `acquisition` (watched points) and `api/runtimeapi` (the real route, HTTP errors).
- Frontend: `tests/frontend/pointPanel.test.ts` (12 tests); frontend 308/308 (296 before);
  `tsc --noEmit` clean.
- Browser (dev app, Go engine, scratch data folder, sample IED, 39 to 42 points) at 1366x768:
  running: card counts, pinned value "11.0 kV", "3 not healthy"; popup ON/OFF, double buttons,
  value Enter, bad input message, command confirm and position change, All healthy ("6 points set
  to healthy", 0 unhealthy); stopped: "Run the IEDs to change points" and edit mode; Add points by
  type, search, tick all, Add N, rename, un-pin, remove (saved revision checked through the
  API). 0 clipped labels in the popup, card and sheet.
- `python studio/tools/verify.py`: file lengths, architecture, documentation, history, specs and
  GOOSE engine PASS; the Go engine section passes every package except `internal/gateway`, whose
  test program Windows blocks on this PC ("Access is denied"; the package was not changed).
  `CGO_ENABLED=0 go test -trimpath -p 2 ./...` shows the same single failure.

## Limitations

- Status, double and value points only (BOOLEAN, Dbpos, numeric MX); no integer-status points.
- Commands to a real device are refused, as Operate does; a panel never changes a real IED.
- No IED marks on the diagram for panel points (the signal-layer files were out of scope).
- The frozen Python engine refuses a project that contains a point panel (as for ADR-0090);
  the unknown-kind message does not name `POINT_PANEL`.
- Owner visual acceptance is pending. The packaged EXE was not built (not requested).
