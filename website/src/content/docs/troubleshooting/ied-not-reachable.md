---
title: "IED not reachable from another PC"
description: "Check the address, the network, the firewall and the port when another PC cannot reach a simulated IED in ScadaBench."
section: "troubleshooting"
order: 2
---

Your IED runs on this PC, but another PC or a real device cannot reach it. Work through these checks in order.

## 1. Is the IED running?

Press **Run** and check that the IED shows **Publishing**. An IED that is stopped cannot answer.

## 2. Is the IED on a network?

An IED set to **This PC only** answers on this PC only. Other PCs cannot reach it at all.

Open **Settings**, then **Network & GOOSE**, then **Put my IEDs on a network**. Choose **ScadaBench Lab** or your network card. See the Network section.

## 3. Is the address right?

1. Find the address of the IED in **Devices** or in the address list.
2. From the other PC, use an address on the same network, for example `192.168.1.50`.
3. Both PCs must be on the same network range. A mask such as `/24` means the first three numbers must match.

## 4. Can the other PC ping it?

From the other PC, open a command prompt and type:

`ping 192.168.1.50`

Use the IED's own address. If you get a reply, the network path works. If you get no reply, check steps 2, 3, 5 and 6.

## 5. Is the port open?

IEC 61850 uses port 102. A firewall on either PC can block it.

- On the ScadaBench PC, check **Windows Firewall** as described on the Firewall prompts page.
- On the other PC, allow the program that connects, or allow port 102 for your test network.

## 6. Is the cable connected?

If the card shows **No cable**, plug the cable into the card or the switch. The list updates by itself.

## 7. Still no answer?

- Press **Stop**, then **Run** again. The address goes back on the card.
- Check the **Activity** list for an address that was refused and the reason.
- If the other PC also runs ScadaBench, open **IED Explorer** there, type the address, keep port 102 and press **Connect**.
