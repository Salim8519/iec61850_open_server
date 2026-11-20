---
title: "Use ScadaBench in a virtual machine"
description: "Let virtual machines on your licensed PC use its license with no key and no internet, with Hyper-V, VMware or VirtualBox."
section: "licence"
order: 4
---

A virtual machine (VM) on your licensed PC needs no key, no file and no internet. It uses the license of the PC it runs on. This works with Hyper-V, VMware and VirtualBox.

## Step 1: Share the license on the host PC

The host PC is the computer that runs the virtual machine.

1. On the host PC, activate ScadaBench first, if it is not activated yet.
2. Open **Settings**, then the **License** tab.
3. Under **Virtual machines**, turn on **Let my virtual machines use this license**.

The switch then shows one of these lines:

- **On: virtual machines on this PC can use this license.**
- **On, but the helper is not running yet.** The helper starts the next time you sign in to Windows.

## Step 2: Allow the virtual machines through Windows Firewall

If the section says **Your virtual machines cannot reach the helper yet: Windows Firewall stops them.**:

1. Press **Allow through Windows Firewall…**.
2. Read **What ScadaBench will change**.
3. Press **Yes, apply**.
4. Approve the Windows prompt (UAC).

The page says **Windows Firewall already lets your virtual machines ask this PC** when this step is done. **Clean up** in **Settings › Network & GOOSE** removes the rule again.

## Step 3: Check the network of the virtual machine

The VM must share a network with the host PC. Use the setting that matches your software:

- **Hyper-V**: use a virtual switch that the host PC can see, such as the **Default Switch**.
- **VMware**: use a **NAT** network or a **Host-only** network.
- **VirtualBox**: give the VM a network adapter set to **Host-Only Adapter**.

If the VM cannot find the host, check its network adapter first.

## Step 4: Activate in the virtual machine

1. In the VM, install ScadaBench and open it.
2. On the activation screen, open **Using a virtual machine?**. You do not need to type anything.
3. Wait a few seconds. The page shows **Looking for a licensed PC on the virtual network…** and then continues by itself.

When it works, **Settings › License** in the VM shows **Licensed through your host PC**.

## Host key

Both PCs show a **Host key**. The two keys must match. Use it to check that the VM uses the right host.

## Notes

- The VM keeps working offline while the host PC is switched on with the option ticked.
- Untick **Let my virtual machines use this license** to stop sharing at any time.
- A VM copied to another computer stops. It needs a license on that computer.
