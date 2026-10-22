# 0176 Bug hunt 2026-10-23, group 1: SLD and runtime

Fixes the confirmed SLD and runtime findings of the 2026-10-23 bug hunt (tester B, checked by the
verifier). Rule for every fix: the smallest change the user notices only because it now works. No
new prompts or required steps, no changed stored data or contract shapes, no change to text the
frozen Python engine shares (Go-vs-Python parity tests pass unchanged).

## Fixed

| id | problem | fix |
|----|---------|-----|
| B17 (high) | A command sent with Test=TRUE to an in-service breaker moved the simulated plant (1 → 2 → 1) and saved `initial_position`, although the IED ignores test commands (IEC 61850-7-4 Annex A). | `runtime/equipment_test_flag.go`: when the options that actually went out (the request's, or the held selection's after Select) carry Test, Go skips the plant feedback and the start-position save, keeps remembering the sent options, emits `equipment_test_command` and answers with an extra `message` "Test command accepted; equipment not moved". `sendEquipment` now also returns the options it used. The sheet shows the message (`equipment/operateWords.ts`). Go-only path: Python has no command options. |
| B2 (high) | The tap-changer sheet read `ied.extended_controls`, which `GET /project` never sends, so every IED showed "0 fitting" and a stored BSC showed "Not mapped". Changing the IED select wiped the binding. | `tap/useTypedControlsOf.ts` reads `GET /controls/typed` per IED (same route as the point panel); `tapControls(ied, typed)` takes that list. `controlChoices` always offers the saved control, so it is never "Not mapped". Picking another IED only changes the list; the binding stays until a control is chosen, with "Still sent to … until you choose a control here." The unit test no longer feeds `extended_controls` on the IED (that hid the gap). |
| N1 / B12 (medium) | After Stop and Run the tap, point panel and measurement values showed the previous run's values while the restarted worker had its SCL values (shown 7, device 5). | `run_start_writes.go` `goOnlyRowBindings`: Run reads back the tap-changer, point-panel and measurement-list sources too, like the mapped states. Relay statuses stay out on purpose: Python does not read them back and the parity scenario has a relay. |
| B1 (medium) | A Cancel the device refuses (model without the CO `Cancel` structure, e.g. the demo DPC) made Go and native forget the selection while the IED still held it; the UI said "Not selected". | Go `EquipmentCancel` and the point-panel cancel `peek` and drop only after `CancelTyped` succeeds; `CancelTyped` forgets the typed select only on success. Native `typed_session.cpp` keeps the control handle after a refused Cancel (until it expires). Object-does-not-exist (client error 22) is said in words: "This device's model has no Cancel for <ref>, so the device keeps the selection until its time runs out. Press Operate to send the command, or wait." The browser goes back to the countdown with that reason ("Still selected: …") instead of "Not selected". |
| B4 (medium) | Enter in a command number box opened the question, and the same Enter pressed its focused Cancel. | `PointCommand.tsx` NumberBox: `preventDefault()` on Enter. |
| B3 | The ENC list preselected the first label ("off"), which looked like the present value. | The list starts on a disabled "Choose a value"; Send works once a value is picked. Command rows have no live value, so there is no present value to preselect (see limitations). |
| B6 | The Tidy note sat inline and moved the toolbar buttons when it disappeared. | The note floats under the Tidy button (`.tidy-wrap`, `.tidy-note` absolute, shared tokens). |
| B7 | Add placed symbols on a fixed grid by node count, often on top of others or out of view. | `equipment/placeNew.ts` `freeSpot` + `useVisibleSpot`: a click places the symbol in the middle of what the canvas shows, moved to the nearest spot that touches no other symbol; it is selected as before. Drag-and-drop is unchanged. |
| B10 | A measurement-list row accepted scale 0; the point panel refuses it. | Go `checkMeasurements` refuses scale 0 with the same sentence style (Go-only kind, no parity text). The measurement editor and bulk bar refuse it before saving (`settingFromText`). |
| B13 | Build from SCD said "no symbol yet" for transformers and infeeds, but both are in the Add menu. | The engine's reason is Python parity text, so it stays. The preview shows, for PTR, `PowerTransformer` and IFL, "Build from SCD does not place this type yet. Add a Tap-changer transformer / an Infeed from the Add menu." (`sldPreview.ts skippedReason`); the list is titled "Not drawn" and the summary says "not drawn … with the reason". |
| B15 | A failed save of "Show Select before Operate step by step" was dropped silently, so the switch looked saved. | `equipment/savedSwitch.ts`: the switch moves at once; a failed save moves it back and the options panel says "Not saved, so the setting went back: <reason>". |

