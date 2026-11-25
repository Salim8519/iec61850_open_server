# ADR-0147: Raised panels: depth and a visual grammar for what can be clicked

Accepted: 2026-11-25. Owner, 2026-11-25: "the app does have depth ... this app just looks white"; from
three concepts the owner chose "Raised panels", then asked for rules on "what user can click and what
looks info". Change [0237](../changes/2026/0237-raised-panels-and-find-interfaces.md). Amends
[ADR-0032](ADR-0032-clean-four-place-ui.md) and [ADR-0094](ADR-0094-higher-contrast-controls.md).

## Problem

The sidebar, top bar, cards, sheets and tables were all #FFFFFF on #ECEEF2, so the whole app sat on
one flat sheet. Buttons, list rows, read-only values and status pills were told apart mainly by
outline, and a round Run/Stop button looked like the round status pills next to it.

## Decision

Five tone levels and four elevations, with no gradients:

| Level | Tokens | Use |
| --- | --- | --- |
| Frame | `--surface` + edge shadow | Sidebar: white, with a soft shadow on its right edge (owner, change 0238) |
| Workspace 0 | `--bg` #D5DBE3 | Steel grey behind panels; never holds content |
| Panel 1 | `--surface` + `--shadow-panel` | Cards, top bar, sheets |
| Well | `--well` #F4F6F9, `--well-edge`, `--shadow-well` | Tables, facts lines, number tiles, read-only fields |
| Floating 2/3 | `--shadow-pop` | Menus, diagram toolbars, popovers, dialogs |

The rules (also in `docs/design/interactions.md`):

- R1 Raised means press: controls carry `--shadow-raise`, lift on hover (`--shadow-hover`) and sink
  when pressed (`--shadow-press`).
- R2 Sunken means read: values and tables sit in wells.
- R3 Flat means label: headers, captions, tabs, menu rows and text links have no shadow.
- R4 Round means status: only status pills are fully round; the Run/Stop button has 8 px corners.
- R5 Blue means you can act: primary, selection, active tab, links, focus and the pen mark.
- R6 State is words, icon and border pattern (unchanged from ADR-0032).
- R7 One filled primary per area; danger is outlined red.
- R8 Controls that send to an IED or the network keep the double border of command badges.
- R9 Hover and the hand cursor only on things that can be clicked.
- R10 Disabled is flat, dashed and grey with readable words (was 45% opacity); the reason is written.
- R11 Editable fields are white with an inner shadow; read-only fields are wells.
- R12 Only the four elevation tokens make shadows. `--shadow-small`, `--shadow` and `--shadow-large`
  stay as aliases so every existing component follows.

## Amendments

- Change 0238 (owner, 2026-11-25: "the side bar should be kept white to avoid huge changes"): the
  sidebar is white again; it gets a soft edge shadow and the open place a raised frame. The rail
  tokens are removed. R8 is applied with `button.wire` (double border) on Operate, Open/Close,
  raise/lower, point commands, Explorer, SCADA master and real-device sends. R9: table rows no longer
  change on hover, because no row is clickable as a whole. Dialogs carry `--shadow-pop`.

## Compatibility

CSS only: tokens, base controls and the shell. No component logic, API, project data or engine
change. Status, point-type and equipment colours are unchanged, so the AA contrast test still holds.
Card frames keep #6B7280 (ADR-0094). The
packaged app and development read the same stylesheets.
