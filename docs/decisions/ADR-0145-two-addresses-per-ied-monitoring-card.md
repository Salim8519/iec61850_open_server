# ADR-0145: Two addresses per IED; the user's address wins; monitoring has its own card

Accepted: 2026-11-22. Owner, 2026-11-22, for 0.1.36: "it forgets the ip we put many times and uses
cid file, also it doesnt allow to use same ip in the ied simulated, ied simulated supposed to be
different than the monitoring, in monitoring it should read the relay over a network interface".
Answers to the clarifying questions: keep two IPs per IED; the typed IP always wins; pick a network
card per IED for monitoring; an empty monitoring IP starts from the simulation IP; only same-mode
clashes are refused. Change [0233](../changes/2026/0233-two-addresses-monitoring-card.md).
Amends [ADR-0029](ADR-0029-lab-network-and-format-11.md) (lab_ip and live) and the conflict rule
used with [ADR-0031](ADR-0031-read-only-live-acquisition.md) and
[ADR-0138](ADR-0138-real-ied-allow-commands.md).

## Problem

- Simulation reads `lab_ip`, monitoring reads `live.ip`. An address typed in one mode was invisible in
  the other, so after a role switch monitoring fell back to the CID address ("forgets the IP").
- Choosing a network card planned every IED from its SCD address and wrote the plan over `lab_ip`;
  the address the user had set was never the starting point (lab card: only while already active).
- The endpoint check compared every IED pair on host:port, so a simulated IED and a monitored relay
  could not share an address although one is served by this PC and the other is read over a card.
- Monitoring connected with no local address; Windows chose the card by its routes.

## Decision

1. **Two addresses.** `lab_ip` (simulation) and `live.ip` (monitoring) stay separate. On a role
   switch an empty one starts from the other, once (`project.seedAddress`): to monitoring, empty
   `live.ip` takes `lab_ip`; to simulation, empty `lab_ip` takes a non-loopback `live.ip`. Nothing
   else writes either field on a role switch, so switching back and forth never loses an address.
2. **The user's address wins in planning.** `netcfg.PlanIED.LabIP` is planned before the SCD
   address for both strategies; the row carries `saved: true` (an IED row shows "saved address").
   On the adapter strategy a saved address outside the card's network falls back to a free one.
   The ScadaBench Lab card rows use `lab_ip` on any card. Typed addresses still win over both.
3. **Same-mode clashes only.** `EndpointConflicts` (save) and the run-time check compare simulated
   with simulated and monitored with monitored. Two monitored relays on one host:port still clash.
4. **Monitoring card per IED.** `live.local` (optional IPv4 of this PC; absent = Windows chooses,
   stored only when set) is sent as `local` to `studio_live` `connect` and to `studio_control`
   `operate`; both call `IedConnection_setLocalAddress(local, 0)` (the SCADA master precedent,
   [ADR-0123](ADR-0123-master-network.md)). Control associations are cached per (card, device).
   The device page offers the PC's card addresses (the SCADA master's list) under "Network card".

## Compatibility

Projects without `live.local` behave as before. The frozen Python engine keeps the old rules (no
seeding, all-pairs conflict, no `local`); the Go engine of dev and release differs on purpose. The
recorded Python vectors contain no mixed-mode conflict, role-switch seeding or `local` case, so
they still match. Native workers must be rebuilt (the release does).
