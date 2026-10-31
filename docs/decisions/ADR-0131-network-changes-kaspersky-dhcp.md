# ADR-0131: The lab service is its own program; no fixed addresses on DHCP cards; one PowerShell per helper run

Accepted: 2026-10-31. Owner, after installing the rebuilt 0.1.30: "the scadabench labnet exe got
blocked … it stuck in repair", "i picked by mistake the wifi, then scadabench loaded all ips there
… i lost wifi connection", "even on ethernet takes longer now", "when we stop the network we
should remove the ip addresses … removing them manually is a hassle". Change
[0205](../changes/2026/0205-lab-service-split-dhcp-guard.md). Amends
[ADR-0120](ADR-0120-this-pc-lab-network.md) §install and the network helper of
[ADR-0074](ADR-0074-go-engine.md)'s port.

## Problems

1. **Kaspersky stopped the lab install.** `scadabench_labnet.exe install` copied its own file to
   `Program Files\ScadaBench\LabNetwork` and registered that copy as an automatic service.
   Kaspersky's behaviour engine (System Watcher) reported `PDM:Trojan.Win32.Generic` twice,
   killed the process, rolled its actions back and deleted the installed program. The file scan
   was clean, so the release gate (ADR-0130) could not see it. Development never met it: the
   project folder is excluded from Kaspersky on the build PC.
2. **Fixed addresses on Wi-Fi.** Windows turns DHCP off on a card that gets a fixed address, so
   the card loses its own address and its connection. Nothing stopped the user picking Wi-Fi.
3. **Two minutes per apply.** The elevated helper started one PowerShell per change (each loads
   the networking modules): 9 changes took 126–161 s. The app waited 120 s for the helper, so it
   reported a failure and journaled nothing although Windows had made every change; Clean up then
   could not see them and the owner removed the addresses by hand.
4. **Switching blocked.** The lab page refused IED addresses that ScadaBench itself had put on
   Ethernet earlier ("is the address of Ethernet").

## Decision

1. **Separate service program.** `scadabench_labsvc.exe` (`cmd/scadabench_labsvc`, only
   `service`) is built beside `scadabench_labnet.exe` and is what Install copies to Program Files
   and registers. `scadabench_labnet.exe` keeps `install` and `uninstall` and never copies itself.
   Repair points an older service at the new program (`UpdateConfig`) and deletes the old copy.
2. **DHCP guard.** The preview reads `IP_ADAPTER_DHCP_ENABLED` for the chosen card (one
   `GetAdaptersAddresses` call). On a DHCP card every address ScadaBench would add gets status
   `adapter_uses_dhcp` with a plain hint, so Apply stays off. Addresses already on the card and
   rows with another problem are unchanged.
3. **One PowerShell per batch.** The helper runs the same commands in the same order, up to 20 in
   one script, each in its own `try/catch` that prints a marker line with its outcome. A change
   without a marker line (crash, timeout, parse error) runs again on its own as before. Failure
   messages are the exception text instead of the last stderr line. The Python-parity runner test
   keeps the one-command-per-change path (`ExecuteRequest`).
4. **The app waits up to ten minutes for the helper**, and a helper result that arrives after
   the app stopped waiting is journaled the next time the journal is read (`catchUp`): any
   `apply-`/`remove-` result whose fingerprint has no journal record and whose request is not
   still awaited. Records are only appended.
5. **Moving ScadaBench's own addresses.** An IED address on another card that the journal says
   ScadaBench added (and that is still on that card) gets lab row status `ours_elsewhere` and does
   not block. Use removes exactly those addresses through the helper (one prompt) before
   switching. Addresses set by the user or another program still block.

## Compatibility

Project data, API bodies and the pipe protocol are unchanged; the lab row gains one status value
and the plan row one. Installed services keep working; Repair moves them to the new program. The
Python engine is frozen and keeps the old helper behaviour; Go now differs from it in helper
batching, failure messages, the wait and the DHCP status (deliberate, Go is what ships). The
lab install is still unsigned: other antivirus products were not tested.
