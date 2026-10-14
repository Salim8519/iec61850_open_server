# Spec: Tidy up the single-line diagram

Decision: [ADR-0040](../decisions/ADR-0040-sld-tidy-up.md). Change: [0063](../changes/2026/0063-sld-tidy-up.md).
Owner request (2026-10-14): a smart "tidy up" that turns a random diagram into a
clear one that looks like a real single-line diagram, for engineers with no time.

## Purpose

One press of **Tidy up** in the Diagram toolbar rearranges the drawing the way an
engineer draws an SLD: busbars across the top, each bay hanging straight down from
them, straight wires, and relay panels and buttons parked to the side. Nothing about
the equipment, its IED links or its wiring changes, and one Undo puts it back.

## Behaviour contract

**What may change:** item positions, which side of an item a wire attaches to (and
which end is the wire's source), and the turn of a symbol (`rotation`). Nothing else:
no item or wire is added, removed, renamed, rebound or reconnected.

**Layout** (pure function `equipment/tidy/tidy.ts::tidyDiagram`)
- Wired items form groups (things connected by wires). Groups sit side by side, left
  to right in the order they were drawn.
- **Busbars:** junctions wired to each other form one bar; a lone junction with four or
  more wires is a bar point. The bar with the most connections is the main busbar, on
  top. Another bar is stacked right under it when two or more of its bays reach a top
  bar through a neighbouring bay (two or three wires: a double busbar or bus coupler).
  A bar right under a top bar (one wire, a bay tee) is not stacked. A group without
  bars starts at the end of its longest run that was drawn highest.
- **Rows:** every item sits at its wire distance from the top bars (junction-to-junction
  wires count as zero). A whole bar always shares one row. Bars at the same depth (bay
  tees) share one row.
- **Columns:** sweeps pull each item under (or over) what it is wired to and reorder
  rows to remove crossings. A run of equipment holds its column; an in-line end (feeder
  end) beats a side branch (earth switch, VT), which steps aside. A tee wired to two
  disconnectors is centred between them. Junction rows finish exactly under what
  feeds them.
- Equal pulls keep the user's left-to-right drawing order. Centres snap to the 20 px
  grid; there is clear space between neighbours and rows.
- **Wires:** items in the same row join right side → left side; items in different rows
  join bottom → top.
- **Symbols:** CB, DS, ES, CT, VT, analog source and trip relay become upright on
  vertical-only wiring and turn a quarter on horizontal-only wiring; mixed or no wiring
  keeps the turn.
- **Parking:** items without wires (relay panels, push buttons, loose symbols) stack in
  columns to the right of the drawing, panels first.
- The result keeps the drawing's top-left corner where it was.
- The layout repeats internally (at most four passes) until it settles, so pressing
  Tidy up again on its result changes nothing.

**Interface** (Diagram place; UI-01 unchanged)
- Toolbar button **Tidy up** next to *Build from SCD*. With two or more items selected
  it reads **Tidy selection**: only those items move, anchored where they were, and only
  wires between them are re-sided.
- Disabled while running, with the reason as its tooltip; disabled with fewer than two items.
- Applies through the normal diagram edit: one undo step, revision-guarded save.
- Shows a plain status line: "Tidied N items. Press Undo to put them back." or "The
  diagram is already tidy." After a change, the view refits the drawing.
- Uses the measured size of each symbol from the canvas, and stylesheet sizes before
  the canvas has measured one.

## Out of scope

- Adding junctions or splitting wires to make a busbar straighter.
- Orthogonal wire routing around symbols (wires stay React Flow smooth-step lines).
- Bay or voltage-level labels.
- Guaranteed crossing-free layouts for arbitrary meshes.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| TIDY-01 | Tidy never adds, removes, renames, rebinds or reconnects items or wires; only positions, wire sides and symbol turns change. | `studio/tests/frontend/sldTidy.test.ts::tidy never adds, removes or relinks anything` |
| TIDY-02 | No two items overlap after tidy. | `studio/tests/frontend/sldTidy.test.ts::no items overlap after tidy` |
| TIDY-03 | The main busbar is one straight top row with its bays below; wires between rows run bottom → top and wires in a row run right → left. | `studio/tests/frontend/sldTidy.test.ts::busbars are a straight top row and bays hang below` |
| TIDY-04 | A run of equipment is one straight column; an earth switch or VT branch steps aside. | `studio/tests/frontend/sldTidy.test.ts::a bay is one straight column with its earth switch beside it` |
| TIDY-05 | A double busbar stacks both bars on top and centres each bay tee between its two disconnectors; bay tees are not stacked as busbars. | `studio/tests/frontend/sldTidy.test.ts::double busbar stacks both bars and centres each bay tee` |
| TIDY-06 | Unwired items are parked to the right of the drawing, relay panels first. | `studio/tests/frontend/sldTidy.test.ts::unwired items are parked to the right, panels first` |
| TIDY-07 | Tidy selection moves only the selected items, keeps them where they were and leaves wires to other items unchanged. | `studio/tests/frontend/sldTidy.test.ts::tidy selection moves only the selected items` |
| TIDY-08 | The same input gives the same output, and tidying a tidied diagram changes nothing (no save, no revision). | `studio/tests/frontend/sldTidy.test.ts::tidy is deterministic and a second tidy changes nothing` |
| TIDY-09 | Symbols turn upright on vertical wires; unwired items keep their turn. | `studio/tests/frontend/sldTidy.test.ts::symbols turn to follow their wires` |
| TIDY-10 | Tidy is one undo step through the normal diagram edit, is disabled while running, and its wording names Undo. | `studio/tests/frontend/sldTidy.test.ts::tidy is one undo step and locked while running` |

## Acceptance evidence

Recorded in change [0063](../changes/2026/0063-sld-tidy-up.md): the
`model_substation.scd` diagram (90 items, 90 wires) with every position scrambled
tidies into two MV boards and a 50 kV double busbar with straight bays in one press,
in an isolated browser instance. 400 random diagrams gave no overlaps and no changed
wiring. Pending: the owner's visual acceptance.
