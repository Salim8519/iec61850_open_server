# Proposed spec: IED Explorer

> **Proposed.** Planned behaviour, not a capability. Agreed with the owner on 2026-10-17
> ([ADR-0054](../../decisions/ADR-0054-tools-place.md)). Model: the Siemens IEC Browser
> polling view and the IEDScout data model browser.

## Purpose

Connect to any IEC 61850 server by address, without a project. Browse its live data model,
watch chosen values and save what was seen, without ever changing the device.

## Behaviour contract

- **Connect.** The user types an IP address (port 102 by default) or opens a result from
  Find IEDs.
  - The backend runs a tool session on `studio_live` (`connect`, `discover`, `read`).
  - One Explorer session is allowed at a time, independent of Run.
- **Tree.** Logical device → logical node → data object → attribute, with FC badges.
  - Each attribute shows its value, quality and timestamp as text and an icon; colour is
    never the only signal.
  - Large models load lazily, one logical device at a time.
- **Watch list.** Add attributes, choose a poll interval (0.5–60 s), then Start/Stop.
  - Report subscription is offered only on report blocks that `choose_access` proves free,
    and the block is released on Stop or disconnect.
  - Values carry their origin (`poll` or `report`).
- **Read-only.** Datasets, report blocks and GOOSE control blocks are shown read-only.
  - Explorer has no write or control path to any device.
  - The page says so in plain words.
- **Export.** The watch list and its values export to CSV.
- **Compare (optional).** "Compare with SCL file" uses `domain/model_compare.compare` and
  the existing plain-English hints.
- **Logs.** Events carry the host and a correlation id; values and full models are not
  logged.

## Reserved rule IDs

| Prefix | Rules (to be written with their tests) |
| --- | --- |
| `EXPLORE` | connect without a project; lazy tree; quality and time as text; watch list polling bounds; free-RCB-only reports released on stop; no write/control route exists; CSV export; SCL compare reuse |

## Open questions

- None. The owner chose read-only for real devices (real-IED access policy).
