# 0127: Speed lane T merged: tool workers die with the server; one GET test-panel owner

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 1, lane T. Branch
`speed-t` (577f0dd) merged into `master` (538d5e1). Stream log:
[speed-t](../../architecture/go-engine/streams/speed-t.md). Not wired into the Go server yet.

## Change

- `internal/tools/worker`: every tool worker (studio_live, studio_control, studio_gateway,
  studio_master) joins a kill-on-close Job Object right after start (`job_windows.go`, no-op
  elsewhere); if assignment fails the worker runs as before.
- `internal/api/toolsapi`: `GET /project/test-panel` removed (runtimeapi owns it and matches
  Python; toolsapi's relied on methods nothing implemented); `PUT` stays. `Runtime.Signal`,
  `Runtime.Source`, `Projects.TestPanel()` dropped from the interfaces. Composing the tools and
  runtime areas no longer fails with "registered twice".

## Evidence

- `TestHardKillLeavesNoChildren` (server killed with TerminateProcess: child gone within ~1 s; fails
  when the job assignment is removed), `TestJobCloseKillsMember`, `TestComposesWithRuntimeRoutes`.
- runtimeapi GET vs frozen Flask in a throwaway replay with three panel entries: 5/5 identical.
- Gateway CPU at 5000 points (`STUDIO_GATEWAY_BENCH=go`, real studio_gateway): idle 0.08 % of a
  core, 1000 changes/s 0.73 %; idle cost is almost all the 10 s safety sweep (0.001 % without it).
- Integrator on `master`: only owned paths; `go vet ./...` clean; tools, toolsapi, runtimeapi,
  gateway packages pass; `verify.py`: see ledger line.

## Limitations

- Gateway service still runs a 10 s `FullSweep` ticker when idle (meets the 0.1 % target; an event
  for project edits the hub does not report would remove it).
- Two identical job helpers (native and tools/worker); `overridden: true` panel case not in goldens.
