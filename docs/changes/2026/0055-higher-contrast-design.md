# 0055: Higher-contrast design and diagram

Date: 2026-10-14. Owner feedback: the app looks mostly white, borders are missing and
the single-line diagram has low contrast. Decision:
[ADR-0034](../../decisions/ADR-0034-higher-contrast-surfaces.md).

Change (styles only, plus three diagram constants):
- `styles/tokens.css`: grey background, darker text and borders, `--line`, `--hover`,
  canvas tokens.
- `styles/base.css`, `layout.css`, `workspace.css`, `ui/ui.css`: 1.5 px outlines on
  buttons, inputs, cards, list rows, popovers and toolbars; tinted card/sheet headings
  and table headers; accent-outlined selection.
- Diagram: `equipment-node.css`, `relay-panel.css`, `canvas-interaction.css`,
  `Canvas.tsx` (wire `#1F2937` 2.5 px, darker dots), `EquipmentSymbol.tsx` (stroke 2.2).

Evidence: browser check of Diagram, Devices, Test and Settings at 1568 px wide on the
isolated data folder; frontend tests 59 pass (contrast test included), build passes.

Limitations: owner visual acceptance pending; edges saved with an older inline colour
keep it until redrawn (new and unstyled wires use the darker colour).
