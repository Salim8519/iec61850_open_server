# 0233 Two addresses per IED; the user's address wins; monitoring has its own card (0.1.36)

Date: 2026-11-22. Owner: "many bugs found in 0136, it forgets the ip we put many times and uses cid
file, also it doesnt allow to use same ip in the ied simulated ... in monitoring it should read the
relay over a network interface". Decision: [ADR-0145](../../decisions/ADR-0145-two-addresses-per-ied-monitoring-card.md).
Layers: Go engine (`internal/project`, `internal/netcfg`, `internal/labsetup`, `internal/runtime`,
`internal/acquisition`, `internal/realcmd`, `internal/tools/control`, `internal/api/platformapi`),
native (`live/mms_session.cpp`, `live/identify.cpp`, `control/control_session.*`), frontend.

## What changed

- **Role switch.** New `project/ied_seed.go` (`seedAddress`), called by `UpdateIED`: an empty
  monitoring address starts from the simulation address and the other way round, once.
- **Clashes.** `project/ied_settings.go` and `runtime/endpoints.go` compare same-mode pairs only.
- **Planning.** `netcfg.PlanIED.LabIP` (filled by `platformapi.SimulatedIEDs`) is proposed before
  the SCD address and marked `saved`; `labsetup/rows.go` uses `lab_ip` on any card.
- **Monitoring card.** `netset.ValidateLive` accepts `live.local` (IPv4, stored only when set);
  `acquisition/session_run.go` sends it with `connect`, `realcmd` and `tools/control` send it with
  `operate`; native live/identify/control bind it with `IedConnection_setLocalAddress`.
- **Screen.** New `features/devices/MonitorCard.tsx` ("Network card": Windows chooses, or one of
  this PC's addresses grouped by card) in the device connection form; `LiveSettings.local`.

## Evidence

- New Go tests: `TestSimulatedAndMonitoredMayShareAnAddress`, `TestRoleSwitchStartsFromTheTypedAddress`,
  `TestMonitoredRelayIntoSimulatedSeedsTheLabAddress`, `TestMonitoringNetworkCard`,
  `TestPlanProposesTheSavedAddress`; `labsetup` `TestTypedAndStoredAddressesWin` updated to the new
  rule. Whole engine `go test ./...` passes (gateway `TestOwnPointsOverIEC104` flaked once under
  load and passes alone; `acquisition/modes` needs `GOTMPDIR=.local/gotmp` because the antivirus
  blocks test programs in the user temp folder).
- Native `studio_live` and `studio_control` build (MSVC Release). Frontend `tsc --noEmit` clean.
- Live, fresh Go engine with the merged Al Kamil project (28 BCUs): E04 to monitoring took its
  simulation address 172.26.0.34; E04 monitored at 172.26.0.35 while E05 simulated there (accepted);
  `live.local` stored; back to simulated kept both addresses; two monitored relays on
  172.26.0.35:102 refused. Plan preview on the Ethernet card: every simulated row proposed its saved
  address; with E04's `lab_ip` 172.26.0.99 and CID 172.26.0.34 the plan proposed 172.26.0.99.
- Release: `compile_go_release.py --version 0.1.36` exit 0, antivirus 19 programs, 0 findings.

## Limitations

- No test against a real relay through a second card yet (the bind is the library's documented
  `IedConnection_setLocalAddress`, as the SCADA master uses for IEC 104).
- The frozen Python engine keeps the old rules.
