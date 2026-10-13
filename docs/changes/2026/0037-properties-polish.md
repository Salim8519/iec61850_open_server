# 0037: Properties without path typing, polished buttons

Date: 2026-09-20. Properties never ask for a typed path: mappings are
search-and-choose everywhere, and buttons follow one hierarchy.

A shared `SignalSearchPicker` (search + report filter + one-click
rows, description first, raw ref collapsed) now serves state mappings,
open/closed contacts and relay adding; `RelaySignalAdder` is a thin
wrapper over it. Chosen bindings render as chips with Change to
re-pick. Command mapping shows a ready chip with cdc, model and
blocked state. `SignalPicker` stays for the Signal Test tab only.

Buttons in properties use one system: Lucide move/remove icons,
ghost icon buttons, `danger-outline` for delete, a single primary per
action zone (`Pulse 1` with a Zap icon), ghost pagers and Resume.
BOOLEAN values use one-tap True/False segments in properties via a
`variant` prop; dropdowns remain elsewhere. Live trigger zones sit on
panel tint. Focus ring and press states follow the primary token.

Evidence:

- `npm run check` passes; 9 frontend node tests pass (3 new
  `signalQuery.test.ts`); backend unit suite: 106 pass; `verify.py`
  passes.

Limitations: visual acceptance pending; Scenarios/Test-tab controls
deliberately untouched.
