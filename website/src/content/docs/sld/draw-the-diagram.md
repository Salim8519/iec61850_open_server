---
title: "Draw the single-line diagram"
description: "Open the SLD place, add breakers, disconnectors, busbars and whole bays with the Add menu, then select, move, rotate, copy and delete symbols."
section: sld
order: 1
---

The **SLD** place is the single-line diagram (SLD). It draws the substation with one line for each
circuit. Each symbol can be linked to an IED signal, so the drawing shows the live state of your
test.

## Add a symbol

1. Click the **SLD** place.
2. Press **Add**. A panel of large tiles opens.
3. Tiles are grouped by purpose:
   - **Switching**: circuit breaker (CB), disconnector (DS) and earthing switch (ES).
   - **Measuring**: current transformer (CT), voltage transformer (VT), source and measurement list.
   - **Protection & control**: relay, trip relay, push button and point panel.
   - **Power in and drawing**: infeed, tap changer transformer and junction.
   - **IEDs and GOOSE**: one IED component for each IED.
   - **Whole bays**: a complete **Line feeder**, **Transformer bay** or **Bus coupler**.
4. Click a tile to add it to the drawing, or drag it onto the drawing.

Each tile has a short hint. Hover over a whole-bay tile to read what it contains.

## Select and move

- Click a symbol to select it. Its properties open in a panel on the right.
- Drag a symbol to move it. Arrow keys nudge the selection.
- Drag a box over empty space to select several symbols at once. Hold **Shift** or **Ctrl** and click
  to add more. Then **Duplicate** or **Delete** them with the buttons in the panel.
- Move the view by dragging with the middle or right mouse button.
- Shortcuts: Ctrl+C copies, Ctrl+V pastes, Ctrl+D duplicates.

In the properties panel, **Rotate** turns the symbol by 90 degrees, and **Duplicate** copies it.
**Delete** removes it.

## Name a symbol

Change the **Name** field in the properties panel. Use the name of the bay or switch in the real
substation, for example `Q1`. The name is shown on the drawing.

## Undo and Redo

- **Undo** takes back your last change to the drawing.
- **Redo** brings it back.
- The drawing is locked while IEDs run. Press **Stop** to change it.

## Tidy the layout

**Tidy up** arranges the whole drawing like a standard single-line diagram: busbars on top and bays
straight down. With symbols selected, the button says **Tidy selection** and arranges only those.
One Undo puts them back.

## Next

- [Build the diagram from the SCD](/docs/sld/build-from-scd/) draws the whole substation in one go.
- [Link symbols to IED signals](/docs/sld/link-to-signals/) makes the symbols show real values.
