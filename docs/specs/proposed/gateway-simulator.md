# Proposed spec: Gateway simulator (IEC 60870-5-104, Modbus TCP, DNP3)

> **Proposed.** Planned behaviour, not a capability. Agreed with the owner on 2026-10-17
> ([ADR-0054](../../decisions/ADR-0054-tools-place.md)). Models: the FreyrSCADA IEC 104 and
> DNP3 simulators (point table, traffic log) and protocol gateways such as SCADA Data Gateway.
> Needs its own decision record for the dependencies before code starts.

## Purpose

Studio acts as the station gateway/RTU, so a SCADA master can be tested against the simulated
station (and, read-only, real IEDs) over IEC 104, Modbus TCP or DNP3.

## Behaviour contract

- **Workers.** There is one native worker per protocol: `studio_gw104`, `studio_gwmodbus`
  and `studio_gwdnp3`. Each worker:
  - is a libiec61850 MMS client toward the IEDs (reports or polling for monitoring; SBO or
    direct operate for commands);
  - acts as a protocol server toward the SCADA master: a lib60870 104 slave, Studio's own
    Modbus TCP server, or an opendnp3 outstation.
- **Commands reach simulated IEDs only.** A command for a point on a real IED is refused with
  the protocol's negative answer:
  - IEC 104: negative confirmation, COT 47;
  - DNP3: NOT_AUTHORIZED;
  - Modbus: exception 01.

  The refusal is enforced in the native worker as well as the backend.
- **Point list.**
  - Proposed automatically from the SCD:
    - Pos → double point;
    - Boolean stVal → single point;
    - mag.f → float measurement;
    - controllable objects → commands.
  - Editable in a table: address, type, IED reference, scale and deadband.
  - Imported and exported as CSV.
  - Unknown references produce diagnostics; values are never invented.
  - Saved in the project, which needs a new project format with a migration.
- **Screens.**
  - protocol settings, with the common choices first and timers under Advanced;
  - a point table with live value, quality, time and last cause;
  - connected masters;
  - a decoded traffic log with filter and pause;
  - a command log that keeps request, applied and observed results separate.
- **Evidence before claims.** Each protocol is claimed only after an independent master has
  passed against it:
  - IEC 104: the lib60870 client example plus the FreyrSCADA client;
  - Modbus: pymodbus;
  - DNP3: the opendnp3 master example.

## Reserved rule IDs

| Prefix | Rules |
| --- | --- |
| `GW` | point proposal mapping; CSV round trip and diagnostics; command refused for real IED (per protocol); command operates simulated breaker; quality/time mapping; traffic log decoding; settings validation; format migration |

## Open questions

- Delivery order: 104 → Modbus TCP → DNP3. Each one is usable on its own.
