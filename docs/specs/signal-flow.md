# Spec: Signal flow

Decision: [ADR-0036](../decisions/ADR-0036-signal-flow-from-scl.md). Change: [0059](../changes/2026/0059-signal-flow.md).

## Purpose

The engineer can see which IED sends which signals to which other IED: GOOSE
publishers and subscribers, report clients and, after D1, Sampled Values streams.
They can then answer questions like "what trips this breaker?" or "who is listening
to this relay?" without reading XML. The view shows only what the SCD **declares**.
Live confirmation comes from [Station health](station-health.md).

## Inputs (IEC 61850-6)

These are read from the original SCL stored in the project, across **all** IEDs in
that file, including IEDs that aren't loaded.

| Element | Used for |
| --- | --- |
| `GSEControl` + `DataSet` + `GSE` address | GOOSE publisher block and its signals |
| `Inputs/ExtRef` (with or without `srcCBName`) | GOOSE subscription and each received signal (`intAddr`) |
| `ReportControl` + `RptEnabled/ClientLN` | Report block and its declared clients |
| `SampledValueControl` + `SMV` address + `ExtRef` | SV stream and subscribers (only once D1 exists) |

It reuses `publisher_catalogue` and `goose_subscriptions` from `scl/goose_links.py`.
It does not write a second ExtRef resolver.

## Behaviour contract

**Graph model**
- **Nodes** are IEDs and control blocks: GoCB, RCB, and SvCB once D1 exists.
- **Edges** run publisher IED → block → subscriber IED. Each edge carries:
  - its signal list (source reference, dataset member, subscriber `intAddr` and `desc`);
  - how it was matched: *named block* (`srcCBName` present) or *matched by data*
    (FCDA match, the existing fallback).
- Each IED node is marked *in project* (loaded) or *in SCD only*.
- The graph is computed by a pure domain module, and the browser only draws it.
  Running the same SCD always gives the same graph.

**Diagnostics** (shown, never auto-fixed)
- **Problem:** an `ExtRef` whose source can't be resolved. This includes inputs without
  `serviceType` that name an IED with no section in the file: one problem per source IED,
  with the count and the first input (Go engine, ADR-0114).
- **Problem:** an `ExtRef` naming a block that doesn't exist.
- **Problem:** a block whose dataset is missing.
- **Info:** a GoCB or SvCB with no subscriber in the file. This is normal when the
  subscriber belongs to another project.
- **Info:** a `ClientLN` naming an IED outside this file, for example a gateway or HMI.
- The IED view also lists diagnostics written on other IEDs that name this IED as a
  whole word, such as a feeder expecting this IED's GOOSE.

**Interface** (Devices place; UI-01 unchanged)
- **Station view** (Devices → Signal flow):
  - Columns show publishers, then blocks, then subscribers.
  - You can filter by IED, type (GOOSE/Reports/SV) and diagnostic, and search by IED or
    signal name.
- **IED view** (DeviceDetail → Signal flow tab): only that IED's incoming and outgoing
  edges.
- Clicking an edge lists its signals in a table, which can be downloaded as CSV.
- Edge type is shown with text and line style, not colour alone: GOOSE solid, report
  dashed, SV double.
- Without live data every edge reads "declared in SCD". Once Station health has
  results, the edge also shows the latest live verdict with its timestamp. A declared
  edge is never presented as traffic actually seen.
- From the diagram, equipment bound to an IED offers "Show signal flow for this IED". The SLD
  [signal layers](signal-layers.md) open the same link details from an IED tag or GOOSE link.
- Stations with more than 60 IEDs open filtered to one IED, so the view stays readable.

## Out of scope

- Editing subscriptions or writing `ExtRef` / `ClientLN`.
- Configuration of the subscriber's internal logic.
- Explicit MMS client associations not declared in SCL (the live client list is in the
  [signal layers](signal-layers.md), not in this declared graph).
- R-GOOSE and R-SV.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| FLOW-01 | GOOSE edges come from `ExtRef` through the existing resolver; one `intAddr` maps to exactly one signal row. | `studio/tests/unit/test_signal_flow.py::test_goose_edges_from_extref` |
| FLOW-02 | Each edge records whether it was matched by named block or by data. | `studio/tests/unit/test_signal_flow.py::test_match_method_is_recorded` |
| FLOW-03 | Report edges come only from `RptEnabled/ClientLN`; a client outside the file is an info note, never an edge. | `studio/tests/unit/test_report_clients.py::test_unknown_client_is_diagnosed` |
| FLOW-04 | IEDs in the SCD that aren't loaded appear as *in SCD only*, never hidden. | `studio/tests/unit/test_signal_flow.py::test_unloaded_ieds_are_marked` |
| FLOW-05 | Unresolved sources and missing blocks or datasets are diagnostics; no edge is guessed. | `studio/tests/unit/test_signal_flow.py::test_unresolved_extref_creates_no_edge` |
| FLOW-06 | The graph is deterministic for the same SCD. | `studio/tests/unit/test_signal_flow.py::test_graph_is_deterministic` |
| FLOW-07 | The browser doesn't compute flow; it only draws the API graph. | `studio/tests/frontend/signalFlow.test.ts::renders api graph without deriving edges` |
| FLOW-08 | Edge type is shown with text and line style, not colour alone. | `studio/tests/frontend/signalFlow.test.ts::edge types have text and line style` |
| FLOW-09 | A declared edge without live evidence never shows a "received" state. | `studio/tests/frontend/signalFlow.test.ts::declared edge is not shown as received` |
| FLOW-10 | The signal list of an edge can be downloaded as CSV with the same rows shown. | `studio/tests/integration/test_signal_flow_api.py::test_edge_csv_matches_rows` |
| FLOW-11 | The IED view keeps only links touching that IED, plus diagnostics on it or naming it as a whole word. | `studio/tests/unit/test_signal_flow.py::test_single_ied_view_keeps_only_its_links` |
| FLOW-12 | Inputs without `serviceType` from an IED that is not in the file are a problem, never silent (Go engine). | `studio/engine/internal/scl/vendor_station_test.go::TestFlowFlagsInputsFromIedsOutsideTheFile` |

## Acceptance evidence

Recorded in change [0059](../changes/2026/0059-signal-flow.md):
- `scd/model_substation.scd`: 8 GOOSE links (feeders and transformers to the bus
  protections), all matched by named block. The 8 problems found are real errors in the
  file: the feeders expect `Bus1TripEventCn`/`Bus2TripEventCn`, while BUS1/BUS2 publish
  `…Cb`.
- `scd/open_substation.scd`: the only GOOSE `ExtRef` names a missing block `GoCB` and is
  listed as a problem. Changing it to `gcbEvents` gives one named edge, and removing
  `srcCBName` gives one data-matched edge.
- A browser walkthrough of the station view, the link detail, the CSV download and the
  per-IED tab.
- Pending: a real vendor SCD with more than 10 IEDs, a live verdict on links (added with
  Station health), and the owner's acceptance.

## Answered questions

1. A `ClientLN` naming a gateway or HMI with no IED section is **info**, labelled
   "outside this file". (Owner delegated the proposed answers, 2026-10-14.)
