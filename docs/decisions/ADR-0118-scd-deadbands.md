# ADR-0118: SCD deadbands (db, zeroDb): off by default, per-IED switch, SCADA writes per valKind

Accepted: 2026-10-24. Owner decisions: off by default with a per-IED "Use SCD deadbands" switch; the
SCD db shown even when off; no rangeC means never applied, with a note; SCADA writes of db/zeroDb
follow the SCD valKind ("Follow the SCD"); values typed by hand always go out. Change
[0190](../changes/2026/0190-nr-access-points-and-deadbands.md). Go engine and the native IED worker.

## Problem

IEC 61850-7-3 MV/CMV: mag is reported only when it moves more than db (in 0.001 % of rangeC
max - min); zeroDb forces values near zero to 0; instMag/instCVal always has the live value.
ScadaBench ignored db, so a 10 Hz generator sent 10 reports per second, and a SCADA writing db got
"access denied" because libiec61850 allows writes only to FC SP/SV/SE. The owner's BCUs carry
db=100000 with rangeC (the whole range): applied silently, their values would almost never be
reported. That is why the switch is off by default.

## Decision

1. **Switch.** IED field `scd_deadbands: true`, stored only while on (projects without it stay
   byte-identical). `GET/PUT /ieds/{id}/deadbands` (Go-only) read and set it; a running IED follows
   at once (atomic flag on the run).
2. **Filter (runtime/deadbands.go).** Only for `source == "generator"` and only when the switch is on
   and the measurement has rangeC with max > min: zeroDb first, then mag (and its q/t) are written
   only when |new - last reported| > band. instMag/instCVal (when the model has it) follows every
   value. Hand-typed values, scenarios, tests and commands are never filtered.
3. **SCADA writes (native `deadband_writes.cpp`).** The start key `writable_deadbands` lists the
   db/zeroDb references whose valKind (instance over type) is Set or not given; the worker installs a
   write handler only on those, accepts INT 0..100000, and reports `deadband_written`; Go stores the
   value, so reads and the filter use it. RO/Conf/Spec and every other CF attribute stay refused.
4. **Display.** The IED overview shows the switch, counts and each measurement's deadband in plain
   words; Data points rows show "db N from the SCD · applied / not applied". All text comes from Go.

## Compatibility

With the switch off the write path does one atomic load per generator sample and nothing else; start
requests differ only by the new key when the SCD has writable deadbands; the native worker changes
only for those writes. Python engine unchanged.
