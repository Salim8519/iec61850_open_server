# 0152: One command stops every ScadaBench process

Date: 2026-10-21. Owner request: one command that stops ScadaBench, its engines and everything it
started.

## Change

- `stop-studio.cmd` (repository root, next to `start-studio.cmd`) runs `studio/tools/stop_all.py`.
- It stops, each with its child processes: `ScadaBench.exe`, `ScadaBench-Portable.exe`,
  `studio_server.exe` and the workers (`studio_ied`, `studio_live`, `studio_probe`,
  `studio_control`, `studio_gateway`, `studio_master`, `studio_goose`, `studio_snmp`, `studio_xsd`),
  plus Python and Node processes whose command line runs this repository's dev launchers, Python
  backend or Vite. `--dry-run` only lists them.
- Never stopped: the network helper (`--network-helper`, it may be changing adapter settings) and
  any Python or Node process outside this repository.

## Evidence

- Unit tests `test_stop_all.py` 3/3 (choice of processes, network helper kept, one id per tree).
- Live: built Go server on port 8790 with the 6 simulated IEDs running (11 processes); dry run
  listed `studio_server.exe`, `stop-studio.cmd` stopped it with all 10 workers, none left.

## Limitations

- Processes started by another Windows user or as administrator need an administrator window; the
  command says so.
- Firewall rules and network aliases are not touched; Settings › Network › Clean up handles them.
