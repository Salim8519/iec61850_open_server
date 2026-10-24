# ADR-0117: IEDs with several access points: simulate the station bus (S1)

Accepted: 2026-10-24. Owner decision: "S1 exact + auto-pick". Change
[0190](../changes/2026/0190-nr-access-points-and-deadbands.md). Go engine only
([ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)); the frozen Python engine
keeps its refusal. Item 3 of [FUTURE-WORK.md](../../FUTURE-WORK.md).

## Problem

NR and NARI relays (PCS-9705, PCS-921S, the owner's FBMR-E02) have three access points with a
server: S1 (station bus: IP address, reports, the LDs a SCADA reads), G1 (process bus GOOSE, LD
PIGO) and M1 (Sampled Values, LD PISV). Import refused every IED with more than one server
("select an access point first") and no screen let the user choose.

## Decision

1. `scl/access_points.go` `stationOf` picks the access point a SCADA uses: the only one with an
   IPv4 address in Communication, else the only one with a ReportControl. When neither tells them
   apart the IED stays refused with a plain reason naming the access points.
2. Only the chosen server is simulated, exactly as the SCD defines it: its logical devices, reports,
   controls and GOOSE blocks. G1 and M1 are not simulated (no MMS, no process-bus GOOSE, no SV).
3. Every reader of logical devices uses the station server (`stationDevices`): import, preview,
   data classes, typed controls, the GOOSE map, inventory, subscriptions, signal flow and the
   substation reader. Communication addresses come from the station's ConnectedAP only.
4. The preview adds `access_point_note`, e.g. "Simulates S1 (station bus, 10.0.115.22), the access
   point a SCADA uses. Not simulated: G1 (process bus, GOOSE: PIGO); M1 (process bus, Sampled Values:
   PISV)." The import dialog shows it under the IED.
5. IEDs with one server are untouched. Export keeps the original SCL unchanged. The Python-twin test
   switch keeps the old refusal so the Python goldens still compare byte for byte.

## Compatibility

Projects that imported before could not hold these IEDs, so no stored project changes. Over 182
local SCL files only the four files with NR/NARI IEDs answer differently, and within them only those
IEDs (corpus dump, change 0190).
