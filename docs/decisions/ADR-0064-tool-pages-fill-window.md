# ADR-0064: Tool pages and Devices fill the window; one filter bar on every list

Accepted: 2026-10-18. Owner request: the tool pages (SCADA gateway & master, Find IEDs, IED
Explorer) and the Devices area need a lot of page scrolling before the list is visible. The owner
chose two mockups: a settings column on the left with the list filling the rest (B), and results
directly under a one-row form (C), plus a filter bar on every list. Change:
[0093](../changes/2026/0093-tool-layout-and-filters.md). Spec: [tool layout](../specs/tool-layout.md).

## Why

- `.page` was the only scroll container and `.content > *` never shrank, so every card stacked at
  its full height. An idle SCADA tool showed the role card, a side diagram, the full settings,
  Advanced and the firewall card before its tabs. The fill layout existed only while running.
- `.table-scroll` used a fixed `max-height: calc(100vh - 330px)`, which gave a second scroll bar
  inside an already scrolling page.
- Find IEDs results sat under the whole form card, Explorer's watch list sat under the node, and
  the Devices list scrolled away with the page.
- Long beginner paragraphs pushed the lists down.

## Decision

- **Fill the window (replaces "pages scroll inside a centred column" for these pages).** An open
  tool (except GOOSE & SV traffic, which is a report page) and Devices use `content fill-page`:
  the page height is the window height and the page itself never scrolls. The window is split into
  panes (`styles/fill.css`): `fill-split` (side column + main), `side-pane` (a column with its own
  scroll), `fill-panel` + `panel-body` (a card whose table or log takes the remaining height), and
  `fill-column` / `fill-grow` for stacked panes. The split row is `minmax(0, 1fr)`, so content can
  never stretch it. Scroll panes contain their absolutely placed children (screen-reader labels).
  The column is centred and at most 1600 px wide.
- **Narrow windows fall back.** Below 1000 px wide or 620 px high the panes stack, the page
  scrolls again and each table is capped at 70 % of the window height. Nothing is ever cut off.
- **SCADA gateway & master.** The header row holds the title, the Act as switch, a "?" and a
  one-line "SCADA ⇄ ScadaBench ⇄ IEDs" caption (the box diagram is gone). The left column holds the
  settings, Advanced, the compact "IEDs are not running · Run the IEDs" line and, kept in view at
  its bottom, the status, inline errors and Start/Connect. The converter shows "1 · Read the
  device" above "2 · Serve to SCADA", each with its own status and button. The tabs fill the rest.
  The running-only status bars are removed: the same column shows both states.
- **Find IEDs** is one toolbar row and a thin progress line; results fill the window, with an
  empty state that says what will appear. **IED Explorer** is a toolbar row, the model tree in the
  left column and the node detail with the watch list pinned below it, each with its own scroll.
- **Devices.** A compact, searchable IED list in its own scrolling column (row in
  `DeviceListRow.tsx` with a `hints` slot), a one-line detail header (name, "N points · N controls
  · N notes", ?, Simulate/Real device/Off), a thin stats line in Overview and a Data points tab that
  opens straight to its full-height table.
- **Hints stay, but behind "?".** Long explanations moved into `HelpTip` (a "?" with a bubble that
  is placed on the window, so scroll panes never cut it off). Empty states keep their plain text.
- **One filter bar.** `ui/FilterBar.tsx` (search, one-of chips with counts, toggles, dropdowns,
  "N of M · Clear") on every list. Filtering is display-only on rows the page already has, except
  the Data points tab, which sends its choices to the existing signals endpoint (below).
- **Remembered filters.** Chips and dropdowns of each list are saved by the backend
  (`/preferences`, `list_filters`) like the guided-tour state, because the packaged app's address
  and port can change between launches. The typed search and Traffic's Pause are not remembered.
- **Compact notices.** The migration and recovery notices are one dismissible line; their details
  open over the page.

## Compatibility

- No project format, protocol, native worker or simulation change. Run, gateway, master and
  Explorer behave exactly as before; only where things are shown changed.
- Backend additions are read-side only: `GET /ieds/{id}/signals` accepts `ld`, `bad_quality` and
  `in_dataset` and returns `devices`; `/preferences` accepts and returns `list_filters`
  (validated, at most 40 lists of 16 short values; damaged entries are ignored). Old clients that
  never send them see identical answers plus the extra field.
- Development and the packaged app share the same code, defaults and stored preferences.
- `GatewayStatusBar.tsx` and `MasterStatusBar.tsx` are removed; their content lives in the column.
- The design rule "Pages scroll inside a centred 1440 px column" still applies to the SLD, Test,
  Settings, the Tools list and the GOOSE & SV traffic page.
