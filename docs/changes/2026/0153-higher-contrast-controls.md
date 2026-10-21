# 0153: Higher-contrast controls and frames (design only)

Date: 2026-10-21. Owner request: clearer visibility and separation so people can tell lists,
buttons, inputs, labels and locked values apart, without an extreme change. The owner picked
option D of nine browser mockups. Decision: [ADR-0094](../../decisions/ADR-0094-higher-contrast-controls.md).
The design history was read first (ADR-0032, ADR-0034, layout.md, changes 0054, 0055, 0097),
and its fixed rules are kept.

## Change (styles only)

- `styles/tokens.css`: new `--control-border`, `--control-border-hover`, `--frame` and
  `--table-head`; darker `--border`, `--text-2` and `--text-3`.
- `styles/base.css`: buttons, inputs and selects use the near-black outline. Darker primary
  button, danger outline in the error colour, underlined text-only buttons (not icon
  buttons), dotted locked fields, solid 3 px focus ring, bold table header with a 2 px rule.
- `styles/layout.css`: 2 px sidebar and top bar dividers, accent bar on the current place,
  grey card frames without the shadow, frame line under card headings.
- `styles/workspace.css`: framed list rows, accent bar on the selected row, no underline in
  the diagram toolbar.
- `ui/ui.css`: framed choices and tiles, accent bar on the selected choice.

Not touched: the diagram drawing (`--border-strong` and the diagram constants), wording,
layout, sizes, behaviour, and the SNMP, gateway and explorer style files.

## Evidence

- Browser, dev app (Go engine), 1366x768 and 1920x1080: Devices, SLD, Test, Settings, Tools
  list, IED Explorer, Find IEDs, SCADA gateway, SNMP devices, GOOSE & SV traffic. The
  clipped-text check (`studio/tools/clip_check.js` selectors) found 0 clipped on all 10
  screens at both sizes.
- Frontend tests 267/267, including the AA contrast test in `uiStructure.test.ts`. `tsc` clean.
- `python studio/tools/verify.py`: all sections PASS.

## Limitations

- Owner visual acceptance is pending.
- The diagram's equipment sheet was not opened with a selected symbol. It uses the same
  shared controls.
- The packaged EXE was not built (not requested).
