# ADR-0127: Lab addresses are on the card only while their IEDs run

Accepted: 2026-10-29. Owner: "in the scadabench lab, the ip will stick there even if u did not
click start" and, after the plain-English proposal, "ok check and confirm" and "if all ok, do the
fix, deploy version 0.1.29". Changes [ADR-0120](ADR-0120-this-pc-lab-network.md) §3 and §6.
Change [0201](../changes/2026/0201-lab-addresses-follow-the-run.md).

## Problem

"Use the lab network" put every simulated IED's address on the ScadaBench Lab card at once, and
nothing took it off again: not Stop, not closing the app, not "This PC only". Only moving an IED,
removing the card or restarting Windows did. The PC kept answering ping at addresses of IEDs that
were not running, which looks like a device that is not there.

## Decision

1. **Use only stores.** "Use the lab network" checks the addresses (clashes, other cards, ARP) and
   switches the project to the card as before, but puts nothing on the card.
2. **Run puts them on.** Before the workers start, the coordinator asks for the addresses of the
   IEDs it is starting (`LabAddresses(guid, starting)` → `EnsureForRun`); IEDs that do not start
   get none.
3. **Stop takes them off.** After a full Stop, a Stop of some IEDs, or a failed Run, the
   coordinator calls `Release`: every address on the card that no running IED and no gateway or
   master own address (ADR-0125) uses is removed. Release runs under the run's cycle lock, so a
   Run and a Release never overlap; adds and removals share one more lock in labsetup.
4. **Exit and launch.** App exit stops the run, so Release runs. At launch the app releases what a
   crashed or killed session left, under the cycle lock (nothing runs yet).
5. **Change an address.** A running IED is stopped and started again: Stop removes the old
   address and Run adds the new one. A stopped IED's new address is only stored.
6. **No new prompt or wait.** The lab network service already holds the permission; each add or
   remove is one pipe call (milliseconds). The page says Ping works after Start, and the summary
   says "Set: … while the simulation runs" until it runs.

The gateway's and the master's own addresses keep the ADR-0125 rules (on the card while chosen).

## Compatibility

No project data or API body changes; `POST /network/lab/use` and `PUT /network/lab/address` keep
their shapes. Ping of a stopped IED's address no longer answers. The Python engine has no lab
card. The journal stays append-only (`lab_journal.jsonl`, `removed` records).
