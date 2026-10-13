# 0031: Concentrator standard sequence evidence and no-op feedback fix

Date: 2026-09-17. Follow-up to change 0028 and
[ADR-0018](../../decisions/ADR-0018-concen-sbow-compatibility.md).

## New runtime evidence

Run-log records (same-PC worker clock, local) prove the observed concentrator
completes the standard enhanced-SBO sequence for Close commands on the standard
profile; the Python backend never emits `trigger`, so `control_requested`
records with `trigger=Oper` are genuine wire `Oper` writes:

- Run `bd5e8ed35b50490ba593111c8e76096c` (trace 3): selection at 22:58:03.483,
  `Oper(true)` 20 ms later, Close applied, selection released as operated.
  Later Open attempts selected but never received an `Oper` and timed out
  after the imported 30,000 ms window.
- Run `a7fc32421c6846daa5b53af67aa46568`: selection at 23:09:45.827, `Oper(true)`
  32 ms later, Close applied. The wire capture for trace 4 ends about 30 ms
  before that `Oper`, which is why earlier diagnosis saw no `Oper` frame.
- Run `28f4855e31e84f73b09e14c2caafe624` (trace 5, inversion enabled):
  selection, adapted Close (no-op), no `Oper`, 30,000 ms timeout.

Capture extras: trace 2 shows the concentrator direct-operating the Trip Relay
`SPC` (`Oper` with no select) and re-initiating a new MMS association mid-run.
Repeats arrive with add-cause 19 (0x13), matching libiec61850.

## Interpretation

The concentrator is standard-capable against this simulator; the inversion
profile consumed the Open as a Close and held the selection until timeout.
Remaining open: Open attempts never produce an `Oper` from the concentrator in
any recorded run. Client-side correction is not available; a fresh Open and
Close capture after this fix decides whether standard behavior suffices or an
explicitly bounded policy is required.

## Correction

`apply_value` no longer rewrites the native feedback attribute, its timestamp
or the revision counter when an accepted command does not change the value. A
real IED never republishes a position it did not change. The no-op case still
emits `control_requested` and `control_applied`, now carrying `changed=false`,
so the browser canvas and event log keep their authoritative refresh while
external MMS observers stop seeing fabricated select-time reports. The CONCEN
compatibility path uses the same code path, so its select-time feedback now
matches a real IED shape: unchanged value, unchanged timestamp, `stSeld=true`.

The `concen_inverted_sbow` binding on CB-1 is unchanged in the saved project;
setting it back to `standard` in Equipment Properties is the operator decision
before the next concentrator capture. No project format change.

## Evidence

- Isolated build `studio/native/build/concentrator-feedback` (the running
  development session kept the previous binaries; no packaged EXE built).
- Protocol suite: 21 passed, including the new
  `test_concen_compat_noop_keeps_native_state_and_timestamp_unchanged`, which
  asserts state, timestamp and pre/post select `Pos.t` equality, plus a single
  `control_applied` with `changed=false`.
- Unit/integration suites: 104 passed. `studio/tools/verify.py`: PASS.
- Limitation: third-party concentrator interoperability remains unproven until
  a fresh two-direction capture is recorded.
