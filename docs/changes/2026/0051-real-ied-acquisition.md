# 0051: Read-only live acquisition from real IEDs and SCD comparison

Date: 2026-10-14. Owner request: StationScout-comparable item 2; owner approved
"reports on free blocks, polling otherwise, no commands" after a plain explanation.
Decision: [ADR-0031](../../decisions/ADR-0031-read-only-live-acquisition.md).
Specification: [Real IEDs](../../specs/real-ieds.md).

Change: `studio_live.exe` (`live/mms_session.cpp`, `live/reports.cpp`,
`common/mms_json.cpp`, shared `common/browse.cpp`/`type_tree.cpp`), backend
`acquisition/` (`real_session.py`, `live_values.py`, `manager.py`),
`domain/model_compare.py`, `domain/access_policy.py`, `domain/live_value.py`,
`api/live.py`, role-aware coordinator start, refusals for commands and pulses to
real IEDs, and the Real IEDs tab.

Evidence:
- `test_live_client` against a separate `studio_ied` stand-in: discovery compared
  with the SCD shows 0 missing and 0 type mismatches; read returns typed values
  and an access error for an unknown point; no write operation exists; an enabled
  block is reported busy and not taken; a reserved free block delivers reports and
  is free again after disconnect.
- Unit tests for comparison, access policy, roles, value conversion and stale marking.
- Browser on an isolated data folder: IED2_PTOC set to *Connect to real device*
  against a stand-in on 127.50.0.240:102 reached Live; comparison showed 143 equal
  points and no differences after control-block attributes were excluded; "Read
  these now" displayed device values with Good quality.

Limitations: evidence uses Studio's own server as the device; real relays,
vendor-specific RCB ownership and Ed1 devices without `Owner` remain release gates.
Polling covers only mapped/test-panel points plus on-demand reads.
