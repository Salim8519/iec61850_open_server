---
title: "SCADA gateway"
description: "Let your SCADA connect to the IEDs of your project over IEC 104, Modbus TCP or DNP3, and send commands to them."
section: "tools"
order: 1
video: "scada-gateway"
---

The SCADA gateway lets your SCADA system talk to the IEDs in your project. ScadaBench acts as the substation gateway. Your SCADA connects to ScadaBench, and ScadaBench answers with the values of the IEDs.

Three protocols are available:

- **IEC 60870-5-104** (IEC 104): the usual link between a substation and a control centre.
- **Modbus TCP**: a simple register protocol used by many PLCs.
- **DNP3**: common in North American utilities.

## Before you start

- Add your IEDs in **Devices**, for example the sample IED **Feeder01**.
- Decide where your SCADA runs. If it runs on this PC, use **This PC only**. If it runs on another PC, use **All networks** (see the Network section).

## Start the gateway

1. Open **Tools**, then **SCADA gateway & master**.
2. Choose **Station gateway**.
3. Pick the protocol: IEC 60870-5-104, Modbus TCP or DNP3.
4. Under **Who can connect**, choose **This PC only** or **All networks**.
5. Press **Start gateway**.
6. Point your SCADA at the address and port shown under **SCADA connects to**.

When you press **Start gateway**, ScadaBench prepares the point list for you and starts listening. It does not start the IEDs.

## Start the IEDs with Run

The gateway and the IEDs are separate. This is on purpose.

- **Start gateway** starts listening for SCADA.
- **Run** starts the IEDs.

If the gateway runs but the IEDs are off, the page says so. Every point is sent to SCADA as invalid until you press **Run the IEDs** (or **Run** at the top). An invalid point is never made up. It is simply marked as not valid.

## Points

Each row in the **Points** tab is one point that SCADA can read or command.

1. Press **Propose from IEDs** to fill the list. ScadaBench keeps only what SCADA usually needs: positions, alarms and trips, measurements, device health and commands.
2. Use **Add points** to add more. You can add points from an IED, made-up points, or rows pasted from Excel.
3. Change any address so it matches your SCADA point list. Every address can be edited.
4. Use **Export CSV** and **Import CSV** to move the list to and from a file. The columns are `type,address,ied,reference,scale,description`.

The point kinds are:

| Kind | Meaning |
| --- | --- |
| Single point | ON or OFF status |
| Double point | Open or closed position |
| Measurement | Measured value (decimal) |
| Integer measurement | Whole number value |
| Single command | ON or OFF command |
| Double command | Open or close command |

You must press **Stop gateway** before you change the point list. The page says "Stop the gateway to change the point list."

## Commands from SCADA

SCADA commands operate the IEDs over real IEC 61850 MMS.

- Commands to **simulated** IEDs work as soon as the gateway runs and the IED runs.
- Commands to a **real** IED need two switches on:
  1. **Allow SCADA commands to real IEDs** in the gateway settings. This is on by default.
  2. **Allow commands** for that device in **Devices**. This is off by default.

You can also send a command from the **Points** tab with a **Send** button.

1. Press the **Send** button of the point.
2. Read the warning. It names the IED and the object.
3. Press **Yes, send** and the command name to confirm. Press **Cancel** to stop.

The warning for a real device is stronger. Make sure the device is safe to switch first.

## Commands tab

The **Commands** tab shows every command in three steps:

1. What was asked.
2. What the IED did.
3. What ScadaBench saw afterwards.

A command from this page is logged as **You · this page**.

## Traffic tab

The **Traffic** tab shows the messages between SCADA and the gateway, newest at the bottom. Use **Pause**, **Resume** and **Clear view** to control it.

The status line shows how many SCADA connections are open, for example **1 SCADA connected** or **No SCADA connected yet**.

## Allowed SCADA address (optional)

You can type one SCADA address under **Allowed SCADA address**. Only that address may connect. Other callers are refused and shown in the **Traffic** tab. Leave it empty to let anyone connect.

DNP3 checks the master address instead, so any SCADA address may connect to it.

## Own address (optional)

Like a real gateway, the gateway can have its own address. Type it under **ScadaBench Lab**. Only programs on this PC can reach that address. For another PC or a virtual machine, choose **All networks** instead.

The ScadaBench Lab card must exist first. See the Network section.

## Stop the gateway

Press **Stop gateway**. The remembered settings stay in place. If the gateway was running when you closed ScadaBench, it starts listening again by itself the next time. The IEDs still start only with **Run**.
