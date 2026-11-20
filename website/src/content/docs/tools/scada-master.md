---
title: "SCADA master"
description: "Use ScadaBench as the SCADA master to test an RTU, gateway or outstation: read every point and send commands after a confirmation."
section: "tools"
order: 2
---

The SCADA master role turns ScadaBench into the SCADA. ScadaBench connects to your gateway, RTU or outstation, reads every point and can send commands.

Use it to check a real or simulated device before you connect the real SCADA.

## Connect to the device

1. Open **Tools**, then **SCADA gateway & master**.
2. Choose **SCADA master**.
3. Pick the protocol:
   - **IEC 60870-5-104**: ScadaBench is the controlling station (104 master).
   - **DNP3**: ScadaBench is the DNP3 master.
   - **Modbus TCP**: ScadaBench is the Modbus client.
4. Type the **Host** address of the device, and its **Port**.
5. Press **Connect**.

The status shows what is happening:

- **Not connected**: nothing is open.
- **Connected**: the link is up.
- **Link lost, trying again**: the device stopped answering. ScadaBench tries again by itself.
- **Stopped with a problem**: read the message under the status.

Press **Disconnect** when you are done.

## Finding the points

- **IEC 60870-5-104** and **DNP3** ask for every value by themselves when they connect. All points appear without typing addresses.
- **Modbus TCP** cannot list its points. Tell ScadaBench which ranges to read, or use **Find registers** to discover them.

## Settings

The main settings are shown first:

- **IEC 60870-5-104**: TCP port and Common address (ASDU).
- **DNP3**: TCP port, Outstation address and Master address.
- **Modbus TCP**: TCP port and Unit ID.

Advanced settings hold the timers and the other protocol options. Change them only if your device asks for it. The defaults work with most devices.

## Sending a command

1. Choose the command kind:
   - **Double command (C_DC)** and **Single command (C_SC)** for IEC 104 breakers and switches.
   - **Regulating step (C_RC)** for a tap changer.
   - **Setpoint** kinds for values such as a voltage setpoint.
   - **Binary output (CROB)** and **Analog output** for DNP3.
   - **Write coil**, **Write register** and **Write float** for Modbus.
2. Type the command address. For IEC 104, the status point and the command point often have different addresses. Check the command address in the device point list.
3. Press **Review…** and read the confirmation.
4. Confirm the command. Press **Cancel** to stop.

The **Command** log shows each command with the device's answer, the last cause and the time.

## Check your point list

Use the point table to see what SCADA received. ScadaBench shows whether each point was received, is missing, or has a different type than you expected.

## Stop or disconnect first

You must **Disconnect** before you switch to another role. The role buttons are greyed out while a connection is open.
