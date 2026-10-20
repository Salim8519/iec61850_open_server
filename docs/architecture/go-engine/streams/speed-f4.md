# Speed lane F4: push client (frontend half of ADR-0087)

Branch `speed-f4` from master 8e24b7a ([speed plan](../speed-plan.md), phase 2). Owns the new
`studio/frontend/src/api/{capabilities,pushStream,pushClient}.ts`, small wiring in `api/liveFeed.ts`,
`state/eventStream.ts` (comment) and `main.tsx`, `studio/tests/frontend/**` and this log.
Contract: [speed-g](speed-g.md) (frame schema, path map, "Frontend notes"); plug-in points:
[speed-f](speed-f.md). No backend, Go, API, project data or text changed.

## What it does

| File | Responsibility |
|---|---|
| `api/capabilities.ts` | Raw `fetch` of `GET /api/v1/capabilities` (not `api()`: Python's JSON 404 must not show the "restart ScadaBench" text). Push only when `push === 1`; any error, 404, HTML or other protocol gives `null`. `pathTopic(paths, path)`: the topic of a polled path. |
| `api/pushStream.ts` | Pure (node-testable). Applies `event: push` frames per topic and returns the exact GET text. |
| `api/pushClient.ts` | One `EventSource` per tab, topic list `PUT`s, fallback and retry, tab id; `installPushClient` for the page. |
| `api/liveFeed.ts` | `PushSource.watch(path, onText, {ask, release})`; Flask's trailing newline ignored in the text compare; a request answered after a frame keeps the frame. |
| `main.tsx` | `installPushClient({feed: liveFeed, put})` before render. |

**Engine choice.** `installPushClient` replaces the event stream factory with a lazy one. `useStudio` calls it
after the session step (the capabilities request needs the session cookie). It fetches capabilities once:

- `null` (Python): it opens today's `sseStream` (`/api/v1/events`) and never calls `liveFeed.setPush`.
  Every poll, interval and request stays as before; the only new request is the one capabilities 404.
- push 1 (Go): `GET /api/v1/push?client=<tab id>&after=<last bus id>`.

**Stream.** Bus events (no `event:` line) go to `handlers.event` exactly like `/api/v1/events`;
`handlers.open` runs on every `onopen`. `event: push` frames go to `pushStream`. On the hello frame the
client goes online: `liveFeed.setPush(source)` (covered paths stop their timers) and the current topic
list is PUT, always, because after more than 30 s away the server has forgotten the tab. A new epoch
drops every held text and the record of what was sent. `onerror`: `setPush(null)` (views poll while
EventSource reconnects by itself). `CLOSED` (or a tab id change): close, wait 1 s doubling to 10 s,
ask capabilities again, reopen with `after=<last bus id>` (or, if push is gone, the plain stream).
A failed PUT is handled like a closed stream.

**Topics.** `covers(path)`: the path without query is in `paths` and the query names only that
topic's `params`, each once (`/tools/gateway/points?protocol=iec104` yes; `/tools/gateway/traffic?after=N`,
`/network/adapters?refresh=1`, `/health?x=1` no). One subscription per topic per tab: a second path of
the same topic takes it (last params win) and the earlier path is released back to polling. PUT body
`{"client","topics":[{"t","params"}],"resync":[..]}` with the watched list, `[]` while the page is
hidden; at most one per 16 ms ("per frame"; `setTimeout`, so it also leaves while hidden), never two
at once (a change during a PUT sends one more after its answer). `unknown` names in the answer are
released to polling.

**Frames** (`pushStream`, port of `internal/push/mirror_test.go`):

- Full: body = `frame.slice(frame.indexOf('"full":') + 7, -1)`; only the members before it are parsed
  (`frame.slice(0, at - 1) + "}"`), so a 1.2 MB table is never parsed or re-encoded here. Always applied.
  `rows` cut the body into head, keyed rows and tail. Offsets are bytes; Flask bodies are ASCII, and a
  body with other characters is converted to string indexes (checked with real Go frames).
