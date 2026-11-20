---
title: "Find IEDs"
description: "Scan a network for IEC 61850 devices on port 102 and see each device's vendor, model and logical devices."
section: "tools"
order: 5
video: "ied-explorer"
---

Find IEDs scans a network for IEC 61850 devices. It checks each address for port 102 and asks every device that answers who it is. It sends nothing else.

## Scan a network

1. Open **Tools**, then **Find IEDs**.
2. Pick the **Network adapter** the devices are plugged into.
3. Type the **Addresses to check**. You can type a network such as `192.168.1.0/24`, one address such as `192.168.1.50`, or a range such as `192.168.1.20-40`. At most 256 addresses are checked at once.
4. Check the **Port**. It is 102 for IEC 61850.
5. Press **Find IEDs**.

The page says how many addresses it will check, for example **Will check 254 addresses, 192.168.1.1 to 192.168.1.254, on port 102.**

Press **Stop** to end the scan early. Press **Scan again** to repeat it.

## Read the results

Each device that answers is listed with:

- its address,
- its vendor, model and firmware,
- its logical devices.

A device that is already in this project is marked **Already in this project**.

Press **Open in Explorer** to open a device in IED Explorer.

To test with one of these devices in a project, import its CID or SCD file in **Devices** and choose **Connect to real device**.

## No device answered

Check these things:

1. The cable is plugged in, and the adapter you picked is the one the devices use.
2. The devices are switched on.
3. The address range matches the devices' network.

If you scan `127.x` addresses on this PC, the simulated IEDs answer only while **Run** is on. Press **Run** first, then scan again.
