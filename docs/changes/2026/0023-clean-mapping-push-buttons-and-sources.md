# 0023: Clean mappings, configurable buttons and analog Source

Date: 2026-09-16. Implements
[ADR-0014](../../decisions/ADR-0014-equipment-actions-and-source-editor.md).

Signal selection now assigns immediately. Mapping-specific Copy, Use and Clear
buttons were removed; state uses a `Not assigned` choice. State and command
remain optional and independent.

Trip relay now has selectable polarity and defaults new nodes to Operate=0 and
Reset=1. A new Push Button supports either a one-command SPC press with a chosen
0/1 value or a latching Boolean status toggle. Status toggles use acknowledged
sample updates and therefore participate in reports and runtime events.

The Source equipment picker accepts numeric MX points and its inspector exposes
the existing manual and automatic measurement generators, including CSV. The
implementation reuses coordinator-owned source logic.

Evidence:

- Sixteen focused unit tests passed for mappings, project format and control rules.
- Two independent native protocol tests passed, covering DPC feedback, inverted
  Trip polarity, one-shot SPC and Boolean latching updates.
- Frontend TypeScript and production Vite build passed.
- Maintained file-length validation passed.
- The complete maintained suite passed all 111 tests, including native loopback
  protocol and portable packaging coverage.

Browser visual acceptance could not run because the local in-app browser harness
could not create its kernel assets. Visual acceptance and third-party
interoperability remain release gates.
