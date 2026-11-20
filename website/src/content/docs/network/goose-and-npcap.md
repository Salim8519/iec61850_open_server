---
title: "GOOSE and Npcap"
description: "How GOOSE works inside ScadaBench, when you need the free Npcap driver, and what the GOOSE status words mean."
section: "network"
order: 4
video: "goose"
---

GOOSE is the fast message that IEDs send each other for trips and interlocks. This page explains when you need the free Npcap driver, and what the GOOSE status words mean.

## GOOSE inside ScadaBench

GOOSE between simulated IEDs in the same ScadaBench project works without Npcap.

## When you need Npcap

Install the free Npcap driver if you want to:

- share GOOSE from ScadaBench with other programs or PCs on the network,
- hear GOOSE from real devices,
- receive Sampled Values.

Get Npcap from **npcap.com**, install it, then press **Run** again in ScadaBench.

ScadaBench shows the state in the **About** page:

- **Npcap installed**: GOOSE can be shared with the network and real devices can be heard.
- **Npcap missing**: GOOSE works inside ScadaBench only.

If you press **Run** without Npcap and try to share GOOSE, the page says: "GOOSE sharing needs the free Npcap driver. Install it from npcap.com, then press Run again."

## Sampled Values

ScadaBench listens to Sampled Values and checks them. It does not publish them. Sampled Values need the ScadaBench Lab network. Without it, the status reads **Lab network only**.

## Sending GOOSE

- A simulated IED with GOOSE in its SCD shows **Sends GOOSE** while it runs.
- An IED without GOOSE in the SCD shows **GOOSE off**, or **No GOOSE in SCD**.

## GOOSE status words

| Status | Meaning |
| --- | --- |
| **Not running** | Press **Run** to start sending and listening. |
| **Not received yet** | ScadaBench is listening. If this stays, check that the publisher runs and is on the same network. |
| **Receiving** | GOOSE from this block is arriving. |
| **Received with a problem** | Messages arrive but do not match the SCD. The page gives the reason. |
| **Stopped arriving** | Messages arrived before but have stopped. The publisher may be off, unplugged or restarted. |

## Reasons for a problem

| Reason | What to do |
| --- | --- |
| **Unreadable message** | Check the publisher configuration. |
| **Configuration revision differs** | The publisher runs a different configuration than the SCD. Update the IED or the SCD so both match. |
| **Different dataset** | The publisher sends a different dataset. Check the GOOSE block dataset. |
| **Different GOOSE ID** | The publisher sends another GOOSE ID. Load the matching configuration. |
| **Publisher needs commissioning** | Finish commissioning the publishing IED. |
| **Messages stopped** | Check the cables and that the publisher is running. |

To see what really arrives, open **Tools**, then **GOOSE & Sampled Values traffic**.