## Evidence

Tests (all pass):

- Go `go vet ./...` and `go test ./...` in `studio/engine` with real workers
  (`STUDIO_NATIVE_BIN` = worktree native build, `STUDIO_PYTHON` = main venv): all packages ok,
  parity and golden tests unchanged.
- New Go tests, each checked to fail without its fix where noted:
  `TestRunReadsBackTapRowsFromTheWorker` (fails without the read-back: no snapshot, tap stays 7),
  `TestGoOnlyRowBindingsLeaveRelaysOut`, `TestTestFlagCommandDoesNotMoveTheEquipment`,
  `TestTestFlagSelectionDoesNotMoveTheEquipment`, `TestRefusedCancelKeepsTheSelection` (the fake
  worker refuses Cancel with client error 22 when `STUDIO_RUNTIME_FAKE_NO_CANCEL=1`), and the
  scale-0 case in `TestMeasurementCrossChecks`.
- Frontend `npx tsc --noEmit` clean; `node --test` 436/436, new `tests/frontend/bugHunt0176.test.ts`
  (reducer `cancelRefused`, saved switch revert, test-command words, saved tap control option,
  Enter/ENC source checks, scale 0, Build from SCD words, free spot, floating tidy note).
- Native build (VS 2019, Release) of all workers, `build_xsd_helper.py`, `build_goose_engine.py`.
- `python studio/tools/verify.py` (main venv).

Browser (dev build of this worktree, Go server on `localhost:8821`, own data folder, 1366x768,
SampleTyped.scd + the demo SCD):

- B13: Build from SCD preview lists IFL1/IFL2 and PT1/PT2 with "Add an Infeed / a Tap-changer
  transformer from the Add menu"; SAR keeps the engine's words.
- B7: Add → Tap-changer transformer landed in the middle of the view, clear of the other symbols.
- B2: tap sheet IED list "SampleTyped — 8 fitting" (4 BSC + 4 ISC), mode list "4 fitting"; after
  choosing ATCC1.TapChg and switching the list to FEED1 the binding stayed with "Still sent to
  CTRL/ATCC1.TapChg on SampleTyped until you choose a control here."
- B12: Raise twice → card "Tap 7 / 1–17"; Stop, Run → card "Tap 5 / 1–17" (the device's SCL value).
- B17: CBR_F1 with Test command on, Close → "Test command accepted; equipment not moved", card
  still "Open", `initial_position` still "open", options remembered.
- B1: CBR_F1 step by step, Open selects (27 s), Cancel → countdown stays with "Still selected:
  This device's model has no Cancel for FEED1LD1/CSWI1.Pos …"; Operate Open then worked
  ("Open done · sent 0"), so native kept the selection.
- B3: both ENC lists start on "Choose a value". B4: typing 50 + Enter in AnOut1 opens the question
  and it stays open.
- B6: Tidy up with the toolbar in one row: the note floats (`position: absolute`), button
  positions identical before and after.

## Limitations and not fixed

- **B6, toolbar wrap with a sheet open:** not fixed. One row needs about 934 px; with a sheet
  open at 1366x768 the canvas leaves about 714 px (less while running, the lock badge takes room).
  Every way to fit it either hides button words (beginner rule) or folds tools into a menu (a new
  step). Owner decision; the floating Tidy note at least stops the buttons moving.
- **B3:** the current ENC value is not preselected: command rows carry no live value. A later
  change could read the control's status (`stVal`) for command rows.
- **B7:** the free-spot check looks at symbols, not wires, so a new symbol can sit on a wire.
- **B1:** the unmount auto-cancel (closing the sheet while selected) still cannot show a refusal;
  the selection then runs out by itself. The demo SCD's DPC type still has no `Cancel` (our file;
  owner may add it).
- **B12:** relay statuses are still not read back at Run (Python parity); the tap reply right after
  Operate can still lag one step (`tap_changer_route.go`, optional item, not changed).
- **B15:** the failed-save path is covered by the unit test only; it needs a failing PUT to show
  in the browser.
- Not in this group or not fixed by plan: B11 (parity texts), B8 (known 0097 limitation), B9, B14,
  B18, B19 (not bugs / not reproduced), B5 and B16 (other groups).
