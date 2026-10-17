# ADR-0054: A fifth place, Tools, for testing and commissioning helpers

Accepted: 2026-10-17. Owner request: add SCADA helper tools of the kind the leading products
have (client/browser, finding IEDs, traffic monitoring, a gateway simulator) in one place
called Tools, after Test.

**Decision**
- **Places.** The sidebar becomes Devices, SLD, Test, **Tools**, Settings. UI-01 changes from
  four to five places. Settings stays last because it is used least.
- **One card per built tool.** The Tools place lists large cards: an icon, a name, one plain
  line and a "What is this?" hint (`copy/tools.ts`). Opening a card shows the tool with a
  way back to the list. Only tools that are delivered are listed. Planned tools live in
  `docs/specs/proposed/` until they are built, so no card ever points to unfinished work.
- **First tool: GOOSE & Sampled Values traffic.** The existing `GooseTraffic` and `SvTraffic`
  cards move from Settings → Network into Tools. Settings keeps one link to them. GOOSE
  publishing per IED stays in Devices.
- **Roadmap agreed with the owner (2026-10-17):**
  1. IED Explorer: browse any IED and poll a watch list, read-only;
  2. Find IEDs: scan a confirmed subnet for port 102 and identify each device;
  3. Traffic inspector extensions;
  4. Gateway simulator speaking IEC 60870-5-104, Modbus TCP and DNP3. It has simulated
     sources plus read-only real IEDs, and commands reach simulated IEDs only.

  Each is specified in `docs/specs/proposed/` and gets its own decision record where a new
  dependency or format change is needed.
- **Safety stays the same.** Tools never write to or command a real IED (ADR-0031 and the
  real-IED access policy).

**Compatibility**
- No backend, native, project-format or dependency change. Projects open unchanged.
- The tour gains a Tools step. Its version is not raised, so the tour does not reopen by
  itself for people who have seen it; Help reopens it.
- The published tutorial videos show GOOSE traffic under Settings. Settings now links to
  Tools, so those videos stay usable until they are re-recorded.
