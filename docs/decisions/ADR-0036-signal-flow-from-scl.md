# ADR-0036: Station signal-flow graph from SCL

Accepted: 2026-10-14. Specification: [Signal flow](../specs/signal-flow.md).

Engineers need to see who sends GOOSE and reports to whom without reading XML.
Studio already resolved GOOSE `ExtRef`s for simulated subscribers, but only for loaded
IEDs, and it never read report clients.

**Decision**
- **Whole file, not only loaded IEDs:** the graph is built from the original SCL stored
  with the project and covers every IED in that file. Loaded IEDs are marked; the others
  read "in SCD only".
- **GOOSE edges** reuse `scl/goose_links.py`. Each input now records how it was resolved:
  `named` (`srcCBName`) or `data` (FCDA match). There is still exactly one resolver.
- **Report edges** come only from `RptEnabled/ClientLN` (`scl/report_clients.py`).
  Clients outside the file are info notes.
- **Code layout:** `domain/signal_flow.py` is pure and deterministic, the browser only
  draws the API graph, and `projects/scl_sources.py` now owns stored-source lookup for
  both SLD import and signal flow.
- **Placement:** the station view and a per-IED tab live in Devices, so the four-place
  rule (UI-01) is unchanged. Diagram equipment bound to an IED offers "Show signal flow".

**Compatibility:** additive. The GOOSE input dictionaries gain a `match` field. There is
no project-format or worker change.
