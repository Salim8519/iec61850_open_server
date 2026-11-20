---
title: "Firewall prompts"
description: "What Windows Firewall asks when ScadaBench is used on a network, and how to allow the gateway or the virtual machine helper through it."
section: "network"
order: 5
---

Windows Firewall can stop other PCs from reaching ScadaBench. This page explains the prompts you may see and how to allow the traffic.

## The Windows question

The first time a ScadaBench program listens for network traffic, Windows may ask whether to allow it. Allow it only on the network you trust, such as your private test network. Never allow it on an operational substation network.

## Gateway on All networks

When the SCADA gateway listens on **All networks**, the page checks Windows Firewall for you.

- **Other computers can reach the gateway on TCP port 102** (or the port you chose): the firewall is open.
- **Other computers cannot reach the gateway yet: Windows Firewall stops them.**: the firewall needs a rule.
- **Windows made a rule that blocks the gateway**: this usually happens when the "allow access?" question was closed. Allow it now.

To fix it:

1. Press **Allow through Windows Firewall…**.
2. Read **What ScadaBench will change**.
3. Press **Yes, apply**.
4. Approve the Windows prompt (UAC).

ScadaBench changes only the rules it needs for the gateway. Press **Clean up** in **Settings › Network & GOOSE** to undo them.

## Virtual machine helper

If your virtual machines cannot find the licensed PC, the **Virtual machines** section shows **Your virtual machines cannot reach the helper yet: Windows Firewall stops them.**

1. Press **Allow through Windows Firewall…**.
2. Read the changes and press **Yes, apply**.
3. Approve the Windows prompt (UAC).

The rule lets only your virtual network reach one port. **Clean up** removes it again. See the Virtual machines page in the Licence section.

## Check failed

If the page says **Windows Firewall could not be checked**, the firewall state is unknown. Check that Windows Firewall is running, then open the page again.

## Other firewalls

If you use a security program with its own firewall, it may block ScadaBench too. Allow ScadaBench and the port in that program, following its own steps.
