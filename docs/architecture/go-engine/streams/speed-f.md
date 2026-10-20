# Speed lane F: the browser idles and repaints only what changed (F1 + F2 + F3)

Branch `speed-f` from master 538d5e1 ([speed plan](../speed-plan.md), phase 2, both engines).
Owns `studio/frontend/src/**`, `studio/tests/frontend/**` and this log. No backend file changed;
no API, contract, project data or text changed. User-visible behaviour is the same, only faster.

## What changed

**Live stores outside React (F1).** `state/frameBatch.ts` (one flush per animation frame, injectable
clock, flush on `visibilitychange`), `state/valueStore.ts` (`Map<"ied:ref", value>`, last value wins,
per-key listeners told once per frame), `state/eventLogStore.ts` (500-entry ring, time text made once
per second, kind words once per event, details on first display, per-kind counters),
`state/eventStream.ts` (the one `EventSource`, behind `setStreamFactory`; `deliverLive` hands each
event to the stores), `state/liveData.ts` (lazy singletons), `state/useLive.ts`
(`useSignalValue(key)`, `useEventCount(kind?)`, `useEventLogVersion()`).
`useStudio` no longer holds `events` or `values`: App re-renders only for project, busy, working,
error, save state, import preview and history. The `values` prop is gone from DevicesPage,
DeviceDetail, DeviceSignals, SignalTable, LiveValueTable, TestPage, TestPoints, Generators,
SourceEditor, EquipmentSheet and RelayStatusEditor; each row/cell reads its own key
(`SignalRow` memo, `LiveRow`, `PointControl`, `RelayStatusRow` fallback, SourceEditor "Value now").
Sidebar "Activity (N)" is a leaf (`ActivityCount`); GooseConflicts uses `useEventCount('goose_conflict')`.

