# 0171: SLD select before operate and typed point panel commands

Date: 2026-10-22. Branch `sld-sbo-0171` (on `release-0120` + the point panel / GOOSE patch,
changes 0165 and 0166). Decision: [ADR-0108](../../decisions/ADR-0108-sld-select-before-operate.md).

## Intent

Command like a SCADA operator: an SBO breaker, disconnector or earth switch is selected first,
a countdown shows how long the selection holds, then Operate or Cancel; command options
(test, interlock check, synchro check, originator) on the sheet; point panel rows for integer
(INC), list (ENC), set-point (APC) and raise/lower (BSC, ISC) commands with the same flow.

## Observed evidence and assumption

- The demo stations (`scd/open_substation.scd`, `scd/model_substation.scd`, the marketing demo)
  mark their breakers `sbo-with-enhanced-security` (ctlModel 4). Today one native call does
  SBOw + Operate (ADR-0069), so an Open/Close works without a visible selection.
- Typed IEC 61850 commands with a real wire Select are agent cmd-61850-0167's work (ADR-0104).
  This branch carries ADR-0104's Go API as adapters (`typed_command.go`, marked
  `// Replaced by ADR-0104 at merge`): Select checks and holds, sends nothing; the Operate after
  it uses today's native SBOw + Operate. So the demo breakers keep switching on this branch,
  and the countdown is real Go state, but **no separate Select is on the wire until ADR-0104
  merges**. Typed values and options are refused in words until then.

## Changes

Go (all logic): selections held in the coordinator with the window sboTimeout minus a margin
(30 s gives 27 s, the 10 s test model 7 s); `POST /equipment/{node}/select` and `/cancel`
(Go-only routes); Operate checks the held selection (action must match, expiry refused in
words, used up once); default requests keep today's exact path; options parsed, checked and
remembered as `data.command_options` (CB/DS/ES only, omittable); point panel `select`/`cancel`
actions on the existing test route, typed values checked against min/max/step and enum labels;
project validation accepts typed controls from `extended_controls` and a unit only on APC rows.
Alignment with ADR-0104's final `extended_controls` (second session): the typed list is read with
ADR-0104's own `scl.CachedExtendedControls` (its SCL reader files copied unchanged), never from the
IED payload; `labels` (label to number) replaces `enum_labels`; `model` is the numeric ctlModel;
BAC is a raise/lower class; `GET /controls/typed` (adapter `controlList`, Go-only) feeds the point
panel through `points/useTypedControls.ts`.

Browser (display and send): state machine `equipment/selectOperate.ts`; shared
`useSelectOperate` hook, `SboCountdown` and `StepButtons` (exported for rtcc-0172);
`SwitchOperate` (direct: unchanged `{action}` body; SBO: select first), folded
`CommandOptionsPanel`; `PointCommand` with number box, label list and Lower/Raise; Add points
lists typed controls with the SE badge and their class; words in `copy/selectOperate.ts`.

Modified existing files and why: `runtime/equipment_control.go` (Operate split into
`planCommand` + `command` so Select reuses the same checks; send step via `sendEquipment`;
options remembered with the start position), `coordinator.go` (selections field), `stop.go`
(Stop ends selections), `point_panel.go` (select/cancel actions), `point_panel_write.go`
(typed send), `api/runtimeapi/routes.go` (two routes, options passed), `app/routes_test.go`
(Go-only routes listed), `project/contract/diagram.go` (omittable `command_options`),
`project/validation.go` (options check), `project/validation_point_panel.go` (typed command
controls, APC unit), `api/types.ts` (ExtendedControl, CommandOptions, optional fields),
`DiagramPage.tsx` (operate returns its promise, optional options), `EquipmentSheet.tsx` (uses
`SwitchOperate`), `points/PointRow*.tsx`, `usePointActions.ts`, `PointAdder.tsx`,
`pointRequests.ts`, `pointDisplay.ts`, `copy/pointPanel.ts` (typed rows), design and reference
documents.

## Evidence

- `CGO_ENABLED=0 go build ./...` clean. `go test ./internal/project/ ./internal/runtime/
  ./internal/api/runtimeapi/ ./internal/app/` passed after the Go refactor (before the new
  routes were added). New tests then run and passed: runtime `-run
  'Typed|Select|Cancel|Direct|OptionChanges|OperateWindow'` (select → operate, wrong action,
  expiry at 7 s in words, cancel, Stop clears, direct model answer keys unchanged, options
  route typed, window 30 s → 27 s), runtime `-run PointPanel` (existing panel tests plus SBO
  row select/send/cancel and INC/ENC value checks), project `-run 'Typed|CommandOptions|PointPanel'`.
- Second session (2026-10-22, after the ADR-0104 alignment), with `GOTMPDIR` inside the worktree
  (antivirus blocks test executables in %TEMP%; no "Access is denied" seen):
  - `CGO_ENABLED=0 go test ./...` in studio/engine: every package passes. Under parallel load
    `cmd/studio_server TestFoldBeforePython` once exceeded its time (passes alone, 21 s) and
    `internal/launcher` needs `STUDIO_PYTHON` (set: passes). Fixed on the way: the select route
    test expected Python-spaced JSON, the BAC value case predated BAC as a raise/lower class.
  - `node --test --experimental-strip-types tests/frontend/*.test.ts`: 400 of 400 pass
    (`interlockHints` now reads the Open/Close buttons from `SwitchOperate.tsx`: disabled only
    while busy or while a selection is held, never by a hint).
  - `npx tsc --noEmit` clean; `check_file_lengths.py` PASS; `verify.py` PASS (file lengths, architecture,
    documentation, history, specifications, GOOSE engine and Go engine tests; one run inside the
    combined check failed under load and passed when run again alone).
- Browser (Go engine built from this worktree, port 8797, own data folder, built frontend):
  demo station `scd/open_substation.scd` (IED1_XCBR, CSWI1.Pos DPC ctlModel 4, no sboTimeout →
  30 s → 27 s window) and ADR-0104's sample `SampleTyped.scd` for typed rows.
  - CB1 Close: "Selected: Close. Operate within 26 s", dashed frame, Operate Close / Cancel;
    Operate Close closed CB1 ("Close done", symbol Closed). Open then waiting: "The selection ran
    out, so nothing was sent. Select again to try once more."
  - `GET /controls/typed` answered 24 typed controls of SampleTyped with ADR-0104's keys; the
    point panel showed ENC (label list), INC (SBO, number box "0 to 10" + Select), APC (number
    box with kV) and BSC (Lower/Raise) rows with the SE badge; Select of INC 4 answered in words
    that typed values arrive with ADR-0104 (nothing sent), as designed.
  - Fixed after looking: the SBO hint's timer icon shrank to 6 px (flex), and the number box
    placeholder was clipped ("0 to 10, ste"); it now shows the range only (full hint in title).
  - Screenshots (session scratchpad `sbo0171/`): `01-cb1-selected-countdown.jpg`,
    `02-cb1-operated-closed.jpg`, `03-cb1-selection-ran-out.png`, `04-panel-typed-rows.jpg`.

## Limitations

- No wire Select/Cancel and no typed values or options on the wire until ADR-0104 merges; the
  adapters say so in words. `extended_controls` keys are ADR-0104's final ones.
- Point panel rows use default options (no options section in the panel).
- Real IEDs remain refused by the SLD and the panel, as before.
- Python engine frozen: it answers 404 to the new routes and refuses projects with
  `command_options` or typed rows, as it refuses point panels.
