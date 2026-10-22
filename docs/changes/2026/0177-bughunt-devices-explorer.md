# 0177: Bug hunt group 2: Devices, Test, IED Explorer and the app shell

Date: 2026-10-23. Source: bug hunt of 2026-10-23 (tester A, verifier A; B5 from verifier B), fix plan group 2.
Rule for every fix: the smallest change the user notices only because it now works. No new prompts or steps,
no change to stored data or API shapes, no change to any text the Python engine shares with Go.

## A1 (high): IED Explorer selected a control just by reading it

**Problem.** Opening a data object (and "Read again", and a re-read after Operate) read every attribute,
command fields (FC CO) included. For a control with "select before operate, normal security" (ctlModel 2),
IEC 61850-8-1 maps Select to an MMS read of `<DO>.SBO`, so the read reserved the switch for Explorer
(reproduced by the verifier: `control_selection ... selected` at the read instant, `timed_out` 3 s later). A
watched SBO would have selected the control on every poll.

**Fix.**
- Go (authoritative, covers every client and the poller): `tools/explorer/sbo_guard.go`. `Service.Read` never
  sends an attribute with FC CO whose last name is `SBO` (dot or `$` form, any case); it answers it with
  `{raw: null, text: "Not read: reading SBO would select the control", status: "warning"}`. When only SBO
  attributes are asked, nothing is sent and the read counter does not move. `SetWatch` keeps such a row (the
  stored watch list keeps its shape) and gives it that value at once; the poller goes through `Read`, so it is
  never polled. SBOw, Oper and Cancel reads have no side effect and are unchanged.
- Frontend (`features/tools/explorer/readSafety.ts`): opening a data object reads no command field at all
  (they only hold what the last command sent); CO rows say "Command field, not read" with a hint, the SBO row
  shows the warning text and its + button is off with the reason "Not added: reading SBO would select the
  control"; `ExplorerTool.addWatch` drops SBO items too. The command panel still gets ctlModel, sboTimeout and
  limits (CF), so Select / Operate are unchanged.

**Python engine.** Frozen and not changed. This is a protective divergence of the Go engine only: Python
still reads SBO when asked. The Go-vs-Python domain oracle (`TestDomainAgainstPython`) covers node views, key
points and value texts, which are unchanged, and passes.

**Evidence.**
- `go test ./internal/tools/explorer/` (with native workers): new `TestReadNeverSendsSBO`,
  `TestWatchShowsWhySBOIsNotPolled`, `TestSelectsOnReadOnlyForCOSBO` plus the existing wire test
  `TestExplorerWatchesReadsAndOperatesARealServer` and the Python oracle: all pass.
- API on the Go server (port 8831, `SampleTyped.scd`): `POST /tools/explorer/read` with all 28 attributes of
  `GGIO2.AnOut1` (what the UI used to send) answered 28 values, SBO = the "Not read" entry; the event stream
  over the same period had 0 `control_selection` events.
- Browser 1366x768: Tools › IED Explorer › GGIO2 › AnOut1 opened and read again: CF/MX values shown
  (ctlModel 2, sboTimeout 3000), SBO row "Check: Not read: reading SBO would select the control" with its +
  disabled, 19 Oper/SBOw/Cancel rows "Command field, not read"; 0 `control_selection` events in the 25 s capture.
  The typed Select through the command panel still selected (see A4).

## A3: Activity flooded by "report created"

Every report a simulated IED builds sends `report_control` with native kind 9 (`RCB_EVENT_REPORT_CREATED`);
TR1/TR2 alone filled the 500-row Activity ring. Frontend only, engine events unchanged:
- `state/eventLogStore.ts` gets an optional `fold` key: events with the same key in the same minute become one
  row with a count (the per-kind counter still counts each event).
- `features/event-log/reportControl.ts`: the fold key (IED + block, kind 9 only) and plain details for every
  RCB kind ("enabled by 127.0.0.1:5000", "TrgOps changed", "buffer overflowed (oldest reports lost)").
- `features/event-log/iedNames.ts`: Activity names the IED ("TR1 ReportControl_00101 / 26 reports created");
  the project read fills the id→name map.
Evidence: browser, 6 running demo IEDs: one row per block per minute (8 rows/min) instead of ~240 rows/min;
`bugHunt0177.test.ts` folds 600 events into 2 rows.

## A2: "Add a sample IED" reused a taken name

`features/devices/devicePick.ts` `freeSampleName`: the first SampleN no IED uses (case-insensitive).
Evidence: with Sample8 present among 7 IEDs (old rule would ask for Sample8, which the engine refuses with
"An IED with this name already exists"), the button added Sample1, no error.

## A6: clipped text at 1366x768

