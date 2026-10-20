# ADR-0092: SNMP devices (simulate agents, read devices)

Accepted: 2026-10-20. Owner: "add snmp simulator client and master, in all versions, of all
types, while keeping ui simple, we simulate snmp3 devices if possible, and we read in master too".
Decisions the owner chose on 2026-10-20 are marked (owner).

## Decision

- **Place:** one new tool, Tools > **SNMP devices**, with a role switch in the header row like the
  SCADA tool (ADR-0064): **Simulate devices** (ScadaBench is the SNMP agent) and **Read devices**
  (ScadaBench is the SNMP manager). Devices and the SLD stay IEC 61850 only; SNMP values on the
  SLD are a later step (owner).
- **Engine:** a separate Go worker, `studio_snmp.exe` (owner), built from the same module as the
  server (`studio/engine/cmd/studio_snmp`, stdlib only, CGO off). It owns every SNMP socket, the
  USM engine state (engine boots/time, salts) and the simulated MIB values; the server owns the
  configuration, the page state and the logs. Framing is the native worker framing (4-byte length
  + JSON, `{"version":1,"id","op"}` requests, `{"event"}` events). One process serves every
  simulated device and the reader; it is started only while SNMP is in use, so SNMP costs
  nothing otherwise, and a crash or a flood never touches the IEDs or the gateway.
- **Protocol:** SNMPv1, v2c and v3 in the first release (owner). GET, GETNEXT, GETBULK, SET,
  v1 Trap, v2 Trap, Inform, Report. USM (RFC 3414) with auth MD5, SHA-1 and SHA-2 (RFC 7860:
  SHA-224/256/384/512) and privacy DES (RFC 3414), AES-128 (RFC 3826) and AES-192/256 in both
  key-extension variants (Blumenthal draft as `aes192`/`aes256`, Reeder/Cisco as
  `aes192c`/`aes256c`). Engine discovery, time window and the usmStats reports are implemented.
  Written from the RFCs in Go; no third-party SNMP library.
- **Simulated devices:** each device has its own IP address on port 161 (owner); the default is a
  free `127.60.0.x` address on this PC. Ready-made profiles: **Network switch** and **GPS time
  server** (owner), plus empty device, import of a `snmpwalk` output and ScadaBench CSV. Values
  are fixed, `uptime` (timeticks since start) or `counter` (grows by a rate). Devices stand alone:
  no link to the IED station (owner). SET from outside changes writable points only. Traps and
  informs go to the device's trap targets when the user presses Send trap (switch: link down/up;
  GPS: sync lost/back; any device: cold start or a custom trap).
- **Reader:** connect = v3 discovery when needed, then a full walk (GETBULK on v2c/v3, GETNEXT on
  v1); key points (system group, interfaces' status, entity/sensor names) show on top and
  everything else is searchable (owner). The shown points are re-read every **1 s** by default
  (owner), changeable; a full walk runs only on Walk again. SET after the usual short
  confirmation. A trap receiver (port 162) shows traps and acknowledges informs. "Save as
  simulated device" turns the last walk into a simulated device.
- **MIBs:** standard IETF/IANA MIB modules are built in for names and value meanings; the user can
  remove any of them and import their own (owner). The library lives in the app data folder
  (`snmp/mibs/`), not in the project: it is reference data shared by all projects.
- **Project data:** `project.snmp = {role, devices[], reader{}}`; community strings and v3
  passwords are stored in the project (owner), shown masked, never logged; exporting a project
  that holds reader passwords shows a one-line warning.
- **Port clashes:** when 161 or 162 is taken (for example by the Windows SNMP or SNMP Trap
  service), the message names the likely owner and offers a free port such as 1161/1162.
  ScadaBench never stops Windows services.

## API (Go only)

`GET/PUT /api/v1/tools/snmp/config`, `POST /tools/snmp/role`, devices
(`POST /tools/snmp/devices`, `PUT|DELETE /tools/snmp/devices/<id>`, `POST .../import`,
`GET .../<id>/points.csv`), `POST /tools/snmp/start|stop`, `PUT /tools/snmp/devices/<id>/value`,
`POST /tools/snmp/devices/<id>/trap`, `GET /tools/snmp` (status), reader
(`POST /tools/snmp/read/connect|disconnect|walk|set|poll`, `GET /tools/snmp/read/values`),
traps (`POST /tools/snmp/traps/listen|stop`, `GET /tools/snmp/traps`), MIBs
(`GET /tools/snmp/mibs`, `POST /tools/snmp/mibs`, `DELETE /tools/snmp/mibs/<module>`,
`POST /tools/snmp/mibs/restore`). Exact bodies: `docs/reference/API.md` section SNMP.

## Compatibility

- Go engine only. The frozen Python engine answers the SNMP routes 404, drops `project.snmp`
  when it saves a project, and its pages hide the tool (capability `snmp`).
- Projects without `snmp` are byte-identical; `/activity` and other parity routes are unchanged.
- The release ships `studio_snmp.exe` beside the other workers (like the Go `studio_goose.exe`
  and `studio_xsd.exe` it is not in the sealed native list) and the MIB licence notice
  `licenses/snmp-mibs/snmp-mibs-NOTICE.txt`.
- Lab-network addresses for simulated SNMP devices (network helper aliases, firewall rule) are a
  later step; until then a device binds to an address that already exists on the PC.
