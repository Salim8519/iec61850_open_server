# Spec: Infeed symbol and live colouring

Decisions: [ADR-0041](../decisions/ADR-0041-infeed-symbol.md),
[ADR-0042](../decisions/ADR-0042-live-colouring.md). Changes: [0064](../changes/2026/0064-infeed-symbol.md),
[0065](../changes/2026/0065-live-colouring.md).
Owner decisions (2026-10-14): power enters through a new **Infeed** symbol that must
not affect any engine (SCL import/export, SLD import, runtime); live colouring is
display only.

## Purpose

While the IEDs run, the single-line diagram shows which wires and busbars are live,
dead or earthed, computed from the real switch positions, like a station HMI.

## Behaviour contract

**Infeed** (`INFEED`)
- A drawing symbol for where power enters: `infeed_type` is `line`, `transformer` or
  `grid`; `infeed_state` is `live` (default) or `dead`.
- It never has a state mapping, command, relay entries or push settings. It is never
  written to SCL, never produced by SLD import and never sent to an IED.
- Project format stays 12. Nodes of other kinds are stored exactly as before: the
  infeed fields are only stored on Infeeds.

**Energisation** (`backend/studio_app/domain/energization.py`, pure)
- Positions come from the authoritative equipment states (Dbpos 1 Open, 2 Closed;
  0, 3, missing or unmapped is unknown).
- CB, DS and ES join the wires on their north/west side with those on their
  south/east side only when Closed. JUNCTION, CT and INFEED join all their wires. A VT
  is an end point. TRIP_RELAY, SOURCE, RELAY and PUSH_BUTTON carry no power; their
  wires have no state.
- A section is **live** when it holds a live Infeed through closed switches,
  **earthed** when it holds a closed ES, **earth fault** when both, **maybe live** when
  it would be live if every unknown switch were closed, otherwise **dead**.
- Computed by the backend only while running; the browser only displays it. Nothing
  is written to any IED.

**Display**
- Wires and junctions show their state by line style plus colour: live thick solid,
  dead thin dashed grey, earthed dash-dot green, maybe live dotted amber, earth fault
  thick double red. A legend names every style in words while running.
- The equipment sheet names the section state in words.
- When stopped, nothing is coloured and the legend is hidden.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| INFEED-01 | An Infeed rejects state mappings, commands, relay entries and infeed fields on other kinds. | `studio/tests/unit/test_infeed.py::test_infeed_rejects_bindings` |
| INFEED-02 | Projects without Infeeds are stored exactly as before (no new keys on existing nodes). | `studio/tests/unit/test_infeed.py::test_existing_nodes_store_no_infeed_keys` |
| INFEED-03 | SCL export is byte-identical with and without Infeeds on the diagram. | `studio/tests/unit/test_infeed.py::test_scl_export_ignores_infeeds` |
| INFEED-04 | SLD import never creates an Infeed. | `studio/tests/unit/test_infeed.py::test_sld_import_never_creates_infeeds` |
| POWER-01 | Closed switches join their two sides; open switches separate them. | `studio/tests/unit/test_energization.py::test_switch_position_splits_sections` |
| POWER-02 | Unknown positions make downstream sections maybe live, never live or dead. | `studio/tests/unit/test_energization.py::test_unknown_position_is_maybe_live` |
| POWER-03 | A closed earth switch earths its section; with a live infeed it is an earth fault. | `studio/tests/unit/test_energization.py::test_earthing_and_earth_fault` |
| POWER-04 | A dead Infeed energises nothing; control and display items carry no state. | `studio/tests/unit/test_energization.py::test_dead_infeed_and_non_power_items` |
| POWER-05 | Energisation is published only while running, only when it changes, and never writes a signal. | `studio/tests/protocol/test_energization_runtime.py::test_energization_events_follow_switching` |
| POWER-06 | Every state has a distinct line style and a legend word, and colours meet AA on the canvas. | `studio/tests/frontend/energization.test.ts::every state has a style and a word` |

## Acceptance evidence

Change [0065](../changes/2026/0065-live-colouring.md): an isolated browser instance with
a grid Infeed, a busbar, a Line feeder and a Transformer bay linked to a simulated IED
coloured live, maybe-live and dead wires as Q1/Q0 switched, with a worded legend.
Pending: the owner's visual acceptance.
