# Current engineering handover

Snapshot: 2026-09-16 after analysis of `commmand_traces_5.pcapng`. This file is the
first operational context for the next LLM session; immutable details are also in
the [engineering ledger](history/INDEX.md).

## 2026-10-14 (latest): SLD Tidy up (change 0063)

- Diagram toolbar **Tidy up** / **Tidy selection** lays the drawing out like a real SLD
  (spec `docs/specs/sld-tidy.md`, ADR-0040). It is a pure frontend function: positions,
  wire sides and symbol turns only; one undo step. Owner visual acceptance pending.

## 2026-10-14 (later): clean UI restructure

- Owner found the UI messy; rebuilt as four places (Diagram, Devices, Test, Settings),
  design system v2, merged duplicates, format 12 (ADR-0032/0033, changes 0053/0054,
  spec `docs/specs/ui-structure.md`).
- 134 owner CIDs (Downloads zips, copied to the session scratchpad only) all import
  and start; PhyComAddr and unnamed report blocks fixed.

## 2026-10-14 update: station-level roadmap A-D (changes 0057-0062)

**Specs**
- The roadmap was specified first in `docs/specs/proposed/` and every spec was then
  promoted: `sld-import`, `signal-flow`, `station-health`, `sampled-values`, `test-mode`.
- Owner decisions:
  - busbars are drawn as JUNCTION and the project format stays at 12;
  - Sampled Values are listened to only.

**Features**
- **A, SLD import:** Diagram → Build from SCD.
- **C, Signal flow:** Devices → Whole station, plus an IED tab.
- **B, Station health:** Devices → Health check.
- **D1, Sampled Values:** Settings → Network.
- **D2, Test mode:** device card and badges.

**Native**
- New `studio_live` operations `sv_start`/`sv_stop`.
- The goID check and SV listener are in `studio_live`.
- An integer-write fix is in `studio_ied`: INT8/16/32 and Enum writes were silently
  ignored before.
- `studio_sv_test_publisher` is test-only and never packaged.
- All of this was built and tested in `studio/native/build/station`.

**Evidence and dev parity**
- Evidence uses Studio stand-ins and the VMware adapter only. Interoperability is not
  claimed.
- The dev backend needs the same native build: rebuild `studio/native/build/Release`
  (README) whenever no run is active.

## 2026-10-14 update: lab network, GOOSE and real IEDs

- Owner asked for StationScout-comparable items 1-3, beginner friendly with hints,
  and spec files that protect features from LLM drift. Read `docs/specs/` first.
- Implemented: changes 0049-0052, ADR-0028..0031, project format 11. The owner's
  autosave was not opened or migrated during this work (an isolated data folder
  was used); it upgrades to format 11 with a recovery archive on next open.
- Normal native build rebuilt with GOOSE and `studio_live.exe` (hashes in 0050).
  Building requires the Npcap SDK (`python studio/tools/fetch_npcap_sdk.py`).
- Evidence on this PC: VMware VMnet1 GUID `{15A6105F-A715-444F-9D7C-AA259A2EED8A}`
  carries GOOSE between processes; set `STUDIO_GOOSE_ADAPTER` to it for wire tests.
- Not done: applying the lab plan on a real adapter (UAC), real relays, and the
  packaged EXE (not built; build it only when the owner asks).

## User intent and safe boundary

- Trip Relay behavior is accepted: Operate sends `0`, Reset sends `1`, Operated
  feedback is `1`, Reset feedback is `0`.
- Diagnose and make breaker commands usable with the observed concentrator, while
  keeping standard IEC behavior for other clients and making every adaptation clear.
- Preserve C03 Y1105 engineering work, recovery archives and all supplied captures.
- Keep every browser tab on one backend/project reference and reject stale engines.
- Do not build or promote the packaged EXE unless explicitly requested.

## Saved project snapshot

- Current autosave: `studio/data/autosave.subsim`; format 8; 821,911 bytes;
  SHA-256 `AF63A518AE924B0ECBD644DFFBF30B22386829EEBEAC6B75C0B1860DB5EC45A7`.
- Project at packet diagnosis: `Untitled substation`, backend session
  `1238fd157af4440887a591d11e8bbae4`, revision 8, one IED `C03_Y1105`, 36,021
  signals, 294 controls, 48 reports and four canvas nodes.
- Backend source reload created session `265090513eff4fbb90728dc43a4e64da`, revision
  2. A post-reload API audit confirms the same IED counts, node IDs, CB mapping,
  compatibility setting and Trip settings. Runtime and IED are stopped.
- `CB-1` node `4feff52d-a01d-4a1f-a270-19bc0c79ef81` maps command
  `C03_Y1105System/Bkr0XCBR1.Pos`, direct state
  `C03_Y1105System/Bkr0XCBR1.Pos.stVal`, and currently stores
  `concen_inverted_sbow`. Its last authoritative value was Dbpos `2` (Closed).
- `TRIP_RELAY-1` maps command `C03_Y1105Master/GGIO2.SPCSO007` and state
  `C03_Y1105System/ConInGGIO1.Ind141.stVal`, with standard compatibility,
  `operate_false` command polarity and `operated_true` state polarity.

Primary pre-format-8 recovery archive:

`studio/data/recovery/before-format-8-migration-bfea448058624db39aea1912ad8a874d.subsim`

It is 821,874 bytes with SHA-256
`7321F76734989ED83950B83CC1FA4C9347D4D56C3F49E29FF68FABB2F47A7B01`.
The original format-6 C03 recovery has SHA-256
`87FB9C7B32A34CA7E19DF4EB4FDF4391ED17370770D225FF2A936423196B45F5`.

