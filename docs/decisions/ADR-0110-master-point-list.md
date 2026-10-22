# ADR-0110: The SCADA master's own point list: Add points, expected vs received, one-click commands

Accepted: 2026-10-22. Owner-approved design ("Add points" for the SCADA master tool, IEC 104, DNP3 and
Modbus). Change [0174](../changes/2026/0174-master-add-points.md). Builds on
[ADR-0061](ADR-0061-scada-roles-master-converter.md) (the master), [ADR-0105](ADR-0105-gateway-add-points.md)
(the gateway's Add points and its type catalog), [ADR-0101](ADR-0101-save-always.md) (save always) and
[ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md) (new backend code in Go only).

## Problem

The master listed whatever the device answered. A SCADA engineer tests a device against a point list: which
addresses must arrive, with which type, and which commands must work. ScadaBench had no such list, could not say
"this address is missing" or "this arrived as the wrong type", and every command was typed by hand.

## Decision

1. **Address-only, like a real SCADA.** The master keeps its own point list: address (IOA, DNP3 index or
   Modbus address), master type, name, scale, offset and unit; Modbus points also store the table, data type
   (int16, uint16, int32, float32), word order and register count; command points store the command type,
   select before operate and the value (or step) that Send uses, and a DNP3 CROB its control code. It is never
   linked to an IED and the master never reads IEDs to decide what it received.
2. **Stored in the project, Go contract.** The master block gains `point_list` (`{iec104: [...], dnp3: [...],
   modbus: [...]}`) and `check_ied` (bool). Both are written only when present, so projects without them keep
   the exact bytes the frozen Python engine produces; the Go project load checks a stored list with the same
   rules as a save and refuses a broken one with the reason (`project/gateway/master_points.go`). Ids are
   `m<n>`. Rules (`internal/gateway/masterpoints/contract.go`, `list.go`): a known type of the protocol, a whole
   address in range (IEC 104 1-16777215, DNP3 and Modbus 0-65535 minus the extra register of a 2-register type),
   name at most 120 characters, unit 16, scale a number other than 0, on/off and position commands only 0/1 or
   1/2, at most 5000 points, and no two points on one address of one space (IEC 104: one IOA space; DNP3: each
   object group by itself, CROBs and analog outputs by their output; Modbus: each table, counting registers).
   The page saves the whole list at once (`PUT /tools/master/list`, one request in flight, latest wins, Undo
   for whole-list actions).
3. **One catalog, two directions.** `addpoints/catalog.go` gains a `Master` field per gateway kind (the master
   type that kind is seen as) and `addpoints/catalog_master.go` holds the master's type list per protocol (most
   used first, protocol type ids as small text, the reported group and/or command kind, Modbus table and data
   type). The gateway answer is unchanged (`json:"-"`). The dialog reuses the gateway's `TypeList` (now generic
   with a `badge` function), `add-points.css`, the dialog frame and `SegmentedControl`.
4. **Add points.** One point, a range (N x type from address X; a float32 or int32 range steps by 2 registers),
   or paste from Excel / upload the engine-made `.xlsx` template (Name, Address, Type with a Type drop-down per
   protocol, a whole-number rule for Address, no IED column) / upload a CSV. Every row is checked before
   anything changes; bad rows carry "Row n: reason". The CSV export (`Name, Address, Type, Scale, Offset, Unit,
   SBO, Value, Word order`) reads back to the same list.
5. **Expected vs received (Go computes, the page displays).** `masterpoints.Compare` puts every listed point in
   one state: Received (value, value after scale/offset with unit, quality, time), Missing (nothing after the
   general interrogation, integrity poll or Modbus read), Wrong type (the same address in the same space, sent
   as another type or read with another Modbus data type), Waiting (not connected, or the first ask not
   finished) or Command (devices do not report commands; a CROB or analog output shows its output status when
   the device reports one). Reported points that no listed point claims are "Not in your list", each with a
   one-click Add (`from-received`). Counts appear as filter chips; each state is words plus a symbol.
6. **Modbus reads follow the list.** Listed points become poll ranges: neighbouring points of one table, data
   type and word order in one read, split to the per-request limit (125 registers, 2000 bits), never reading an
   unlisted address; ranges the user's own ranges already cover are not read twice. The user's ranges and Find
   registers are unchanged and read first.
7. **One-click commands.** The engine builds each command point's command (`kind, address, value, select`,
   `code`, `word_order`); Send opens the existing command form at its REAL-device confirmation (`review`), so it
   goes through `POST /tools/master/command` with the same checks, logging, Repeat automatically and Address +1.
8. **Copy from gateway.** Fills the list from the project's gateway list of the same protocol: the address, the
   type as the master sees it (`MasterOfGateway`) and the name as the gateway shows it (IED name plus the point
   name, words only). `from: {protocol, address, kind}` remembers the source gateway point; already copied
   points are left out, an address the list already uses is left out with a note.
9. **Check against IED (test aid, off by default).** For copied points, `Manager.Mapped` (gateway service, not
   the master) finds the gateway point by protocol, address and kind and reads what its own mapping holds now
   (the IED value through the runtime, or an own point's made-up value) and the gateway scale.
   `CheckAgainstIED` answers ✓ "Both say Closed.", ✗ "IED says Closed, master got Open: check the double point
   mapping." (numbers: within 1e-4 relative or 0.01, scaled values within 0.5) or "–" with why there is nothing
   to compare. It is a separate column labelled as a test aid.

### Routes (Go only; `toolsapi.MasterGoOnlyRoutes`, which also lists the 0169 cyclic routes)

| Route | Body | Answer |
| --- | --- | --- |
| `GET /tools/master/list` | - | `{protocol, points, types, check_ied, auto_ranges}` |
| `PUT /tools/master/list` | `{points}` | the same, after the checks (400 with "Point n: reason") |
| `PUT /tools/master/list/check-ied` | `{on}` | `{check_ied}` |
| `POST /tools/master/list/new` | a point, or `{type, count, from, name}` | plan |
| `POST /tools/master/list/rows` | `{text}` (pasted rows or CSV) | plan + `columns` |
| `GET/POST /tools/master/list.xlsx` | POST `{data: base64}` | template / plan + `columns` |
| `GET /tools/master/list.csv` | - | the list as CSV |
| `POST /tools/master/list/copy-gateway` | `{}` | plan + `notes`, `gateway_points` |
| `POST /tools/master/list/from-received` | `{keys}` | plan |
| `GET /tools/master/list/compare` | - | `{rows, not_in_list, counts, asked, connected, check_ied}` |

A plan is `{rows: [{number, point, problem, cells?}], points, ok}`; `points` is empty unless every row is ok.

## Compatibility

No format version change: the two keys are optional and absent from existing projects. `GET/PUT
/tools/master/config` and every frozen route answer byte-identically (`TestRoutesAgainstFlask`). The frozen
Python engine drops `point_list`/`check_ied` when it saves a project and answers the new routes 404: a list made
with the Go engine is lost if the project is saved by the Python engine (Go is the default engine). The native
`studio_master.exe` is unchanged: auto ranges use the existing `poll` request. No new dependency (excelize is
from ADR-0105).
