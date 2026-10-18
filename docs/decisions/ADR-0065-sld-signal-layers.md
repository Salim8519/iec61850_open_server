# ADR-0065: SLD signal layers for GOOSE and MMS clients

Accepted: 2026-10-18. Spec: [signal layers](../specs/signal-layers.md). Change:
[0094](../changes/2026/0094-goose-mms-signal-layers.md).

**Problem**

The owner wants to see GOOSE and IEC 61850 client activity on the single-line diagram without
distraction. Until now GOOSE was only visible as a table (Tools › Traffic) and as a declared
graph (Devices › Signal flow), and nothing showed which SCADA or test program reads an IED.
Real vendor files make naive views wrong:

- identity: many files use APPID 0000 everywhere and two IEDs can share a multicast MAC, so a
  GOOSE stream is identified only by its control block reference (goCBRef = IED name + LD +
  block), with goID, dataset and confRev as checks. MMS clients are identified by IP address;
- GE Multilin marks blocks `Function="Disabled"` in its own XML namespace; most such blocks
  have no dataset and are never sent;
- a GE C650 names its publishers only inside a packed ExtRef `intAddr`
  (`LD|IED|n|MAC|type|…`), without `srcCBName`, and those publishers are often in no file;
- the same small publisher IEDs are copied into every bay unit's CID.

**Decision**

- One **GOOSE link map** per project: publisher → control block → subscribers, built by a pure
  domain module from all stored SCL files (IEDs deduplicated by name, preferring the file the
  IED was loaded from). Every GSEControl is listed with a status: sendable, disabled in file,
  no dataset, no address, not GOOSE. Links to blocks that are not sent stay visible with that
  status; publishers in no file are *remote*. Vendor data is read as hints and never rewritten.
  `GET /goose/links` serves the map joined with the GOOSE listener's last state per goCBRef.
- Diagnostics are words with a fix hint, never import failures: APPID 0, a duplicate APPID,
  a shared MAC, a remote publisher, a subscriber expecting a block that is not sent.
- Two **remembered switches** in the SLD toolbar: *GOOSE messages* and *Who is reading*. They
  are stored as backend preferences (`sld_goose_layer`, `sld_readers_layer`), because the app's
  address and port change between launches. They change only the view.
- While a switch is on, small grey **IED tags** are placed beside the topmost equipment bound
  to that IED (state, command or relay binding); IEDs outside the project and unbound IEDs sit
  in a column at the left edge, outside ones dashed. Tags are computed, never saved: no project
  format change. They scale up when zoomed out so they stay readable.
- **Pulses**: one quiet dot (~0.7 s) along a faint curve publisher → subscriber, then the link
  fades (~2 s), with a short label such as "FEED1 → BUS1 · XSWI3.Pos Closed". Solid dot =
  stNum increase seen by the listener; hollow dashed dot with "(expected)" = inferred from a
  publisher value change that belongs to a sent block's dataset while no listener runs. The
  first frame never pulses, a retransmission never pulses, and a link pulses at most once per
  second. Reduced motion shows a static highlight. Animations use transform and opacity only.
- **Problems persist** on the subscriber's tag as words with a dashed border ("⚠ GOOSE lost
  from FEED1", "configuration revision differs"), never as colour alone. Energization colours
  are untouched; tags and links live in a separate overlay above the drawing.
- Clicking a tag or link opens a side sheet with the signal-flow details (block, dataset,
  APPID, MAC, confRev, inputs) and "Open in Signal flow".
- *Who is reading* shows "◉ 2 read" on a simulated IED's tag; hovering lists each client's IP,
  the logical nodes it reads and the report blocks it enabled. An external MMS control shows a
  single short "‹ip› operated ‹object›" on the tag. ScadaBench's own command client is not
  counted as a reader.
- Devices: a quiet list hint ("↑ sends to 2 · ↓ gets from FEED1 · ⚠ 1 not heard") and two
  Overview tables, *GOOSE partners* and *Read by clients*, each in its own small component so
  the parallel Devices layout work (ADR-0064) is not disturbed.
- Live events reach only the overlay store; equipment nodes are not re-rendered by GOOSE or MMS
  traffic, and per-second MMS summaries are kept out of the Activity list.

**Compatibility**

- No project format change. Preferences gain two keys (0 or 1). `/goose/links` and
  `/mms/access` are new read-only endpoints.
- The SLD toolbar wraps instead of running under the "drawing is locked" pill or the sheet.
- A GE block marked Disabled is no longer published by the simulator (the device would not
  send it either); the import says so in words.

**Rejected**

- Identifying GOOSE by APPID or MAC: wrong for real files.
- Saving tags as diagram nodes: would change the project format and every saved diagram.
- One pulse per frame or per retransmission: noise, not information.
