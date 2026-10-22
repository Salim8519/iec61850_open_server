# ADR-0108: Select before operate and typed commands on the SLD

Accepted: 2026-10-22. Owner request: the single-line diagram and the point panel must command
like a SCADA operator does: Select, a visible countdown, then Operate or Cancel for devices
whose control model asks for it, and integer, list, set-point and raise/lower commands in the
point panel. Builds on [ADR-0102](ADR-0102-sld-point-panel.md) (point panel) and the typed
IEC 61850 commands of ADR-0104 (agent cmd-61850-0167, merged separately).

## Problem

Since ADR-0069 a canvas Open/Close on an SBO control sends Select (SBO or SBOw) and Operate in
one native call. The operator never sees the selection, cannot cancel it and cannot set the
originator or the check bits. The point panel sends only SPC and DPC true/false.

## Decision

### Go engine (all logic)

- `runtime/typed_command.go` has exactly the ADR-0104 types (`CommandOptions`, `TypedCommand`,
  `SelectResult`) and `SelectTyped`, `OperateTyped`, `CancelTyped`. On this branch they are
  temporary adapters marked `// Replaced by ADR-0104 at merge`:
  - SelectTyped checks the IED (real and skipped IEDs refused as in Operate), the control and
    its SBO model and answers the model's `sbo_timeout_ms` (30 s when 0). It sends nothing:
    the native canvas client still performs Select and Operate together when the Operate
    follows. Nothing claims a separate wire Select until ADR-0104 lands.
  - OperateTyped sends Boolean SPC/DPC with default options through that client.
  - Other values and non-default options are refused in words ("… arrive with the typed
    IEC 61850 commands (ADR-0104). Nothing was sent."), never dropped.
  - CancelTyped has nothing to cancel on the wire.
- Selections are held by the coordinator (`sbo_selections.go`), keyed by node (and panel row),
  with the control, the selected action or value, the options and the end time. The window the
  browser counts down is the select timeout minus a margin (3 s of a timeout above 6 s, a
  tenth otherwise): 30 s gives 27 s, so the IED never receives an Operate after its own timer.
  Stop ends every selection.
- Operate (`equipment_control.go`, `equipment_select.go`):
  - a live selection must match the action ("… is selected to close. Press Operate to close
    it, or Cancel first.") and is used up by the typed Operate with the selected options;
  - an expired one is refused: "The selection of CB1 ran out after 27 s, so nothing was sent.
    Press Close again to select it." (event `equipment_select_expired`);
  - without a selection and without options, the request and the native call are exactly
    today's (direct models, scripts, push buttons, Python parity);
  - options without a selection go through the typed Operate.
- Command options: `{test, interlock_check, synchro_check, orcat 0..8, orident ≤ 64}`, unknown
  keys refused. The last options sent are remembered in the node as `data.command_options`
  (contract `command_options.go`, omittable: written only after options were sent once, and
  only on CB, DS and ES).
- Point panel (`point_panel.go`, `point_panel_write.go`, `point_panel_select.go`,
  `typed_values.go`): command rows may use a control of `controls` (SPC, DPC) or of the optional
  `extended_controls` list (INC, ENC, APC, BAC, BSC, ISC) that ADR-0104's `scl.ExtendedControls` reads
  from the stored SCL (never part of the IED payload; the browser reads it from
  `GET /controls/typed?ied_id=`). Values: true/false (SPC, DPC), a whole
  number (INC), a number (APC), an enum label (ENC), "higher"/"lower" (BSC, ISC); `min`, `max`
  and `step` bound INC and APC, messages name the range. New actions on the existing
  `POST /equipment/{node}/test`: `select` and `cancel`. `send` on a selected row must send the
  selected value. A Boolean SPC/DPC without a selection keeps today's path. Events
  `point_panel_selected`, `point_panel_select_cancelled`. Project validation accepts typed
  controls on command rows and a `unit` only on an APC row.

### Routes

| Route | Body | Answer |
| --- | --- | --- |
| `POST /api/v1/equipment/{node}/select` | `{"action":"open"\|"close","options"?:{…}}` | `{node_id, action, model, selected: true, expires_ms}` |
| `POST /api/v1/equipment/{node}/cancel` | any | `{node_id, cancelled, action?}` |
| `POST /api/v1/equipment/{node}/operate` | unchanged; optional `options` | unchanged |
| `POST /api/v1/equipment/{node}/test` (panel) | `{"action":"select"\|"cancel","row":…,"value"?:…}` | `{…, selected, model, expires_ms}` / `{…, cancelled}` |

Select and cancel are Go-only routes (`runtimeapi.GoOnlyRoutes`); the frozen Python engine
answers 404 and still refuses point panels.

### Browser (display and send only)

- `equipment/selectOperate.ts`: the state machine idle → selecting → selected → operating /
  cancelling → idle, with expiry at the deadline; late answers revive nothing.
- `features/diagram/sbo/useSelectOperate.ts` and `SboCountdown.tsx`: the shared hook and
  component (also for the tap changer of agent rtcc-0172). The held selection reads
  "Selected: Close. Operate within 27 s" with a timer icon, a dashed accent frame and the
  buttons *Operate Close* and *Cancel*; outcomes have an icon and words (never colour alone).
  `StepButtons.tsx`: ▼ Lower / ▲ Raise.
- Sheet: `SwitchOperate.tsx` replaces the Open/Close buttons. Direct models send at once with
  the body `{action}` byte for byte; SBO models select first. A folded *Command options*
  section (`CommandOptionsPanel.tsx`) is sent only once used.
- Point panel: `PointCommand.tsx` gives INC/APC a number box (APC with the row's unit), ENC a
  list of labels, BSC/ISC Lower/Raise; SBO rows select, direct rows keep the one confirmation.
  Add points lists typed controls with the kind badge (SE) and their class (INC, ENC, APC, BSC).

## Compatibility

- Projects, requests and native calls without SBO selections or options are unchanged; parity
  vectors are untouched.
- `data.command_options` and typed command rows are Go-only: the frozen Python engine refuses a
  project that has them (extra keys), as it refuses point panels.
- `extended_controls` entries are ADR-0104's final shape: `{ref, status, cdc, ctl_val_kind,
  ctl_val_ref, value_kind (int|enum|float|step), model (numeric ctlModel 0..4, as in `controls`),
  sbo_timeout_ms, min, max, step, labels {label: number}, trans_ind}`. This branch carries
  ADR-0104's SCL reader byte for byte (`scl/controls_extended.go`, `controls_extended_read.go`, the
  two builder lines), so they merge without change. `runtime/typed_command.go` (adapters and
  `TypedControls`) and `runtimeapi/typed_routes.go` (`controlList`) are marked
  `// Replaced by ADR-0104 at merge`; ADR-0104's files of the same names replace them.

## Limits

- Until ADR-0104 merges there is no separate wire Select or Cancel, typed values are refused
  and options cannot be sent.
- Options of point panel rows are the defaults; the panel has no options section.
- Real IEDs stay refused on the SLD and in the panel (ADR-0067, real-IED policy).

## Revision (change 0173, owner decision)

Showing the selection as a second step surprised users: in a station the select / execute
exchange happens between the SCADA or HMI and the IED, and the operator gives one command. The SLD
switches and point panel rows therefore send in one press by default; for an SBO control the engine
selects and operates on the wire in the same request (ADR-0104). The countdown with Operate and
Cancel is kept for testing behind "Show Select before Operate step by step" in Command options,
stored as the Go-only preference `sld_sbo_steps` (absent from the default document, so the frozen
Python engine's answers are unchanged). The IED Explorer, a protocol test tool, stays step by step.
