# ADR-0104: Typed IEC 61850 commands beyond SPC/DPC (Go engine only)

Accepted: 2026-10-22. Change [0167](../changes/2026/0167-typed-iec61850-commands.md). Builds on
[ADR-0011](ADR-0011-typed-equipment-controls.md), [ADR-0067](ADR-0067-run-single-ied-and-test-buttons.md),
[ADR-0069](ADR-0069-standard-commands-and-listening-endpoint.md) and the Go-only rule of
[ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md).

## Problem

Simulated IEDs and ScadaBench's own client handled only SPC and DPC (on/off). Tap changers (BSC/ISC),
set points (APC/BAC), integer and enumerated commands (INC/ENC, e.g. `Mod`) were invisible, and Select,
Operate and Cancel could not be sent as separate steps with options (test, interlock/synchro check,
originator). The Python engine is frozen, so the IED model it builds (and its golden vectors) must not
change.

## Decision

1. **Read, never store.** `scl.ExtendedControls(source, ied)` reads INC, ENC, APC, BAC, BSC and ISC
   from the IED's stored SCL with the model builder's rules (`scl/controls_extended*.go`), cached per
   IED/source like `DataClasses` (ADR-0095). The IED model, its `controls` and warnings stay
   byte-identical; no project data changes. A control is listed when ctlModel is not status-only and
   its feedback/ctlVal leaves have the class's types; others are skipped silently.
2. **Worker start key `extended_controls`** (only when the list is not empty). The worker installs a
   typed binding per entry and advertises `typed_controls`; the Go engine requires it (the stale-worker
   text is unchanged). The vendor stack runs the four control models, select timeout and cancel.
   Feedback: INC/ENC/ISC status = ctlVal; APC `mxVal.f` = ctlVal; BSC Tcmd higher/lower moves
   `valWTr.posVal` ±1 within -64..63 or the SCL's minVal/maxVal (`transInd` stays false: the simulated
   drive arrives at once); BAC moves `mxVal.f` by stepSize. Out of range → AddCause 26 (inconsistent
   parameters), at a tap limit → 7 (step limit). Test commands are acknowledged and change nothing.
   Per control the coordinator can set simulated blocks (`control_checks`): `interlock_blocked` refuses
   a command that asks for interlock check (AddCause 10), `synchro_blocked` one that asks for synchro
   check (11), `blocked` everything (9). orCat, orIdent and ctlNum are in the control events.
3. **Client: separate typed steps.** `studio_control` adds `typed_select`, `typed_operate`,
   `typed_cancel` (capability `typed_commands`) with ctlVal kinds bool, int, enum, float (APC
   AnalogueValue.f) and step (Tcmd). A Select is kept per endpoint and reference until operate, cancel
   or sboTimeout, on its own associations; `canvas_operate` and the Tools `operate` are untouched.
4. **Go API and routes** (Go-only): `Coordinator.SelectTyped/OperateTyped/CancelTyped`
   (`runtime/typed_command.go`), `POST /controls/select`, `/controls/cancel`, `/controls/checks`,
   `GET /controls/typed`; `POST /controls/operate` keeps `{ied_id, ref, value: bool}` on the old path
   and takes the typed path when `options` or a non-boolean value is present. Only simulated IEDs of
   the run are commanded, exactly as before. IED Explorer gets `POST /tools/explorer/command`
   (select/operate/cancel) under its existing real-IED policy (one confirmation; cancel needs none).
   An Operate after an expired Select answers "Select expired after N s. Press Select again."; an
   Operate without any Select selects and operates in one step as before.
5. **Explorer panel**: list pick for `Mod` (standard modes; MMS carries no enum names), number box for
   INC/ISC/APC, ▲ Higher / ▼ Lower for BSC/BAC, Select → countdown → Operate/Cancel for SBO models, and
   a folded "Command options" box.

## Compatibility

Python parity and goldens are unchanged (`controls`, warnings, IED model, `canvas_operate`, Tools
`operate`, every existing route body). A Go engine with an older `studio_ied` is refused as stale; the
Python engine does not need `typed_controls`. Limitations: APC/BAC use `AnalogueValue.f` only (no `i`),
Edition 1 `setMag` is not read, Explorer names enumeration values only for `Mod`, and operTm
(time-activated operate) is not sent.
