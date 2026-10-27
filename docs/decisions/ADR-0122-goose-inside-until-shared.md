# ADR-0122: GOOSE stays inside until the user shares it (back to ADR-0073)

Accepted: 2026-10-27. Owner, on 0.1.28: "what happened to GOOSE preventing from going outside the
network? before, default they are inside the lab ... can we do it as before?" and "Yes, restore the
old GOOSE card: inside by default, Also share needs a confirm; keep the Npcap line and the
nobody-there hint". Replaces decision points 2 and 4 of
[ADR-0119](ADR-0119-goose-follows-the-ieds.md); restores [ADR-0073](ADR-0073-goose-engine-go.md).
Change [0194](../changes/2026/0194-goose-inside-until-shared.md).

## Problem

ADR-0119 sent GOOSE to the IEDs' network without a click and asked only when a real device was seen.
The owner found GOOSE going out on "VMware Network Adapter VMnet8" because that adapter was picked
under Advanced, while the card said IEDs on this PC only keep it inside. A simulated trip or
interlock that leaves the PC without a clear decision is the risk ADR-0073 was written to prevent.

## Decision

1. **Inside by default, in every network mode** (this PC only, the ScadaBench Lab network, a real or
   virtual card). Simulated IEDs always hear each other inside ScadaBench.
2. **Sharing is one confirmed choice.** "Also share with the network" opens the warning (stronger
   when real devices are likely); GOOSE goes out only after the user confirms with the adapter shown,
   stored as `goose_reach: "outside"` and `goose_share_guid`. One click on "Inside ScadaBench only"
   stops it. An adapter chosen under Advanced only says where sharing would go; it is locked while
   sharing, as before.
3. **Kept from ADR-0119:** the guard (3 s listen, own echoes dropped, network GOOSE never resent, held
   if a real device uses the same name, rate limit); `studio_goose` holding sharing next to a real
   device until confirmed for that adapter; the red Npcap line with its download link; the "nothing
   else on this network" hint; the top-bar pill only while GOOSE really goes out.
4. **One stored meaning again.** The Go engine reads `goose_reach` exactly as the Python engine does;
   the 0.1.28 `goose_local` key is dropped when a file is read, and `auto`/`local` are refused.

## Compatibility

Projects saved by 0.1.28 open unchanged; their GOOSE stays inside until the user shares it again, so
a lab-network test that relied on the automatic sending needs one confirmation. Projects shared
before (stored `outside`) keep sharing. Python engine unchanged.
