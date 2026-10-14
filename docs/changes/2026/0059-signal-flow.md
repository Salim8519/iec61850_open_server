# 0059: Signal flow between IEDs

Date: 2026-10-14. Roadmap feature C. Specification: [Signal flow](../../specs/signal-flow.md).
Decision: [ADR-0036](../../decisions/ADR-0036-signal-flow-from-scl.md).

## Change

**Backend**
- `scl/report_clients.py` and `scl/flow_sources.py` read the file. `goose_links.py`
  records the match method for each input.
- `domain/signal_flow.py` builds the graph.
- `projects/signal_flow.py` and `api/signal_flow.py` provide `GET /signal-flow/sources`,
  `GET /signal-flow?source=&ied=` and `GET /signal-flow/edges/<id>.csv`.
- `projects/scl_sources.py` is shared with SLD import.

**Frontend**
- **Devices → Whole station → Signal flow** shows count tiles and filters (file, IED,
  type, search).
- The table has Sender, Message and Receivers columns. GOOSE uses a solid outline and
  reports a dashed one, always with a text label.
- Clicking a receiver opens the link detail: the signal table, how the link was matched
  and a CSV download.
- The file's problems and notes sit in disclosures.
- Each IED has a **Signal flow** tab. The diagram equipment sheet has **Show signal flow**.

## Evidence

- Tests: `test_signal_flow.py`, `test_report_clients.py` and `test_signal_flow_api.py`
  (pytest 216 passing overall), plus `signalFlow.test.ts` (frontend 64 passing).
- Browser, `model_substation.scd`:
  - 6 IEDs and 8 GOOSE links.
  - FEED1 → BUS1 lists the two `XSWI3.Pos.stVal` values going to BUS1's `PDIF_in_swi_0`
    and `in_nor_0` inputs.
  - The CSV matches the rows exactly.
  - It found a real error in the file: the feeders subscribe to `Bus1TripEventCn`, but
    BUS1 publishes `Bus1TripEventCb`. The bus-trip GOOSE therefore has no receiver, and
    the IED tab of BUS1 lists those 4 problems.

## Limitations

- No live verdict on links yet (that comes with Station health).
- SV links come with the Sampled Values feature.
- Not yet checked on a real vendor SCD with more than 10 IEDs.
