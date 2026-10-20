# 0131: Speed lane G merged: Go push stream; gateway without idle timer

Date: 2026-10-20. [ADR-0087](../../decisions/ADR-0087-push-stream-and-live-store.md); speed plan
phase 2 (G1-G3). Branch `speed-g` (15cd28f, 81c8921) merged into `master` (8e24b7a). Stream log:
[speed-g](../../architecture/go-engine/streams/speed-g.md). Not wired into the Go server yet
(lane I); frontend push client not written yet.

## Change

- New `internal/push`: `/api/v1/capabilities`, `/api/v1/push` (one SSE stream per tab, hello frame
  with epoch, bus bytes as `/events`, topic frames), `PUT /api/v1/push/topics` (32 tabs max,
  subscriptions survive 30 s after disconnect). Topics: health, goose.links, goose.overview,
  goose.share, sv, live.sessions, mms.access, test.panel, scenario.runs, activity, gateway and
  master status/config/points/commands/traffic, explorer, scan; keyed by exact GET path.
- Hooks: `OnChange` in gateway/service, tools/explorer, tools/scan; `tools/numlog.Watch`.
- Gateway: 10 s `FullSweep` ticker removed; `ProjectChanged` re-checks points on project edits.

## Evidence

- Full frames equal the Flask body: liveapi Python scenario (7 live topics), runtimeapi goldens
  (2), toolsapi Flask oracle (12 tools topics; explorer not in Flask scenarios); one added byte
  fails 35 comparisons. Every polled GET is a topic.
- Stream: after 200 events incl. bursts, bus bytes identical to `/events`; push frames carry no
  id; heartbeat; closed tab releases topics; reconnect replaces stream. Client algorithm
  reference (`mirror_test.go`) over 300 random edits rebuilds the exact GET text.
- Idle: 10 tabs over 2 s, 0 s Go CPU; hidden tab = 0 timers, no value-hub subscription.
- Gateway, real studio_gateway, 5000 points: idle 0.08 % -> 0.00 % of a core; 1000 changes/s
  0.73 % -> 0.38 %; IEC 104 wire test passes; role edit reaches the gateway via ProjectChanged.
- Integrator on `master`: `go vet ./...` clean; push, api, gateway, tools packages pass;
  `verify.py`: see ledger line.

## Limitations

- Point tables use a 1 s gap (route body ~22 ms at 5000 points); an incremental gateway table
  would allow 100 ms. Health/goose/sv/live/scan re-read every 1 s while running and subscribed
  (native counter events would remove it). Traffic `?after=` pages stay polled. Files touched
  outside the lane list: test-only push tests in three api areas, `tools/numlog` hook.
