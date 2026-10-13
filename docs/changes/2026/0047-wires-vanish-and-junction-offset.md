# 0047: Wires vanishing until refresh; junction dot off its terminals

Date: 2026-09-29. Owner report with screenshot: canvas wires sometimes
disappear until the page is refreshed or the run starts; the screenshot also
shows junction terminals detached from the junction dot.

Cause (wires): React Flow 12.11 `adoptUserNodes` resets a node's terminal
bounds (`internals.handleBounds`) whenever it receives a new node object
without `measured`. Studio drops React Flow's dimension changes (to keep
measurement out of saves), so every select, move or edit re-projected nodes
without `measured`, forcing a reset and re-measure. When a second update lands
in the same render batch as that re-measure, React Flow's observer effect does
not re-run, bounds stay empty and every wire on that node stops rendering
until a remount (refresh, Run). Not reproduced on demand (timing race); the
reset itself is proven by a test against the installed library.

Fix: `equipment/measured.ts` remembers React Flow's measured size per node
(view-only, never persisted or in history); `stableFlowNodes` attaches it, so
replaced nodes keep their bounds and no longer re-measure on each click. The
drag draft is only cleared when it exists.

Cause (junction): the 54 px symbol overflowed the 18 px junction box to the
lower right. The junction renders at 18 px with a larger dot radius.

Evidence: frontend node tests 50 pass (`terminalBounds.test.ts` shows bounds
kept with the remembered size and dropped without it); build passes;
Playwright on an isolated copy of the owner's autosave: 8/8 wires after 40
random clicks/drags/undos and after six two-tab synchronisations; junction dot
centred between its terminals (2x screenshot); editing script from 0046
repeated with identical results.

Limitations: the race could not be triggered on demand, so the fix is proven
at the mechanism level; owner confirmation in daily use pending.
