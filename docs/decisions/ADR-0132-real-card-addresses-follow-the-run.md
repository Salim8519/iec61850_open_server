# ADR-0132: Approved Ethernet addresses follow the run through the ScadaBench network service

Accepted: 2026-10-31. Owner: "when we stop it should remove the ips right? The ips goes back when
hit start", "the problem is the annoyance of the IPs", "can you clean them up when we close the
program?", "how about the other cards when we stop and start?" and, after the proposal, "lets make
best user experience and smoothest". Change [0206](../changes/2026/0206-real-card-addresses-follow-the-run.md).
Extends [ADR-0127](ADR-0127-lab-addresses-follow-the-run.md) to real cards; builds on
[ADR-0131](ADR-0131-network-changes-kaspersky-dhcp.md).

## Problem

On a real card (Ethernet, a VMware card) the elevated helper added every IED address at Apply and
the addresses stayed until Clean up or a Windows restart: one prompt to add, one to remove, the
addresses answering while nothing ran, and blocking a later switch to another card. Removing them
at Stop or at exit through the helper would ask Windows every time. Running the whole app as
administrator was rejected (security, locked-down customer PCs, a prompt at every launch).

## Decision

1. **Approve once.** When the ScadaBench network service speaks protocol 3 and the chosen card is
   not the lab card, Apply writes an *approval* per IED address instead of adding it: one DWORD
   value `"{card GUID} address" = prefix` under the service's `Parameters\Approved` key, which only
   administrators can write. It is an ordinary journaled plan item (`approval`), shown in the
   preview with its exact command, removed by Clean up (approval and address). An approval that
   already exists is "keep". SNMP device addresses keep the old alias item: those devices start
   from their own page, not with Run.
2. **Start adds, Stop removes, without a prompt.** The pipe protocol gains `card` (a GUID). For a
   real card the service changes only approved addresses, never on a DHCP card, never an address
   another adapter has. The coordinator's existing hooks do the rest: `EnsureForRun` puts the
   starting IEDs' approved addresses on, `Release` (after Stop, at launch, at exit) takes off every
   approved address no running IED uses.
3. **Fast and safe on the wire.** Windows' own duplicate check makes a new address usable only
   after 3.3–3.5 s. The service instead asks the card's network once (ARP from the card's own
   address on that network, all addresses in parallel, 0.7 s at most; a device answers in
   milliseconds). An address a device answers for is refused; the rest are added ready at once.
   Where the card has no address on that network or no cable, Windows' check runs and the service
   waits (6 s at most) until each address can really be listened on.
4. **Repair updates the service.** `Install` now waits for the old service process to exit,
   retries the copy for 5 s and restarts the old service if the copy still fails (the first Repair
   to the same file name failed and left the service stopped). The lab page offers Repair to any
   service older than protocol 3.

## Compatibility

A protocol 1–2 service, or none, keeps the old behaviour exactly (aliases at Apply). Project data
and API bodies are unchanged; plan items gain the `approval` kind, lab journal records the
`real_address` kind. The pipe stays limited to what an administrator approved: any local program
can at most put the approved addresses on or off their card. The Python engine is frozen and has
no lab service.

## Amendment (2026-11-20, change 0221)

Owner: "do all of them" (speed and network trials N6 and N7). Item 3 changes for a card without a
link: the service no longer adds an address there (it used to add it, report it added and leave it
tentative, so no IED could listen on it); the Run stops with "“Ethernet” is disconnected … Plug the
cable in, then press Start again". An address Windows is still checking when the 6 s wait ends is
taken off again and reported instead of reported as added. The 0.7 s ARP question stays as
decided: treating a late ARP answer as "not asked" (trial N7) would add Windows' 3.3–3.5 s check to
every real-card Start for a device slower than 0.7 s, which real devices are not; it was dropped.
Change [0221](../changes/2026/0221-network-failures-named.md). The installed service gets this
only with the next installation (Repair), as for every service change.

## Amended by ADR-0141 (2026-11-20)

Item 2: an approved IED address the service put on at Start stays after Stop while the project
stays open (the next Start adds nothing and skips the ARP question); it comes off when the project
changes, at exit and after a kill at the next launch. [ADR-0141](ADR-0141-lab-addresses-owned-and-kept.md), change [0222](../changes/2026/0222-lab-addresses-owned-and-kept.md).
