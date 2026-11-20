# ADR-0142: Spare IED processes; SCADA master Connect does not wait for the interrogation

Accepted: 2026-11-20. Owner, 2026-11-20: Run and Stop are slow; "do all of them, once i wake, i
should see all ready and no coding left, just testing" (speed trials R1-R7; the owner approved
implementing the winners that change a decision the same night). Amends
[ADR-0133](ADR-0133-faster-run-start-offline-page.md) (adds a spare beside item 4; item 2 stays:
R1 was measured and dropped); changes the answer of `POST /tools/master/connect`. Change
[0223](../changes/2026/0223-run-stop-trials.md); evidence in the
[Run and Stop trial details](../testing/speed-trials-runstop-2026-11-20.md).

## Problem

Measured on the owner's PC against today's code (alternating rounds, other agents loading the PC):

- **R3.** Every Run started one `studio_ied` per simulated IED in the middle of the busy start;
  each process did its own start-up and, in the packaged build, its licence check (machine ID and
  the signature of `studio_server.exe`). Process start was 19-28 % of Run in Step 0, and with the
  IED addresses kept for the session (ADR-0141) it is a bigger share of what is left.
- **R7.** Connect as SCADA master answered only after the general interrogation, its end and the
  counter interrogation (`interrogate`, up to 60 s). An outstation without counters, as many are,
  never confirms the counter interrogation, so Connect waited t1 (15 s) every time although every
  value had already arrived.

## Decision

1. **Spare IED processes (R3).** While the simulation is stopped the coordinator keeps one spare
   `studio_ied` per simulated IED of the project (after Stop, after a failed Run, and after the
   project opened or changed, once the idle start-data read of ADR-0133 item 8 ends). Spares are
   started one after another and greeted with the new worker request `hello`, which runs the
   licence check and answers `{"ready": true}`. A Run takes a spare for each IED and starts a
   process only when none waits. A spare holds no model and no address; the request each worker
   then receives (`start`, values, everything after) is unchanged, so the wire is unchanged. A
   spare is never used when `studio_ied.exe` changed after it started (size or time), a spare that
   died is skipped, and every spare is closed at shutdown before the last Stop.
2. **Connect does not wait for the interrogation (R7).** `POST /tools/master/connect` answers once
   the link is open ("connected"); the interrogation it starts runs in the background and its
   values arrive as before through `master_values` events. What needed the finished interrogation
   waits for it instead: the protocol converter maps the received points only after it (gateway
   Start in converter role, the automatic connect at launch before the gateway's automatic start),
   so it serves the same points as before. Disconnect during an interrogation ends the worker at
   once instead of queueing "stop" behind it (the link closes either way); a failure of an
   interrogation that Disconnect ended is not shown.

## Considered and not adopted

- **R1: load the models while the addresses are added, bind after.** Built (a worker request
  `load` and a split start) and measured: on the busy PC the 20 model loads slowed the address step
  and each other, and Run was 3 % faster on the small project and 10 % slower on the large one.
  With the addresses kept for the session (ADR-0141) there is little left to overlap. ADR-0133
  item 2 stays as it is.
- **R5: show "stopped" before the lab clean-up.** Built and measured after ADR-0141: Stop's
  clean-up no longer removes IED addresses and costs about 2 ms (last worker gone to `stopped`
  0.017 s today, 0.016 s with R5; click to `stopped` 0.096 s on both sides). Not adopted.
- **Keeping models loaded between Runs** stays rejected (ADR-0133): a spare without a model costs
  about 1 MB of private memory, a loaded one 52-58 MB.

## Compatibility

- Wire: unchanged (spares only move when the process starts; the MMS server, reports and GOOSE
  start with the same `start` request; checked with an independent client and the native wire
  comparison).
- Worker protocol: `studio_ied` accepts `hello`. An older worker refuses it, so no spare is kept
  and Run starts processes as before.
- Processes: up to one idle `studio_ied` per simulated IED while ScadaBench is open and stopped
  (about 1 MB private memory each, no processor time).
- API: `POST /tools/master/connect` answers before the interrogation; the
  `last_interrogation` and message of the master status follow when it ends.
