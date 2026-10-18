# 0095: IED Explorer connect never leaves the live worker stuck on "Already connected"

Date: 2026-10-18. Spec: [IED Explorer](../../specs/ied-explorer.md) (EXPLORE-09).
Found in [0093](0093-tool-layout-and-filters.md) and recorded there as a known issue.

## Problem

`studio_live` keeps one MMS association and answers a second `connect` with "Already connected".
Two connects that overlapped (React StrictMode runs ExplorerTool's "open from Find IEDs" effect
twice in development) both reset, then both asked the worker to connect: the first associated, the
second got "Already connected" and set the Explorer state to `failed`. `disconnect()` only told the
worker to disconnect when the state was `connected` or `lost`, so every later Connect started from
a still-associated worker and failed the same way ("Could not open an IEC 61850 association…")
until the worker process was killed.

## What changed

- `studio/backend/studio_app/tools/explorer.py`: connects are serialised by a `connect_lock`;
  a connect to the device that is already connected (live worker) returns the current status;
  `disconnect()` always sends `disconnect` to a running worker, whatever the Explorer state
  (`session.close()` in the worker is a no-op without an association). The worker stays warm.
- `studio/frontend/src/features/tools/explorer/ExplorerTool.tsx`: a `useRef` remembers the
  "Explore" target already opened, so the effect's second run in StrictMode only refreshes status
  and does not connect again. Production behaviour is the same: one connect per target.
- No native, protocol or project-format change.

## Evidence

- New `studio/tests/unit/test_explorer_connect.py` (stand-in worker with MmsSession's one-
  association rule): connect after a failed state, refused device then retry, two overlapping
  connects, disconnect from `failed`. With the old `explorer.py` 3 of 4 fail ("Already connected").
- New `test_explorer_reconnects_after_overlapping_and_failed_connects` in
  `studio/tests/protocol/test_explorer_wire.py` against the real `studio_live.exe` and a
  `studio_ied.exe` stand-in: with the old code the two overlapping connects end `failed`; with the
  fix both succeed, a connect from a stale `failed` state succeeds and browsing works.
- Python unit + integration 389 pass, 8 skipped; protocol `test_explorer_wire.py` 7 pass (native
  build from the main checkout via `STUDIO_NATIVE_BIN`); frontend `node --experimental-strip-types
  --test studio/tests/frontend/*.test.ts` 140 pass; `tsc --noEmit` clean; `verify.py` PASS.

## Limitations

- The StrictMode double effect was not re-run by hand in the dev app with a device; the frontend
  guard is covered by `tsc` only, and the backend now tolerates overlapping connects anyway.
- A connect to the same device while one is still in progress waits up to the worker's 15 s
  connect timeout before returning.
- Not tested against a real vendor IED.
