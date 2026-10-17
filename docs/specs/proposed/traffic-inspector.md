# Proposed spec: Traffic inspector extensions

> **Proposed.** Planned behaviour, not a capability. Agreed with the owner on 2026-10-17
> ([ADR-0054](../../decisions/ADR-0054-tools-place.md)). It extends the delivered Tools card
> "GOOSE & Sampled Values traffic" ([tools-page](../tools-page.md)).

## Purpose

Explain what a GOOSE or Sampled Values stream did over time, not only its current state.

## Behaviour contract

- **Per-stream timeline.** It records stNum/sqNum changes, retransmission spacing, late or
  missing frames against the time allowed to live, and the simulation/test bits.
  - Detection runs in the native listener and the backend (`goose_supervision.py`,
    `sv_supervision.py`). The browser only displays it.
- **SCL match.** Mismatches against the SCD are shown with the existing plain-English
  reasons (`goose_consistency.py`).
- **Recording.** "Record to file" writes a pcap of the watched streams when Npcap is
  installed. Otherwise the button is disabled with the reason.
- **Limits.** Timelines are bounded per stream, and the oldest entries drop first with a
  visible notice.

## Reserved rule IDs

| Prefix | Rules |
| --- | --- |
| `INSPECT` | timeline fields; late/missing detection in backend only; SCL match reuse; pcap only with Npcap; bounded history with notice |
