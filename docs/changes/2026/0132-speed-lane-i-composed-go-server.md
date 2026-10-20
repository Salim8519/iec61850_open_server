# 0132: Speed lane I merged: one composed Go server serving every Flask route

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 1, lane I. Branch
`speed-i` (a5f7fb1..6b01223) merged into `master` (c9ecf1c). Stream log:
[speed-i](../../architecture/go-engine/streams/speed-i.md). Python stays the default engine;
`run_dev_engine.py --engine go` runs the Go server in development. No EXE built.

## Change

- `internal/app`: composition of project service, runtime coordinator, live acquisition, tools,
  gateway/master, platform routes and the push stream ([ADR-0087](../../decisions/ADR-0087-push-stream-and-live-store.md));
  startup order Fold -> project service -> background recovery -> runtime; graceful close (push,
  gateway/master, live, coordinator, explorer/scan, service, lease); parity build (`-tags parity`)
  and inspect operations for the golden replay harness.
- `cmd/studio_server`: `--native-exe`, `--fold-journal`, clean shutdown on signal.
- `studio/tools/run_dev_engine.py`: folds a Go journal before starting Python.
- `.gitignore`: `studio/engine/build/`.

## Evidence

- `TestFlaskParity` through the composed server 290/290; `TestEveryPythonRouteServed` 127/127
  Flask routes (plus 3 push routes), no GET answers 501.
- `TestComposedClickPath` (template, diagram, Run, 3 operates, Stop, Close, restart: position kept,
  journal folded); events identical to Python (60 global, 26 and 9 per IED);
  `TestStartupGateHoldsUntilRecovery`, `TestFoldBeforePython`, `TestComposedPushWiring` pass.
- 60 s composed stress: ~400k requests, 71k SSE frames, no stall, no 500.
- Golden replay `dual` (python-http vs go) and `golden --engine go`: 4/4 scenarios, 0 differences.
- `STUDIO_ENGINE=go` pytest: 434 passed / 104 skipped / 7 failed / 16 errors (Python 446/104); all
  23 traced to the missing XSD helper, missing parity seams and the `service/stop` inspect op.
- Same demo station, both engines: 23/27 GET routes byte-identical, 4 differ only in uuid order or
  `age_ms`. Timings (real workers): operate click-to-symbol p50 156 -> 56 ms (max 204 -> 68);
  diagram save 10.1 -> 5.7 ms; import/open 31/47 -> 19/17 ms; Run 4 IEDs 495-528 -> 291 ms.
- Integrator on `master`: only owned paths; `go vet ./...` clean; app and studio_server pass;
  `verify.py`: see ledger line.

## Gaps (Go not yet a full replacement)

1. `studio_xsd.exe` not built: SCL import/open without a recorded verdict answers 503 (needs the
   owner's OK to download libxml2 2.11.9 + libiconv sources).
2. Stop 4 IEDs slower than Python (286 vs 159 ms): live monitors closed one after another.
3. First canvas command after start waits 0.1-0.5 s for `studio_control` to start.
4. Live values are not saved back into the project (Go shows initial values after restart).
5. Per-lane JSON writers not merged with httpapi's (not byte-identical).
6. liveapi goldens not re-run through the composed server.
7. Lab mode preflight lists no address when Windows' address lookup fails (Python raised).
8. Parity build lacks callback seams (fault, object, update opener, network) and some inspect ops.
9. Two engines on one host share the session cookie (a tab of one signs the other out).
10. `TestComposedEventsMatchPython` failed once (fixed port base 21300 suspected), 5 reruns pass.
11. Click-to-symbol <= 50 ms needs a kept command association: owner decided to keep it closed.
