# 0218 Real IEDs are read only until Allow commands, and the device's command setup is checked

Date: 2026-11-11. Owner: "when ied is in monitor mode, why we can send commands?", then chose a
switch per IED in Devices (off by default), IED Explorer left as it is, the gateway needing its
own switch and the IED's switch, the user making sure the IED pings, and "both together": a check
of the device against the CID and a test of command reading over every vendor file available.
Decision: [ADR-0138](../../decisions/ADR-0138-real-ied-allow-commands.md) (updates ADR-0057).
Layers: Go engine (`project/netset`, `gateway/service`, `acquisition`, `api/liveapi`), frontend.

## Problem

"Monitor real device" promised read only, but since ADR-0057 the gateway passed SCADA commands to
every real IED while its one switch was on, and that switch is on by default. A user watching a
live bay could operate a breaker through the gateway without knowing it. Nothing showed when the
device's own control setup differs from the CID, or that the device is in Local.

## Change

- **Allow commands** per real IED (`live.allow_commands`). Off by default; stored only while on, so
  every existing project and settings answer keeps its bytes. Devices › Overview shows the switch
  with one plain sentence for on and off; it is saved with Save connection and locked while running.
- **Gateway.** A real IED's command points are monitor-only (the native worker answers SCADA
  negatively) and page or SCADA commands are refused with "<IED> is a real device with commands
  off; turn on Allow commands in Devices" until the IED's switch is on. The gateway's own switch
  still works: both must be on. Explorer, SLD and Test are unchanged.
- **Commands on the device.** After connecting (and on Refresh health) the live session reads, read
  only, `ctlModel` of every CID control, `sboTimeout` of the SBO ones and `Loc.stVal` of LLN0 and
  of each control's node. Each control is `match`, `differs` (with the reason, e.g. "the device
  has this object as status only") or `unreadable`. A failed read is kept as the check's error and
  never ends the session. Go-only route `GET /live/{id}/controls`; Devices shows "N commands
  checked: …", a Local banner and the rows that differ.
- **Vendor command reading** is now a test over every SCL file available, and the instance
  `ctlModel` form (name or number) has a fixed test.
- Words: the role hint, the Devices hint and the gateway switch hint say where commands come from.

## Send from Devices (same day)

The owner then asked where commands are sent, saw they were only in Tools, and said: "make the
ied has its own allow switch to send commands". Devices › Data points now starts with a
Commands strip. With Allow commands off it says how to turn them on; while the IED is not live
it says to press Run; when both are fine each SPC/DPC command of the CID has Close…/Open… (or
On…/Off…). A button only opens the REAL-device warning (the gateway's wording); "Yes, send"
calls the Go-only `POST /live/{id}/operate`. `internal/realcmd` checks again (confirmation,
real role, switch, command in the CID, on/off value, live session), sends through the shared
`studio_control` client (device's own model, originator station), reads the feedback point for up
to 3 s and logs `real_command_requested`, `real_command_applied`/`real_command_failed` and
`real_command_observed` with one correlation id. The gateway switch does not apply here; steps and
set points are not offered yet.

## Evidence

- Go: `go test ./internal/...` pass except three tests unrelated to this change that fail on this PC
  only: `runtime` TestPointPanelRefusesWhatDoesNotFit and TestAvrLowersAHighVoltageOnItsOwn (ports
  23201/23901 are in a Windows excluded port range), `licensing/hostkey` rate-limit test (passes
  1 of 3 runs on this tree, passes on a clean HEAD worktree; timing), and `runtime`
  TestOperateTiming and `licensing` TestScenariosMatchPython (each failed once in a full run under
  load and passed alone; the latter also passes on a clean HEAD worktree). `verify.py`: file
  lengths, architecture, documentation, history, specifications and offline UI pass; its Go step
  shows only these failures. New tests:
  - `project/netset/allow_commands_test.go`: off by default, stored only while on, only on/off.
  - `gateway/service/real_commands_test.go`: both switches needed (five cases); monitor-only flag.
  - `acquisition/control_check_test.go`: every class (match, model differs, select time differs,
    status only, unreadable), Loc only from LLN0 and the control's node, sboTimeout only for SBO,
    typed controls checked once; a failed read leaves an error and the session goes on.
  - `scl/controls_vendor_test.go`: every command read has model 1 to 4, a feedback point of the
    same object and a whole-ms select time; instance ctlModel wins as a name or number.
  - `api/liveapi` parity replay: the Go-only read is answered as unreadable so the recorded Python
    conversation, the live status and health stay byte-identical (`TestRoutesMatchPython` pass).
- Independent device (`TestRealIEDSession`, STUDIO_REAL_IED=1): `studio_live` to a separate
  `studio_ied.exe`: control check total 4, match 4, differs 0, unreadable 0, 3 Loc points read.
- Vendor corpus (local `vendor_cids/` + `test_cids/`, never committed): 40 manufacturers, 744 IEDs,
  68,091 switching commands and 10,296 other commands read, none failing a check (ABB, Siemens,
  SEL, Schneider, GE, Efacec, Hitachi, NR, Reinhausen, Phoenix, OMICRON and others). 41 files the
  importer refuses are left to the import tests (ADR-0114).
- Two engines from this source on this PC (scratch data): one simulated the demo IED1_XCBR at
  127.50.0.2, the other monitored it as a real device. Devices showed the switch off with its
  sentence; ticking it and Save connection stored `allow_commands: true`; after Run the card read
  "2 commands checked: 2 as in the CID, 0 different, 0 not readable". Gateway page command with both
  switches on: "operated: The device selected and operated it (sbo-with-enhanced-security)". With
  the IED's switch off: both its command points monitor-only and the command refused with "IED1_XCBR
  is a real device with commands off; turn on Allow commands in Devices".
- Devices sending (two engines, demo IED1_XCBR as the device): switch off, the strip said so and
  `POST /live/{id}/operate` answered 400 "IED1_XCBR has Allow commands off; turn it on in Devices
  first"; switch on and live: Close… showed the REAL-device warning, Yes, send Close gave "The
  device selected and operated it (sbo-with-enhanced-security). Position: …CSWI1.Pos.stVal is
  closed", Open gave "… is open"; the device's own engine read stVal 1 (open) afterwards; the run
  log holds requested, applied and observed per command with one correlation id. Screenshots
  taken with headless Chrome at 1366x768 (the app's browser pane does not draw while the
  desktop window is hidden). `realcmd` tests: every refusal in words, request/applied/observed
  events, the device's own refusal, a position that does not move. `gateway/service`
  TestGatewayServesValuesAndCommandsOverIEC104 failed once in a parallel run and passed 3 of 3
  alone.
- Frontend: all 485 tests pass (`guideCopy.test.ts` updated to the new wording, one new test in
  `iedRoles.test.ts`); `tsc --noEmit` clean.

## Limitations

- No third-party IED was available: the device check is proven against ScadaBench's own simulator.
  Vendor support covers reading commands from SCL files, not operating those devices.
- A project that relied on gateway pass-through to a real IED must turn that IED's switch on once.
- Not in a release yet; no EXE was built.
