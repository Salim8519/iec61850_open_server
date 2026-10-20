# ADR-0087: Push stream (Go engine) and frontend live store

Accepted: 2026-10-20. Owner: "make the app ultra fast, ultra lightweight on CPU and RAM"
([speed plan](../architecture/go-engine/speed-plan.md) phase 2). Evidence: changes
[0130](../changes/2026/0130-speed-lane-f-frontend.md) (frontend live store) and
[0131](../changes/2026/0131-speed-lane-g-push.md) (Go push stream); logs
[speed-f](../architecture/go-engine/streams/speed-f.md), [speed-g](../architecture/go-engine/streams/speed-g.md).
Numbers 0077-0086 stay reserved for the proposed Go migration ADRs.

## Need

Value-to-paint <= 20 ms and 0 requests per minute when idle. Today the UI polls 24 paths every
0.3-5 s, and every SSE event or signal re-rendered the whole app.

## Decision

- **Frontend live store (both engines).** Values, the event log and mode badges live in stores
  outside React with per-key listeners and one flush per animation frame. Every poller goes through
  `liveFeed` (one request per path, no overlap, unchanged answers ignored, paused while hidden).
  `liveFeed.setPush` and `eventStream.setStreamFactory` let a push source replace polling.
- **Push stream (Go engine only).** `GET /api/v1/capabilities` announces it; Python answers its
  JSON 404, so the frontend keeps polling there. `GET /api/v1/push?client=<tab>&after=<id>` is one
  SSE stream per tab: bus events with the exact bytes of `/api/v1/events`, plus `event: push`
  frames (no `id:`). A topic is a polled GET route, keyed by its exact path; a full frame carries
  that route's exact body, keyed lists travel as text deltas (`rev`/`base`) that rebuild the exact
  GET text. `PUT /api/v1/push/topics` holds the tab's subscriptions (only mounted, visible views;
  `[]` when hidden stops all work). Coalescing: leading edge, then at most once per 100 ms
  (health and activity 250 ms, point tables 1 s); unchanged bytes are never sent.
- Services report changes through `OnChange` hooks; the gateway's 10 s safety sweep is replaced
  by the `project_changed` event, so an idle server runs no timers.

## Compatibility

Routes, bodies and `/api/v1/events` are unchanged; the frontend shows the same text it would
have fetched. Both engines show the same data; only timing improves. Python is untouched.
