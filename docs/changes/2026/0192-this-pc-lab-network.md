# 0192: This PC lab network: each IED at its own address without a VM

Date: 2026-10-26. Owner: "instead of VM use a smart method to isolate network"; IED addresses are
all different and the user changes them at any time; "it should act like a real network IP, e.g.
ping 192.168.216.1 should work"; warn about the Ethernet and its network, prevent clashes, zero
headache during installation; "test, debug and make sure it works yourself". Decision:
[ADR-0120](../../decisions/ADR-0120-this-pc-lab-network.md). Go engine only.

## Change

| File | Why |
| --- | --- |
| `engine/internal/labnet/` (new) | address rules, pipe protocol and request handler (pure); card from `netloop.inf` through SetupAPI, /32 add/remove through `CreateUnicastIpAddressEntry` (preferred at once, usable as source), pipe server/client (server checked by service PID), service, install/uninstall, ping, source-pinned ARP |
| `engine/cmd/scadabench_labnet/` (new) | `install`/`uninstall --result FILE` behind the UAC prompt, `service` under the SCM; any crash still writes a result file |
| `engine/internal/labsetup/` (new) | rows and clash rules, background ARP answers (60 s, keyed by the real card's address), Use, Change (stop/move/start), Run re-add, install/remove through the existing elevator, `network/lab_journal.jsonl` |
| `engine/internal/api/labapi/` (new), `app/wire_lab.go` (new), `app/compose.go`, `wire_platform.go`, `wire_runtime.go`, `app/routes_test.go` | seven Go-only routes under `/network/lab`; Run's lab address lookup puts missing card addresses back first |
| `frontend/src/features/network/LabNetwork.tsx`, `LabRows.tsx`, `lab.css` (new); `api/lab.ts`, `copy/labNetwork.ts` (new); `settings/NetworkSettings.tsx`, `network/NetworkSetup.tsx`, `copy/hints.ts` | the "Lab network: this PC only" card (create, rows with editable address, status pill with the engine's hint, free-address button, Ping, Use, Remove); banner names the lab card; the ordinary card picker no longer offers it |
| `tools/build_labnet.py` (new), `run_dev_engine.py`, `compile_go_release.py`, `go_release_steps.py`, `packaging/installer.iss`, `tests/unit/test_go_release.py` | built like the VM helper, shipped beside the workers; the uninstaller removes service, program and card (one prompt, only when the service exists) |
| `docs/specs/lab-network.md` (NET-15..17), `docs/reference/API.md`, `FILE-MAP.md` | contracts |

## Evidence (owner's PC, Windows 11 Pro 26300, not elevated; Wi-Fi 192.168.118.112/20 up, Ethernet 192.168.40.111/24 unplugged)

- **Measured before building (proof script, removed afterwards).** KM-TEST card from `netloop.inf`:
  4.4 s, no reboot, status Up without cable, MAC `02-00-4C-4F-4F-50`. Aliases /32 in 0.3 s each.
  With `SkipAsSource` (what lab mode used) `ping 192.168.216.1` failed "General failure"; with it
  off, replies `<1 ms TTL=128`; TCP 102 bind/connect 0.5 ms; moving 10.20.0.5 to 10.20.0.50 177 ms;
  internet stays on Wi-Fi with the Wi-Fi source. Npcap: a frame sent on the card reached a second
  handle at once. `SendARP` without a source answers for an address on the card with the card's own
  MAC (false "in use"); with the Wi-Fi address as source the gateway 192.168.112.1 answered in
  143 ms and a free address got no answer after about 3 s.
- **Dev app (worktree, Go engine), demo station `scadabench_demo.scd` (6 IEDs, 172.16.1.x).**
  "Create the lab network card": one UAC prompt; card `{63B34BF8-…}` "ScadaBench Lab", metric 9999,
  DHCP off, IPv6 unbound; service `ScadaBenchLabNet` running from
  `C:\Program Files\ScadaBench\LabNetwork` (Users read/execute only); journal `lab_card applied`.
  "Use the lab network for 6 IEDs": no prompt, six /32 addresses, project in lab mode on the card,
  banner "on the ScadaBench Lab network (this PC only)". Status route 15 ms.
- **Run**: six `studio_ied` listening on 172.16.1.20/.21/.30/.31/.40/.41:102. Windows `ping`
  replies; **independent libiec61850 client (`mmscli`)** connects to 172.16.1.20:102 and reads the
  model.
- **Change while running** (page, Enter): FEED2 172.16.1.21 to 192.168.216.1: listener moved, old
  address refused (mmscli err 5), `ping 192.168.216.1` replies, mmscli connects and reads
  `FEED2LD1/LLN0` ST with the new start time; journal `applied` 192.168.216.1, `removed`
  172.16.1.21. Move back through the API: 1,985 ms (stop, address, start). No lab address in the
  persistent store (they vanish at reboot).
- **Clashes**: TR1 to FEED1's address 400 "FEED1 also uses 172.16.1.20…"; to Wi-Fi's own address
  400; to 127.0.0.5 400. Typed 192.168.112.1 (the Wi-Fi gateway): `checking`, then `real_device`
  "A real device on Wi-Fi uses …" with free address 192.168.112.130; 192.168.118.200: `same_range`.
- **Ping route**: 7 ms for a card address, `ok:false` after about 0.5 s for an unused one.
- **GOOSE** ("goes to ScadaBench Lab with your IEDs", 0191 auto): an independent Npcap listener on
  the card counted 18 frames in 30 s from six sources `02:a7:ac:00:00:01..06` (3 each, the demo
  file's heartbeat).
- **After a reboot**: Stop, FEED1's address removed through the service (ping times out), Run: the
  address was added back first (journal), all six IEDs started in 3.0 s, mmscli reads FEED1.
- Bugs found and fixed while testing: the hardware-ID list came back empty (a NUL in a Go string)
  and crashed Create under UAC (test `TestHardwareIDIsOneMultiString`; the program now writes a
  result file even when it crashes); a normal user may not open the SYSTEM service process, so the
  server is now checked by service PID.

- **Repair, Remove, Create again** (page/API, one UAC prompt each): an unanswered prompt expired after
  about 2 minutes and the app said "Windows did not get permission, so nothing was changed"; Repair
  kept the same card and all six addresses (journal `skipped`, 22.5 s incl. the prompt); Remove left
  no service, no Program Files folder, no service key, no KM-TEST card and the project in "This PC
  only" (journal `removed`, 4.7 s); Create made a fresh card `{247696AF-…}` and Use put the six IEDs
  back without a prompt. The project stayed in lab mode over 2.5 minutes of watching; an earlier
  switch back to "This PC only" while the owner was using the page could not be reproduced (only
  "Back to this PC only" and the card setup do that).
- **Design** (owner: "the UI became crowded, I liked the old design; highlight the lab network, don't
  make it crowded"): the page keeps its old layout; the lab network is one card with an accent
  border, a "New" tag until used, one line and one button; the addresses are folded under "Show the
  N addresses" and open by themselves when an IED needs attention; rows show IED, address, check
  and Ping, with the file address and a hint only when they matter. On the lab card the banner's
  button reads "Use a real network card". At 1366x768 nothing in the card is clipped; a very narrow
  window scrolls the table instead of clipping the card.
- **Tests**: `go test` labnet, labsetup, labapi, api/..., app, netcfg pass; `go vet ./...` clean;
  frontend 488/488; three `runtime` tests that bind 127.63.1.x:21101/21121 fail with WinError 10013
  because Windows currently excludes TCP 21096-21195 on this PC (Hyper-V/WinNAT); master 90d24b30
  fails the same way, so this is the machine, not this change. In this worktree `selftest` fails
  under `verify.py` with "No module named flask" because the worktree has no `studio/.venv` and
  `engine_tests.py` falls back to the system Python; run directly with the venv it passes, as on master.
  `verify.py`: lengths, architecture, documentation, history, specifications and GOOSE engine PASS;
  the Go step failed only for that reason. With `STUDIO_PYTHON` set to the main venv, toolsapi, app
  (Flask parity), httpapi, pyjson and selftest pass; only the three port-range `runtime` tests remain.

## Limitations

- That the real Wi-Fi/Ethernet cards never answer ARP for an address on the lab card (strong host
  model) was not proven from a second machine.
- SNMP devices are not put on the lab card yet (the page handles simulated IEDs); with SNMP devices
  use the ordinary card setup.
- The banner chooses its words by the stored card name "ScadaBench Lab".
- No packaged build was made (owner rule); the installer's uninstall step was not run.
