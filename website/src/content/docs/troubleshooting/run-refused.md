---
title: "Run refused"
description: "What to do when ScadaBench will not start the IEDs: address in use, an address refused by Windows, or a disconnected network card."
section: "troubleshooting"
order: 1
---

When you press **Run** and the IEDs do not start, ScadaBench gives a reason. Read the message on the page. The **Activity** list keeps the same reason with each IED and address.

Find the case that matches your message below.

## Case 1: Address or port in use

The message says another program is using the port or the address. For example:

> Feeder01 cannot listen on 127.50.0.2:102 because another program is using it.

IEC 61850 uses port 102. Only one program can use it at an address at a time.

What to do:

1. Close other IEC 61850 tools that may be open: other IED simulators, IEDScout, and an old copy of ScadaBench.
2. In ScadaBench, press **Stop** for the IEDs, then **Run** again.
3. If the message names a program, close that program. Otherwise, check that no other copy of ScadaBench or another SCADA test tool is open.

## Case 2: Address refused by Windows

The message names the address and gives a reason. Common reasons:

| Reason in the message | What to do |
| --- | --- |
| The address is not an address of this PC now | Open **Settings › Network & GOOSE** and press **Repair** on the ScadaBench Lab card, or pick the card again. Then press **Run**. |
| Windows does not allow the port | Check your security programs and Windows Firewall. See the Firewall prompts page. |
| Windows error number (for example `10013`) | Note the number. Email **support@scadabench.com** with the Activity list. |

A message for a network card can also say that the address is already on another adapter. Pick another address for that IED. The Lab network page explains the status words.

## Case 3: Card disconnected

The message says the card has **No cable**, or that the address could not be added because the link is down.

1. Plug the cable into the network card or switch.
2. Wait a few seconds. The list updates by itself.
3. Press **Run** again.

Addresses are never added to a card with no link. Nothing is changed while the card is down.

## Where to see the reason

- The message under the **Run** button.
- The **Activity** list, which names each IED, its address and the reason.

If you still cannot start the IEDs, see the Troubleshooting pages for an IED that cannot be reached, or for a network address that is gone.
