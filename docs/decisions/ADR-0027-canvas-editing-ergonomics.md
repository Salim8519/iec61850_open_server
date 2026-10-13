# ADR-0027: Canvas editing ergonomics — multi-select, clipboard, terminal targets

Accepted: 2026-09-29. Owner request: make the design area snappier and easier
to select, copy, modify, paste and duplicate, and make terminals (anchors)
much easier to hit with the mouse without hurting other gestures.

- Selection is a list of node IDs owned by `App`. The canvas highlight and the
  properties panel still project the same selection: one ID shows that
  object's properties; several show a count with Duplicate and Delete.
- Pointer model follows design tools: left-drag on empty canvas draws a
  selection box (partial overlap selects); Space+drag, middle- or right-drag
  pans; Shift/Ctrl-click toggles an object; the minimap pans and zooms.
- Keyboard (Substation tab, not while typing or in a dialog): Ctrl+A, Esc,
  Ctrl+C, Ctrl+X, Ctrl+V, Ctrl+D, Delete/Backspace, arrows (20 px, Shift
  100 px), Ctrl+Z/Y. Copy and select work while running; every structural
  shortcut follows the existing runtime lock.
- Paste/duplicate (`equipment/clipboard.ts`) creates fresh node and wire IDs,
  keeps only wires whose both ends were copied, offsets by 40 px per paste and
  gives each copy a unique label (`CB-1` → next free `CB-n`, otherwise
  `<name> copy`). All other data, including signal mappings, is copied
  exactly as the existing single-object Duplicate already did.
- Terminals keep their 9 px dot but get a 25 px invisible hit zone (junctions
  14 px so their centre stays draggable), grow on hover, and the connection
  snap radius is 36 px. Wires have a 24 px pick width and a distinct selected
  style. The group-selection rectangle no longer covers terminals.
- Drag frames move a local canvas draft; the diagram, undo history and save
  update once on drop, so a drag is one undo step and one save.

Compatibility: no contract, project-format, backend or native change. Left-drag
on empty canvas no longer pans (use Space, middle or right drag). The internal
clipboard lives in the browser tab only.

Amendment (2026-09-29, change 0048): junctions use a 40 px transparent drag
grip with stubs drawn only to wired terminals, replacing the 22 px box whose
terminal zones left almost no body to grab.
