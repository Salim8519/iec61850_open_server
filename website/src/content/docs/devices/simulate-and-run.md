---
title: "Simulate and run IEDs"
description: "Choose Simulate, Monitor real device or Skip for each IED, run one IED or every IED, and read the run state at the top of the window."
section: devices
order: 2
video: simulation
---

Every IED in ScadaBench has a **role**. The role decides what ScadaBench does with it when you press
Run.

## The three roles

Open an IED in the list. Under its name, the **What ScadaBench does with this device** control
shows three choices:

| Role | What it means |
| --- | --- |
| **Simulate** | ScadaBench plays this IED. SCADA and other IEDs connect to it. This is the default. |
| **Monitor real device** | ScadaBench reads the real IED on your network. See [Monitor a real device](/docs/devices/monitor-real-device/). |
| **Skip** | The IED stays in the project, with its drawing and GOOSE, but it does not start when you press Run. |

Change the role only while IEDs are stopped. The message **Stop the run to change what ScadaBench
does with this device** explains a grey button.

## Run every IED

1. Press **Run** at the top right. It starts every IED that is not skipped.
2. The run state shows **Running · 3 IEDs**. A live dot marks the places that have running IEDs.
3. Press **Stop** to stop them all.

## Run one IED

You can start one IED and leave the others as they are.

1. Click the IED in the list.
2. Press **Run this IED**. Only this IED starts.
3. Press **Stop this IED** to stop it again. The other IEDs keep running.

When some IEDs run and others do not, the top bar says **Running 1 of 3 IEDs**. Devices shows a
line such as **2 of 3 running**, with a button **Run the other one** (or **Run the other 2**). Press it to
start the rest.

## What a running simulated IED does

- It answers IEC 61850 on its own address, port 102.
- It sends its reports to clients that subscribe.
- It sends its GOOSE messages while the lab network is in use (see the Network section).
- It keeps its data points, so you can change them from the [Test](/docs/test/test-points/) place
  or from the [SLD](/docs/sld/operate-and-test/).

## Add more sample IEDs

Press **Add a sample IED** under the list. A new sample IED is added with a free name. Use a sample to
try ScadaBench without a file.

## Put an IED on a real network

By default a simulated IED answers on this PC only. To let other computers reach it, open
**Settings** and choose a network adapter, or press **Put on a network** on the IED's address line.
The settings are explained in the Network section.
