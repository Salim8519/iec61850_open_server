# ADR-0143: Station gateway and SCADA master run at the same time

Accepted: 2026-11-22. Owner, 2026-11-22, for 0.1.36: "it should allow master and stations work at
same time". Amends [ADR-0061](ADR-0061-scada-roles-master-converter.md) and [ADR-0064](ADR-0064-tool-pages-fill-window.md)
(one role at a time, role switch locked while anything runs; wording of change 0093).
Change [0231](../changes/2026/0231-gateway-and-master-together.md).

## Problem

"Act as" allowed one role at a time: the role switch was refused while the gateway ran or the
master was connected, the gateway refused to start under the SCADA master role, and the master
refused to connect under the Station gateway role. The two are separate programs
(`studio_gateway.exe` serves SCADA, `studio_master.exe` reads a device) with their own ports,
settings and logs, so the lock only stopped useful work: testing ScadaBench's own gateway with its
own master, or serving the station to SCADA while reading an RTU.

The screen locked the whole switch, so the role in use looked greyed out too ("why is it
inactive?", owner, 2026-11-21), and it treated a master in the `failed` state as busy although no
program ran and the Tools card showed nothing running.

## Decision

1. **Roles are views; the programs run on their own.** The Station gateway and the SCADA master
   may run at the same time. Switching between the two never stops, disconnects or restarts
   anything. The gateway starts and the master connects under any role (the role checks in
   `Manager.launch` and `Master.open` are removed); under every role except Protocol converter the
   gateway serves the station's own point list.
2. **Protocol converter stays on its own.** It uses both programs together (the master reads the
   device, the gateway serves the received points). The engine refuses to enter it while either
   program runs and to leave it while either runs (`gateway.RoleChangeRefusal`, two messages).
   So while anything runs, "the gateway serves the converter's points" holds exactly when the
   role is `converter`.
3. **The engine says which buttons are locked.** `GET /activity` carries `role_locks` (role ->
   reason) only while a role is locked, so the usual answer stays byte-identical to Python (same
   pattern as `address_gone`, change 0221). The locks change only with the gateway and master
   state, which already push `activity`, because a role change between gateway and master leaves
   them the same. The browser greys out only the locked buttons and shows the reason on the button.
4. **What runs is visible.** Each running role carries the shared green live dot plus "(running)"
   for screen readers and "Running now." in its tooltip (never colour alone). `SegmentedControl`
   gains per-option `disabled` and `live`; other uses are unchanged.
5. The master's new values tell the gateway table to refresh only under the converter role (the
   station gateway's table does not show them), so a master beside the gateway adds no table reads.

## Compatibility

- Projects: no format change; the saved `role` keeps its meaning (the view, and converter mode).
- API: `POST /tools/gateway/role` now refuses only converter changes while something runs (new
  messages); `/tools/gateway/start` and `/tools/master/connect` no longer refuse because of the
  role. `/activity` adds `role_locks` while a role is locked.
- Python engine: frozen (ADR-0074/0076) and not changed; it keeps one role at a time. The shipped
  and development engine is Go (ADR-0083), so product behaviour is the same in both modes.
- Ports: the gateway and the master can use the same port number (the master connects out); the
  master may connect to ScadaBench's own gateway at `127.0.0.1`.
