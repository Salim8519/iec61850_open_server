# 0130: Speed lane F merged: live stores, one project read, memo fixes, polling hygiene, virtual rows

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 2 (F1-F3), both
engines. Branch `speed-f` (5063ce4) merged into `master` (26b2b22). Stream log:
[speed-f](../../architecture/go-engine/streams/speed-f.md). Frontend only: no backend, API, text or
project-data change.

## Change

- Stores outside React (`state/frameBatch.ts`, `valueStore.ts`, `eventLogStore.ts`,
  `eventStream.ts`, `liveData.ts`, `useLive.ts`): values and the event log no longer flow as props;
  each row/node reads its own key, flushed once per animation frame.
- `GET /project`: `state/coalesce.ts` keeps one request in flight plus one waiting; startup read
  right after the session; replayed reload events older than the last read are ignored.
- Diagram: memoised toolbar and lock note; mode badges through a keyed store; signal layers quiet
  while off; unchanged GOOSE link maps ignored.
- `api/liveFeed.ts` on `sharedPoll`: one request per path, no overlap, unchanged answers call
  nobody, paused while hidden; every poller moved (intervals unchanged).
- `ui/virtualRange.ts`, `ui/VirtualRows.tsx`: Activity log, traffic log, Explorer watch list/tree.
- `app/perf.ts` behind `#perf`: click-to-paint marks, long-task/long-frame observers.
- Plug-in points for the Go push stream: `eventStream.setStreamFactory`, `liveFeed.setPush`.

## Evidence

- Own launcher instance (demo station on 127.67.0.2-5:20102), CPU 88-93 % from other agents,
  A/B by swapping `dist`: `GET /project` at startup 5/3 -> 1/1; idle stopped Devices React commits
  30 -> 0/min; idle running SLD 110 -> 0/min; 40 signals/s on SLD 474 -> 0 commits per 20 s;
  Activity open 3.0 -> 0.25 s dropped frame time per 10 s; long tasks 0-1 -> 0.
- Operate click-to-paint unchanged (p50 ~208 ms): bound by the Python backend's `equipment_state`
  publish, paint follows ~30 ms later; `#perf` agrees with the independent harness.
- Frontend tests 208/208 (19 new); `npm run build` clean; integrator re-ran both on `master`.

## Limitations

- Requests per minute unchanged (same intervals by design until push replaces polling).
- Gateway/master/converter point tables re-render all rows on the page when any value changes;
  `useMmsClients` re-renders the device overview on any IED's client list change.
- The replay-age filter assumes browser and backend share the PC clock (always local today).
