# ADR-0136: Leave nothing on the network

Accepted: 2026-11-04. Owner: "add more reliable way to turn off the ips that are set by us on the
network and we forgot about them … this should happen the moment we switch the cards, the app
closed, or closed by force and user re open", "i clicked stop and the ip addresses still there
looks like a trash", "Fix them bro safely now start", "We should leave nothing", "But only when app
closes it ask to clean up, or user stops, or user reopen app and ask to clean if any ips found
left over".
Builds on [ADR-0125](ADR-0125-gateway-own-lab-address.md),
[ADR-0127](ADR-0127-lab-addresses-follow-the-run.md) and
[ADR-0132](ADR-0132-real-card-addresses-follow-the-run.md).

## Problem

Measured on the owner's PC (2026-11-04):

- Something other than ScadaBench switched the network service to manual start at 01:02, so after
  a restart it was stopped. Apply on Ethernet then fell back to the old helper path without a word:
  38 permanent-until-restart `alias` items (14 IED, 24 SNMP) and an address-list firewall rule.
  Stop removes only what the service added, so all 38 stayed.
- SNMP device addresses on real cards always used `alias`, even with the service running (ADR-0132
  left it as a follow-up).
- Nothing told the user that addresses were left behind, or offered to remove them.

## Decision

Clean-up happens at exactly three moments, and only there: **Stop**, **closing the app** and
**opening it again**. No background program removes anything on its own (owner's choice).

1. **One owner, no fallback.** On a real card only the network service adds addresses: IED and
   SNMP addresses both go through approvals (ADR-0132). The service adds them at Start (devices: at
   their start) and removes them at Stop, at exit and at the next launch, with no prompt. When the
   service is missing, stopped or too old, Network setup never plans `alias` items. It says so in
   plain English and offers one action: turn the network helper on (one Windows prompt: automatic
   start, then start).
2. **Stop and close.** Stop and a normal exit already release every address the service holds
   (ADR-0127, ADR-0132). SNMP devices now do the same, so after Stop or close nothing of ours stays
   on any card.
3. **Reopen asks.** After a forced close, the next launch releases what the service holds, as
   before. Then it looks for anything else ScadaBench's journal says it added that is still
   present: older `alias` addresses and address-list firewall rules. If it finds any, it asks
   once: "ScadaBench found N addresses it left on your network. Remove them?" Yes runs the
   existing Clean up (one Windows prompt). The user's own addresses are never listed, because only
   journaled items are.
4. **Temporary by nature.** Service and helper addresses live in Windows' active store only
   (`CreateUnicastIpAddressEntry`, `-PolicyStore ActiveStore`), so a Windows restart clears them
   too.

## Compatibility

- The pipe protocol is unchanged (3). Plans saved by older versions keep working. Their `alias`
  items become leftovers that the reopen check offers to remove.
- Firewall rules are not addresses. Replacing the per-plan address-list rules with fixed
  per-program rules is a separate decision. Until then, the reopen check offers leftover rules for
  removal with the addresses.

## Evidence

Change [0214](../changes/2026/0214-leave-nothing-on-the-network.md).
