# 0174: SCADA master "My point list": Add points, expected vs received, one-click commands

Date: 2026-10-22. Decision [ADR-0110](../../decisions/ADR-0110-master-point-list.md); spec rules MASTER-14 and
MASTER-15 in [scada-master.md](../../specs/scada-master.md). Go engine only (the frozen Python engine answers the
new routes 404 and keeps its behaviour). Built on the merged base of 0165, 0166, 0168 and 0169.

## What changed

- **My point list** (Tools › SCADA gateway & master › SCADA master, first tab; the old Points tab is now
  *Everything received*). The master's own address-only list: name, address, type, scale/offset/unit, Modbus
  table/data type/word order/registers, command type/select/value. Never linked to an IED.
- **Add points** dialog built from the gateway's pieces: *One point*, *A range*, *Paste from Excel* (paste rows,
  *Download template (.xlsx)* with a Type drop-down and no IED column, *Upload filled template*, *Upload CSV*).
  Every row is checked by the engine first; bad rows show "⚠ Row n: reason"; Add stays off until all are right.
  More › Export CSV / Import CSV / Download Excel template.
- **Expected vs received** computed in Go: ✓ Received (value, value after scale with unit, quality, time),
  ✗ Missing, ⚠ Wrong type (with "The device sends a Double point (M_DP_NA_1) at IOA 20006, not a Single point."),
  … Waiting, ↗ Command; *+ Not in your list* rows with Add / Add all. Counts on the filter chips.
- **Modbus**: listed points add their own reads (merged neighbours, capped per request); manual ranges and Find
  registers unchanged; *What to read* lists the automatic reads.
- **Send** on a command point opens the existing command form straight at its REAL-device confirmation, with the
  saved type, address, select before operate and value; Repeat automatically and Address +1 stay.
- **Copy from gateway** (address, master type, name as the gateway shows it; `from` keeps the source address).
- **Check against IED** (test aid, off by default): ✓ / ✗ with a plain reason / – with why, from the gateway
  service's own mapping.

### Existing files changed

