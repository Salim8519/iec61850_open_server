# Spec: UI structure and design system

Decision: [ADR-0032](../decisions/ADR-0032-clean-four-place-ui.md),
[ADR-0054](../decisions/ADR-0054-tools-place.md) (fifth place, Tools) and
[ADR-0033](../decisions/ADR-0033-feature-consolidation-format-12.md).
Owner request (2026-10-14): the app must feel clean and organised like an iPhone,
be easy for beginners and older users, and still be pro level.

## Behaviour contract

- Five places in a left sidebar: **Devices, SLD, Test, Tools, Settings** (order and name:
  [ADR-0048](../decisions/ADR-0048-desktop-app-window.md), Tools: [ADR-0054](../decisions/ADR-0054-tools-place.md)); Devices opens first. The top bar
  shows the page title (the SLD page spells out "Single-line diagram (SLD)"), one short subtitle, the File menu and one Run/Stop button.
- Text and controls are large: 15 px base text, 40 px controls, 44 px primary
  actions, AA contrast. Sizes are in rem so browser zoom scales everything.
- Each form shows the common choice first. Rare options live in one "Advanced"
  disclosure (`ui/Disclosure.tsx`).
- Tool pages and Devices fill the window and every list has the shared filter bar
  (`ui/FilterBar.tsx`, [tool layout](tool-layout.md), ADR-0064).
- One implementation per job: `features/test/signalActions.ts` is the only code
  that sets, pulses or resumes simulated values or saves a generator;
  `ValueControl` and `SourceEditor` are the only widgets for that.
- Hidden legacy kinds TRANSFORMER, LOAD and BUSBAR are retired (format 12).
- Running dots ([ADR-0059](../decisions/ADR-0059-separate-run-running-dots.md)): each place shows
  its own dot while something runs there: IEDs (Devices, SLD), sequences (Test), tools (Tools).
  The words are in the tooltip and screen-reader text; a paused Run is amber. Tool cards name
  what runs.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| UI-01 | The sidebar has exactly the five places, in order, each with a plain subtitle. | `studio/tests/frontend/uiStructure.test.ts::sidebar has exactly five places` |
| UI-02 | Base text is at least 15 px, small text 13 px, controls 40 px and primary actions 44 px. | `studio/tests/frontend/uiStructure.test.ts::tokens keep text and controls large` |
| UI-03 | Text, accent and status colours meet WCAG AA contrast on the surfaces they use. | `studio/tests/frontend/uiStructure.test.ts::token colours meet AA contrast` |
| UI-04 | Polarity, push-button modes and colours are only shown inside Advanced. | `studio/tests/frontend/uiStructure.test.ts::rare equipment options live under Advanced` |
| UI-05 | Retired kinds are rejected by the backend and absent from the frontend; old projects drop them with a notice. | `studio/tests/unit/test_diagram_drawing.py::test_retired_kinds_are_rejected`, `studio/tests/unit/test_diagram_drawing.py::test_format_12_migration_retires_legacy_kinds_and_their_wires`, `studio/tests/frontend/uiStructure.test.ts::retired equipment kinds are gone from the frontend` |
| UI-06 | Value writes and generator saves have exactly one implementation each. | `studio/tests/frontend/uiStructure.test.ts::one value control and one generator editor` |
| UI-07 | Real vendor files import even when they use PhyComAddr attributes or unnamed report blocks. | `studio/tests/unit/test_real_world_scl.py::test_phycomaddr_attributes_import`, `studio/tests/unit/test_real_world_scl.py::test_unnamed_report_block_is_skipped_with_warning` |
| UI-08 | The drawing place is labelled SLD and its heading spells out single-line diagram. | `studio/tests/frontend/uiStructure.test.ts::the drawing place is called SLD and spelled out in its heading` |
| UI-09 | Each place shows its own running dot with words; a running gateway does not mark the IEDs as running. | `studio/tests/frontend/running.test.ts::each place gets its own running dot with its own words` |
