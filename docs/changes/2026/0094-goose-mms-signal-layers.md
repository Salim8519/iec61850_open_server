# 0094: GOOSE and MMS signal layers on the SLD, GOOSE inputs, MMS client access

Date: 2026-10-18. Decisions: [ADR-0065](../../decisions/ADR-0065-sld-signal-layers.md),
[ADR-0066](../../decisions/ADR-0066-goose-inputs-and-mms-access-events.md). Specs:
[signal layers](../../specs/signal-layers.md), [GOOSE](../../specs/goose.md).
Follows change [0091](0091-goose-between-machines.md).

Owner request: see GOOSE and IEC 61850 client activity on the single-line diagram without
distraction; GOOSE identity is the control block (IED + LD + block), clients by IP.

## What changed

- **Phase 0, SCL.** One GOOSE link map per project from all stored files, IEDs deduplicated by
  name. GE `Function="Disabled"` blocks are not published and are listed as "not sent (disabled
  in file)"; vendor attributes are kept. A packed C650-style `intAddr` is read as a hint
  (publisher, MAC, type). Publishers in no file are remote. APPID 0, duplicate APPID and shared
  MAC are hints. `GET /goose/links`.
- **Phase A, UI.** SLD switches *GOOSE messages* and *Who is reading*, remembered in backend
  preferences. IED tags beside bound equipment (never saved), outside/unbound IEDs in a left
  column, readable when zoomed out. One pulse per real stNum change (max one per link per
  second), hollow "(expected)" pulses without a listener, persistent problems in words, static
  highlight with reduced motion. Tag/link → side sheet with signal-flow details. Devices list
  hint and Overview tables *GOOSE partners* and *Read by clients*. The SLD toolbar now wraps.
- **Phase B, engine.** A simulated subscriber applies received GOOSE to the attributes its own
  ExtRefs bind (`intAddr` resolved to exactly one runtime-editable point), through the normal
  sample path; any listener problem or dataset size mismatch refuses the frame.
- **Phase C, engine.** `studio_ied` reports the client `ip:port` on connection, report-block and
  control events, and a per-second `mms_access` summary (reads per logical node, commands,
  enabled report blocks). Reads are observed by wrapping libiec61850's own access check, so
  answers are unchanged. ScadaBench's own command client is not shown as external.

## Evidence

All on this PC; public samples only (`scd/open_substation.scd`, `scd/model_substation.scd`)
for anything shown; the owner's private vendor files were used only in local tests.

| Check | Result |
| --- | --- |
| Native build (worktree `studio/native/build`, VS 2019) | `studio_ied`, `studio_live`, `studio_probe`, `studio_control` built without new warnings |
| IED to IED over the wire (VMware VMnet1, `test_goose_inputs_wire.py`) | IED2_PTOC published `gcbEvents`; `studio_live` received it; IED1_XCBR's `CILO1.EnaCls.stVal` read **false → true** with `studio_probe` (independent MMS client) after the publisher's `PTRC1.Tr.general` went true |
| tshark on VMnet1 during that run | 3 frames: dst 01:0c:cd:01:00:02, VLAN 1 prio 4, APPID 0x1002, gocbRef `IED2_PTOCGenericIO/LLN0$GO$gcbEvents`, confRev 2, stNum 1→2 with allData `[False, False]` → `[False, True]` (capture SHA-256 4cadd814…ca25, kept locally) |
| confRev mismatch | listener expecting confRev 3: only `goose_input_refused` (conf_rev_mismatch); subscriber value unchanged |
| MMS answers unchanged | reading three nonexistent objects gives data access error 10 from the build **without** the observer (main checkout) and the build **with** it (identical) |
| MMS reads from an independent client | `studio_probe` reads appear in `mms_access` per logical node; report block enabled by the client listed; control carries the peer and counts as a command; ≤ 8 summaries in the test |
| App walkthrough (temporary data folder, `model_substation.scd`) | switch state survived a reload; tags beside FEED1/FEED2/TR1/TR2 breakers; changing TR1 `XCBR1.Pos` gave "TR1 → BUS1 · XCBR1.Pos Closed (expected)" hollow pulses to BUS1 and BUS2; FEED1 tag "◉ 1 read" with hover "127.0.0.1 (this PC): reads LLN0, XCBR1, XSWI2 …; reports on LLN0.ReportControl_00101"; external control showed "127.0.0.1 operated CSWI1.Pos on FEED1"; Devices rows "↑ sends to 2" / "↓ gets from 4" and BUS1 GOOSE partners table ("Not watched" in loopback mode) |
| Private vendor files (local only) | every file reads; all bound inputs of the bay units resolve to their own data points (14 in one unit); disabled blocks never sendable; remote publishers found from packed inputs; APPID 0 reported as a hint |
| Suites | unit + integration 401 passed, 14 skipped; protocol with `STUDIO_GOOSE_ADAPTER` = VMnet1: 51 passed, 24 skipped (gateway, master and SV test-publisher programs were not built in this worktree; those areas are untouched); frontend `node --test` 129 passed; `tsc` clean; `verify.py` PASS |

## Limitations

- No physical switch or VLAN trunk: VMware strips VLAN tags on receive (change 0091).
- No listen-only lab mode yet: GOOSE is watched only while at least one IED is simulated in lab
  mode, so the SLD shows "expected" pulses or "Not watched" otherwise.
- No Sampled Values on the SLD; no third-party GOOSE publisher or subscriber in this change.
- Data-attribute writes (SP/CF/DC) are not counted as client writes; controls and report
  blocks are.
- A short-lived client that connects, reads and leaves within one second never appears as a
  reader (the list shows connected clients).
- An `intAddr` that is not a data point of the subscriber (for example `Op1`, `PDIF_in_swi_0`
  in the public sample) is shown but not applied, so the sample station's BUS IEDs do not
  change their own data on GOOSE.
- The read observer depends on the private MMS server layout of the pinned libiec61850 commit.
- Screenshots were checked in the in-app browser; none are stored in the repository.
- The packaged EXE was not built (not requested).