- Delta: applied when `base == held rev` (head/tail, remove, add at the end in order, upsert, rebuild
  `head + rows.join(",") + tail`); ignored when `rev <= held rev`; otherwise `resync` (PUT `resync:[t]`)
  and further deltas are ignored until the full frame arrives. A missing row text also asks resync.
- `status` (route answered non-200): the view is asked to read once over HTTP (`ask`), so the error
  shows exactly as a failed poll (same `apiText` handling of 402/403/404).
- Parameter change while the old parameters are subscribed (protocol switch): frames of that topic are
  dropped until the PUT carrying the change is answered (frames of the old parameters may be on the
  wire). If a full frame was dropped meanwhile and none is held, the client asks resync once.

**Tab id.** `tab-<uuid>` in `sessionStorage` (`studio-push-tab`), so a reload reuses its subscriptions.
A duplicated tab copies sessionStorage: tabs ask each other once over a `BroadcastChannel` and the
newcomer takes a new id; its stream restarts with it on the next error.

## Evidence

2026-10-20, Ryzen 9 5900HS, other agents' builds running.

**Tests** (`node --experimental-strip-types --test studio/tests/frontend/*.test.ts`): 227/227
(208 before; 19 new in `pushStream.test.ts` 7, `pushClient.test.ts` 9, `liveFeedPush.test.ts` 3;
helpers `pushFakes.ts`). `npm run build` (tsc + vite) clean (the >500 kB chunk warning is the same on
master). `check_file_lengths.py` PASS (largest new file `pushClient.ts`, 274 lines). `studio/tools/verify.py`
PASS (all seven checks, Go tests included).

