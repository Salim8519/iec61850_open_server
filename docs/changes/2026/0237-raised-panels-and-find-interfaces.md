# 0237 Raised panels (depth and click rules); Find IEDs lists each interface once

Date: 2026-11-25. Owner: "the app does have depth ... this app just looks white"; chose concept 1
"Raised panels" and asked for rules on "what user can click and what looks info"; "the ied explorer
should be sees the interface not each interface with its ip". Decision
[ADR-0147](../../decisions/ADR-0147-raised-panels.md). Layers: frontend styles, Find IEDs
(`features/tools/find`). No API, engine, project-data or native change.

## What changed

- **Tokens** (`styles/tokens.css`): steel workspace `--bg` #D5DBE3, dark rail tokens, wells
  (`--well`, `--well-edge`), four elevations (`--shadow-raise/hover/press/panel/pop/well`); the old
  shadow names are aliases.
- **Controls** (`styles/base.css`, `ui/ui.css`): buttons are raised, lift on hover and sink when
  pressed; tabs, menu rows, segments, sidebar places, tree rows and text links stay flat. Disabled
  buttons are flat, dashed and grey with readable words instead of 45% opacity. Inputs have a light
  inner shadow; disabled inputs are wells. Segmented controls are a sunken track with the chosen one
  raised; number tiles are wells; choice cards are raised.
- **Shell** (`styles/layout.css`, `styles/workspace.css`): dark rail sidebar with the open place
  raised; raised top bar and cards; the Run/Stop button has 8 px corners (only status is round);
  tables and the device facts line are wells; diagram toolbars and the side sheet are elevated.
- **Find IEDs** (`features/tools/find/ranges.ts`, `FindTool.tsx`): the adapter list has one entry
  per connected interface, whatever number of IP addresses it has ("Ethernet 2 (2 networks)"). The
  first network fills the range; an interface on several networks shows each network as a chip
  ("10.0.0.0/24 · this PC 10.0.0.5") that fills the range. Two addresses in one /24 count once. A
  choice remembered per address before this change ("guid|ip") still finds its interface.
- Docs: `docs/design/layout.md`, `docs/design/interactions.md`, spec `find-ieds.md` FIND-02.

## Evidence

- Frontend tests 516/516 (new: interface grouping and memory fallback; rail, workspace and well
  contrast; disabled is dashed not faded; Run/Stop not round). `tsc --noEmit` clean.
- Chromium on the built frontend with the API mocked from `engine/internal/runtime/testdata/project.json`
  (the Go server needs Windows for its lock): Devices, SLD and Tools render with the rail, raised
  panels and wells; Find IEDs offered `Ethernet 2 (2 networks)`, `Wi-Fi`, `This PC` for an Ethernet
  adapter with 10.0.0.5, 10.0.0.21 and 192.168.1.7, the saved `eth|10.0.0.21` choice reopened
  Ethernet 2, and the second chip set the range to 192.168.1.0/24.

## Limitations

- Not yet seen in the real app on Windows with a running engine, nor at 80% fit.
- Components that set their own colours or borders were not reviewed one by one; any that look out of
  place under the new rules get fixed as found.
- The device "re-import CID" button the owner mentioned does not exist yet; the owner chose to skip it
  for now.