- `engine/internal/gateway/addpoints/catalog.go`: `Master` field per gateway kind (master direction, hidden from
  the gateway's JSON).
- `engine/internal/gateway/service/master_ask.go`: the Modbus poll reads `pollRanges` (user ranges, then the
  list's reads).
- `engine/internal/project/gateway/master.go`: keeps `point_list`/`check_ied` when stored (`master_points.go`).
- `engine/internal/api/toolsapi/routes.go`: registers the list routes.
- `engine/internal/app/routes_test.go`: allows `toolsapi.MasterGoOnlyRoutes`; this also fixes the base, where
  the 0169 cyclic routes failed `TestEveryPythonRouteServed` with `STUDIO_PYTHON` set.
- `frontend/.../master/MasterTool.tsx`: the *My point list* tab (opened first when the list has points).
  `MasterCommandForm.tsx`: optional `review` (open at the confirmation). `ModbusRanges.tsx`: shows the
  automatic reads. `add/TypeList.tsx`: generic with a `badge` function (gateway default unchanged);
  `add/addPlan.ts`: `splitTypes` generic.
- `tests/frontend/gatewayAdd.test.ts` (TypeList badge assertion), `toolLayout.test.ts` (new `master-list`
  filter name).
- `docs/specs/scada-master.md`, `docs/reference/API.md`, `FILE-MAP.md`, indexes.

## Evidence

- Go unit: `go test ./internal/gateway/masterpoints/` 10/10 (contract and spaces, range steps and clashes,
  from-received, paste with/without title and bad rows, CSV export → import and Excel template fill → upload
  round trips, compare states/reasons/counts/command, Modbus data type and DNP3 spaces, poll ranges merged and
  capped, copy from gateway in all three protocols, Check against IED reasons); `toolsapi`
  `TestMasterListRoutes`; `service` `TestModbusPollAddsTheListRanges`, `TestGatewayMappedReadsTheIEDBehindAGatewayPoint`;
  `project/gateway` `TestNormalizeMasterKeepsAStoredPointList`.
- `CGO_ENABLED=0 go test -trimpath ./...` with `GOTMPDIR` in the worktree and `STUDIO_PYTHON` set: 69 packages
  ok, 0 failed (includes `TestRoutesAgainstFlask` parity and `TestEveryPythonRouteServed`).
- Frontend: `node --test --experimental-strip-types tests/frontend/*.test.ts` 403/403 (5 new in
  `masterList.test.ts`); `npx tsc --noEmit` clean; `vite build` ok.
- End to end, two Go servers from this worktree (`--no-browser`, data in `.local/e2e-0174/a` and `b`, demo
  station `scadabench_demo.scd`, IED ports 2102+ and 3102+): A = station gateway with 374 proposed points on
  127.0.0.1:24174 (IEC 104), 25174 (Modbus), 26174 (DNP3) with its simulation running; B (port 8875) = SCADA
  master with its own simulation for the IED check.
  - IEC 104: Copy from gateway → 374 points (names "FEED1 PTOC1 Op", `from` set, no IED fields). One single
    removed, one double retyped as single, one made-up IOA 10999 added. Connect → counts received 358, missing 1
    ("Nothing received at IOA 10999 after the general interrogation."), wrong type 1, command 14, not in list 1
    (`single:10001`, from-received plan ok). Check against IED → 357 ✓ ("Both say Intermediate."), 1 ✗, 14 –.
    Send on FEED1 CSWI1 (C_DC 50001, SBO, value 2) → "select: confirmed; execute: confirmed; termination:
    activation terminated"; gateway A logged "operated: The device selected and operated it
    (sbo-with-enhanced-security)", observed "FEED1LD1/CSWI1.Pos.stVal is now closed (ON)"; B's list then shows
    M_DP 20001 "closed (ON)" and the IED check ✗ "IED says Intermediate, master got Closed: check the double
    point mapping." (B's own simulation was not operated: two separate simulations, as expected).
  - Modbus: copy → 374 (discrete 100, holding float32 220, int32 12, uint16 28, coils 14), no manual ranges; the
    list made 8 reads (float32 split 62/62/62/34); connect → 374 received, 0 missing; IED check 359 ✓, 1 ✗ (the
    breaker operated above), 14 –; coil Send → "write: the device confirmed the write".
  - DNP3: copy → 374 (binary 100, double-bit 28, analog 232, CROB close/trip 14) → 374 received; IED check 359 ✓, 1 ✗ (the breaker operated above), 14 –;
    CROB trip/close Send → "select and operate: accepted by the outstation"; the double-bit input then read
    "closed (ON)". (A first run found the double-bit inputs left out: they had been put in the binary index space;
    fixed, DNP3 numbers each group by itself, and pinned in the unit test.)
  - Browser at 1366x768 on B: screenshots in [0174-media](0174-media/) (`my-point-list.jpg` with the IED check
    column, `wrong-type.jpg`, `command-rows.jpg`, `send-confirm.jpg`, `send-done.jpg`, `add-dialog.jpg`,
    `paste-checked.jpg` with three bad rows in red with reasons). Clipped-text check (clip_check.js selectors plus
    the list's state, type, name and action cells): 1997 checked, 0 clipped; the table fits its panel (817 of
    816 px, sub-pixel).

## Limitations

- Our master, our outstation and the "IED check" all come from ScadaBench; no third-party SCADA or RTU was used.
  The two-simulation setup means the IED check compares with the master side's own simulation.
- The frozen Python engine drops `point_list`/`check_ied` if it saves the project, and has none of the routes.
- Paste/CSV text with `;` inside a name is split on `;` when the text has no tabs (Excel copies with tabs).
- Command points are not reported by IEC 104 devices, so they stay "Command" (DNP3 CROB/analog outputs show their
  output status when the device sends one). Modbus types are int16/uint16/int32/float32 (no uint32 list type).
- Names, scale and unit are edited in the list; type and address are changed by removing and adding the point.

## Merge onto 0173 (commands everywhere)

- Built on 0.1.21 plus the first integration patch; merged onto master after 0173. The gateway catalog of 0173
  has tap position, step command and set point kinds; each now has a master type, so Copy from gateway covers them:
  IEC 104 M_ST → step, C_RC → regulating, C_SE → set point float/scaled/normalised by the point's format;
  DNP3 tap → analog, CROB raise/lower → CROB close/trip, g41 → analog output float/int32/int16 by format; Modbus
  tap → input int16, set point → holding register float32/int16/uint16/int32 by format
  (`addpoints.MasterOfGatewayPoint`, `masterpoints/copy_typed_test.go`).
- One `MasterGoOnlyRoutes` (0174's list includes 0169's cyclic routes); the app route test keeps 0173's other
  Go-only lists.
- Evidence after the merge: go build, vet, full `go test ./...` with natives and Python parity, frontend 427/427,
  tsc, verify and a browser check of My point list on the merged build (below the ledger entry).
