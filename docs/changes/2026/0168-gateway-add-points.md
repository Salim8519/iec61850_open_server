# 0168: Gateway "Add points": from an IED, made up, pasted from Excel or an Excel template; smart Propose

Date: 2026-10-22. Decision [ADR-0105](../../decisions/ADR-0105-gateway-add-points.md); spec rules GW-12 to GW-14 in
[gateway.md](../../specs/gateway.md). Go engine only (the frozen Python engine answers the new routes 404).

## What changed

- **One Add points dialog** (Tools › Gateway › Points › *Add points*, the app's modal dialog): *From an IED*,
  *Simulated (made up)* and *Paste from Excel*. Every way ends in "Add N points"; the points join the list, save
  at once (ADR-0101) and the note offers Undo.
- **From an IED**: choose the IED (search, Simulated / Real device badge) → the type (one type list: "Most used",
  "More types", badge, plain name and protocol type ids such as `M_SP 1/30`, `g12v1`, `FC2`) → tick the IED's
  signals or commands that fit the type (engine-filtered, searchable, "usual" marks what Propose would pick, points
  in the list are shown as such) → names from the SCL description and next free addresses, both editable →
  "Also add its status point" (ticked) pairs each command with its status (IEC 104 20003 ↔ 50003) → a real device
  shows "This is a real device. SCADA commands will operate it." and needs "I understand". Commands are IED points:
  SCADA commands on them operate the IED through the existing `service/commands.go` path.
- **Simulated**: one point (the ADR-0091 form) or a range (N × type from address X, numbered names).
- **Paste from Excel**: paste rows with or without a title row; columns address / type / IED / point / name /
  scale are recognised (or guessed); bad rows show "⚠ Row n: reason"; Add stays off until every row is right.
  *Download template (.xlsx)* gives an engine-made workbook with Type, IED and Point drop-downs (Point lists only
  that IED's points of the chosen type through hidden named ranges); *Upload filled template* checks every row
  the same way. CSV import/export is unchanged.
- **Propose from IEDs** opens a preview: one tick per IED with "N points · M new", *Add only new points* (default
  when the list has points; keeps every name and address) or *Replace the IED points* (own points stay); names from
  the SCL and every command with its status at the paired address.
- Engine: new package `studio/engine/internal/gateway/addpoints` (catalog, candidates, plan, rows, own range,
  smart, xlsx); `gateway/addresses.go` exports the existing block/cell/free-address rules; routes in
  `api/toolsapi/add_points.go` and `smart_propose.go`. New dependency excelize v2.9.1 (BSD-3-Clause) and its
  imports; licence files in `studio/packaging/third_party/go-licenses/`, shipped by `go_release_steps.py`.

### Existing files changed

- `studio/engine/go.mod`, `go.sum`: excelize v2.9.1 and its modules.
- `api/toolsapi/routes.go`: registers the Add points routes. `own_points.go`: the Go-only route list names them;
  `points/new` takes `count`/`from`/`name` (a range). `points.go`: `propose` with `mode` goes to the smart preview;
  without it the answer is unchanged.
- `frontend/.../PointTable.tsx`: Add points dialog and the smart preview instead of the inline form and the direct
  Propose. `PointToolbar.tsx`: the button reads *Add points*. `AddOwnPoint.tsx`: `bare` (no own title row) inside
  the dialog.
- `frontend/src/copy/gateway.ts`: the empty-list words and the point list "?" say *Add points*.
- `studio/tests/frontend/ownPoints.test.ts`: the toolbar/propose shape check follows the new flow.
- `studio/tools/go_release_steps.py`: ships the Go module licences.
- `docs/specs/gateway.md` (GW-12..14), `docs/reference/API.md`, `FILE-MAP.md`, `docs/decisions/INDEX.md`.

## Evidence

- `go test ./internal/gateway/addpoints/` (CGO_ENABLED=0): 13/13 pass (candidates, plan/pairing, refusals,
  real IED + Modbus/DNP3 pairing, own range, type resolution, paste with/without title row, bad rows, against the
  list, smart replace, smart add-only-new, template drop-downs, template generate → fill → save → upload round trip).
- `go test ./internal/api/toolsapi/` new route tests `TestAddPointRoutes`, `TestExcelTemplateRoutes`,
  `TestSmartProposeRoute` pass with the existing `TestOwnPointRoutes`; Propose without `mode` keeps its old body.
- `go test -run Linked ./internal/gateway/service/` with `STUDIO_NATIVE_BIN` = the app's native build: 2/2 pass.
  `TestLinkedCommandPointOperatesTheIED` (a command planned from the IED → `Send` → `Operate` on
  `BayCTRL/XCBR1.Pos`, status observed "closed (ON)"); `TestLinkedCommandOverIEC104` (real `studio_gateway.exe`
  + the independent lib60870 master `studio_test_iec104_master`: C_DC on planned IOA 50001 confirmed, spontaneous
  M_DP on paired IOA 20001 = 2).
- Full `CGO_ENABLED=0 go test ./...`: 60 packages ok; `internal/gateway`, `store/modelcache`,
  `store/projectfile` blocked by the antivirus ("Access is denied" on the test binary, no test ran);
  `launcher` (needs STUDIO_PYTHON) and `licensing/hostkey` passed on a rerun; `netcfg` failed only on its live
  PowerShell timing test (PowerShell answered after 15 s, limit 20 s; unrelated to this change).
- Frontend: `node --test --experimental-strip-types studio/tests/frontend/*.test.ts` 378/378 (5 new in
  `gatewayAdd.test.ts`); `npx tsc --noEmit` clean, both before the last edit (dialogs switched to the app's native
  `<dialog>` as Devices › Delete).

- After the native `<dialog>` switch and the copy fix (empty-list and "?" hint now say Add points): tsc clean,
  frontend 378/378, `check_file_lengths.py` PASS. `verify.py` (GOTMPDIR inside the worktree so the antivirus lets the test binaries
  run): file lengths, architecture, documentation, history, specifications and the whole Go engine suite (now
  including `internal/gateway`) PASS; the GOOSE engine timing test `TestSendIsHandedOverWithoutWaitingForATimer`
  failed once under the parallel load (p50 476 µs) and passed on its own rerun (module not touched).
- End to end (dev server from this worktree on port 8797, Vite 5195, fresh data folder, demo station
  `scadabench_demo.scd` FEED1/FEED2/BUS1/BUS2/TR1/TR2, browser at 1366×768): Add points › From an IED › FEED1 ›
  Double command › ticked CSWI1 and CSWI4 → rows 50001 + status 20001, 50002 + status 20002 (pairing shown with
  "↳ status of the command above"), *Add 4 points* → "4 points added.", saved. Run, gateway on 127.0.0.1:24168,
  `studio_test_iec104_master gi dc:50001:2`: interrogation M_DP 20001/20002 = 0, C_DC 50001 ACT_CON + ACT_TERM
  (positive), spontaneous M_DP_TB 20001 = 2; Commands tab: "operated: The device selected and operated it
  (sbo-with-enhanced-security)", observed "FEED1LD1/CSWI1.Pos.stVal is now closed (ON)". On the same server:
  pasted rows answered "Row 2: FEED1 has no signal …", "Row 3: address 20001 is already used by another point"
  (ok false, nothing added); template download 200, 17 445 bytes `.xlsx`; smart `add_new` kept the 4 points and
  added 370 ("FEED1 62/58 new" etc.).

## Limitations

- Only the six existing gateway kinds; other protocol types (counters, set points, step positions …) are for a
  later change (one catalog line per protocol plus the worker).
- The template's Point drop-down needs Excel or LibreOffice with `INDIRECT` support; Google Sheets does not
  evaluate named-range `INDIRECT` drop-downs. Upload never replaces the list; it adds rows.
- Paste rows are not editable in the dialog: fix them in Excel and paste again. In "From an IED" a status point's
  address follows its command (change it in the list afterwards).
- The Python engine has none of this (404 / old Propose).
