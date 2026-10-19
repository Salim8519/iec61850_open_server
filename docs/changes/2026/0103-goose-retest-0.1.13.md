# 0103: GOOSE host ↔ VM retest on 0.1.13, and one new finding

Date: 2026-10-19. Evidence only, no code changed. Retests [0102](0102-goose-visibility-and-timing.md)
(ADR-0070) on the installed test build 0.1.13 on both machines.

## Setup

- **Host:** installed 0.1.13 with the owner's project, lab mode on VMnet8, driven through its API with
  the owner's permission. Windows had dropped the host's lab aliases (192.168.98.21-28). Restoring
  them needs an administrator approval, so the lab addresses of C17_Y230, C11_Y120, C06_Y130 and
  C12_Y220 were cleared for the test. Their GOOSE still goes out on VMnet8 (publishing needs only
  lab mode and the adapter); MMS answered on this PC only. Afterwards all 9 addresses were restored
  exactly.
- **VM:** installed 0.1.13 (192.168.230.130). Its aliases were also gone. The owner re-applied the
  network setup in the VM (one Windows approval) and Run started C02_BCU and C19_Y1005.

## Observed

| Test | Result |
| --- | --- |
| Host → VM, SLD (VM, GOOSE layer on) | On each host change of C17_Y230 ConInGGIO1.Ind101: a fading line arcs into C19_Y1005's tag, label "C17_Y230 → C19_Y1005 · stNum N" (N 3, 7, 8 captured), tag line "← C17_Y230 · stNum N"; nothing remains after ~2 s |
| GOOSE partners (VM, C19_Y1005) | C06_Y130, C11_Y120, C12_Y220, C17_Y230 all "Receiving" (0.1.12: "Outside project") |
| VM → host | C19_Y1005 ConInGGIO1.Ind101 set to True in the VM's Data points: host watch stNum 1 → 2 at 09:00:39 UTC, "receiving" |
| Clipped names (VM SLD) | "PUSH_BUTTON-2/3/4" shown in full on two lines (0.1.12: "PUSH_BUTTO…") |
| Lab IED without address (host) | C09_BCU: "Only programs on this PC can reach it: it has no address on the lab network yet" |
| C02_BCU "4 not heard" | Unchanged and correct (its SCD senders C06_BCU … C17_BCU run nowhere) |

## New finding (not fixed)

- **Reachability is shown from the project, not from Windows.** Both machines lost their lab aliases
  (Windows keeps them only until a restart or adapter reset), yet both apps still said
  "2 IEDs reachable from “Ethernet0”" or "9 … VMnet8", and Settings said "Your published IEDs are
  on “Ethernet0”". The user learns this only when Run fails with "… needs address …, which is not
  set up". The app should compare with the adapter's real addresses and offer *Put them back*.
- Typing an address that the VM's adapter already has showed "Free · typed by you", not
  "Already on adapter", and the preview wanted a firewall rule. Not investigated further.

## Limitations

- No wire capture this time; host-side results are from the host app's own GOOSE watch.
- The moving dot itself was not caught in a still photo; the line, label and tag line were.
- Photos (customer IED names) were sent to the owner only and are not committed.
