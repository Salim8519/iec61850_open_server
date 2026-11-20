---
title: "Address gone and disconnected messages"
description: "What the messages Address gone, No cable, Connection lost and Stopped arriving mean, and what to do about each one."
section: "network"
order: 6
---

ScadaBench shows short status words when a network address or a connection goes away. This page explains each one.

## Address gone

**Running, address gone** means the gateway is listening on one address of this PC, and Windows no longer has that address.

ScadaBench checks the address every 2 seconds. The page shows a message that names the address, for example:

> Address gone: 198.18.21.50 is no longer on this PC

SCADA cannot connect while the address is gone.

Common causes:

- the ScadaBench Lab card was removed or repaired,
- Windows restarted and the address was not put back,
- another program took the address off,
- you pressed **Clean up** while the gateway was running.

What to do:

1. Press **Stop gateway**.
2. Press **Start gateway** again. The address goes back on the card.
3. If it still fails, open **Settings › Network & GOOSE** and press **Repair** on the ScadaBench Lab card.

The status clears by itself when the address is back.

## No cable

**No cable** means Windows reports that the network card is not connected.

- Plug the cable into the card or the switch.
- The list updates by itself. You do not need to press a button.
- If you start the IEDs without a cable, their addresses are not added to the card. Plug the cable in, then press **Start** again.

## Not connected and Connection lost

These two words mean an IED or a device is not linked to ScadaBench.

| Word | Where you see it | Meaning |
| --- | --- | --- |
| **Not connected** | IED Explorer, SCADA master | Nothing is open. Press **Connect**. |
| **Connection lost** | IED Explorer, real IED | The link broke. Check the cable and the address, then press **Connect**. |
| **Link lost, trying again** | SCADA master | The device stopped answering. ScadaBench tries again by itself. |
| **Stopped with a problem** | SCADA master | Read the message under the status. |
| **Old value** | A real IED's values | The connection was lost. The value may have changed since. |

ScadaBench never shows a lost value as fresh. A point with no fresh value is sent to SCADA as invalid.

## Stopped arriving

**Stopped arriving** is a GOOSE status. Messages came before but have stopped. The publisher may be off, unplugged or restarted. See the GOOSE and Npcap page.

## Lab card addresses not on the card

If the card no longer shows an address that ScadaBench used, the page says it was not on the card (Windows restarted, the card was repaired or another program took it off). There is nothing to take off, so ScadaBench skips it.
