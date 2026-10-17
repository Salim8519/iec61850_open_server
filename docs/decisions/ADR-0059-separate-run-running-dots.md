# ADR-0059: Run and the gateway are separate, running dots, commands from the gateway page

Accepted: 2026-10-17. Owner request after 0.1.5:
- "why whole IEDs and gateways run? they should be separated run";
- a green dot on each tab so the user knows what is running;
- the points visible without scrolling the page;
- "let user send commands, with warning".

**Decision**

- **Separate runs.**
  - Gateway Start only prepares the point list (when it is empty) and listens. It never starts
    the IEDs.
  - Run (top bar) only starts the IEDs. It never starts the gateway.
  - After a restart, a gateway that was running starts listening again (ADR-0057), still
    without starting the IEDs.
  - While the gateway runs without IEDs, the page says that every point is sent as invalid and
    offers a separate "Run the IEDs" button.
  - This supersedes the "Gateway Start starts Run when it is stopped" part of ADR-0057.
- **Running dots.**
  - Each sidebar place shows its own dot when something runs there:
    - Devices: IEDs running;
    - SLD: live diagram;
    - Test: sequences running;
    - Tools: the gateway listening, IED Explorer connected, or Find IEDs scanning.
  - A paused Run shows an amber dot.
  - The words are in the button's tooltip and in screen-reader text, so colour is never the
    only signal.
  - Tool cards show a green badge that names what runs ("Gateway: IEC 104 on port 2404 ·
    1 SCADA master").
  - The page reads `GET /activity` every 2 s (plain facts only); the IED Run state comes from
    the project as before.
- **Gateway layout while running.** A one-line status bar replaces the settings card ("Settings"
  opens it again). The open tool fills the page height, and the Points, Traffic and Commands
  panel takes the rest, with its own scroll and a sticky table header. The page itself does
  not scroll.
- **Commands from the page.**
  - Command rows in Points offer Close/Open (double) or On/Off (single).
  - A button only opens a warning that names the IED and object, with stronger wording and
    border for a real device. Only "Yes, send" calls `POST /tools/gateway/command` with
    `confirm: true`.
  - The backend runs it exactly like a SCADA command: same checks, same "Allow commands to
    real IEDs" switch, originator "station", and the same request/applied/observed log entry
    marked "You · this page".
  - Rows wait until the point list is saved.

**Compatibility**
- No project format, dependency or native change.
- Existing projects keep their gateway settings. A remembered gateway still resumes, but the
  IEDs now need Run.
- The website's 0.1.5 text ("starts the IEDs") describes 0.1.5 and must be updated with the
  next release.
