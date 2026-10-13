# 0025: Format-6 Trip polarity migration and stable live canvas

Date: 2026-09-16. Implements
[ADR-0016](../../decisions/ADR-0016-format-6-trip-and-live-canvas.md).

Project writes now use format 6. Opening formats 1 through 5 preserves and verifies
the exact input archive before assigning every legacy Trip relay Operate=0 / Reset=1.
The visible migration notice names each relay whose effective polarity changed and
links the recovery file. Explicit format-6 inverse polarity remains unchanged.

Live equipment states now use keyed subscriptions instead of a canvas-wide value
object. React Flow nodes, callbacks and configuration retain stable identities across
signal and Event Log traffic. Projected nodes have safe initial dimensions before
measurement, and the one selected ID still drives both canvas highlight and Properties.
Control feedback log rows show the exact control reference, raw `ctlVal`, interpreted
action and whether mapped feedback changed.

Evidence:

- The complete maintained Python suite passed all 115 tests.
- Independent native MMS probes passed for reset-high false/true control, shared
  relay mappings and separately selected feedback points.
- Nine frontend contract tests passed, including 1,000 live updates with repeated
  selection changes, per-node notification isolation and explicit inverse polarity.
- Frontend TypeScript and the production Vite build passed.
- File-length, architecture and documentation verification passed.

The supplied packet capture remained local and was not added to project data.
No packaged executable was built. Interactive browser visual acceptance and
third-party interoperability remain release gates.
