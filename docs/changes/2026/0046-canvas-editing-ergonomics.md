# 0046: Canvas editing ergonomics — multi-select, clipboard, terminal targets

Date: 2026-09-29. Owner request: snappier design area; easier select, copy,
modify, paste and duplicate; terminals much easier to hit. See ADR-0027.

Changes: selection became an ID list (`App.tsx`, `flowNodes.ts`); box select,
Shift/Ctrl-click, right/middle/Space pan and a pannable minimap (`Canvas.tsx`);
drag frames update a local draft and commit once on drop; shortcuts in
`useCanvasShortcuts.ts`; pure copy/paste/remove/nudge in
`equipment/clipboard.ts`; multi-selection summary with Duplicate/Delete in
`Inspector.tsx`; terminal hit zones, hover/connect feedback, wire and selection
styles in `styles/canvas-interaction.css`.

Evidence: frontend node tests 47 pass (4 new clipboard tests, selection-set
update to the canvas stability test); `npm run build` passes. Playwright on an
isolated data directory against the production build: a wire started 6 px
outside the terminal dot and released 10–20 px from the target terminal
connected; box select selected 2 and the panel showed "2 items selected";
Ctrl+C/Ctrl+V gave 4 nodes/2 wires with fresh IDs and labels CB-3, CB-4;
Ctrl+D gave 6/3; Esc cleared; Shift-click selected 2; group drag moved both
80 px; Delete removed the selection and attached wires; a junction dragged by
its centre moved without creating a wire; no page errors; selection alone did
not create a project revision.

Limitations: arrow nudges handled by React Flow (focused node or selection
box) are one undo step per press; nudges from an unfocused selection are
grouped. Clipboard is per tab (no cross-tab or OS clipboard). Owner visual
acceptance pending.
