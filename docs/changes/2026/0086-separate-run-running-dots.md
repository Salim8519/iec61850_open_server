# 0086: Run and the gateway are separate, running dots, page commands, full-height points

Date: 2026-10-17. Decision: [ADR-0059](../../decisions/ADR-0059-separate-run-running-dots.md).
Specs: [Gateway simulator](../../specs/gateway.md) (GW-06, GW-10),
[UI structure](../../specs/ui-structure.md) (UI-09).

Owner requests after 0.1.5:
- "why whole IEDs and gateways run? they should be separated run";
- a green dot on the tab buttons so it is clear what is running;
- the points visible without scrolling the page;
- "let user send commands, with warning".

## Change

- **Backend.**
  - `gateway/manager.py`: Start no longer starts Run, and `point()` also finds saved points
    while the gateway is stopped.
  - `send()` runs a page command through the executor, so it gets the same checks, switch and
    log as a SCADA command (originator "station").
  - `gateway/commands.py`: `run()` never raises; observed positions read "ON/OFF" or
    "closed (ON)".
  - `api/gateway.py`: `POST /tools/gateway/command` needs `confirm: true`.
  - `tools/activity.py` with `GET /activity`: gateway, Explorer, scan and sequences as plain
    facts.
  - Point rows carry `ied_role`.
- **Frontend.**
  - `app/running.ts`, `useActivity.ts` and `copy/running.ts`: a running dot per place in
    `Sidebar.tsx` (tooltip and screen-reader words; amber when paused), and badges on the
    tool cards.
  - Gateway:
    - `GatewayStatusBar` (one line while running) and `GatewayNotices` (IEDs off, with a
      separate "Run the IEDs" button);
    - the full-height panel (`.content.tool-fill`, flex, sticky table header);
    - `CommandSend` (Send buttons plus the warning) and `gatewayCommands.ts`;
    - the command log shows "You · this page".
  - Copy: "Allow commands to real IEDs" now covers SCADA and page commands, and the texts say
    that Run and the gateway are separate.

## Evidence

- **Tests:** 360 unit + integration and 50 protocol tests (7 skipped need hardware) pass.
  - `test_gateway_runtime.py`: Start leaves `run_state` "stopped"; `/activity` shows the
    gateway on port 12406; a page command without confirm returns 400, and with confirm it
    operates the breaker, logged "manual".
  - The 111 frontend tests include the new `running.test.ts` (dots per place, no implicit
    Run, commands only from the warning). tsc is clean.
- **Source app on its own port (18766), temporary data, two sample IEDs, 1440×900:**
  - after Start gateway the Run state stayed "stopped" and the "Run the IEDs" notice showed;
  - after Run, dots appeared on Devices, SLD and Tools;
  - "On" from a command row went through the warning and was operated, and the log shows
    "is now ON";
  - the page did not scroll (`scrollHeight` equals `clientHeight` = 781 px, with the
    recovery banner shown), whether the IEDs were off or running;
  - the tool card shows "Gateway: IEC 104 on port 12404 · 0 SCADA masters".

## Limitations

- The real-device warning style was checked in code and tests only; no real relay was
  connected.
- A remembered gateway still resumes by itself after a restart (owner rule from ADR-0057), now
  without the IEDs.
- Not in an installer yet. The live website's 0.1.5 text says Start "starts the IEDs"; update
  it with the next release.
