---
title: "ScadaBench Lab card"
description: "Create the ScadaBench Lab network card, put each simulated IED at its own address, and use Ping, Repair and Remove."
section: "network"
order: 2
video: "lab-network"
---

The ScadaBench Lab card lets each simulated IED answer at its own address from the SCD file. You do not need a virtual machine. Ping, your SCADA and GOOSE on this PC all reach the IEDs.

Nothing goes out on your Ethernet or Wi-Fi.

## Create the card (first time only)

1. Open **Settings**, then the **Network & GOOSE** tab.
2. Press **Put my IEDs on a network**, then choose **ScadaBench Lab**.
3. Press **Create the lab network card**.
4. Approve the Windows permission prompt (UAC). It may open behind the ScadaBench window.

The page says **Waiting for Windows: approve the permission prompt (UAC).** when Windows is working. Creating the card needs no download and no restart.

## Use the card

1. Press **Use the lab network**. This needs no Windows prompt.
2. Press **Run** to start the IEDs.

When you press **Run**, the addresses of the IEDs that start go on the lab card. When you press **Stop**, the addresses come off the card again. The next **Run** puts them back.

## Read the address list

Each IED has one row with its address and a status:

| Status | Meaning |
| --- | --- |
| **Ready** | The address can be used. |
| **Same range as a real card** | Note only. The address is in the same range as a real card. |
| **A real device uses this address** | A device on the network already answers. Pick another address. |
| **Used twice** | Two IEDs have the same address. Give each one its own. |
| **Address of this PC** | The address belongs to this PC's own card. Pick another. |
| **Moves to the lab card** | ScadaBench put this address on a real card earlier. **Use** moves it to the lab card. |
| **Address not valid** | Type an address like `192.168.1.21`. |
| **No address** | Type an address for this IED. |

Press **Show the 3 addresses** (or the number you have) to see the full list.

## Change an address

1. Click the address of the IED.
2. Type another address and press **Enter**.

A running IED moves to the new address at once. The SCD file is never changed.

To go back to the address from the file, press **Use file address**.

## Ping an IED

1. Press **Run**.
2. Press **Ping** and choose the IED.

A reply looks like this: `Reply from 192.168.1.50: time<1ms TTL=128`. If the IED is not running, there is no reply. Press **Run** first.

## Repair the card

After an update, the card may need its service updated. The page then says **Start and Stop can be faster: press Repair to update the lab network service.**

1. Press **Stop** to stop the simulation.
2. Press **Repair**. Windows asks for permission once.

## Remove the card

1. Stop the run.
2. Press **Remove the lab network card**. Windows asks for permission once.

Your IEDs go back to **This PC only**. The card, its addresses and its service are removed.

## Notes

- You can switch the network only while the run is stopped.
- The SCL file stays the same, whatever address you type.
