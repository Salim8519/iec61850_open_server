---
title: "SNMP devices"
description: "Simulate network switches and GPS clocks that answer SNMP, or read any SNMP device and receive its traps."
section: "tools"
order: 6
video: "snmp"
---

SNMP is how switches, GPS clocks and UPS units report their health. The SNMP devices tool works in two roles:

- **Simulate devices**: ScadaBench plays the device. Your SCADA or network monitor reads it and receives its traps.
- **Read devices**: ScadaBench reads a real or simulated SNMP device, shows its values with names and receives its traps.

Both roles can run at the same time.

## Simulate a device

1. Open **Tools**, then **SNMP devices**.
2. Choose **Simulate devices**.
3. Press **Add device** and pick one of:
   - **Network switch (8 ports)**
   - **GPS time server**
   - **Empty device**
   - **Copy of a device (snmpwalk file)…**
   - **From a ScadaBench CSV file…**
4. Press **Start devices**.

Each device answers SNMP on its own address, port 161, until you press **Stop devices**. The devices start again by themselves when ScadaBench opens.

If Windows' own SNMP service already uses port 161, ScadaBench tells you. Then change the **Port** to 1161 in the device settings.

To reach a device from another PC, put your project on a network first. See the Network section. Each SNMP device then gets its own free address on the same card as the IEDs.

## Values, settings, traps and requests

Each device has four tabs:

- **Values**: each value with its name. Values change like the real device.
- **Settings**: name, address, port, SNMP versions and users.
- **Traps**: traps the device received or sent.
- **Requests**: the SNMP requests the device answered.

Changes are saved as you make them. If the devices are running, each saved change restarts them.

## SNMP versions

- **v1 and v2c** use a community name, for example `public`. Managers that use the read community can only read.
- **v3** uses a user name with a security level:
  - **No password**: anyone who knows the user name can read.
  - **Password only**: messages are signed, so they cannot be changed or faked.
  - **Password + encryption**: messages are signed and encrypted.

Passwords need at least 8 characters.

## Read a device

1. Choose **Read devices**.
2. Type the address and the community or user of the device.
3. Press **Connect**.

ScadaBench reads every value once with a full walk, shows the important values on top and reads them again by itself. Traps the device sends are listed too.

Values are named from MIB files when they are available.

## Delete a device

Select the device, open its menu with **More actions for this device**, and press **Delete device**. Its settings and values are removed from the project.
