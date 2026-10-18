# ADR-0068: "Be this IED" wording and a one-page network setup

Accepted: 2026-10-18. Spec: [lab network](../specs/lab-network.md). Change:
[0098](../changes/2026/0098-publish-wording-one-page-network.md).

**Problem**

The owner set IEDs to "Real device" expecting ScadaBench to publish them on the real network.
"Real device" actually means the physical IED publishes and ScadaBench reads it, so the network
setup then failed with "There are no simulated IEDs to put on the network". The setup itself was a
four-step wizard with a strategy question (SCD addresses or free addresses) that has an obvious
answer, a prefix box, a clipped check column, and it lived under Settings where nobody looks.

**Decision**

1. **Mode names.** The stored roles stay `simulated` / `real` / `off` (no project format change).
   They are shown as **Be this IED** (ScadaBench publishes as this IED, the default),
   **Read real IED** (the real IED publishes, ScadaBench reads it) and **Leave out**. Pills say
   "Publishing", "Be this IED · stopped", "Reads real IED", "Left out". Backend messages use the
   same words.
2. **Reachability is always visible.** A role line under each device header says who publishes,
   the address, and "reachable from this PC only" with *Put on a network*, or the network card.
   The top bar shows "N IEDs reachable from this PC only" or the card name; it opens the setup.
3. **One-page setup.** Network card list, one row per published IED with the SCD address,
   a check per row, one sentence for the Windows changes (exact list in a disclosure) and one
   button "Put N IEDs on <card>". The planner adds `alternative` (one free address on the card's
   network) to rows that cannot be used, computed with the existing free-address planner.
   The strategy cards, the prefix box and the four steps are removed.
4. **No published IED** shows each IED's mode and a *Make them publish* button, never a bare error.

**Compatibility**

Safety is unchanged: the fingerprinted preview, the single UAC approval, non-persistent aliases,
the narrow firewall rule and cleanup work as before (NET-01…NET-12). The API keeps `strategy`
(the page sends `scd`); `alternative` is an additional field. Projects need no migration.