**GET /project coalesced (F1).** `state/coalesce.ts`: one run in flight, one trailing; every caller
waits for a run that started after its call (`await refresh()` in `perform` still sees the result
of its action). Startup reads the project once right after the session step (not at the stream's
`open`: the Waitress SSE response's first bytes can take ~1 s, which delayed first paint). Reload
events (`runtime`, `resync`, `worker_failed`, `live_status`, `project_changed` from another tab)
whose timestamp is more than 250 ms older than the start of the last project read are already in
that answer and ask nothing (the backend publishes after applying a change; browser and backend
share this PC's clock). The first stream open only hydrates equipment; a reconnect reads both.

**Diagram memo (F2).** DiagramPage: `toolbar` and the partial-run `lockNote` are `useMemo`'d
(`add`/`addBay`/`openFile`/`openDevices` stable), so `Canvas` (memo) skips unrelated renders.
Mode badges: `equipment/keyedStore.ts` + `useModeBadges` write the 3 s `/health` answer key by key;
`EquipmentNode` reads `useModeBadge(id)` (was a context object rebuilt every 3 s that re-rendered
every node). `useSignalLayers` gives the page only the switches (`store.layers`, stable object);
OutsideSenders and SignalSheet subscribe themselves. `SignalLayerStore`: identical map ignored,
`mms_access` / GOOSE updates stored quietly (no notify) while their layer is off. `useModes` sets
state only when modes or simulation sources changed (the answer's check times change every poll).

**Polling hygiene (F3).** `api/liveFeed.ts` on top of `api/sharedPoll.ts`: one request per path for
all views at the fastest interval asked (intervals kept as before), never overlapping, raw-text
compare so an unchanged answer is not parsed and calls nobody, timers paused while the page is
hidden (one ask per polled path when it shows again), `refresh(path)` always delivers, an answer
after an error is always delivered, `subscribeTask(key, every, task)` for incremental feeds (null
= nothing new). `api/client.ts` gained `apiText` (same 402/403/404 handling). `api/useLiveFeed.ts`
hook. Every poller moved: activity, app/update, /live, /mms/access, /goose, /goose/share, /sv,
/network/adapters?refresh=1, /project/test-panel, /scenarios/runs, explorer status, scan status,
gateway status/config/points/commands/traffic, master status/config/points/commands/traffic,
/health, /goose/links. `grep setInterval src` only finds `sharedPoll.ts`.

**Virtual rows (F2/F5).** `ui/virtualRange.ts` (pure; measured heights for wrapping rows) and
`ui/VirtualRows.tsx` (`useVirtualRows`, `SpacerRow`, `SpacerBlock`): EventLog (500, newest first,
no array copy), TrafficLog (up to 1000), Explorer WatchList and ModelTree (flattened rows with
`aria-level`; CSS keeps the node indent). Paged tables keep their pagers.

**#perf (F2).** `app/perf.ts`: `#perf` in the address (kept in sessionStorage because the session
step clears the hash) turns on `markClick` (DiagramPage.operate), `markPaint` (EquipmentNode, on a
position change, after two frames), long-task and long-animation-frame observers and a larger
resource-timing buffer; `window.__studioPerf.report()` gives n/p50/p95/max and API request counts.

## Evidence

Own instance: product launcher (no `--dev`) from this worktree with the main checkout's venv python,
port 20617, temp data folder, workers copied to a scratch folder, demo station
`scd/open_substation.scd` (4 IEDs) moved to 127.67.0.2-5:20102, SLD built from the SCD, QA1 bound to
`CSWI1.Pos`; 1366x768, pane visible. Instrumentation independent of the app: the bundle is loaded
into a blank same-origin page after installing a React DevTools hook (commit count), a `fetch`
wrapper (requests), a longtask observer and a MutationObserver + double rAF on QA1 (click-to-paint).
The laptop ran other agents' builds (CPU 88-93 %), so A/B runs were interleaved by swapping
`dist` (before = master 538d5e1 build).

| Measure | Before | After |
|---|---|---|
| GET /project in the first 3 s, stopped | 3 | 1 |
| GET /project in the first 3 s, running (bus replay) | 5 (69-111 ms) | 1 |
| Idle stopped, Devices: requests / React commits per min | 30 / 30 | 30 / 0 |
| Idle running, Devices: requests / commits per min | 73* / 100 | 82 / 20 (one per /health answer; measured before the `useModes` compare, not re-measured) |
| Idle running, SLD: requests / commits per min | 50 / 110 | 51 / 0 |
| 40 signal/s (4 sine generators), SLD: commits per 20 s | 374; 474 (A/B) | 0 |
| Same, Devices > IED2 > Data points (100 rows) | 432; 406 (A/B) | 360 (each only the 8 changed rows) |
| Same + Activity open (500 rows): frames / dropped frame time per 10 s | 505 / 2.8 s; 461 / 3.0 s (A/B) | 602 / 0.25 s (19-28 rows drawn) |
| QA1 operate click-to-paint p50 / p95 (n=10) | 202 / 739; 231 / 573 (A/B) | 208 / 2342 (one backend stall; #perf report identical) |
| Long tasks (all runs) | 0-1 | 0 |

\* before was measured once with the pane hidden (throttled timers); intervals are unchanged, so
the nominal rate is the same 82/min (activity 30, health 20, mms/access 20, links 12). Requests per
minute do not drop with polling (same intervals by design); what drops is overlap, parsing and
rendering, and everything stops while the window is hidden. Fewer requests need the push stream.

Click-to-paint is backend-bound: in the slow series the harness's own EventSource saw
`control_applied` 347-739 ms and `equipment_state` 421-864 ms after the click; the paint followed
the event by ~30 ms (two frames). The frontend part is unchanged-to-better; a faster click needs the
native/Go lanes.

Tests: `node --experimental-strip-types --test studio/tests/frontend/*.test.ts` 208/208 (189 before,
19 new in `liveStores.test.ts`, `liveFeed.test.ts`, `speedLaneF.test.ts`; helpers `fakeClock.ts`).
`npm run build` (tsc + vite) clean. `studio/tools/verify.py` PASS (one Go `internal/updates`
TempDir-cleanup flake on Windows on the first run, PASS on rerun; not frontend). Smoke-checked in
the browser: SLD operate, Test points/Generators (live "Value now"), Devices data points, Activity
scrolling, gateway points/traffic, IED Explorer tree and watch list.

## For the Go push lane (F4/G1-G3): how to plug in

- **Bus events:** `setStreamFactory((handlers) => { ...; return { close } })` in
  `state/eventStream.ts`. Call `handlers.open()` on every (re)connect and `handlers.event(e)` for every
  bus event with the unchanged JSON shape (`id`, `kind`, `timestamp` in seconds, payload). Nothing
  else changes: values, Activity, layers, equipment and reloads already flow through `deliverLive`.
- **Topics:** `liveFeed.setPush({ covers(path), watch(path, onText) })` in `api/liveFeed.ts`. For a
  covered path the timer stops and every frame is the **full JSON text of `GET /api/v1{path}`**
  (byte-identical to the poll answer; query strings are part of the path, e.g.
  `/tools/gateway/points?protocol=iec104`). Equal text is ignored, so resending a full frame is safe.
  Deltas (`rev`/`base`) must be resolved into the full text inside the push client (F4) before
  `onText`. `watch` returns stop; `setPush(null)` falls back to polling.
- Incremental feeds (`gateway-traffic`, `master-traffic` keys via `subscribeTask`) stay polled
  until the push client offers them as a topic of pages.
- Capabilities: the push client decides from `GET /api/v1/capabilities`; Python keeps polling.

## Open items

- Polled point tables (gateway/master/converter) re-render all rows of the page when any value
  changes (no re-render when nothing changed); per-row memo would need the row editors extracted.
- `useMmsClients` re-renders the Devices overview whenever any IED's client list changes.
- Data points: one React commit per frame while values change (small, only changed rows).
- The 250 ms clock margin assumes backend and browser on the same PC (true for the product).
- Waitress buffers small SSE writes (~1 s to first bytes seen here); backend, not this lane.
