---
title: "GOOSE not seen"
description: "Find out why a GOOSE block shows Not received yet, Received with a problem or Stopped arriving, and how to fix it."
section: "troubleshooting"
order: 3
video: "goose"
---

GOOSE is not seen when a block shows **Not received yet**, **Received with a problem** or **Stopped arriving**. Work through the steps below.

## Not received yet

ScadaBench is listening, but no message has arrived.

1. Check that the publisher runs. Press **Run** for the publishing IED.
2. Check that the publisher and the subscriber are on the same network setting. Both must be on **This PC only**, or both on the ScadaBench Lab card, or both on the same real network card.
3. If the GOOSE must reach another PC or a real device, install the free Npcap driver and share GOOSE with the network. See the GOOSE and Npcap page.
4. If you use Wi-Fi, switch to a cable. Wi-Fi often drops GOOSE.
5. Open **Tools**, then **GOOSE & Sampled Values traffic**, and check what really arrives.

## Received with a problem

Messages arrive, but they do not match the SCD. The page gives the reason:

- **Configuration revision differs**: the publisher runs a different configuration than the SCD. Update the IED or the SCD so both match.
- **Different dataset**: check the dataset of the GOOSE block.
- **Different GOOSE ID**: load the matching configuration.
- **Publisher needs commissioning**: finish commissioning the publishing IED.
- **Unreadable message**: check the publisher configuration.

## Stopped arriving

Messages came before but have stopped. The publisher may be off, unplugged or restarted.

1. Check that the publisher is switched on and running.
2. Check the cable.
3. Press **Run** again for the publishing IED.

## Npcap is missing

GOOSE works inside ScadaBench without Npcap. Sharing GOOSE with other PCs and hearing real devices needs Npcap. The About page shows **Npcap missing** until you install it.
