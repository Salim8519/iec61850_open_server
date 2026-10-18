# ADR-0062: Open the station gateway through Windows Firewall with the network helper

Accepted: 2026-10-18. Found in the two-machine test (change 0089).

**Problem**

On "All network adapters" the gateway reported "Listening on 0.0.0.0:2404", but no SCADA on
another computer could connect:
- ScadaBench never created a firewall rule for `studio_gateway.exe`;
- Windows had already answered its own "allow access?" question for that program with enabled
  inbound **Block** rules ("TCP Query User{GUID}…studio_gateway.exe"). The question appears once,
  and closing it leaves those Block rules behind.

A Block rule wins over any Allow rule, so adding an Allow rule alone is not enough. Users would
see a running gateway and a silent timeout on the SCADA side.

**Decision**

- Use the same contract as the lab network (ADR on the network helper): an exact, fingerprinted
  preview; nothing changes until the user confirms; the elevated helper applies it after one
  Windows (UAC) approval; every change is journaled; Settings › Network › Clean up undoes it.
- When the gateway listens on "All network adapters", a "Windows Firewall" card appears in the
  gateway tool. It checks without elevation (`GET /network/gateway-firewall`) and lists:
  - **allow**: one inbound TCP rule `ScadaBench-Gateway-<hash of program path>`, group
    "ScadaBench", for the gateway worker program only. It has no port filter, so one approval
    covers IEC 104, DNP3 and Modbus. This matches what Windows' own "Allow" button does.
  - **unblock**: each enabled inbound Block rule that names exactly this program (TCP or Any).
    The helper **disables** it, and clean-up **enables** it again. It never deletes a Windows or
    user rule. Before disabling, the helper re-checks the live rule (inbound, Block, same program).
- `POST /network/gateway-firewall` applies only a preview the user saw. If Windows changed since,
  the request is refused with the new list.
- The helper's validation accepts only:
  - gateway rule names matching `ScadaBench-Gateway-[0-9a-f]{12}` in group "ScadaBench";
  - the program `studio_gateway.exe` beside the trusted IED worker;
  - Windows rule names without quotes, so they cannot leave their PowerShell string.

**Compatibility**

- No project format change. Existing lab-network journals stay valid. New journal kinds are
  `gateway_firewall` and `unblock`, and clean-up handles both.
- The user-visible IED firewall rule name changes from "Substation Studio simulated IEDs (MMS)"
  to "ScadaBench simulated IEDs (MMS)". The rule group and names are unchanged, so clean-up
  still finds old rules.

**Evidence and limits**

- Unit tests: `tests/unit/test_gateway_firewall.py` (plan, helper validation, journal, preview
  then apply).
- Two machines, 2026-10-18: FreyrSCADA 104 client, FreyrSCADA DNP3 master, modpoll and
  ScadaBench 0.1.7 on the VM reached the host gateway only after this change was applied with
  UAC. Before it, every connection timed out.
- Clean-up of the gateway rule and re-enabling the Block rule still need a UAC approval on the
  host. That run is recorded in change 0089 when it is done.
