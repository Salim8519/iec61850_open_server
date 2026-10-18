# ADR-0061: SCADA roles: station gateway, SCADA master and protocol converter

Accepted: 2026-10-18. The owner noticed a professional gap (2026-10-18). Real SCADA has two sides,
and ScadaBench only played the answering side ("slave" / server). The owner asked for:
- each side to be simulated;
- the side that gets data from another machine to find all points by default, with the user
  choosing what to check.

Answers:
- roles: SCADA master, gateway pass-through, "what else it can play";
- "master discovers all";
- protocols: IEC 104, DNP3 and Modbus TCP;
- one tool with a role switch;
- "be careful with the correct words, and industry words".

**Terminology (each protocol keeps its own official words)**

| Protocol | Side that asks and commands (SCADA) | Side that answers (gateway, RTU, IED) |
| --- | --- | --- |
| IEC 60870-5-104 | Controlling station ("104 master"), TCP client | Controlled station ("104 slave"), TCP server |
| DNP3 | Master | Outstation (DNP3 does not say "slave") |
| Modbus TCP | Client (formerly "master") | Server (formerly "slave") |
| IEC 61850 MMS | Client | Server |
| GOOSE / SV | Subscriber | Publisher |

A station gateway (data concentrator) has two sides. Downstream, toward the IEDs, it is a
client or master. Upstream, toward SCADA, it is a slave, outstation or server. A protocol
converter is a gateway whose downstream side is also 104, DNP3 or Modbus. The UI uses these
words per protocol, and the glossary explains each term. A frontend test rejects "DNP3 slave",
"Modbus master" and "Modbus slave" in the copy.

**Decision**

- **One tool, renamed "SCADA gateway & master".** It has an "Act as" switch:
  - **Station gateway** (as before);
  - **SCADA master**;
  - **Protocol converter**.

  The role is stored in the project. Switching is refused while the gateway runs or the master
  is connected. A small "Which side am I?" diagram shows where ScadaBench sits.
- **New native worker `studio_master`.** It is licensed (`require_valid_license` on connect)
  and sealed like the other workers. It has three clients:
  - **IEC 104 controlling station** (lib60870 `CS104_Connection`): STARTDT, station
    interrogation (C_IC), single, double and setpoint commands, optionally time-tagged, with
    select/execute and ACT_CON/ACT_TERM, and clock synchronisation;
  - **DNP3 master** (opendnp3): integrity poll (class 0123), class 1/2/3 event polls,
    unsolicited events, CROB (latch, pulse, trip/close) and analog outputs, select-and-operate
    or direct operate;
  - **Modbus TCP client** (own MBAP code): range reads FC1-FC4, writes FC5/FC6/FC16, and
    "Find registers". Find registers is a bounded probe: at most 10,000 addresses and at most
    10 requests per second. It reads blocks and splits them only where the server refuses, and
    it only reads.

  Values come back as plain records (group, type, address, value, raw quality, time, cause). The
  backend words the quality in one place. Traffic lines reuse the gateway's rate-limited
  traffic code.
- **Discover all by default.**
  - IEC 104 and DNP3 interrogate at connect and every N minutes (default 15; 0 means only at
    connect).
  - Modbus reads the user's ranges every poll interval.
  - Every discovered point is watched (ticked). The user can untick points, name them,
    filter, and export or import names as CSV.
  - A dropped link is retried every 10 s until the user presses Disconnect.
  - A connected master reconnects by itself after the app restarts (owner rule: remember).
- **Commands from the master view.** A command form (with prefilled rows) is sent only after a
  REAL-device warning, with `confirm: true` (owner policy: real commands are allowed after a
  confirmation). Each command is logged with how the device answered: confirmation, refusal or
  termination. Interrogations, polls and Find registers only read.
- **Protocol converter.**
  - Every watched device point is served upstream: same address when it is free in the
    upstream protocol, otherwise the next free address, with a note.
  - Values are invalid upstream while the device link is down or the device marks them bad.
  - Upstream commands are relayed to the device through the master: DNP3 binary outputs and
    Modbus coils automatically, and IEC 104 commands through user "command routes" (104
    command addresses cannot be discovered).
  - The existing "commands to real devices" switch applies.
  - After a restart the master reconnects before the upstream side resumes.
- **Project format 14.** The gateway section gains `role` and a `master` block: protocol, host,
  settings per protocol, Modbus ranges, command routes, unticked keys, labels and
  `auto_connect`. 13 → 14 adds the defaults with a migration notice and a recovery copy. Older
  apps refuse format 14, so nothing is dropped silently.

**Compatibility**

- Existing projects open as Station gateway, exactly as before.
- `GET /activity` gains a `master` entry. The tool card and sidebar dot name the master
  connection.
- New native program and packaging: `studio_master.exe` is in both specs, in `SEALED` and in
  the protected build targets.
- No new third-party dependency: lib60870 and opendnp3 are already in use (ADR-0057).
- Not claimed: interoperability with third-party SCADA, RTUs or gateways. The evidence is
  limited to `studio_gateway`, an independent Python Modbus server and the independent lib60870
  104 master that drives the converter.
