# 0205: Lab service passes Kaspersky; no fixed addresses on DHCP cards; fast helper; switching moves ScadaBench's own addresses

Owner (installed 0.1.30 rebuilt in [0204](0204-antivirus-clean-release.md)): "the scadabench labnet
exe got blocked … it stuck in repair", "i picked by mistake the wifi … i lost wifi connection",
"even on ethernet takes longer now", "network dns is now 40% instead of 70% before", "when we stop
the network we should remove the ip addresses … removing them manually is a hassle". Decision:
[ADR-0131](../../decisions/ADR-0131-network-changes-kaspersky-dhcp.md).

## Diagnosis (owner's PC, Kaspersky Total Security 21.3, bases 2026-10-31 04:53)

- `avp.com REPORT SW2`: at 12:24 and 12:29 `scadabench_labnet.exe install` was "Detected:
  PDM:Trojan.Win32.Generic", terminated and rolled back; at 12:33 the installed
  `_internal\native\scadabench_labnet.exe` was deleted. The service copy in Program Files scanned
  clean on demand. Every earlier lab install ran from the excluded project folder.
- Network helper requests: `apply-d7d88…` (Wi-Fi, 10 changes) 12:27:18 → result 12:29:59;
  `apply-1ddaf…` (Ethernet, 9 changes) 12:35:06 → result 12:37:12. The helper process started
  4 s after the request, so the time was the helper: one PowerShell per change. The app waits
  120 s, so both applies failed in the app and neither reached `journal.jsonl`.
- Wi-Fi was `Dhcp Enabled`; after the Wi-Fi apply the owner had to set it back to automatic.
- One PowerShell per change vs one script, 9 `New-NetIPAddress` (not elevated): 36.2 s vs 3.9 s.

## Change

| Area | Files |
| --- | --- |
| Separate service program | `cmd/scadabench_labsvc/main.go` (new), `cmd/scadabench_labnet/main.go` (no `service` mode; installs `scadabench_labsvc.exe` from beside itself), `internal/labnet/protocol.go` (`ServiceProgram`), `client_windows.go`, `install_windows.go` (`pointAt`: older service moved to the new program, old copy deleted), `tools/build_labnet.py` (both programs), `tools/compile_go_release.py` |
| DHCP guard | `netcfg/iphelper_windows.go` (`DHCPEnabled`), `iphelper_other.go`, `iphelper_words.go`, `dhcp_guard.go` (new), `service.go` (`DHCP` field), `service_plan.go`, `frontend/src/copy/status.ts` |
| Helper batches | `netcfg/helper_batch.go` (new: `ExecuteBatch`, `BatchScript`), `helper_runner.go` (`HelperMain` uses batches, parity test keeps `ExecuteRequest`), `elevation.go` / `elevation_windows.go` (wait 10 min) |
| Late results | `netcfg/late_results.go` (new: `catchUp`), `service_apply.go` (requests awaited are skipped), `service_cleanup.go` (`Leftovers` catches up first) |
| Switching to the lab card | `netcfg/our_addresses.go` (new: `OurAddresses`, `RemoveOurAddresses`), `labsetup/ours.go` (new: `ours_elsewhere`), `labsetup/service.go`, `apply.go` (Use frees them first), `rows.go`, `app/wire_lab.go`, `frontend/src/api/lab.ts`, `copy/labNetwork.ts` |
| Tests | `netcfg/dhcp_guard_test.go`, `helper_batch_test.go` (fake and real PowerShell), `late_results_test.go`, `hostkey_firewall_test.go` (shell answers batches), `helper_test.go`, `labsetup/ours_test.go` |

## Evidence

- Go: `netcfg`, `labsetup`, `labnet`, `app`, `cmd/...` pass; `tsc --noEmit` clean.
- Kaspersky: both new programs scanned clean by the release gate (report-only, copies in %TEMP%).
  The new `scadabench_labnet.exe` and `scadabench_labsvc.exe` were placed in the installed
  0.1.30 folder; the owner pressed Repair and approved the prompt: service `ScadaBenchLabNet`
  running from `"C:\Program Files\ScadaBench\LabNetwork\scadabench_labsvc.exe" service`, the old
  copy gone, no Kaspersky event afterwards. Owner: "now it is fine".
- DHCP: `DHCPEnabled` on this PC: Wi-Fi and Bluetooth true, Ethernet, the lab card, VMware and
  VPN cards false. Dev app plan preview on Wi-Fi: every row `adapter_uses_dhcp`, `ready` false.
- Batches in real PowerShell: four commands with two failures give ok/fail per change with the
  exception text; a parse error in one command falls back to one PowerShell per change.
- Switch on a copy of the owner's data with the new engine: `catchUp` journaled the two late
  requests; the 7 IED rows showed `ours_elsewhere` ("ScadaBench put 192.168.230.21 on “Ethernet”
  earlier…"); Use with one prompt (owner approved) removed the 7 addresses from Ethernet in 10 s
  from request to result, prompt included, journaled them `removed`, switched the project to the
  card. Ethernet kept the owner's 192.168.40.111.

## Limitations

- Not released: the live 0.1.30 still ships the self-copying lab program; Kaspersky users who
  create the lab card meet the block until the next release.
- Other antivirus products were not tested; the programs remain unsigned.
- The elevated batch was measured once (7 removals, 10 s with the prompt). DNS Client load
  during an apply was not measured again.
- Ethernet still holds two SNMP addresses (192.168.40.21, 192.168.230.26) from the installed app;
  they are not IED rows, so the switch did not take them.
- Real cards still keep their addresses between runs (one prompt to add, one to clean up); the
  owner asked for the smoother model next.
