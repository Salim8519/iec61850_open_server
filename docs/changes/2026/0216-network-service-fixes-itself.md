# 0216 The network service fixes itself

Date: 2026-11-04. Owner, after "Network helper is off" blocked Put IEDs on Ethernet: "what if users
suffered the same? im pro and im not able to find the issue or how to resolve", "possible to fix in
the app?", "i was doing on the ethernet not the scadabench", "the fix should be clear to all if
needed". Amends [ADR-0136](../../decisions/ADR-0136-leave-nothing-on-the-network.md).
Layers: Go engine (`netcfg`, `labsetup`, `labnet`, `cmd/scadabench_labnet`), frontend copy.

## Problem

Change 0214 stopped the silent fallback to permanent addresses when the ScadaBench network service
cannot add addresses on a real card. The rows then said "Network helper is off" and pointed to
another page. On the owner's PC another program had set the service to manual start, so Ethernet
could not be used without finding Repair on the lab card page. A user who only uses Ethernet and
never created the lab card had no service at all and hit the same wall every time.

## Change

- **The plan sets the service up in its own prompt.** On a real card, when the service is missing,
  stopped or too old, the address plan's first Windows change is "Set up the ScadaBench network
  service…" (`service_repair`). It runs `scadabench_labnet.exe install` (the program Create and
  Repair use) inside the same administrator prompt, before the approvals it reads, and passes the
  program's reason on when it fails. The helper accepts it only for the program beside the
  workers and the service name `ScadaBenchLabNet`. Clean-up never lists it.
- **Start turns a stopped service back on.** When a Run (or an SNMP device start) needs the
  service and it is installed but stopped, ScadaBench asks Windows once and runs the new
  `scadabench_labnet.exe start` (start with Windows again, start now), then continues. A cancelled
  prompt stops the Run with "Windows did not get permission, so nothing was changed."
- **Words.** The review lists the step as "Network service". The lab page says "The lab network
  service is off. Run turns it back on (Windows asks once), or press Repair." "Network service
  needed" remains only when the installation lacks `scadabench_labnet.exe` (reinstall).
- The banner and notice drafted earlier this day for turning the helper on by hand were removed
  before release.

## Evidence

- Go: `go test ./internal/labsetup/ ./internal/netcfg/ ./internal/labnet/ ./internal/netaddr/
  ./internal/app/ ./internal/api/...` pass, with new tests:
  - `labsetup/turn_on_test.go`: a stopped service is turned on once before Start adds the approved
    address; a running one is not asked again; a cancelled prompt stops the Run; This PC only and
    a card without approvals never ask.
  - `netcfg/service_repair_test.go`: the step comes first, only with approvals, runs only our
    program, its key and description, never a leftover; another folder or service is refused.
- The step's exact PowerShell ran through `ExecuteBatch` with a stand-in program: success gives
  ok, failure gives the program's reason ("the lab network service did not start: …"), and no
  result file is left in TEMP. This caught a parse error (the helper's `| Out-Null` after an `if`
  block), fixed by making the step one script block.
- Scratch engine on this PC with the service stopped (manual start), at 1366x768: Network setup ›
  Ethernet shows the SNMP row "Free" (was "Network helper is off") and the review lists "Set up the
  ScadaBench network service…" first, kind "Network service".
- Frontend: `tsc --noEmit` clean, 483 tests pass.

## Limitations

- The full Apply and Start with a real Windows prompt were not clicked through here: they raise
  the owner's administrator prompt. The owner's next Apply on Ethernet, or Start, does it.
- `install` also creates the ScadaBench Lab card when the PC has none, because the service is
  installed with it (ADR-0120). Nothing goes on that card unless the project uses it.
- A stopped service that is also too old is turned on at Start but stays too old until the next
  Apply or Repair updates it.
