---
title: "GOOSE and Sampled Values traffic"
description: "See which GOOSE and Sampled Values messages really arrive on the network, and compare them with your SCD."
section: "tools"
order: 7
video: "goose"
---

The traffic tool shows which GOOSE and Sampled Values messages are really on the network. It checks every GOOSE block and Sampled Values stream of your project against what arrives.

Open it from **Tools**, then **GOOSE & Sampled Values traffic**.

## What you need

- **GOOSE** works inside ScadaBench with nothing extra.
- **Sampled Values** need the free Npcap driver and the ScadaBench Lab network. Without them, the page shows **Lab network only**.
- To see GOOSE from other PCs or real devices, install Npcap too. See the GOOSE and Npcap page in the Network section.

## Read the status

Each GOOSE block shows one state:

| State | Meaning |
| --- | --- |
| Not running | The block is not running. Press **Run** to start sending and listening. |
| Not received yet | ScadaBench is listening but no message has arrived. |
| Receiving | Messages from this block are arriving. |
| Received with a problem | Messages arrive, but they do not match the SCD. The page gives the reason. |
| Stopped arriving | Messages arrived before but have stopped. |

If a state stays on **Not received yet**, check that the publisher runs and is plugged into the same network.

## Where GOOSE goes

The **Where GOOSE goes** view shows which IEDs receive each GOOSE block. Use it to check that a trip or interlock message reaches the right IED.

## Tips

- Compare the traffic with what you expect from the SCD.
- Wi-Fi often drops GOOSE from simulated IEDs. Use a cable when you can.
- The traffic view only reads messages. It does not change them.
