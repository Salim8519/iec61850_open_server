# 0035: Single-IED relay picker, legacy palette and no auto-open launcher

Date: 2026-09-20. Three user-requested improvements in one pass.

Relay: each protection relay now picks exactly one IED in its header
(`RelayStatusEditor`); new signals inherit that IED, so adding is one
search-and-click step instead of IED/report/signal paging. The new
`RelaySignalAdder` searches with the existing `q` backend parameter
(150 ms debounce), pages 50 rows, and shows a blue dot with a primary
"+ Add" for triggerable signals versus a gray dot for display-only
ones; text badges stay authoritative. Switching the header IED
reassigns all rows by exact `ref` after one confirm; missing refs are
dropped and listed for re-picking. Mixed-legacy relays show a warning
badge until reassigned. Trigger rows keep per-row busy Pulse 1 / Pulse 0
plus Set and Resume.

Palette: LOAD, BUSBAR and TRANSFORMER are hidden from the library
(`registry.ts` + `LEGACY_KINDS`) while old diagrams still render;
click/drag creation is blocked and the Inspector shows a legacy badge.
CT/VT are untouched. Backend contracts still accept the kinds, so no
project migration was needed.

Launcher: Studio never opens browser tabs automatically. `entrypoint.py`
and `run_dev.py` only print the address; `--no-browser` remains as a
no-op. Backend reloads no longer spawn a tab per save.

Evidence:

- `npm run check` passes; 6 frontend node tests pass (3 new
  `relayIed.test.ts`); backend unit suite: 106 pass; `verify.py` passes.
- `test_dev_launcher.py` updated to assert the printed address and no
  `webbrowser` dependency.

Limitations: no independent MMS pulse capture this pass; visual
acceptance of the new adder and legacy badges is pending.
