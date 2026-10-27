# ADR-0124: The station gateway shows where SCADA connects and who may

Accepted: 2026-10-27. Owner: "master and station both need ip, right? what u suggest" and "ok go
ahead". Follows [ADR-0123](ADR-0123-master-network.md). Change [0197](../changes/2026/0197-gateway-network.md).

## Problem

In the station gateway role the column said only "Other computers can reach the gateway on TCP
port N": the engineer could not see which address to type into SCADA, could not keep the gateway
on one card, and could not test a gateway that answers only known SCADA addresses.

## Decision

1. **"Network" replaces "Who can connect"**: This PC only, All networks, or one address of this PC
   grouped by card (the ADR-0123 list: no IED, lab or link-local addresses). `listen` keeps the two
   values the Python engine reads; a card is stored as `listen: "network"` plus `listen_address`.
2. **"SCADA connects to A:port (card)"** under it, worked out in Go: 127.0.0.1 for This PC only, the
   chosen address, or for All networks the default-route address plus how many more the PC has.
3. **"Allowed SCADA address"** under Advanced (`allowed_scada`, empty = anyone). IEC 104 refuses
   other callers in the connection request handler and Modbus TCP at accept; both log the refusal
   in Traffic. DNP3 shows that it checks the master address instead: opendnp3 has no accept filter
   and the vendor tree is never edited.
4. **Go-only keys survive a project load.** `listen_address`, `allowed_scada` and the master's
   `local` (ADR-0123) are kept when stored and checked as IPv4; a project without them normalises
   to the Python bytes; a damaged value stops the load with the reason.

## Compatibility

Projects keep This PC only or All networks as before. The Python engine ignores the new keys and
listens as `listen` says. The firewall check is unchanged (it follows `listen`).
