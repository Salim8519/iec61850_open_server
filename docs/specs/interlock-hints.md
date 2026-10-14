# Spec: Friendly interlock hints

Decision: [ADR-0043](../decisions/ADR-0043-interlock-hints.md). Change: [0066](../changes/2026/0066-interlock-hints.md).
Owner decision (2026-10-14): interlocking is **hints only, friendly**. Studio never
blocks a command: not its own Operate, not an external MMS client.

## Purpose

Before an engineer switches, Studio says in plain words when the drawing suggests the
operation is unsafe, like a reminder from a colleague. It also notes in Activity when
an external client (SCADA, gateway) switches against such a hint.

## Behaviour contract

- Hints are computed by the backend (`domain/interlock_hints.py`, pure) from the
  drawing and the authoritative switch positions, on the same sections as live
  colouring, and published with the `energization` event while running.
- Rules (CB, DS, ES only):
  - **load**: a disconnector whose bay breaker (reached through junctions and CTs
    only) is Closed: "Open Q0 first: a disconnector must not switch load current."
  - **unknown**: that breaker's position is unknown: "Check that it is open first."
  - **earth**: closing an earth switch on a part that is live or may be live, or
    closing a breaker/disconnector whose one side is earthed while the other has power.
- The equipment sheet shows the hints under Open/Close in an amber dashed box with an
  icon and words. Identical open and close hints are shown once. Open and Close stay
  enabled; they are only disabled while a request is busy.
- The box always says the hints come from the drawing, not the IED, and that Studio
  never blocks a command. Studio never claims IEC interlocking (CILO).
- When an external MMS client switches against a hint, Activity gets one
  `interlock_hint_observed` entry naming the switch, the client origin (or "An external
  client") and the hint. Studio's own Operate is not reported as external.
- The native per-control `blocked` flag stays unused.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| HINT-01 | A disconnector warns when its bay breaker is closed, or its position is unknown. | `studio/tests/unit/test_interlock_hints.py::test_disconnector_warns_about_closed_breaker` |
| HINT-02 | Closing an earth switch on a live or maybe-live part warns; on a dead part it does not. | `studio/tests/unit/test_interlock_hints.py::test_earth_switch_on_a_powered_part` |
| HINT-03 | Closing a switch that would join power to an earthed part warns. | `studio/tests/unit/test_interlock_hints.py::test_closing_onto_an_earthed_part` |
| HINT-04 | Hints never block a command, and only external switching against a hint is noted in Activity. | `studio/tests/protocol/test_interlock_hints_runtime.py::test_hints_never_block_and_external_switching_is_noted` |
| HINT-05 | The sheet never disables Open/Close for a hint and always names the hint source. | `studio/tests/frontend/interlockHints.test.ts::hints never disable switching and always name their source` |

## Acceptance evidence

Change [0066](../changes/2026/0066-interlock-hints.md): in an isolated browser instance
F1-Q1 showed "Before you open or close: Open F1-Q0 first …" with both buttons enabled;
`studio_probe` opening XSWI1 was accepted and Activity noted "An external client opened
F1-Q1 …". Pending: the owner's visual acceptance and real SCADA clients.
