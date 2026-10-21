# 0151: Own names and changeable signals in SNMP values and gateway points; Explorer tabs

Date: 2026-10-21. Owner request with screenshots: a value or point could not be named or moved to
another OID/IED/path once it was in the list, and IED Explorer showed two half-height boxes on
the right. Three Explorer layouts were drawn; the owner chose tabs (small screens). Decisions in
[ADR-0092](../../decisions/ADR-0092-snmp-devices.md) and
[ADR-0064](../../decisions/ADR-0064-tool-pages-fill-window.md) (amendments of this date).

## Change

- SNMP values (Simulate devices › Values): every value has an own **Label** (box above the MIB
  name, saved on Enter or when leaving the box) and a pencil that turns the OID into a box
  (Enter saves, Escape cancels, a MIB name works too; an OID the device already has is refused
  and the box stays open). Add value has a Label field. Search also finds labels.
- Go: `model.Point.Label` (`label`, omitted when empty, at most 120 characters, never on the
  wire); ScadaBench CSV has an optional last column `label` (files without it still import).
- Gateway point list: **Name** is the first column and stays while selecting rows (it is the
  existing `description` field, so CSV and project data are unchanged). "IED · path" has a
  **Change** button: a row under the point with the shared signal search (only value kinds that
  fit the point type) or, for commands, the IED's controllable objects. Signals another point of
  the same type already serves show "Added ✓" and cannot be chosen twice. Save point list
  checks the result as before.
- IED Explorer: the right side is one full-height panel with tabs "<logical node>" and
  "Watch list (N)"; both stay mounted, so opened data objects and the watch list survive a
  switch. Picking a node in the tree shows its tab.

## Evidence

- Go: `internal/snmp/...` tests pass; the CSV round trip now carries a label with a comma.
- Frontend 267/267 (new `snmpRowMatch` test), `tsc` clean, `vite build` OK.
- `verify.py`: every step PASS except Go, where Windows refused to run the `internal/gateway`
  test program from the temp folder ("Access is denied", as in 0150); built elsewhere
  (`go test -c`) the gateway tests PASS.
- Browser at 1366x768 on the built frontend with a Go server on a spare port (the usual dev port
  8765 was taken by another session's file server): SNMP label "Vendor identity" saved and in the
  CSV export; OID moved to 1.3.6.1.4.1.99999.77.0, duplicate 1.3.6.1.2.1.1.1.0 refused with a
  message, then `SNMPv2-MIB::sysObjectID.0` resolved back; label cleared again. Gateway table fits
  without sideways scroll (816 of 816 px); point 10002 changed to `PhsTocPTOC2.Str.general` and a
  name saved, then restored (3,240 points, no duplicate ids); taken signals show "Added ✓".
  Explorer connected to 127.50.0.2, node ACsrcMMXN1 with Hz open kept across tab switches,
  watch list tab polling 40 values.

## Limitations

- The backend still accepts two IED points on the same signal and type (for example from a CSV);
  only the Change row prevents it. Their ids are then equal.
- Changing a label or OID of a running SNMP device restarts it (same as the other value settings).
- The signal search in the Change row is the shared one: it searches descriptions and short
  references, not every part of a full reference.
- Python engine: SNMP is Go-only; the gateway and Explorer changes are frontend-only.
