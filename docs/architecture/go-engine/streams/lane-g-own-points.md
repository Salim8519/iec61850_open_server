# Lane G: gateway own points (made-up points for simulation)

Branch `worktree-agent-aa44ecb9b75738463` from master 91c4fbb. Owner request: keep the automatic
"Propose from IEDs" and let the user add any made-up point for simulation. Go engine and React
frontend only; the frozen Python backend (`studio/backend`) is unchanged. No EXE was built.
The integrator owns the change record, the ledger entry, the indexes and the ADR number.

## What

An **own point** is a gateway point with `"source": "manual"`: no IED, no reference, a `value`
that ScadaBench serves with good quality (quality 0, `source_ok` true) while the gateway runs.

| Part | Behaviour |
|---|---|
| Stored point | `{kind, source:"manual", ied:"", ref:"", address, scale, description, value, [target], id}`; id `own:<n>` (IED ids always end in `:<kind>`, so they never clash). A well-formed id keeps its number; a missing or repeated one gets the next free number. |
| Values per type | single 0/1 (true/false accepted), double 0-3 (true=2 closed, false=1 open), float any finite number, integer whole number in int32; commands store the last command (single 0/1, double 1 open / 2 close; none until the first command). |
| Validation (`gateway.ValidatePoints`) | Same type, address range, scale and address-clash rules as IED points; the IED lookup is skipped. Bad value: "Point N (own point Name): the value must be ...". A command's `target` must be an own single/double point of the list, otherwise a problem and the target is dropped. IED points are validated exactly as before (Python oracle unchanged). |
| Save / load | The project normaliser keeps stored points as they are, so `source`, `value`, `target`, `id` survive save, export and reopen (`TestOwnPointsSurviveSaveAndReopen`). |
| CSV | Extra columns `source,value,id,target` are written **only** when the list has own points; an IED-only list exports byte-identically to Python. Old six-column files import unchanged. A row with `source` = `manual` (or `own`) is an own point; value accepts numbers and on/off/true/false/open/closed. |
| Serving | `ownStore` holds the served values (reset at Start from the saved list, cleared at Stop). The syncer reads own points from it instead of the IED hub; a change marks only that point, so it leaves in the next 100 ms batch as a spontaneous / event change, like an IED value. |
| Live change | `PUT /api/v1/tools/gateway/points/value {point_id, value}`: checked by type, sent to SCADA at once when running, and saved in the point list (works while stopped too, and survives a restart). Command points refuse a typed value. |
| New point | `POST /api/v1/tools/gateway/points/new {protocol, kind, points}` answers the next own id and the first free address of the type's block (the same block starts as Propose). |
| Propose | `POST /tools/gateway/propose` takes an optional `keep` list: own points stay after the proposal; one whose address a proposed point now uses moves to the next free address, with a note. Without `keep` the answer is unchanged. |
| Commands | A command (SCADA or the page's Send buttons) on an own command point always succeeds: no device is operated, the value is stored, and the own status point it drives (`target`) follows (single ON=1/OFF=0, double close=2/open=1), which SCADA sees as a spontaneous change. Commands tab: applied "done: own point, no device is operated", observed "<status> is now closed (ON)". The real-IED commands switch does not apply (it is not a real device). A Send on an own monitoring point is refused "this point is not a command". |
| Point table | Own rows: `ied_name` "Own point", `ied_role` "own", `monitor_only` false, `value_text` like IED rows (status words, value x scale), commands "Last command: close" / "No command yet". |

Frontend: **Add point** beside Propose from IEDs opens a small inline form (name, type in plain
words, address pre-filled by the backend, starting value as OFF/ON switch, position list or
number, scale for measured values, "Changes this status" for commands) with a hint "An own point
is a made-up value you control. SCADA sees it like any other point ...". Own rows show "Own
point" with "you set the value" / "drives <name>", and a value control that stays usable while
the gateway runs (OFF/ON switch, position list, number box sent on Enter or leaving the box).
A filter chip **Own points**; the IED dropdown also lists "Own point". Propose keeps own points.

## Evidence (2026-10-20, this worktree)

- `GOMAXPROCS=4 go test -trimpath -p 2` with `STUDIO_PYTHON` (main checkout venv) and
  `STUDIO_NATIVE_BIN` (main checkout `native/build/Release`): `internal/gateway` (incl.
  `TestRulesAgainstPython`, `TestCSVAgainstPython`, 5 new own-point tests),
  `internal/gateway/service` (incl. the existing IEC 104 wire test and the new
  `TestOwnPointsOverIEC104`), `internal/api/toolsapi` (incl. `TestRoutesAgainstFlask` and the new
  `TestOwnPointRoutes`), `internal/app` (incl. `TestEveryPythonRouteServed` with the two new
  Go-only routes listed in `toolsapi.OwnPointRoutes`), `internal/project/...`: all PASS. The
  `internal/gateway` binary had to be built with `go test -c` and run from the scratch folder
  because Windows Defender refused the fresh test executable ("Access is denied").
- `TestOwnPointsOverIEC104`: real `studio_gateway.exe` with an independent lib60870 master
  (`studio_test_iec104_master.exe`, 127.0.0.1, port from `testutil.FreePort` >= 20000): general
  interrogation reads the own single, double and float next to an IED measurement; a value typed
  through `SetOwnValue` arrives spontaneously (COT 3) as 99.5; `dc:50002:2` select + execute on an
  own double command is confirmed and moves the own breaker to 2 (COT 3); Commands tab and the
  saved values are checked.
- `python studio/tools/verify.py`: file lengths, architecture, documentation, history,
  specifications and GOOSE engine PASS; the Go suite passed in every package except
  `internal/gateway`, whose fresh test binary Windows Defender refused to start ("Access is
  denied", the known antivirus issue); that package passes in full when built with `go test -c`
  and run from the scratch folder (above).
- Frontend: `npm run check` (tsc) clean; `node --experimental-strip-types --test
  studio/tests/frontend/*.test.ts`: 235/235 including the new `ownPoints.test.ts`;
  `check_file_lengths.py` PASS.
- Browser check at 1366x768 on a separate Go server (port 28765, temporary data folder, gateway
  port 22404; the owner's app on 8765/5173 was not touched): Propose, then Add point for a
  status, a position, a command driving the position and a measurement; Save; the Own points
  chip; Start; OFF/ON switch while running; a 104 master on 22404 read the own points with
  quality 0 (IED points 192 = invalid, IEDs not running) and received the ON change as
  M_SP_TB COT 3; the page's Close command set the position to "Closed (ON)" and logged it; after
  a server restart the values were still there; CSV export carried the four extra columns.

## Differences from Python

Python has no own points (expected). Same project file in the Python engine: the points stay
stored, but its Start refuses them ("its IED is not in the project") and serves the rest; its
CSV import reports own rows as "no IED named ''"; the two new routes answer 404 there. IED points,
their CSV, Propose without `keep`, and every Flask route answer exactly as before (parity tests).

## Limitations / open points

- Quality is always good; there is no way yet to make an own point invalid or questionable
  (could be a later "quality" choice). Time tag: the change time (0 at Start, so the native
  gateway stamps the send time).
- A command on an own point is always accepted (no select-before-operate model, no refusal
  simulation). A command drives at most one own status point.
- The Add point address suggestion skips used cells but a later Propose may still move an own
  point (with a note), never an IED point.
- A hand-edited project with an own point without `id` gets `::<kind>` from the frozen project
  normaliser and a new `own:<n>` at validation; the next Save point list fixes it.
- Contract change (two Go-only routes, new point fields) needs an ADR number from the integrator
  (proposed text: "Gateway own points: Go-only extension, backwards compatible stored data and
  CSV; Python engine ignores them").
- The point table still scrolls sideways at 1366x768 as before; own rows add no new clipped text.
