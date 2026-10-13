# 0048: Junction body easy to grab

Date: 2026-09-29. Owner report: the junction body is hard to catch for moving;
its terminals and body conflict. Amends ADR-0027.

Cause: the junction box was 22 px with four terminal hit zones of 14 px, which
left about an 8 px free square to start a move.

Change: the junction box is 40 px (transparent, move cursor, soft hover and a
visible selected outline). `JunctionBody` draws the dot plus stubs only to
terminals that carry a wire (`junctionSides.ts` via React Flow
`useNodeConnections`), so wires still meet the dot. Terminal hit zones stay
small at the box edge; the centre is the drag grip. Saved projects are
unchanged; an existing junction's dot appears 9 px lower-right of its old
centre because positions are the box's top-left corner.

Evidence: frontend node tests 51 pass (new `junctionSides` test); build
passes. Playwright on an isolated copy of the owner's autosave (zoom 0.68):
drag started at a 5×5 grid over the junction box moved it from 20/25 points;
the 5 others are the terminals themselves; every drag undid cleanly and all 8
wires remained. Screenshot shows stubs only on wired sides and the selected
outline.

Limitations: owner visual acceptance pending.
