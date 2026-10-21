# 0134: Speed lane F4 merged: frontend push client (Go engine), polling unchanged on Python

Date: 2026-10-20. [ADR-0087](../../decisions/ADR-0087-push-stream-and-live-store.md); speed plan
phase 2 (F4). Branch `speed-f4` (7fe0226, 256ccc7) merged into `master` (7294f81). Stream log:
[speed-f4](../../architecture/go-engine/streams/speed-f4.md).

## Change

- `api/capabilities.ts`: raw `GET /api/v1/capabilities` after the session step; push only when
  `push === 1`; `pathTopic()` holds the coverage rules (path in `paths`, only the topic's params).
- `api/pushStream.ts` (pure): full-frame body by offset (never re-encoded), keyed deltas, rev/base
  rules, resync, epoch reset, status frames re-read over HTTP, parameter changes.
- `api/pushClient.ts`: one `/api/v1/push` EventSource per tab (bus events to the existing handlers,
  push frames to `pushStream`), `liveFeed.setPush`, topic PUTs debounced per 16 ms and `[]` while
  hidden, polling fallback on errors with 1-10 s backoff and `after=` resume, tab id per tab.
- `api/liveFeed.ts` watch hooks; `main.tsx` installs the client.

## Evidence

- Frontend tests 227/227 (19 new). 399 frames recorded from the Go push hub (376 keyed deltas over
  300 random edits, `1.0`, escapes, `é`, a 500 status frame) rebuild the expected GET bodies
  exactly (398/398 SHA-256); JS port of the random-edit test, 1000 edits with 10 % frames lost:
  exact text after every applied frame; 9 of 10 mutations caught (the 10th is a redundant guard).
- Python unchanged (same launcher, `dist` swapped): idle running 82 requests/min and idle stopped
  30/min identical per path; one extra `capabilities` request (404) at load; one `/api/v1/events`.
- Integrator on `master`: frontend tests 227/227, `npm run build` clean. `verify.py` on 7294f81:
  6/7 PASS; the Go suite failed once in `liveapi TestRoutesMatchPython` (not frontend code): under
  load the initial live-poll `signal` arrived after its step closed and was counted in the next
  step (same events, same order). It passes 30/30 alone; harness fix assigned to lane I2.

## Limitations

- Not yet tested end to end against the composed Go server (11-step plan in the log, next phase).
- On Python the browser console shows one "Failed to load resource: 404" line for the
  capabilities probe (not a JavaScript error). Traffic `?after=` views stay polled; the hide PUT
  can take ~1 s in a throttled background tab.
