# Real VM test: VM uses the host's license (2026-10-22)

Release 0.1.20, installed build. Host: the owner's laptop (Windows 11, licensed, lifetime key).
VM: VMware Workstation Pro guest (Windows 10) at 192.168.230.130, reached over RDP.
Change: [0156](../changes/2026/0156-activate-once-vm-host-key.md), decision ADR-0099.

## Steps the user follows (recorded)

1. **Host:** activate ScadaBench with the key (once, as usual).
2. **Host:** Settings > License > Virtual machines > tick **Let my virtual machines use this license**.
   The firewall line shows "Windows Firewall already lets your virtual machines ask this PC" (or offers
   the previewed rule, which needs one Windows prompt).
3. **VM:** install ScadaBench and open it. No key, no file, no internet needed.
   - If the VM was already open on the activation screen, it continues on its own; otherwise it opens
     straight into the app.
4. **VM:** Settings > License shows **Licensed through your host PC** and the host key (`wvPO2…SdbE`,
   the same as on the host).

## Observed

| Check | Result |
| --- | --- |
| VM had no license files (moved to a backup folder beforehand) | yes |
| Host helper `scadabench_hostkey.exe` serving | 00:44:54, on 192.168.230.1 (VMnet8), 172.25.96.1, 169.254.143.146, port 47862 |
| VM opens without the activation screen | yes |
| VM Settings > License | "Licensed through your host PC", host key wvPO2…SdbE |
| VM Run | not refused for the license; it stopped because the VM's lab address 192.168.230.41 was not set up (network, unrelated) |

## Findings

- **Bug (small), now addressed:** right after ticking the box, the host showed "On, but the helper is not
  running yet. It starts the next time you sign in to Windows." The helper was in fact started (log 00:44:54)
  and served the VM. The status was read before the helper took its lock. Turning sharing on (and the
  refresh at app start) now waits up to 3 seconds for the helper to take its lock before Settings reads the
  state. Test: `TestTurningSharingOnWaitsForAHelperThatIsStillStarting`
  ([0157](../changes/2026/0157-vm-box-and-guide.md)). Not yet re-checked on a real PC with a new build.
- **UX gap (owner), now addressed:** the VM activation screen explained VMs only in one line at the very
  bottom, below the offline steps. It now has a closed, bold "Using a virtual machine?" box right after
  "Server & offline options" (three numbered steps, a "Looking for a licensed PC" line, a note); the screen
  keeps looking for the host every 6 seconds while the box is closed. Checked by the frontend copy and
  structure tests and by screenshots of the running app (the website guide images); not yet tried on a real
  VM with a new build ([0157](../changes/2026/0157-vm-box-and-guide.md)).
- Not yet covered on a real VM: simulator engines running under the host proof (needs the VM lab
  network set up), stopping the helper (VM must stop), copying the VM to another PC (must stop).
- Screenshots of this run show the owner's private desktop and engineering files; they are not used
  for public material. Tutorial images come from the demo station only.