## Breaker protocol contract

- Imported `CB-1` is DPC control model 4 (enhanced SBO), timeout 30,000 ms.
- Standard command meaning is Boolean false=Open and true=Close. Feedback Dbpos is
  1=Open and 2=Closed.
- Standard enhanced SBO requires successful `SBOw(value)` followed by matching
  `Oper(value)`. Selection alone is not movement.
- The explicit `concen_inverted_sbow` profile is non-standard. Only exact origin
  `CONCEN` on mapped DPC model 4 is affected. Successful `SBOw` is applied
  immediately and inverted; a later matching `Oper` is suppressed as a duplicate.
- Normal rebuilt worker capability markers are `versioned_updates`,
  `atomic_control_feedback`, `control_selection_events` and
  `concen_sbow_compatibility`.

## Why trace 5 did not Open

Capture `commmand_traces_5.pcapng` is 9,582,264 bytes, spans 23:10:35.627709 to
23:10:50.916939 AST, and has SHA-256
`544C0C8591B354A19B6CF4AECA809C9E14499A3F9A88976A2F659C7B560B0ABD`.
Keep it local.

At 23:10:40.370307, frame 994 sends a write to
`C03_Y1105System/Bkr0XCBR1$CO$Pos$SBOw` with Boolean `false` and origin `CONCEN`.
Frame 997 at 23:10:40.370728 is the successful MMS write response. No matching
`Oper` occurs in this 15.289-second capture.

Run `28f4855e31e84f73b09e14c2caafe624` records the consequence in
`studio/data/logs/28f4855e31e84f73b09e14c2caafe624/events-0000.jsonl`, lines 194-199:

- selection accepted with `auto_applied=true`;
- raw requested value false was adapted to true;
- native action was Close and feedback value was Dbpos 2;
- `changed_count=0` because the breaker was already Closed;
- authoritative equipment state remained 2.

This is not packet loss or an MMS rejection. The enabled inversion profile caused
the failed Open. The earlier `commmand_traces_4.pcapng` was identified by the user
as a Close attempt but also used origin `CONCEN`, `SBOw(false)` and no `Oper`.
Therefore those two stated intentions are indistinguishable on the wire. No
deterministic adapter can infer both Open and Close from the same object, origin,
trigger and Boolean value.

Do not silently replace inversion with “toggle current state”: it would make a
repeated Open/Close command move in the wrong direction. The preferred correction
is for the client to send `SBOw(false)+Oper(false)` for Open and
`SBOw(true)+Oper(true)` for Close. If that cannot be changed, obtain an explicit
policy decision and record its failure mode before adding a new compatibility
profile and project-format migration.

## Implemented repository contracts

- Formats 1-7 migrate to format 8 with an exact recovery archive; non-standard
  compatibility is never enabled by migration.
- Browser tabs share visible session/revision identity, reload remote mutations and
  reject stale diagram writes. An inactive Vite origin redirects to the active one.
- SSE reconnect uses `GET /api/v1/equipment/states` for authoritative hydration.
- Future runtime evidence uses compact append-only event segments plus append-only
  run indexes; no app-owned log pruning. Legacy run folders remain untouched.
- Engineering changes use a compact chained append-only ledger with intent, reason,
  files, expected results and evidence, deliberately without a status field.

## Evidence and build identity

- Normal worker: `studio/native/build/Release/studio_ied.exe`, 495,616 bytes,
  SHA-256 `F231F1290CC2AFD356C7D5D7B46AFDC8E961887169D9CCCA8B77825A55E96CF9`.
- Normal probe: `studio/native/build/Release/studio_probe.exe`, 310,272 bytes,
  SHA-256 `908EDE15BBB817BD9F86C79C43922258E6E574FF8B2D29E697271B1A99849D52`.
- Final pass: all 126 maintained Python/native tests and all 9 frontend contract
  tests passed; the production frontend build and `studio/tools/verify.py` passed.
- The deferred CONCEN probe case passed 10 consecutive runs after its bounded poll
  fix. The worker binary was not rebuilt because its source/behavior did not change.
- No packaged EXE was built. Browser visual acceptance and third-party
  interoperability remain release gates.

## Next incremental work

1. Read [LLM start here](llm/START-HERE.md) and query history for `controls` and
   `trace-5`.
2. Restart the development session so the worker picks up change 0031
   (no-op accepted commands no longer rewrite the native attribute, timestamp or
   revision; `control_applied` carries `changed=false`; isolated build
   `studio/native/build/concentrator-feedback` verified with 21 protocol and
   104 unit/integration tests). Reconfirm the autosave hash before mutation.
3. In Equipment Properties set CB-1 compatibility back to `standard` (run-log
   evidence: the concentrator completed standard `SBOw`+`Oper` sequences for
   Close in runs `bd5e8ed35b50490ba593111c8e76096c` and
   `a7fc32421c6846daa5b53af67aa46568`; the inversion profile consumed the Open).
4. Capture one complete Open and one complete Close from the concentrator with
   the standard profile and the fixed worker. Open attempts never received an
   `Oper` in recorded runs; the fresh capture decides standard-sufficient or an
   explicitly bounded policy.
5. Add independent probe tests for both directions, repeated commands, no-op commands,
   selection timeout/disconnect and reconnect hydration (no-op coverage added in 0031).
6. Update ADR/change record/history ledger, run focused tests and
   `python studio/tools/verify.py`.
