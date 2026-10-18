# 0101: GOOSE between host and VM on 0.1.12, and five findings

Date: 2026-10-19. Evidence only, no code changed. Spec: [GOOSE](../../specs/goose.md),
[signal layers](../../specs/signal-layers.md). Follows [0091](0091-goose-between-machines.md).

Owner request: "0.1.12 installed in the VM, now start testing GOOSE; the host has 0.1.12 too".

## Setup

- **Host:** installed ScadaBench 0.1.12, the owner's project (9 IEDs), lab mode on VMware VMnet8
  (aliases 192.168.98.21-28). Driven through its own API with the owner's permission, because the
  app window is a browser window that the test tooling may only look at.
- **VM** (192.168.230.130): installed 0.1.12, lab mode on Ethernet0, C02_BCU and C19_Y1005
  running. Driven through Remote Desktop.
- **Independent check:** tshark on the host VMnet8 adapter.
- To avoid two publishers of one block, only host IEDs the VM does not run were started:
  C17_Y230, C11_Y120, C06_Y130, C12_Y220 (*Run this IED* through the API).

## Observed

| Test | Result |
| --- | --- |
| VM → wire | C02_BCU and C19_Y1005 GoCB01, stNum 1, TAL 30 s, a frame every 10 s (SCD MaxTime 10 s) |
| Host → wire | The four GoCB01 blocks, source MAC of VMnet8, 802.1Q priority tag VLAN 0 priority 4 as in the SCD, TAL 180 s, one frame per MaxTime 60 s (SCD) |
| Host hears the VM | C19_Y1005 "receiving" on the host |
| VM hears the host | VM GOOSE tool: C06_Y130, C17_Y230, C11_Y120, C12_Y220 "Receiving"; C19_Y1005's "4 not heard" cleared |
| Change host → VM | C17_Y230 ConInGGIO1.Ind101 set on the host: stNum 3 → 4, sqNum 0-3 burst; the VM tool showed "state 4" |
| Change VM → host | C19_Y1005 Ind101 set to True in the VM's Data points: burst stNum 2, sqNum 0-3 at 01:18:33.217-.241, then every 10 s; host shows stNum 2 |
| Loss and recovery | C19_Y1005 stopped on the VM: host "timeout" 31 s after the last frame (TAL 30 s); *Run this IED*: "receiving", stNum 1, within 2 s |
| C02_BCU "4 not heard" | Correct: its SCD subscribes to C06_BCU, C11_BCU, C12_BCU, C17_BCU, which no machine runs |
| SLD GOOSE layer (VM) | Tags C02_BCU ⚠4 and C19_Y1005, "8 senders outside project"; on each host change only the grey label "C17_Y230 → C19_Y1005 · stNum N" appears near the tag for about 1 s |

## Findings (not fixed here)

Re-checked with own files and fixed in [0102](0102-goose-visibility-and-timing.md). Finding 1 turned
out narrower: at normal zoom the dot did move; it vanished only when zoomed out.

1. **No visible GOOSE movement for senders outside the project.** The owner: "i dont see goose
   moving". Screenshots every ~0.6 s caught only the label, never the dot. By design no line is drawn
   for an outside sender, and the dot (r 5) starts left of the leftmost equipment. Together the
   change is almost invisible. Idle repeats draw nothing, by design.
2. **Lab-mode IED without a lab address claims the lab network.** C09_BCU (imported after the host's
   lab setup) has no lab address: the page says "Clients connect to 127.50.0.2:102 · On the lab
   network adapter". The role line says "reachable from this PC only", and the top bar counts it in
   "9 IEDs reachable from VMware Network Adapter VMnet8" (8 are).
3. **Host burst spacing exceeds TAL.** Host burst gaps were 10, 15 and 15 ms with TAL 12 ms
   (3 × MinTime 4 ms). The VM's gaps were 8, 5.5 and 10 ms. It looks like Windows timer granularity
   on the host, and a strict subscriber may briefly flag the stream.
4. **GOOSE partners status says "Outside project" while frames arrive.** VM C19_Y1005's partners
   table shows "Outside project" for all four host senders, which the GOOSE tool shows as
   "Receiving". The receive state is not in the table.
5. **Clipped SLD labels.** VM push buttons read "PUSH_BUTTO…".

Not a fault: the owner's "Run does nothing on the host" was the Run button of the full-screen
Remote Desktop window (the VM restarted, CB-1 back to Open, latches released). The host received no
click. The SCD gives every block APPID 0x0000, and C09_BCU and C17_Y230 share
01-0C-CD-01-00-00. ScadaBench sends these SCD values unchanged.

## Limitations

- No third-party GOOSE device. The listener is ScadaBench on both sides, plus tshark.
- Captures and the project stay on the owner's machines (customer model names) and are not committed.
- The host left C17_Y230, C11_Y120, C06_Y130 and C12_Y220 running for the owner.
