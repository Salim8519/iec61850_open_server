---
title: "Monitor a real device"
description: "Connect ScadaBench to a real IEC 61850 IED to read its live values. It reads only, unless you turn on Allow commands for that IED."
section: devices
order: 3
video: real-devices
---

Use **Monitor real device** when the real relay is on your network. ScadaBench connects to it as a
client and shows its live values. It does not change the device.

## Read only by default

- ScadaBench only reads a real IED.
- It sends **no commands** unless you turn on **Allow commands** for that IED, and the gateway also
  allows commands to real IEDs.
- **IED Explorer** is a separate tool. It asks before every command, see the Tools section.

## Set up the connection

1. Click the IED in the **Devices** list.
2. Choose **Monitor real device** under **What ScadaBench does with this device**.
3. In the connection block, check the **Device address**. ScadaBench fills it from the SCD file. Type
   another address only if the device uses a different one. The line under the box says **From the
   SCD file** or **Typed by you**.
4. Check the **Port** (usually 102).
5. Under **How ScadaBench reads values**, choose one option:
   - **Reports (recommended)**: ScadaBench uses a report block only if the IED shows it as free. It
     tells you which one, and gives it back on Stop. If none is free, it asks for values instead.
   - **Ask every second**: the safest choice. ScadaBench only asks for values and never touches the
     report blocks. Updates are slower.
6. Press **Save connection**. The button is grey until you change something.

## Connect and read

1. Press **Run** at the top right. Real IEDs connect with the simulated ones.
2. Wait until the IED shows **Live**. Until then its commands stay grey.
3. Open the **Data points** tab to see the values. If the connection is lost, a value is shown as
   **Old value**: it may have changed since the last update.
4. Press **Stop** to disconnect. ScadaBench gives back every report block it used.

You can also press **Connect** and **Disconnect** in the IED's details, while the other IEDs keep their
state.

## Allow commands (optional)

Commands are for testing with the real equipment. Turn them on only when you mean to operate it.

1. Open the IED's **Overview** tab.
2. Tick **Allow commands**. The text under the tick explains what is on.
3. Press **Save connection**.
4. Press **Run** and wait for **Live**.
5. Press **Send commands…**. A pop-up lists each control, such as `Close` and `Open` for a breaker.
6. Press a command. A red box asks you to confirm. Press **Send** only if you are sure.
7. The result says what the device answered and where the position went.

Each command goes to the real device with its own control model. The device checks it, and it can
refuse it. Turn **Allow commands** off again when you are done.

Commands from SCADA through the gateway reach the real IED only when both switches are on: here,
and in the gateway settings in the Tools section.

## Before you operate

ScadaBench reads how the device takes each command, and whether it is in **Local**. A device in
Local refuses remote commands until someone switches it to Remote. ScadaBench only reads this; the
device always has the last word.
