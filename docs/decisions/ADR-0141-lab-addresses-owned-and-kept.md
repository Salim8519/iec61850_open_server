# ADR-0141: Only this copy's lab addresses are removed; device addresses follow their device; IED addresses stay for the session

Accepted: 2026-11-20. Owner, 2026-11-20: "plan to include these ones too, these are part of 0135
now, plan, check" and "do all of them, once i wake, i should see all ready and no coding left,
just testing" (speed and network trials N3, N4, N5, N9, N10 and the Lab-card finding of the same
night). Amends [ADR-0125](ADR-0125-gateway-own-lab-address.md) (item 3),
[ADR-0127](ADR-0127-lab-addresses-follow-the-run.md) (items 3 and 4) and
[ADR-0132](ADR-0132-real-card-addresses-follow-the-run.md) (item 2); touches
[ADR-0136](ADR-0136-leave-nothing-on-the-network.md) (the moments of clean-up) and settles the
fix plan's Part B item "installed and portable copies remove each other's addresses"
([fix plan](../testing/fix-plan-2026-11-19.md)). Change [0222](../changes/2026/0222-lab-addresses-owned-and-kept.md).

## Problem

Measured on the owner's PC on today's code (2026-11-20, details in
[locks and ownership trials](../testing/speed-trials-locks-2026-11-20.md)):

- **Copies remove each other's addresses.** Every Release (Stop, Use, exit, launch) took off the
  Lab card every address its own copy did not use, also the ones another ScadaBench copy had just
  added (the installed app and a development app, or two data folders). In 10 rounds every one of
  40 actions of one copy took the other copy's 8 running addresses off; that night other agents'
  Runs failed with "MMS bind failed" 7 times.
- **N3.** Choosing the gateway's own address put it on the card first and saved the choice after,
  while Release (IED and SNMP Stop) used another lock: 80 of 200 choices lost the address to this
  engine's own Release.
- **N5.** A starting SNMP device's address counted as kept only once the device ran: 47 of 200
  starts failed ("not an address of this PC") while an IED stopped and started.
- **N4.** The launch clean-up and the gateway's automatic start ran unordered: in 1 of 60 launches
  the clean-up took the address the gateway had just started on ("Address gone").
- **N9.** The gateway's and master's own address stayed on the card after Stop and after exit, in
  10 of 10 cycles, until the next launch or a Windows restart.
- **N10.** Every Run added and every Stop removed every IED address (Windows changes, DNS Client
  work: 13-14 % of Run and 38 % of Stop on the Lab card, 160 Windows address changes and 50-61 DNS
  Client CPU seconds per 10 Run/Stop rounds of 8 IEDs; on a real card also the 0.7 s ARP question).

## Decision

1. **Removal follows this copy's journal, on the Lab card too** (as ADR-0140 item 2 decided for
   real cards). Release takes off only addresses whose newest `lab_address` record in this copy's
   `lab_journal.jsonl` is `applied` or `requested`. An address another copy, another program or an
   older version put there is never touched. A journaled address that is no longer on the card is
   journaled `skipped` ("Not on the card ...") so a later copy's address at the same place is never
   taken for ours. Exit and the next launch still remove every leftover this copy journaled; the lab
   page's **Take them off now** does the same on request (below).
2. **One lock for every add and every removal (N3).** Run, Release, and the gateway's, master's
   and SNMP devices' own addresses take the same lock (`card`) around the reads that decide a
   change and the change itself; page operations keep their own lock.
3. **A device's address follows the device (N9, N3, N5).** The gateway's and the master's own
   address is checked and stored when it is typed (nothing goes on the card then), goes on the card
   at Start/Connect and comes off at Stop/Disconnect and at exit. The device reports the address as
   held (`Projects.InUse`) *before* it is added, so no Release in between takes it off; an SNMP
   device counts as held from the moment its start puts its address on, not only once it runs.
   The clash checks still use the chosen addresses (`Reserved`): no IED can take the gateway's
   address while the gateway is stopped. Replaces ADR-0125 item 3 ("on the card while chosen").
4. **Launch clean-up first (N4).** At launch the clean-up of what an earlier session left finishes
   before the master, gateway and SNMP devices start by themselves.
5. **IED addresses stay for the session (N10).** Off for now (owner, 2026-11-20: "for the time being the stop removes them"; `labsetup.keepForSession = false`): Stop takes IED addresses off as ADR-0127 item 3 and ADR-0132 item 2 say; the rest of this item describes the switched-off mechanism. An IED address this run of ScadaBench put on a card
   (Lab card, or an approved real-card address, ADR-0132) stays there after Stop while the project
   stays open; the next Run adds nothing. It comes off when the project's addresses change while
   nothing runs (another project opened, File › New or Close, This PC only, the IED moved or
   removed), when the network is switched (Use), at exit, from the lab page's **Take them off
   now**, and after a kill at the next launch (the new session keeps nothing and the journal says
   the address is ours). Replaces ADR-0127 item 3 ("Stop takes them off") and ADR-0132 item 2's
   "Release (after Stop ...) takes off every approved address no running IED uses" for IED
   addresses. Gateway, master and SNMP device addresses are not kept (item 3).
6. **The page says it.** While stopped IEDs' addresses are kept, the lab page shows "N addresses
   stay on the lab card until you close this project or ScadaBench, so the next Start is quicker.
   Take them off now" (`GET /network/lab` `kept`, `POST /network/lab/cleanup`, refused while IEDs
   run). The Use hint and the ping hint say when addresses go on and come off.

## Why the gateway and the IEDs differ

The gateway and master run rarely and for a long time; their address stayed on the card for days
(after exit too), which is the leftover the owner called "trash" (ADR-0136). IEDs start and stop
many times a session; keeping their addresses until the project closes removes most Windows address
changes (and the DNS Client work and, on a real card, the 0.7 s ARP question per Start) without
leaving anything after the project or the app closes.

## Compatibility

- No project data change. API: `POST /tools/gateway/network/own` and
  `/tools/master/network/own` no longer put the address on the card (they check and store it);
  `GET /network/lab` gains `kept`; new `POST /network/lab/cleanup`. The journal gains `skipped`
  records for `lab_address` (append-only, as for `real_address`).
- An address an older ScadaBench left on the Lab card without a journal record (or in another data
  folder) is no longer removed by this copy; Windows removes it at the next restart, as before for
  real cards (ADR-0140 item 2). The older copy itself still removes everything it does not use until
  it is updated.
- ADR-0136's clean-up moments change for IED addresses only: Stop no longer removes them; closing
  the project, exit and reopening still leave nothing.
- The Python engine is frozen and has no lab card.
- Change 0227 (no new decision, item 1 applied to lost answers): a lab add whose answer was lost
  while the service stayed busy, and that the card does not show yet, is journaled `requested`
  instead of `failed`, and a lost removal counts as done only by the service's own list; the next
  Release then takes off an address the service put on late, or journals `skipped` for one that
  never came.