- **Real Go frames.** `fixtures/push/push-frames.json` (399 frames, 158 kB) was recorded from the Go hub
  itself (`internal/push`, fake clock): hello, full frames with `rows`, 376 keyed deltas over 300
  rounds (adds, edits, removals in the middle, a reorder that forces a full frame, `1.0`, `\"`, `\\`,
  `</x>`, escaped `é` and one raw `é` row to check byte offsets), a whole topic and a 500
  status frame. Each step carries the SHA-256 of the GET body; `pushStream` reproduces all 398. The
  generator is `fixtures/push/frames_gen.go.txt` (copy it into `internal/push` as a `_test.go` to
  regenerate; it is not compiled in place, so lane G's package is untouched).
- **mirror_test port.** 1000 random edits through a JS frame builder of the same schema, with 10 %
  of frames lost: every applied frame rebuilds the exact text; lost frames end in `resync` and a full
  frame (≈ 680 deltas applied, ≈ 140 resyncs).
- Rules: full/delta, `base == held`, `rev <= held` ignored, gap → one resync, full always applied
  (lower rev too), epoch reset, untracked topic, empty list, status, the parameter gate, Flask newline,
  late HTTP answer vs frame, `covers()` cases, PUT once per frame and never two at once, `[]` while
  hidden, last params win, stream error → polling, `CLOSED` → retry with capabilities and `after=`,
  404 capabilities → plain stream and no `setPush`.
- **Mutation check** (one change at a time, then restored): tail cut, byte offsets, base check, full
  slice, race guard, newline, gate, hidden list, retry: each fails 1-3 tests. Removing the PUT guard in
  `dirty` alone survives because `flush` checks it again (equivalent).

**Python unchanged** (own instance: product launcher without `--dev`, main checkout venv, port 20641,
temp data folder, workers copied to scratch, public demo station `scd/open_substation.scd` remapped to
127.67.0.2-5:20102; same server, `dist` swapped between master 8e24b7a's build and this one; built-in
browser, Devices page, counts from resource timing):

| Window | master | speed-f4 |
|---|---|---|
| First 20 s after load (running) | 34: activity 10, app/update 1, equipment/states 1, goose/links 4, health 7, license/status 1, mms/access 7, preferences 2, project 1 | 35: the same + `capabilities` 1 (404) |
| 60 s idle, running | 82: activity 30, goose/links 12, health 20, mms/access 20 | 82: identical per path |
| 60 s idle, stopped | 30: activity 30 | 30: activity 30 |
| Event stream | one `GET /api/v1/events` | one `GET /api/v1/events`, no `/push` |

No error banner. The browser console shows one "Failed to load resource: 404" line for the
capabilities request (a browser log of every non-2xx fetch; no JavaScript error).

## For lane I (once the Go server is composed): end-to-end test plan

Nothing here ran against Go: the push routes are not composed yet. Own Go instance, temp data, demo
station on 127.67.x.y, built-in browser, DevTools network list:

1. **Wiring.** `GET /api/v1/capabilities` → 200 `push:1` with `paths` for every topic in speed-g's
   table. `/api/v1/push` and `PUT /api/v1/push/topics` pass the same security middleware as
   `/api/v1/events` (EventSource sends only the session cookie; the PUT goes through `api()` with
   `X-Studio-Frontend`), and the stream has the write-timeout exemption. Also through the Vite proxy
   (`run_dev.py`): SSE not buffered, PUT proxied.
2. **One stream.** Exactly one `GET /api/v1/push?client=tab-…&after=0` per tab, no `/api/v1/events`;
   Activity, values, SLD states and reloads behave as with `/events` (same bus bytes).
3. **Subscriptions.** The PUT bodies list exactly the covered paths of the mounted views (Devices:
   health, goose.links, mms.access, activity; Test: test.panel, scenario.runs; Tools: gateway/master
   status, points with `protocol`, commands; Find: scan; Explorer). Switching the gateway protocol
   sends one PUT with the new `protocol`; no old-protocol rows ever show.
4. **Requests per minute** with the Devices page idle, stopped and running: expect 0 polls of covered
   paths (only `/app/update`, `/network/adapters?refresh=1` while open, traffic `?after=` pages and
   once-per-view config reads). Compare with the Python numbers above.
5. **Byte identity.** For each topic, in the console: `fetch('/api/v1'+path).then(r => r.text())` at
   rest equals what the push client holds (`liveFeed` cache; a temporary log in `onText` is fine for
   the test). Gateway points at 5000 rows: deltas apply, the table repaints only changed rows.
6. **Hidden tab.** Minimise: one PUT with `topics: []` (within ~1 s; background timers are throttled),
   server instances drop to 0. Restore: one PUT with the list, one full frame per topic, no poll burst.
7. **Restart.** Kill and restart the Go server: the views poll while it is away; hello with a new
   epoch → one PUT, full frames, polling stops. After a 30 s+ outage the subscriptions come back too.
8. **Resync.** Inject a lost delta (debug build) → one PUT `resync:[t]`, one full frame, then deltas.
9. **Duplicated tab** (right click on the tab > Duplicate): the two tabs show different
   `client` ids and their streams stay up (no 1 s replace ping-pong in the server log).
10. **Limits.** 33rd tab: 503 → that tab keeps polling and retries with backoff (≤ 10 s).
11. **Python again** with the same build: capabilities 404, numbers identical to the table above.

## Open

- Traffic views (`TrafficLog`, `?after=` pages) stay polled; switching them to the `gateway.traffic` /
  `master.traffic` page topics (keep lines with `seq` above the last shown) is a follow-up.
- Capabilities are re-read only after a closed stream, not on every new epoch (same build, same paths).
- A frame for a path that lost its topic to another path is not shown; that path polls (correct, not
  push-fast). Two views of one topic with different parameters at once are rare (none today).
- `PUT []` on hide leaves after up to ~1 s because hidden tabs throttle timers.
- Change record, ledger entry, ADR-0087 link and the streams index line are left to the merge
  (as for lanes F and G).
