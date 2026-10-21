# 0150: Network setup checks faster, long waits show their steps, file address always shown

Date: 2026-10-21. Owner request with answers to questions; decisions in
[ADR-0093](../../decisions/ADR-0093-network-check-speed.md).

## Measured first (this PC)

- Listing network cards (IP Helper, Go): 0.3 s, not the cause.
- `Get-NetFirewallRule -Name 'Prefix*'` through PowerShell: 4.2-5.3 s per call; Network setup
  made two per check (IED and SNMP rules), the gateway card one.
- Address check: about 3.2 s for an address nobody answers (Windows' ARP wait), 16 at a time.
- SCL files: preview 0.015 s and import of 6 IEDs 0.04 s over HTTP (public `scd/model_substation.scd`);
  Go parse/import of 120 IEDs (500 KB) 0.12 s; XSD check 0.2 s first time. IED addresses are
  stored in the project at import, so Network setup never re-reads files. The wait people saw
  while opening was a 14 px spinner in the top bar.

## Change

- Go `netcfg`: `RuleNames` reads ScadaBench's rule names (MMS, SNMP, gateway) from the local
  firewall rule store in about 6 ms, PowerShell only when the store cannot be read.
- `ARPProber` asks 64 addresses at once and reports each answer; `PreviewProgress` and Go-only
  `GET /network/plan/progress` give steps, answered/asked and each row's state.
- Frontend: shared `StepList`; Network setup shows it with live rows (Checking… / Free / Taken /
  Checked by Windows when it is added) on the first check and while listing cards; opening a
  project and reading/importing an SCL file show it in the middle of the window (in the import
  window while importing), after 400 ms. Every IED row shows "File address" with "Use file
  address" when another address is in the field.

## Evidence

- Go: live Windows test, rule store equals PowerShell for all three prefixes (6 ms against
  4.2-4.7 s); fallback and case tests; progress test (steps, row states, preview unchanged);
  64-at-once test; all netcfg tests incl. Python parity vectors; platformapi and app routes.
- Frontend 266/266 (new `stepList` tests), `tsc` clean.
- Browser at 1366x768 on a scratch Go server (empty data folder, public SCD): Default Switch
  card checked; firewall step 7 ms, one silent typed address 3.2 s; typed address then "Use file
  address" restores 172.16.1.20; step list with "12 of 30 addresses" and row states one line each
  (synthetic progress injected in the page); file-open panel "Reading station.scd" with ticked
  "Save your diagram" (preview slowed in the page).

## Limitations

- An address nobody answers still takes Windows' full ARP wait (about 3 s); a shorter wait was
  not chosen. A re-check after typing keeps the small "Checking again…" line.
- Blocking-rule and clean-up firewall checks still use PowerShell.
- Not measured with large customer SCDs (private data is not used); the import panel shows the
  time of each step so a slow file shows where the time goes.
- Python engine: no progress route (frozen); the steps then show without counts.
- `verify.py`: the Go step reports FAIL because the antivirus blocked the `gateway` and
  `modelcache` test programs, and `gateway/service` IEC 104 wire tests failed in some runs
  (one of two runs passed); that package does not depend on any file changed here.
