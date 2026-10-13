# ADR-0031: Read-only live acquisition from real IEDs

Accepted: 2026-10-14. Owner decision after a plain explanation: use report blocks
like the app does today, but only free ones; preview, release on Stop, poll
otherwise, never command real IEDs. Specification: [Real IEDs](../specs/real-ieds.md).

- New native target `studio_live.exe` (same framed IPC as the worker). Its
  operation table is `hello`, `npcap_status`, `adapters`, `goose_start`,
  `goose_stop`, `connect`, `discover`, `read`, `rcb_status`, `rcb_reserve`,
  `rcb_release`, `disconnect`, `stop`. The only writes are `Resv`, `RptEna`, `GI`.
- Model discovery code moved from the probe to `native/common/` and is shared.
- The backend `acquisition/` package runs one supervised session per real IED:
  connect, discover, compare with the SCD (`domain/model_compare.py`), choose
  access (`domain/access_policy.py`), reserve free blocks, poll the rest, reconnect
  with backoff and mark values stale on loss. Values keep device quality/time and
  carry source `live_report`, `live_poll` or `live_goose`.
- Real IEDs are never started as simulated workers; operating equipment bound to
  them is refused in plain words; endpoint conflicts stop Run.
- Comparison ignores control-block service attributes (GO, GS, MS, US, RP, BR,
  LG, SV FCs) because they are not engineering data points.

Compatibility: role defaults to `simulated`, so existing projects are unchanged.
Third-party IED interoperability is not claimed; evidence uses a stand-in worker.
