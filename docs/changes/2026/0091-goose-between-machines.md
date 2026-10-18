# 0091: GOOSE between two machines, IED to IED, and three fixes

Date: 2026-10-18. Spec: [GOOSE](../../specs/goose.md). Follows change 0050, which had only
virtual-adapter evidence on one PC.

Owner request: test GOOSE between the two machines and between IEDs.

## Setup

- **Host (192.168.98.1, VMware VMnet8).** The source app ran on port 18766 with a temporary data
  folder. It was put in lab mode on VMnet8 after one owner UAC approval.
- **VM (192.168.98.130).** ScadaBench 0.1.8 (installed), lab mode on Ethernet0, driven through
  its API.
- **Independent check.** tshark 4.x on the host and Windows `pktmon` on the VM.
- **Engineering data.**
  - `scd/open_substation.scd` and `scd/model_substation.scd` (public samples) only.
  - The owner's installed app was running its own project in lab mode on the same adapter. Its
    frames were ignored, and none of its names are recorded.

## Observed

| Test | Result |
| --- | --- |
| VM IED2_PTOC publishes → host (tshark) | APPID 0x1002, dst 01-0C-CD-01-00-02, gocbRef, dataset, confRev 2, stNum 1, sqNum 0→4, TAL 15 s: all match the SCD |
| VLAN tag | On the VM wire (pktmon): VLAN 1, priority 4, as in the SCD. Received on the host untagged, because the VMware virtual switch strips 802.1Q tags (the owner's own host frames lose their tag the same way). This is not a ScadaBench fault. |
| Host ScadaBench supervises the VM's GOOSE | IED2_PTOC as "Real device" on the host: "receiving", no problem. The owner's streams are listed as "unknown". |
| **IED to IED across machines** (`model_substation.scd`) | Host simulates FEED1 (publishes Feed1SwiPosCtl); VM simulates BUS1, which subscribes to it through ExtRef. On the VM, BUS1's subscription to FEED1 is "receiving". The VM's BUS1 Bus1TripEventCb is received on the host. Blocks not running anywhere (FEED2, TR1, TR2) show "waiting". |
| A change travels by GOOSE | Host FEED1 XSWI3.Pos.stVal set to 2: on the VM, stNum 1→2 and sqNum reset to 0. VM BUS1 PTRC1.Tr.general set to true: on the host, stNum 1→2. |
| Stream loss and recovery | VM BUS1 stopped: the host stays "receiving" within the 30 s time allowed to live, then shows "timeout". Restarted: "receiving" again with stNum 1. |
| SCD diagnostics | FEED1's ExtRef to "Bus1TripEventCn" (the SCD names the block "Bus1TripEventCb"), and open_substation's ExtRef to IED2 "GoCB", are reported as not publishable instead of being guessed. |

## Bugs fixed (each has a test)

1. **sqNum frozen on the page.** Retransmissions on the wire carried sqNum 4, 5 … while the page
   kept showing 0, because only a state change carried sqNum. `goose_monitor.py` now takes sqNum
   from the periodic stats (`tests/unit/test_goose_monitor_stats.py`). Verified live: the page
   followed the wire (14, 16 against 15).
2. **The free-address picker proposed a taken address.** On the VM it proposed 192.168.98.21,
   which the owner's host app was already using, and only reported the clash. The picker now
   picks again around addresses that answer on the network, up to 5 rounds. Typed addresses are
   reported and never changed (`tests/integration/test_network_api.py`).
3. **Stale real-IED text.** "Commands to real IEDs are not enabled in this version, so ScadaBench
   never operates real equipment" is no longer true. It now says that this page only reads, and
   that IED Explorer and the gateway operate real equipment after a confirmation (`guideCopy.test.ts`,
   `test_real_ied_refusal_text.py`, `test_live_roles.py`).

## Limitations and follow-ups

- **A simulated subscriber does not use received values.** By design (GOOSE-08), received GOOSE
  only drives LGOS supervision. A simulated BUS1 does not trip or interlock on FEED1's GOOSE, and
  neither sample SCD has LGOS.
- **Listening needs a simulated IED.** GOOSE listening needs lab mode, and lab mode refuses to
  start with only real IEDs. Watching a real station's GOOSE without simulating anything needs a
  "listen-only" lab mode (decision record to write).
- **The picker adds new addresses instead of reusing ScadaBench's own.** After a project change,
  it proposed a new alias and a new firewall rule (one more UAC prompt) instead of reusing an
  address ScadaBench already owns and no IED uses.
- **Not tested:** a physical switch with VLAN trunking, a third-party GOOSE publisher or
  subscriber (IEDExplorer's GooseSender/GooseExplorer is available on the VM), and Sampled Values.
- **The VM still runs installed 0.1.8,** so fixes 1–3 reach it with the next build.
