---
title: "This PC only or a network"
description: "Choose where your simulated IEDs answer: only on this PC, on the ScadaBench Lab card, or on a real network card."
section: "network"
order: 1
---

Your simulated IEDs can answer in three places. Pick the one that matches what you want to test.

## The three choices

| Choice | Who can reach the IEDs | Windows changes | Good for |
| --- | --- | --- | --- |
| **This PC only** (loopback) | Programs on this PC only | None | The simplest test. Start here. |
| **ScadaBench Lab** | Programs on this PC, at each IED's own address from the SCL file. Ping works. | One Windows prompt, once, to create the card | Testing with real addresses on one PC, without a virtual machine |
| **A real network card** (Ethernet or Wi-Fi) | Other PCs and real devices on that network | Addresses are added to the card. Windows asks for approval. | Testing with other devices on your network |

## This PC only

- Each IED listens on a private `127.50.0.x` address.
- Only this PC can reach these addresses.
- Nothing in Windows is changed.

This is the default. Choose it when your SCADA and the IEDs both run on this PC.

## ScadaBench Lab

- Each IED answers at its own address from the SCD file, for example `192.168.1.50`.
- Ping, your SCADA and GOOSE on this PC reach the IEDs.
- Nothing goes out on your Ethernet or Wi-Fi.

See the ScadaBench Lab page in this section for the steps.

## A real network card

- The IEDs are placed on the card you choose, such as **Ethernet**.
- Other PCs and real devices on the same network can reach them.
- ScadaBench checks each address before it uses it. Windows asks for approval once, and you can see every change first.

See the Real network cards page in this section for the steps.

## Change the choice

1. Open **Settings**, then the **Network & GOOSE** tab.
2. Press **Put my IEDs on a network**.
3. Pick **This PC only**, **ScadaBench Lab** or a network card.
4. Follow the steps on the screen.

To go back to this PC only, press **Use this PC only**. The page also has **Back to this PC only**.

The IEDs must be stopped to change the network. Press **Stop** first.

## Important

ScadaBench is a laboratory tool. Do not connect it to a live operational substation network.
