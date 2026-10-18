# 0097: Clean SLD, equipment test buttons, run or connect one IED

Date: 2026-10-18. Decision: [ADR-0067](../../decisions/ADR-0067-run-single-ied-and-test-buttons.md).
Specs: [run one IED, test buttons](../../specs/run-single-ied.md),
[signal layers](../../specs/signal-layers.md). Follows change
[0094](0094-goose-mms-signal-layers.md) and release [0.1.10](0096-release-0.1.10.md).
(Number 0095 is used by an unmerged branch.)

Owner feedback on 0.1.10: Devices cut "Simulated"; the SLD showed a big column of "outside
project" sender stickers and cryptic "C03" tags; breakers without a command link could not be
moved for testing; GOOSE said "Not watched"; Run started every IED with no way to run or
connect a single one.

## What changed

1. **No clipped text.** Devices rows put the status pills on their own line under the name and
   data points. SLD IED tags always show the full IED name (short names removed, also in pulse
   labels and the Devices GOOSE hint). New reusable check `studio/tools/clip_check.js`: at
   1366×768 it fails when any pill, chip, tag or label has `scrollWidth > clientWidth`.
2. **Clean SLD.** Outside-project senders no longer get canvas tags. The toolbar shows
   "N senders outside project ▾" next to the switches; a small list gives each name and *Import
   its CID* (opens the normal import). Their GOOSE is one dot from the canvas edge to the
   receiver tag for ~2 s, then nothing remains; pulse rules (one per real stNum change, max one
   per link per second, reduced motion) are unchanged.
3. **Test buttons** for every equipment kind, only while running and only for simulated IEDs,
   in the side sheet and a new right-click menu: CB/DS/ES Open · Closed · Moving · Bad state;
   relay panel rows on/off; trip relay Trip · Reset; CT/VT Set value · Bad quality; Source
   Live · Dead when bound; push button unchanged. `POST /equipment/<node>/test` writes through
   the sample writer (same path as Devices data-point editing); no command is sent. The command
   link is optional: "Needs: command" became the hint "Add a command link to operate it like SCADA".
4. **Compact GOOSE status.** A link problem is one small ⚠ on the receiver tag. The tag's sheet
   lists one line per link ("← TR1 · Trafo1SwiPosCtl · Needs lab network"), reason on hover, and
   one *Set up lab network* button (to Settings › Network). "Not watched" became "Needs lab
   network" (This PC only), "Not checked" (lab without Npcap) and "Outside project"; Devices
   GOOSE partners use the same words.
5. **Run or connect one IED.** Devices header: *Run this IED* / *Stop this IED* for simulated,
   *Connect* / *Disconnect* for real IEDs. The coordinator keeps one run with a set of running
   IEDs (`simulation/run_selection.py`, `Supervisor.add/stop_some`, `LiveManager.connect/
   disconnect`, `MmsAccess.forget`); endpoint conflicts are checked against running IEDs.
   `GET /project` returns `running_ieds`; the top bar says "Running 1 of 2 IEDs"; a stopped
   IED's SLD tag says "not running" and its test section "IED not running". Any running IED
   locks the drawing and roles, like a full Run.

## Evidence

- `python studio/tools/verify.py`: all PASS.
- Python unit/integration/packaging: 415 passed, 10 skipped. Protocol (native, independent `studio_probe`):
  68 passed, 9 skipped, including the new `test_run_single_ied.py` (one IED runs alone, the other refuses
  the connection, joins, leaves; test buttons write Dbpos 10/00/11/01 read back by
  `studio_probe`; no `control_applied` event; a real IED is refused).
- Frontend: 153 tests pass (new `testButtons.test.ts`, updated `signalLayers.test.ts`,
  `kindInfo.test.ts`); `npx tsc --noEmit` clean.
- Dev app in the in-app browser (backend on port 8775, fresh data folder, public samples only:
  `scd/open_substation.scd` IED1_XCBR + IED2_PTOC with its SLD, `scd/model_substation.scd` BUS1 +
  TR1):
  - *Run this IED* on IED1_XCBR: top bar "Running 1 of 2 IEDs", list pills "Running" /
    "Simulated" on their own line. `studio_probe` reads IED1 (Pos.stVal 01) and gets a refused
    connection (error 5) from IED2. After adding IED2 and stopping IED1: IED1 refused, IED2
    answers; the SLD tag says "IED1_XCBR not running" and the sheet says "IED not running".
  - QB1 (no command link) sheet: "Ready", hint "Add a command link…", test *Closed* →
    `studio_probe` reads XSWI2.Pos.stVal 10. Right-click QA1 › *Closed* → XCBR1.Pos.stVal 10.
  - With both switches on: only equipment and small full-name tags; toolbar "3 senders outside
    project ▾" (FEED1, FEED2, TR2, each with *Import its CID*). BUS1 sheet: "← FEED1 ·
    Feed1SwiPosCtl · Outside project", "← TR1 · Trafo1SwiPosCtl · Needs lab network", one *Set
    up lab network* button that opens Settings.
  - `clip_check.js`: ok (0 clipped) on Devices and SLD at 1366×768 and 1920×1080, and on IED
    Explorer, Find IEDs, SCADA gateway & master and GOOSE & SV traffic at 1366×768.

## Limitations

- The edge dot for outside senders needs listener evidence (lab network with Npcap); it was
  checked by unit tests and code, not on a real wire in this change.
- Real-IED *Connect*/*Disconnect* reuses the existing real session code; it was not exercised
  against a real device here (no lab device attached).
- CT/VT/relay/trip/source test buttons are covered by unit tests; the public samples have no
  bound CT/VT, so only switch buttons were checked live.
- Unbound project IED tags still sit in a small left column that can be under the toolbar at
  the top-left after *Fit view*; pan to reach them.
- Changing roles or the drawing still needs everything stopped. No EXE was built.
