# ADR-0120: This PC lab network: every IED on its own address without a VM

Accepted: 2026-10-26. Owner: "instead of VM use a smart method to isolate network ... IP addresses of
IEDs are different, we can't put a fixed ScadaBench IP address, user can change it at any time ...
it should act like a real network IP, e.g. ping 192.168.216.1 should work"; warn about the Ethernet
and its network, prevent clashes, zero headache during installation. Amends
[ADR-0029](ADR-0029-lab-network-and-format-11.md) (lab mode) for one adapter only. Change
[0192](../changes/2026/0192-this-pc-lab-network.md). Go engine.

Amended by [ADR-0127](ADR-0127-lab-addresses-follow-the-run.md): an IED's address is on the card
only while the IED runs (Use stores, Run adds, Stop/exit/launch remove).

## Problem

To test IEDs at their real SCL addresses the owner needed a VM: loopback mode only offers
`127.50.0.x`, and lab mode puts aliases on a real card, which must have a cable, is not isolated and
can clash with a live relay. The Windows loopback card (KM-TEST) already appeared as `virtual_lab`
in the adapter list, but lab mode added the SCL address with its SCL prefix and `SkipAsSource`.
Measured on the owner's PC (change 0192): with `SkipAsSource` on every address of the card Windows
has no source address, so `ping 192.168.216.1` and any client without an explicit bind fail with
"General failure"; a /24 also installs subnet routes on the card.

## Decision

1. **One app-owned card, "ScadaBench Lab".** A Microsoft KM-TEST Loopback Adapter created from the
   driver Windows ships (`%WINDIR%\INF\netloop.inf`, nothing downloaded, no reboot; 4.4 s here).
   Interface metric 9999, IPv6 unbound, never a gateway or DNS. It is always "connected", so its
   addresses work with no cable, and nothing on it leaves the PC.
2. **Addresses are the IEDs' own.** Each simulated IED answers at its SCL address (or the one the
   user typed), added as a `/32` with `SkipAsSource` **off** and not persistent (gone at reboot).
   `/32` installs no subnet route, so the real cards' routes never change; Windows takes a source
   address from the outgoing card only, so these addresses are never the source of real traffic.
   Measured: ping, TCP 102 (0.5 ms connect) and GOOSE through Npcap (a second handle sees the frame
   at once) all work; internet traffic keeps the Wi-Fi source.
3. **Change at any time, no prompt.** A tiny Windows service, `ScadaBenchLabNet`
   (`scadabench_labnet.exe`, LocalSystem, copied to `%ProgramFiles%\ScadaBench\LabNetwork`, where
   only administrators can write), accepts on the pipe `\\.\pipe\ScadaBenchLabNet` (SYSTEM and
   administrators full, interactive users read/write) exactly three requests: `status`, `add` and
   `remove` one IPv4 unicast address on the card whose GUID the installer stored in the service's
   own registry key. It refuses loopback, multicast, link-local, 0.x and broadcast addresses, any
   other card and an address another card already has. Before it sends anything the app checks
   that the pipe's server process is the process Windows runs `ScadaBenchLabNet` in (process ID
   from the Service Control Manager, which any user may ask; a normal user may not open a SYSTEM
   process to read its path). Adding an address takes milliseconds; a running IED is stopped and
   started again on its new address (about 2 s measured for one IED).
4. **One Windows prompt, once.** "Create the lab network card" runs `scadabench_labnet.exe install`
   through the existing UAC elevation: card, settings, service. The same program's `uninstall`
   (Remove the lab network card) removes service, files and card. Both are journaled in the
   append-only `network/lab_journal.jsonl` (kinds `lab_card`, `lab_address`); the older
   `journal.jsonl` and its clean-up stay as they are.
5. **Clash protection.** An address is refused when another IED uses it or another card of this PC
   has it. When it lies inside the network of a connected real card (Ethernet, Wi-Fi) ScadaBench
   asks that card by ARP **with the card's own address as source** (without it Windows answers
   from the lab card itself, measured); a device that answers is a real device: the row is marked
   as a warning and a free address of the same /24 is offered, but the address stays usable (this
   PC then reaches the simulator). Inside a real card's range without an answer is only
   information. Answers are kept 60 s and keyed by the real card's address, so a new network is
   asked again by itself.
6. **Runs by themselves.** At Run, missing addresses (after a reboot) are added through the
   service before the usual preflight. All decisions are made in Go; the browser only shows them.
7. **Stored values stay Python-readable.** The project keeps `mode: "lab"`, the card's GUID and
   `lab_ip` per IED, which the Python engine already understands. The ordinary card picker no
   longer offers "ScadaBench Lab"; the new page owns it.

## Compatibility

Existing loopback and lab projects are unchanged. A KM-TEST card the user created by hand is not
owned and is still listed as before. The service only exists after the user's "Create" and its
UAC approval; it is removed by "Remove the lab network card". Python engine unchanged.

## Not decided here

ARP replies from real cards for addresses on the lab card (Windows' strong host model should not
send them) are not yet proven against a second machine; recorded as a limitation in 0192.
