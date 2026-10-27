# ADR-0125: The station gateway may have its own address on the lab network

Accepted: 2026-10-27. Owner: "all network and specific network and ip should be in same drop down
list, reduce the hassle", "allow gateways to work on the scadalab network" and "yes go ahead, build
the gateway part". Extends [ADR-0124](ADR-0124-gateway-network.md) and
[ADR-0120](ADR-0120-this-pc-lab-network.md). Change [0198](../changes/2026/0198-gateway-own-lab-address.md).

## Problem

With the ScadaBench Lab card every simulated IED answers at its own address on this PC, but the
station gateway could only listen on an address this PC already had (ADR-0124). On one PC the
gateway therefore never looked like its own device, as a real gateway does in a station.

## Decision

1. **One list.** The gateway's Network list keeps All networks (default), This PC only and the
   addresses by card, and adds an "Own address" group with the addresses typed before, plus
   "Type an address…" as the last item. Picking it opens a box and Add in place.
2. **Same rules as an IED address** (labsetup, Go): a unicast IPv4 address that no IED, no other
   card of this PC and no other device of ScadaBench uses, and no real device on a connected card
   answers by ARP. Unlike an IED row a real device refuses the address (no alternative offered).
3. **On the lab card through the lab network service** (no Windows prompt), journaled in
   `lab_journal.jsonl` as `lab_address` with the name "Station gateway". The gateway puts it back
   before it listens (addresses are gone after a Windows restart) and takes it off the card when
   another network is chosen; the address stays in the list.
4. **The lab page sees it.** An IED row with the gateway's address is blocked ("Station gateway
   uses …") and "Use the lab network" no longer removes it from the card.
5. **Stored Go-only** in the gateway block: `own_addresses` (list) beside `listen: "network"` and
   `listen_address`; kept on load and checked as IPv4 like the ADR-0124 keys.
6. **Hints.** Under an own address: only programs on this PC reach it; for another PC or VM choose
   All networks. Without the lab card: create it first.

## Compatibility

Projects without `own_addresses` load to the same bytes. The Python engine ignores the key and,
seeing `listen: "network"`, listens on all cards. The SCADA master gets the same list in a later
change.

## SCADA master (change [0199](../changes/2026/0199-master-own-lab-address.md))

Owner: "yes, do the master part now" and "custom ip is always as scadabench network lab". The master's
Network list gets the same "ScadaBench Lab" group and "Type an address…" with the same checks. The
address is the one the master connects from (`local`), kept in the master block as `own_addresses`,
put back on the card before Connect and before each reconnect, and taken off when another network
is chosen. The gateway and the master cannot take each other's address. A warning shows when the
device is reached through a real card, because an own address reaches devices on this PC only.
Typed addresses are listed as "ScadaBench Lab" in both lists.
