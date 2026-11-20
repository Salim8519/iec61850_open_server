---
title: "Protocol converter"
description: "Read a gateway or RTU as a master in ScadaBench, then serve the same points to your SCADA over another connection."
section: "tools"
order: 3
---

The protocol converter sits between a device and your SCADA. ScadaBench reads the device first, as a master. It then serves the same points upstream to your SCADA, as a slave, outstation or server.

Use it when you want to test a converter, or when your SCADA cannot talk to the device directly.

## Step 1: Read the device

1. Open **Tools**, then **SCADA gateway & master**.
2. Choose **Protocol converter**.
3. In the left column, type the device **Host** and **Port**, and pick its protocol.
4. Press **Connect**.
5. Wait until the device values appear under **Device points**.

Untick any device point you do not want to serve.

## Step 2: Serve to SCADA

1. Set the upstream side in the column **Converter settings**.
2. Press **Start serving**.
3. Connect your SCADA to ScadaBench.

Press **Stop serving** when you are done.

The **Served to SCADA** tab shows what your SCADA sees. Each watched device point is listed with its upstream address.

## Command routes

IEC 104 devices do not say which addresses accept commands. For each command your SCADA may send, add a route:

1. Press **Add route**.
2. Type the **Upstream address** that your SCADA uses.
3. Pick the **Upstream command** (double or single command).
4. Pick the **Device command** and type the **Device address** it becomes.
5. Press **Save routes**.

DNP3 binary outputs and Modbus coils pass through without a route.

## Passing commands to the device

The switch **Pass SCADA commands to the device** controls whether commands reach the device.

- **On**: commands from your SCADA on the relayed points go on to the real device.
- **Off**: the device is only read. Your SCADA gets a negative answer for every command.

## Tabs

- **Device points**: the points read from the device.
- **Served to SCADA**: what your SCADA sees.
- **Command routes**: the routes for IEC 104 commands.
- **SCADA traffic** and **Device traffic**: the messages on each side.
- **Commands**: every command in three steps: what was asked, what the device did, and what ScadaBench saw afterwards.
