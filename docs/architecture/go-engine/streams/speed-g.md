# Speed lane G: push stream (G1 core, G2 live topics, G3 tools topics)

Branch `speed-g` from master ab06a94 ([speed plan](../speed-plan.md), phase 2). Owns the new
`studio/engine/internal/push`, small hooks in `internal/gateway/service`, `internal/tools/explorer`,
`internal/tools/scan` and `internal/tools/numlog` (a `Watch` hook), and test-only files
`push_topics_test.go` in `api/liveapi`, `api/runtimeapi`, `api/toolsapi` (plus three lines in
toolsapi's Flask replay). Not touched: `internal/app`, `cmd/studio_server`, the frontend, Python.

## What the push stream is

Go-only. Python answers `/api/v1/capabilities` with its JSON 404, so the frontend keeps polling.

| Route | Answer |
|---|---|
| `GET /api/v1/capabilities` | `{"epoch":"<boot id>","paths":{"/health":{"params":[],"t":"health"},...},"push":1,"topics":[sorted names]}` |
| `GET /api/v1/push?client=<tab id>&after=<bus id>` | SSE: bus events exactly as `/api/v1/events`, plus topic frames |
| `PUT /api/v1/push/topics` | body `{"client":id,"topics":[{"t":name,"params":{..}}],"resync":[names]}`; answer `{"epoch","topics":[accepted],"unknown":[names]}` |

- Tab id: 1-64 of `A-Z a-z 0-9 - _ .`. Otherwise 400 `{"error":"Send the tab id as client"}`.
- The cursor rule is the same as for `/events`: `Last-Event-ID` header, else `?after=`, Python `int()`.
- A PUT replaces the tab's whole subscription list. Unknown names are reported, not refused.
- The PUT may come before or after the stream opens. A tab's subscriptions live 30 s
  (`Grace`) after its stream closes, so a reconnect keeps them.
- A second stream with the same tab id replaces the first (the old one ends).
- At most 32 tabs. Beyond that, 503 `{"error":"Too many app windows are open; close one and reload"}`.

## Frame schema (for lane F)

Stream text, in order: `retry: 1000\n\n`, then the hello frame, then everything else.

- **Bus events:** `id: N\ndata: <json>\n\n`. The bytes are identical to `/api/v1/events`:
  same ring, replay, `resync` and `: heartbeat\n\n` after 10 s quiet.
- **Topic frames:** `event: push\ndata: <frame>\n\n`. They have **no** `id:` line, so
  EventSource's `Last-Event-ID` stays the bus cursor.
- **Hello:** `{"t":"hello","epoch":"<boot id>","client":"<tab id>"}` is sent first on every
  connect. A different epoch means the server restarted: drop every topic state. Full frames
  follow anyway.

**Full frame:** `{"t":name,"rev":N[,"status":S][,"rows":{"head":H,"keys":[..],"ends":[..]}],"full":<body>}`.

- `full` is the exact body of the polled GET, without Flask's trailing newline.
- `full` is always the **last** member. The exact text is therefore
  `frame.slice(frame.indexOf('"full":') + 7, -1)`. Use this, not `JSON.stringify`, which
  loses `1.0`, escapes and key order. The first `"full":` in a frame is always that member:
  `t`, `rev`, `status` and the `rows` keys before it are JSON strings or numbers.
- `status` appears only when the route did not answer 200. The body is then its JSON error,
  as the poll would have rejected.
- `rows` appears only for keyed topics:
  - `head` is the length of the text up to and including the list's `[`.
  - `ends[i]` is the end offset of row `i`. Row `i` starts at `head` (i = 0) or at `ends[i-1] + 1`.
  - `tail` is the text from `ends[last]` (or from `head` when the list is empty), starting at `]`.
  - Offsets count bytes. Flask bodies are ASCII, so bytes equal UTF-16 units.
- `rev` restarts at 1 when the server restarts or the topic's params change.
  **A full frame is always applied.**

**Delta frame (keyed topics only):** `{"t":name,"rev":N,"base":B[,"head":"<text>"][,"tail":"<text>"][,"upsert":{"<key>":"<row text>"}][,"remove":[keys]][,"add":[keys]]}`.

- All texts are JSON strings that hold the exact bytes.
- To apply:
  1. Replace `head` and/or `tail` if present.
  2. Delete the `remove` keys.
  3. Append the `add` keys in order at the end.
  4. Set `rows[key] = upsert[key]`.
  5. Rebuild the text: `head + keys.map(k => rows[k]).join(",") + tail`. This is the GET text
     byte for byte. The server only sends a delta when the list is compact and the result is exact.
- Rules for the client:
  - `base == held rev`: apply the delta.
  - `rev <= held rev`: ignore it (it was already shown through a newer full frame).
  - Anything else: PUT `resync:[name]`. The full frame of the current rev follows.
- When a tab is slow, unsent deltas collapse into the newest full frame, so the client
  never sees a gap it could apply wrongly.

**Path → topic** (`paths` in `/capabilities`; paths are relative to `/api/v1`):

| Path | Topic | Kind | Re-read on |
|---|---|---|---|
| `/health` | `health` | full; `checked_at` changes alone are not sent | live_\*, goose_\*, sv_\*, mms_access, Mod/Beh/Sim values; 1 s while running; gap 250 ms |
| `/goose/links` | `goose.links` | full | goose_\*; 1 s while running |
| `/goose` | `goose.overview` | full | goose_\*; 1 s while running |
| `/goose/share` | `goose.share` | full | goose_share, goose_conflict, goose_uplink |
| `/sv` | `sv` | full | sv_\*; 1 s while running |
| `/live` | `live.sessions` | full | live_\*, goose_status; 1 s while running |
| `/mms/access` | `mms.access` | full | mms_access |
| `/project/test-panel` | `test.panel` | full | every value change, source\*, pulse_\*, scenario_\*, live_\* |
| `/scenarios/runs` | `scenario.runs` | full | scenario_\* |
| `/activity` | `activity` | full | gateway/master/explorer/scan changes and events; gap 250 ms |
| `/tools/gateway` | `gateway.status` | full | gateway hook, gateway_\* |
| `/tools/gateway/config` | `gateway.config` | full | project_changed |
| `/tools/gateway/points` (param `protocol`) | `gateway.points` | keyed `points` by `id` | every value change, gateway/master points hooks; gap 1 s |
| `/tools/gateway/commands` | `gateway.commands` | keyed `entries` by `seq` | gateway commands log |
| `/tools/gateway/traffic` (no `after`) | `gateway.traffic` | keyed `entries` by `seq` | gateway traffic log |
| `/tools/master`, `/tools/master/config` | `master.status`, `master.config` | full | master hook, master_\* |
| `/tools/master/points` | `master.points` | keyed `points` by `key` | master points hook; gap 1 s |
| `/tools/master/commands`, `/tools/master/traffic` | `master.commands`, `master.traffic` | keyed by `seq` | master logs |
| `/tools/explorer` | `explorer` | full | explorer hook, explorer_\* |
| `/tools/scan` | `scan` | full | scan hook; 1 s while scanning |

`runtime`, `ied_runtime`, `project_changed`, a bus `resync` and a value-hub resync re-read
every subscribed topic. The default gap is 100 ms: a change after a quiet period is read at
once, and a stream of changes at most once per gap.

## Frontend notes (lane F)

- `liveFeed.setPush({covers, watch})`. `covers(path)` is true when the path without its
  query is in `paths` and the query names only that topic's `params`.
  - `/tools/gateway/points?protocol=iec104` is covered.
  - `?after=` traffic pages are not.
- One subscription per topic name per tab. The last params win. Watchers of one path share it.
- Send the current list on every change of mounted views. Send `[]` when the tab is hidden,
  so the server computes nothing for it.
- `onText` gets the rebuilt full text. Identical repeats can still be dropped client-side.
  The server sends nothing when bytes are unchanged.
- Traffic: the topic is the newest page without `after` (300 entries, keyed by `seq`).
  `TrafficLog` can switch to it by keeping lines with `seq` above its last.
  Until then those views keep polling, as agreed.
- Bus events: parse the `data:` lines without `event: push` exactly as today. `handlers.open()`
  runs on every (re)open. The hello frame tells whether the epoch changed.
- A Go reference of the client algorithm is `internal/push/mirror_test.go`. It is checked
  over 300 random edits (`TestKeyedRandomEditsRebuildExactText`).

## Wiring notes (lane I)

```go
area := push.New(push.Options{
	Bus:    bus,                                   // the same *events.Bus as events.Area
	Values: func(n func(ied, ref string)) func() { // value hub (speed-r S6)
		return coordinator.Subscribe(func(c runtime.Change) { n(c.IEDID, c.Ref) })
	},
	Topics: topics,
	Listen: func(e *events.Event) {                // replaces the gateway's 10 s sweep
		if e.Kind == "project_changed" { gatewayManager.ProjectChanged() }
	},
})
topics, missing := push.Catalog(push.IndexRoutes(liveRoutes, runtimeRoutes, toolsRoutes),
	push.CatalogOptions{Running: func() bool { return coordinator.State() != "stopped" },
		Scanning: scanner.Scanning})
// build topics first, then area; log `missing` (empty when all three areas are mounted)
gatewayManager.OnChange(area.Changed); master.OnChange(area.Changed)
explorerService.OnChange(area.Changed); scanner.OnChange(area.Changed)
routes = append(routes, area.Routes()...)
```

- `IndexRoutes` takes the areas' raw `[]api.Route`. Topic bodies come from those exact
  handlers, called in-process. A panic in one answers Werkzeug's 500 text, as the server would.
- Call `ProjectChanged` on **every** `project_changed`. Without it, a role edit reaches SCADA
  only with the next value change, because the 10 s sweep is gone.
- The push routes need the same security middleware as `/api/v1/events`. They are long-lived
  like `/events`, so the same write-timeout exemption applies.
- `area.Close()` is for tests only. The area lives as long as the process.

## Design

- One shared instance per topic and query. The body is built once per change and the frame
  encoded once for every tab. The last subscriber leaving closes the instance: no timer, no
  value-hub subscription.
- Every bus event is read once by one goroutine and marks topics by kind. Value-hub
  notifications mark value-driven topics. Hooks call `Area.Changed(name)`. Marks never block.
- Per-instance `throttle`:
  - The first mark after a quiet gap fires at once.
  - Marks during a build or within the gap give one trailing build. Only that build arms a timer.
- One build goroutine reads routes in turn and compares the bytes; `Ignore` drops a member
  from the comparison. A change bumps `rev`; keyed topics then diff their rows.
- Per-tab outbox: at most one entry per topic. Collisions become "send the newest full".
  `sent` revision tracking drops duplicates.
- Bus events go out as soon as they arrive. After a write that carried events, the stream
  gathers for 16 ms, so a burst leaves in few writes.
- Idle state:
  - every goroutine blocks on a channel;
  - each stream has only the 10 s heartbeat wait, as `/events` does;
  - the 1 s re-reads run only while `Running`/`Scanning` is true and a tab subscribes.

## Gateway, explorer, scan hooks

- `service.Manager` and `service.Master` have `OnChange(fn)`. They report the names `gateway.status|points|commands|traffic`
  and `master.status|points|commands|traffic`: on start, stop, message, client changes,
  worker failure, link, values, probe and interrogation, and on every log `Add`/`Update` via
  `numlog.Log.Watch`. Status includes `traffic_last`, so traffic also reports status.
- `explorer.Service` and `scan.Service` have `OnChange(fn)`, reporting `"explorer"` and `"scan"`.
  `scan.Service.Scanning()` is new.
- **Sweep removed:** `FullSweep`, `SweepInterval` and the ticker are gone. `Manager.ProjectChanged()`
  marks all points and their quality/timestamp dependencies, and reports `gateway.points`.
  Python diffed every 100 ms, so an edit now reaches SCADA as fast as it did there (once wired).

## Evidence (2026-10-20, Ryzen 9 5900HS, GOMAXPROCS=4, `go test -p 2 -trimpath`)

- `go vet ./...` is clean. `go test -p 2 -trimpath ./...` is green (all packages), with
  `STUDIO_PYTHON` set. `check_file_lengths.py` passes. The largest new file is `hub_test.go`
  at 283 lines.
- **Full frame == Flask body:**
  - `liveapi` replays Python's scenario and answers each topic GET through the push path:
    health 2, goose.links 2, goose.overview 2, goose.share 3, sv 2, live.sessions 2,
    mms.access 2 frames.
  - `runtimeapi` goldens: test.panel, scenario.runs.
  - `toolsapi` Flask oracle: activity, gateway.status/config/points (5)/commands/traffic,
    master.status/config/points/commands/traffic, scan. `/tools/explorer` is not in the Flask
    scenarios, because it warms a worker.
  - Mutation check: one extra byte in `FullFrame` failed 35 comparisons.
- **Coverage:** `push.Coverage` per area. Every polled GET is a topic, and every topic is a
  polled GET, except the two config routes, which are read once per view.
- **Stream** (`stream_test.go`):
  - bus bytes are identical to `/api/v1/events` after 200 events, a burst included, with
    non-ASCII and escapes;
  - push frames carry no `id:` line;
  - heartbeats arrive;
  - a closed tab releases its topics after Grace, and no re-read runs afterwards;
  - the same tab id replaces the old stream;
  - capabilities, PUT and unknown topics behave as described.
- **Hub/throttle** (fake clock):
  - leading edge, then trailing;
  - 500 ms of marks every 5 ms gives 6 builds;
  - a mark during a long build gives one more build;
  - 50 changes within the gap give one read;
  - unchanged and ignored-member bytes are not sent;
  - two tabs share one read per change;
  - queued deltas collapse into a full frame;
  - a hidden tab (`[]`) cancels the value subscription and leaves 0 timers and 0 instances;
  - grace expiry and reconnect work.
  - `push` passed 8 runs in a row with `-count=8`.
- **Idle:** 10 tabs with open streams and subscribed topics, over 2 s: **2 allocations (the
  measurement's own) and 0 s Go CPU** (`TestIdleTabsCostNothing`).
- **Costs:**

  | Benchmark | Result |
  |---|---|
  | `BenchmarkKeyedBuild5000` (1.2 MB table, one row changed: cut, compare, delta) | 3.4 ms, 47 allocs |
  | `BenchmarkChangedNoMatch` | 250 ns, 0 allocs |
  | `BenchmarkTable5000` (the route body itself) | 22 ms |

  At 22 ms per table, the point tables use a 1 s gap: never more work than today's 1 s poll,
  and nothing when values do not change.
- **Gateway** (`STUDIO_GATEWAY_BENCH=go`, real `studio_gateway` from the main checkout's
  Release, read only, 5000 points, 20 s windows):

  | Changes/s | Before (speed-t, with sweep) | After |
  |---|---|---|
  | 0 (idle) | 0.016 s (0.08 %) | **0.000 s (0.00 %)** |
  | 1000 | 0.73 % | 0.38 % (0.23 % tick-based) |

  - `TestGatewayServesValuesAndCommandsOverIEC104` passes.
  - `TestProjectEditReachesGatewayWithoutSweep`: no update for 300 ms without an event. After
    `ProjectChanged`, `source_ok` false reaches the gateway.

## Open

- `gateway.points` costs 22 ms per build at 5000 points (route body, not push). An
  incremental table in the gateway service would allow a 100 ms gap. That is a later
  measured change.
- The traffic views keep `?after=` polling until lane F switches them to the page topic.
- Live sessions, goose and sv still need the 1 s re-read while running (frame counters have
  no event). Native counters events (N-lanes) could remove it.
- Topic triggers come from reading each route's inputs. A missed trigger shows up only as a
  late view; the 1 s re-read hides it while running. A resync or reconnect always re-reads.

## Proposed ADR text: "Push stream and frontend live store"

> **Push stream (Go engine).** The Go engine serves `GET /api/v1/push`, one SSE stream per
> tab. It carries the bus events with the exact bytes of `/api/v1/events`, plus topic frames
> that replace polling. A topic is a polled GET route; its full frame is that route's exact
> body. Keyed lists travel as text deltas that rebuild the exact GET text, with a revision
> and base per topic. A tab subscribes only to what mounted, visible views show
> (`PUT /api/v1/push/topics`). Changes are coalesced (leading edge, then at most once per
> 100 ms; health 250 ms; point tables 1 s), and unchanged bytes are never sent. `/capabilities`
> announces the stream; the Python engine does not, so the frontend keeps polling there.
>
> - *Need:* value-to-paint ≤ 20 ms and 0 requests per minute when idle (speed plan phase 2).
>   Polling costs 24 paths every 0.3-5 s.
> - *Compatibility:* routes, bodies and `/api/v1/events` are unchanged, and the frontend
>   shows the same text it would have fetched. Both engines show the same data; only the
>   timing improves.
> - *Consequences:* services report changes through `OnChange` hooks. The gateway's 10 s
>   safety sweep is replaced by the `project_changed` event, so an idle server runs no timers.
>   The frontend's live store gets its text from the push client or from polling, behind one
>   interface (`liveFeed`).
