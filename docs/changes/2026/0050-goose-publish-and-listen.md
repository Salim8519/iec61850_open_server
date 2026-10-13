# 0050: GOOSE publishing, listening and supervision

Date: 2026-10-14. Owner request: StationScout-comparable item 1 (GOOSE), with
Npcap installed by the user. Decision: [ADR-0030](../../decisions/ADR-0030-goose-with-user-installed-npcap.md).
Specification: [GOOSE](../../specs/goose.md).

Cause: GOOSE was forced off in `native/CMakeLists.txt`; the importer only warned
that GSEControl was retained but not active; no receiver existed.

Change:
- Native: pinned Npcap SDK injection, delay-loaded Npcap, `worker/goose/publishing.cpp`,
  stdout protection in `worker/ipc/framing.cpp`, `live/goose_traffic.cpp` receiver.
- SCL: `scl/communication.py` (ConnectedAP address, GSE), `scl/goose_controls.py`
  (GoCB + `GC`/`PA` lines), `scl/goose_links.py` (ExtRef subscriptions, LGOS).
- Backend: `runtime/worker_settings.py`, `domain/goose_supervision.py`,
  `acquisition/goose_monitor.py`, `GET /goose`. Frontend GOOSE traffic card.

Evidence:
- Feasibility: two processes exchanged raw and libiec61850 GOOSE frames on the
  VMware VMnet1 host-only adapter (5/5 valid frames) before any Studio code.
- `test_goose_wire::test_simulated_publisher_reaches_listener` with
  `STUDIO_GOOSE_ADAPTER={15A6105F-A715-444F-9D7C-AA259A2EED8A}`: IED1_XCBR from
  `scd/open_substation.scd` published `gcbEvents`; `studio_live` received confRev 2,
  goID `events`, dataset `IED1_XCBRGenericIO/LLN0$Events`; writing Pos.stVal=2
  raised stNum and delivered Dbpos 2. Passed on the isolated and normal builds.
- Import correctly diagnoses that IED1's ExtRef expects `IED2_PTOC/GenericIO/GoCB`,
  which the sample SCD does not define.
- Normal build SHA-256: `studio_ied.exe` 0a71bef8…f98b, `studio_live.exe`
  a8fd8a2e…7c337, `studio_probe.exe` 54ac9ce8…92c9. Existing 22 protocol tests pass.

Limitations: only virtual-adapter evidence on one PC; no physical switch, VLAN
priority or third-party subscriber/publisher yet. Pcap device indexes are resolved
just before use and could change if adapters appear in between. Opening a pcap
adapter can take seconds, so GOOSE becomes active shortly after Run.
