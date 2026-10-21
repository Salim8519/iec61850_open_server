# ADR-0094: Higher-contrast controls and frames

Accepted: 2026-10-21. Owner feedback on [ADR-0034](ADR-0034-higher-contrast-surfaces.md):
the white-and-black look is right, but people still cannot tell at a glance what is a list,
a button, a field they can type in, a label or a locked value. The owner compared nine
mockups of the same screen and chose option D, "High contrast". This is a design change
only: no wording, layout, size, behaviour or data changes.

- New tokens: `--control-border` #374151 (buttons, inputs, selects, table header rule),
  `--control-border-hover` #111827, `--frame` #6B7280 (cards, list rows, choices, tiles,
  sidebar and top bar dividers), `--table-head` #E5E7EB.
- Changed tokens: `--border` #D3D8E0 → #B6BDC7, `--text-2` #3F4856 → #1F2937,
  `--text-3` #5B6472 → #374151 (still two text levels, both AA).
- Buttons: near-black outline. The primary button is the darker blue #1D4ED8 with a #1E3A8A
  outline. A danger outline uses the error colour. Text-only (ghost) buttons are underlined
  except icon buttons and the diagram toolbar, whose frame already marks them as buttons.
- Inputs, selects and textareas: near-black outline, black on hover. A locked field has a
  dotted outline on grey, never only faded (no gradient).
- Focus: a 3 px solid blue ring with a 2 px gap.
- Cards: grey frame without the soft shadow and a frame-coloured line under the heading.
  Tables: bold dark header text on #E5E7EB with a 2 px dark rule.
- Selection: list rows and choices add a 5 px accent bar inside the left edge. The sidebar's
  current place has a 4 px accent bar. The shell dividers are 2 px.

Kept unchanged on purpose: `--border-strong` #A9B1BD, which the diagram (equipment cards,
relay and measurement chips, signal-layer tags, zoom controls) uses, and every diagram
constant from ADR-0034/0045/0065. The page grey #ECEEF2, the one blue accent, 15 px text,
40/44 px controls and the status rule (words + icon + border pattern) also stay.

Compatibility: display only. No project data, API, engine or protocol change. Feature
styles that use the shared tokens follow automatically. The SNMP, gateway and explorer
styles were not edited.
