---
title: "IED Explorer"
description: "Connect to any IED by its address, browse its data model, watch live values and send a command after one confirmation."
section: "tools"
order: 4
video: "ied-explorer"
---

IED Explorer is an IEC 61850 client. Use it to look inside an IED, see its values and, if needed, operate a breaker or switch.

Explorer works with real IEDs and with simulated IEDs. A simulated IED answers at its address while **Run** is on.

## Connect

1. Open **Tools**, then **IED Explorer**.
2. Type the address of the IED in **IED address**, for example `192.168.1.50`.
3. Check the **Port**. IEC 61850 uses port 102.
4. Press **Connect**.

Press **Disconnect** when you are done.

For a simulated IED on this PC, type its address from the project, for example a `127.50.0.x` address. Press **Run** first.

## Browse the data model

1. Open a logical device.
2. Open a logical node, for example `XCBR1`.
3. Open a data object to see its values.

Each level loads only when you open it, so even large IEDs open quickly. Use **Filter logical nodes** to find a node by name.

Open data objects refresh by themselves every 2 seconds. Press **Read all now** to read them again at once.

**Datasets** and **Report blocks** are listed only. Explorer never enables or reserves a report block, so your SCADA and gateways keep theirs.

## Watch list

Watch the values you care about in one list.

1. Press **+** next to any value to add it to the **Watch list**.
2. Press **Start watching** to refresh the list at the interval you pick under **Refresh every**.
3. Press **Stop watching** to pause it.
4. Press **Export CSV** to save the values to a file.
5. Press **Clear the watch list** to empty it.

## Send a command

Explorer sends a command only after you confirm it.

1. Open the control, for example the breaker position.
2. Choose the command. The page shows **Command:** and what the device reads now.
3. Read the confirmation: `Send "..." to ... on ...?`
4. Press **Yes, send** and the command name to send it. Press **Cancel** to stop.

The command uses the device's own control model. If the device asks for select before operate, Explorer does that for you.

## When the IED does not answer

- The page says **Refreshing paused because the IED did not answer.** Press **Read all now** to try again.
- A status of **Connection lost** means the link is down. Check the cable and the address, then press **Connect**.
- **The device closed the connection. Press Connect to try again.** means the device ended the session.