- Import button (Devices list): when the list's scrollbar appears, the text needs 168 px and has 134. It now
  wraps as "Import" over "CID / SCD" (12 px sides; the button text is unchanged for scripts and the tour).
- Data points search: min 190 px, placeholder fits (needs 122 px, has 227). Same check found the IED Explorer
  data object search clipped ("…attributer"): it now has a 270 px minimum (needs 201, has 227), bar stays one row.
- Value lists (`.signal-select`): `min-width: max-content`; with the SP filter the selects are 110 px for
  "False (false)" (was 90 px, "False (fa").
Measured in the browser at 1366x768 with 6 IEDs.

## A5: menu stayed open with a TypeError on resize

`ui/menuAway.ts`: a resize always closes the menu; a scroll checks only real page elements. Browser: Help menu
open, `resize` dispatched, menu closed, no page error.

## A4: two contradicting messages after a selection ran out

`CommandPanel` clears the old "Selected" answer when the countdown ends. Browser (AnOut1, sboTimeout 3 s):
during the selection "Selected "1". Operate within 2 s"; after it, only "The selection ran out before Operate.
Press Select again."

## A7: grey buttons while running did not say why

Titles from `copy/devices.ts`: "Stop the run to import IEDs", "Stop the run to add a sample IED", and on
Simulate / Monitor real device / Skip "Stop the run to change what ScadaBench does with this device"
(also on the empty-page buttons). Verified in the DOM while running.

## A8: generator saved a bad cycle time

`features/test/sourceCheck.ts`: under 0.1 s (or empty) shows one red line "Cycle time must be at least 0.1
seconds." under the fields, the field is `aria-invalid`, Save is off with the same reason. The engine text
("Source period must be at least 0.1 seconds") is unchanged (parity). Browser: −5 and 0 → line shown, Save off,
inputs stay aligned.

## A10: detail showed an IED the filter hides

`shownSelection`: when the chosen IED is hidden by the list filter, the first shown IED is chosen (unless it was
asked for from another place, for example "Signal flow" from the SLD). Browser: "Has problems" on, reload →
FEED1 selected and shown, not the hidden SampleTyped. Selection and tab are still not remembered (see below).

## A12 and A9: wording

- Import dialog "1 IED found" (`countOf`).
- Overview "Address in the SCD **not in the file**" instead of "None".
- Find IEDs on This PC (127.x) with no answer: "No simulated IED answered. Press Run so the simulated IEDs
  answer on 127.50.0.x, then scan again." Other ranges keep the cable advice.
- A9: "Missing DataTypeTemplates" is shown as "This file has no data type section (DataTypeTemplates), so
  ScadaBench cannot build the IED model. Export the SCD or CID again from your engineering tool with the data
  types included." (frontend mapping in `copy/devices.ts`; the engine text and golden files are unchanged).
  Browser: an SCD without DataTypeTemplates showed the new text in the banner and the dialog.

## B5: "Engine connected" while the engine was gone

`api/connectionHealth.ts`: every API answer marks the page connected, a failed request (or 401: the session is
gone) marks it lost; no extra requests, the page's own polls are the evidence. `app/EngineStatus.tsx`: the
sidebar shows "Connection lost, values may be old" (red dot plus words; the hint stays in the title when the
sidebar is narrow) and one line over the page says the same with what to do. Browser: engine stopped → within
seconds the sidebar and the banner switched; the longer text wraps in the sidebar (not clipped).

## Tests

- `studio/tests/frontend/bugHunt0177.test.ts` (10 tests, A1–A12, B5): every helper above plus source checks.
  All frontend tests: 437 pass. `npx tsc --noEmit`: clean.
- Go: `go vet` and `go test ./internal/tools/explorer/` with `STUDIO_NATIVE_BIN`: pass.
- `python studio/tools/verify.py`: file lengths, architecture, documentation, history, specifications, GOOSE
  engine and Go engine (pyjson vs CPython) all PASS.

## Limitations and not fixed

- A1: Python engine (frozen) still reads SBO if a client asks; the frontend no longer asks. An SBO already in a
  saved watch list shows the "Not read" row; it is not removed, so the stored list keeps its shape.
- A3: the folded row keeps the time of its first report in that minute; the count grows in place.
- A10: the selected IED and tab are still not remembered across reload (only the list filter is); the page now
  only makes sure the shown detail matches the list.
- B5: no grey style on old values; the banner and sidebar say it. While the window is hidden no polls run, so
  the state updates when it is shown again.
- Not fixed (fix plan): A11 and A13 (beginner polish with no rule broken; experts need raw values and CO
  internals), A14 (no invented value; a file-check note would be a separate change), A15 (harness artifact;
  WinError text is Python parity), A16 (feature request: New project / Save as).
