---
title: "Build the diagram from the SCD"
description: "Build the single-line diagram from the Substation section of an SCD file in one step, preview it, then apply it to your drawing with one Undo."
section: sld
order: 2
video: sld
---

An SCD file already describes the substation: its bays, switches and the IEDs that control them.
ScadaBench can draw that for you.

## Before you start

- Import the SCD file in [Devices](/docs/devices/import-cid-scd/). The Build button needs an SCD.
- Stop the IEDs if they run. The drawing is locked while they run.

## Build the diagram

1. Click the **SLD** place.
2. Press **Build from SCD**. The window **Build the diagram from the SCD** opens.
3. If the project has more than one SCD source, choose the one you want in the list.
4. Choose how to combine it with your drawing:
   - **Add beside my drawing**: keeps your drawing and places the new one next to it.
   - **Replace my drawing**: replaces your drawing with the new one.
5. Press **Preview**. ScadaBench reads the file and shows what it found.
6. Read the preview. Open each section that has items:
   - **Needs a link**: symbols that still need an IED signal. You link them later.
   - **Not drawn**: parts of the file that cannot be drawn, with the reason.
   - **Notes about the file**: anything unusual that ScadaBench noticed.
7. Press **Apply to diagram**. The whole drawing is added as one step.

If you change the project after the preview, the window says **The project changed after this
preview. Press Preview again.** Press **Preview again** before you apply.

## Undo a build

Press **Undo** once. The drawing goes back to how it was before the build.

## Tidy after a build

A built diagram can look crowded. Press **Tidy up** to arrange it in standard single-line order.
One Undo puts it back. See [Draw the single-line diagram](/docs/sld/draw-the-diagram/).
