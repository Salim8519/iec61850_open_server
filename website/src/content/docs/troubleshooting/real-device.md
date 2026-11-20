---
title: "Real device not answering"
description: "What to do when a real IED does not answer, shows old values, or gives the result unknown after a command."
section: "troubleshooting"
order: 4
video: "real-devices"
---

This page covers a real IED that does not answer, shows old values, or gives an unknown result after a command.

## The device does not answer

The status shows **Trying again**, **Could not connect** or **No address**.

- **No address**: type the device address in the connection card.
- **Trying again**: the device did not answer. Check the cable, the address and that port 102 is open. ScadaBench keeps trying.
- **Could not connect**: check the same things, then press **Connect** again.
- **The device closed the connection. Press Connect to try again.**: the device ended the session. Press **Connect**.

Check the network: the real device and this PC must be on the same network range. See the Network section.

## Values are old

A status of **Old value** means the connection was lost. The value may have changed since.

- ScadaBench never shows a lost value as fresh.
- A point without a fresh value is sent to SCADA as **invalid**.
- Press **Read all now** in IED Explorer, or press **Connect** again to read the values.

## Result unknown after a command

ScadaBench shows a message such as:

> Bay1 did not answer in time; result unknown — check the position before trying again

This means the command was sent, but the device did not confirm the result in time. The command may or may not have happened.

1. Do **not** send the command again straight away.
2. Look at the device or the equipment. Check the position, for example on site or in the device's own display.
3. Open the **Commands** tab. It shows three steps: what was asked, what the device did, and what ScadaBench saw afterwards.
4. Send the command again only when you know the current position.

## A command is refused

If a command to a real IED is refused, check both switches:

1. **Allow commands** for that device in **Devices** must be on. It is off by default.
2. In the SCADA gateway settings, **Allow SCADA commands to real IEDs** must be on. It is on by default.

The refusal message explains which switch is off.

## Commands from SCADA stop working

Check that the device is live (status **Live**) and that its **Allow commands** switch is on. A device that is connecting or trying again cannot take commands.
