# 0193: The lab network is a choice in the network card list

Date: 2026-10-26. Owner: the separate lab card is bad design; use the network card drop-down list,
highlight loopback and give the lab network another colour; "send design first", then "build it and
make new video"; keep version 0.1.28. Decision: [ADR-0121](../../decisions/ADR-0121-network-card-list.md).

## Change

| File | Why |
| --- | --- |
| `frontend/src/features/network/NetworkCardPicker.tsx`, `picker.css` (new) | listbox drop-down: "On this PC" group (loopback blue, lab teal, icon and tag each), real cards below, virtual ones lighter; keyboard and outside-click like `Menu` |
| `features/network/NetworkSetup.tsx` | uses the picker; picking the lab shows the lab panel with Cancel in the same row |
| `features/network/LabNetwork.tsx`, `lab.css` | from a card to a panel: `choose` (line, folded addresses, create/repair/use) and `active` (folded addresses in the banner); Use is hidden while in use; the list closes after the work, not inside it |
| `features/settings/NetworkSettings.tsx` | separate lab card removed; addresses in the banner on the lab network; button "Change network" |
| `copy/labNetwork.ts` | picker labels and tags, first-time line |
| `marketing/videos/lessons_lab.py` | the lesson picks the lab network from the list |

## Evidence (dev app, Go engine, demo station, 6 IEDs)

- From "This PC only": Put my IEDs on a network, the list opens with both rows highlighted and
  tagged; picking ScadaBench Lab shows "Ready: 6 IEDs …" in about 0.1 s; Use switches the banner to
  the ScadaBench Lab network in 0.2 s with no Windows prompt; the banner shows the 6 addresses, Ping
  and the remove link. Keyboard: ArrowDown opens, arrows move, Enter picks, Escape closes.
- At 1366x768 nothing is clipped (screenshots of the list, the lab choice and the banner).
- Frontend 488/488 (the display-fit test caught a `70vh` in the first picker CSS; removed).

## Limitations

- Mouse clicks in the browser pane by screenshot coordinates landed on another row because the
  pane's screenshot frame is scaled; clicks by element pick the right row (test tooling, not the app).
