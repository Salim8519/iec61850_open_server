---
title: "Real network cards"
description: "Put your IEDs on Ethernet or Wi-Fi: approve each address once, read what Windows asks, and remove the changes with Clean up."
section: "network"
order: 3
---

Use a real network card when other PCs or real devices must reach your simulated IEDs. ScadaBench adds the IED addresses to the card you choose. Windows asks for approval once.

Use a cable when you can. Wi-Fi often drops GOOSE from simulated IEDs.

## Put the IEDs on your card

1. Open **Settings**, then the **Network & GOOSE** tab.
2. Press **Put my IEDs on a network**.
3. Pick your network card, such as **Ethernet**. If the list is old, press **Refresh the network cards**.
4. Wait while ScadaBench checks the addresses. The page says **Checking the addresses…**
5. Press **Show the exact Windows changes** to read the list.
6. Press the button that starts with **Put** and names your card, for example **Put 3 IEDs on “Ethernet”**.
7. Approve the Windows prompt (UAC) when it appears. The prompt may be behind the ScadaBench window.
8. When the page says **The lab network is ready**, press **Done**.

## Read the change list

The review shows one row for each change:

| Result | Meaning |
| --- | --- |
| **Will be added** | ScadaBench adds this address when you press the put button. **Clean up** removes it later. |
| **Already there** | The address already exists. ScadaBench uses it and never removes it. |
| **Will be removed** | ScadaBench added this earlier and removes only this. |
| **Will be switched on again** | ScadaBench had switched off a Windows rule. **Clean up** switches it back on. |

After Windows answers, the result is **Done** or **Failed**. **Failed** means Windows refused this change.

## Address problems

| Message | What to do |
| --- | --- |
| **No cable** | Plug the device or switch into this port. The list updates by itself. |
| **Taken on the network** | Another device answers on this address. Pick a different one. |
| **Card gets its address automatically** | The card uses DHCP, so a fixed address would cut its connection. Pick **ScadaBench Lab** or a card with a fixed address. |
| **Used by another adapter** | Another card of this PC has this address. Pick another address. |
| **No free address** | Type an address in another network. |
| **Free** | No conflict. The address will be added. |

Press **Cancel** to leave without changes.

## Clean up

ScadaBench remembers every change it made. To remove them:

1. Open **Settings**, then the **Network & GOOSE** tab.
2. Find **ScadaBench network changes** and press **Clean up…**.
3. Read the list. Press **Remove these changes** to undo them, or **Keep them** to stop.

Clean up removes only what ScadaBench added. Addresses that were already there stay.

## Go back to this PC only

Press **Back to this PC only** in the network settings. The IEDs then answer on this PC again.
