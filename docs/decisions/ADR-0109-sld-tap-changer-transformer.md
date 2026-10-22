# ADR-0109: SLD tap-changer transformer (RTCC)

Accepted: 2026-10-22. Owner request: a power transformer with an on-load tap changer on the SLD:
Raise/Lower, the tap position, Auto/Manual and a simulated voltage regulator (90), all logic in Go.
Same shape as [ADR-0102](ADR-0102-sld-point-panel.md) (Go-only kind, omittable fields, the Python
engine refuses it). Builds on the typed controls of change 0167 ([ADR-0104](INDEX.md), built in
parallel) and leaves select-before-operate to the shared SLD SBO hook of change 0171.

## Problem

The SLD has no power transformer that does anything: `TRANSFORMER` was retired in format 12
(ADR-0033) and the infeed's transformer drawing (ADR-0041) is drawing-only, with one terminal. A
tap changer test (raise, lower, tap limits, Auto blocking manual steps, a regulator reacting to a
voltage) needs a symbol with two sides, live tap and mode values and commands.

## Decision

A new diagram kind `TAP_TRANSFORMER` (Add > Power in and drawing > Tap-changer transformer). A new
kind, not the retired `TRANSFORMER`, because the format 12 migration still deletes nodes of that
name. Two-winding only (3-winding was not cheap: three sides, two ratios, two regulators).

### Contract (`project/contract/tap_changer.go`, `project/validation_tap_*.go`)

`data.tap_changer` (omittable; a new transformer gets every default, other kinds lose it):

| Field | Default | Rule |
| --- | --- | --- |
| `tap_source` | null | `{ied_id, report_ref, ref}`; whole-number ST signal (BSC `valWTr.posVal`, INS/INC/ISC `stVal`) |
| `command_mode` | `none` | `none`, `step` (one BSC/ISC `step_control`), `spc_pair` (`raise_control` + `lower_control`, two different SPC) |
| `step_control`, `raise_control`, `lower_control` | null | `{ied_id, ref}`; looked up in `controls`, then `extended_controls` (ADR-0104) |
| `mode_source` | null | Boolean ST (true = Auto) or whole-number ST (= `auto_value` means Auto) |
| `mode_control` | null | SPC (true = Auto) or ENC (`auto_value` / `manual_value`) |
| `auto_value`, `manual_value` | 1, 0 | -1000..1000 |
| `voltage_source` | null | numeric MX signal (LV voltage) |
| `min_tap`, `max_tap`, `nominal_tap` | 1, 17, 9 | -64..64; min < max; min <= nominal <= max |
| `avr` | see below | `{enabled, target_kv, bandwidth_pct, delay_s, step_pct, hv_rated_kv, lv_rated_kv, source_kv}` |

`avr` defaults `false, 11.0, 1.5, 30.0, 1.25, 132.0, 11.0, 132.0`; target, ratings, band and step
above 0, delay 0..3600 s, band <= 50 %, step <= 20 %; enabled needs a tap position and a command. A
transformer refuses `state_mapping` and `control_binding`. Removing an IED clears its bindings, a
command left incomplete becomes `none` and the regulator is switched off; the transformer stays.

### Runtime (`runtime/tap_changer*.go`, pure domain `runtime/tapchanger`)

- Live rows `<node>:tap`, `<node>:mode`, `<node>:lv` (plant states, like relay rows) and `<node>:avr`
  (regulator state). Real IEDs poll the three signals (`acquisition/watched_tap_changer.go`).
- `POST /equipment/{node}/operate` with `raise`/`lower`/`auto`/`manual` (no new route).
  `tapchanger.Route`: BSC -> `OperateTyped` value `"higher"`/`"lower"`; ISC -> the new tap number;
  SPC pair -> the raise or lower SPC with `true` through the canvas command path. Limits refuse a
  step past min/max. A manual step in Auto is refused. Then, for a tap position on a running simulated
  IED that is not the step control's own status, the new tap is written as `control_feedback`.
- Auto/Manual: with a mode command, that command (+ feedback to a simulated status); without one and
  without a status, the regulator's own mode (per run, starts Manual).
- Regulator (every 0.1 s cycle): LV = source x (LV/HV rated) x (1 + (tap - nominal) x step %);
  written to a simulated `voltage_source` when it changes (write source `tap_changer`, which pauses
  a generator on that point). In Auto, out of band (|LV - target| > band % of target) starts the
  delay; after it one raise (low) or lower (high) is sent from a goroutine, and the next step waits
  the whole delay again. Back in band, Manual or a limit cancels the delay. Every transition is one
  `avr_decision` event with run (run log), node, IED and correlation ids; the step's command events
  share its correlation id and carry `origin: "avr"`.
- Never a real IED: the regulator acts only when every point it uses is on a running simulated IED
  (else `blocked`, logged once), and the command path refuses real devices (`commandRefusal`).
- Energisation: power passes through a transformer both ways (HV `:a` north/west, LV `:b`
  south/east); an earthed side does not earth the other side (`plant/energize_transformer.go`).

### Typed commands until ADR-0104 is merged

`runtime/typed_command.go` holds exactly the API of change 0167 (`CommandOptions`, `TypedCommand`,
`SelectResult`, `SelectTyped`, `OperateTyped`, `CancelTyped`) with temporary adapters that answer
"Step commands (BSC or ISC controls) are not available yet". It is marked
`// Replaced by ADR-0104 at merge`. The frontend reads the optional `extended_controls` of an IED.

### Frontend

The canvas card (132 px) shows HV/LV, "Tap 9 / 1–17", an Auto (solid, robot icon) or Manual (dashed,
hand icon) badge, and while running the LV voltage, the regulator line and ▲ Raise / ▼ Lower. The
sheet has an Operate section (Raise, Lower, Auto, Manual, disabled buttons say why) and the bindings
with the shared signal search picker. Every command goes through `sendTapCommand` (one function, a
direct operate); change 0171's `useSelectOperate` wraps it at merge.

## Compatibility

- Documents without a transformer keep their bytes; other kinds are unchanged. The unknown-kind
  error text and the Python golden vectors are unchanged (`kindChoice` accepts the kind first).
- The frozen Python engine does not know the kind (ADR-0076): it refuses a project with one, like
  a point panel. Remove transformers before switching a data folder back to Python.
- Energisation of diagrams without a transformer is unchanged (CPython vectors still identical).

## Limits

- Two-winding only; no parallel transformers (master/follower), no line drop compensation, no
  tap-change time or transInd pulse simulation; the source voltage is a setting, not a live value.
- BSC/ISC steps and ENC mode commands need ADR-0104; until then use an SPC pair.
- The regulator's own Auto/Manual is not saved; every run starts in Manual.
- No SBO on the SLD until change 0171.
