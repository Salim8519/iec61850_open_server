# ADR-0034: Higher-contrast surfaces and diagram

Accepted: 2026-10-14. Owner feedback on design v2 ([ADR-0032](ADR-0032-clean-four-place-ui.md)):
the user "sees white most of the time", borders are missing and the single-line
diagram has low contrast. A reference screenshot showed framed cards with visible
1.5 px outlines on a soft grey background.

- Background `#ECEEF2`; sidebar `#F7F8FA`; white cards, sheets and list rows on top.
- Borders `#D3D8E0` (cards, rows) and strong `#A9B1BD` (buttons, inputs, popovers), drawn
  with the shared `--line` token (1.5 px). Hover darkens borders to `#8F98A6`.
- Text `#0F172A`, secondary `#3F4856`, tertiary `#5B6472` (all AA on white and background).
- Card and sheet headings sit on `--surface-2`; table headers on `#EEF1F5` with a strong rule.
- Selected nav item, segmented option and list row are outlined in the accent.
- Diagram: canvas `#E9EDF2` with darker dots `#B4BCC8`; wires `#1F2937` 2.5 px; symbol
  strokes 2.2; equipment cards outlined `#7D8796` 1.5 px with 12 px bold labels; state
  pills, relay panels and terminals use darker outlines.

Compatibility: display only. No project data, API or behaviour change. Status still
uses words + icon + border pattern; colour only repeats it.
