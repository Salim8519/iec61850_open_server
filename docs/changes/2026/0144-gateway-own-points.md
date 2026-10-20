# 0144: Gateway own points: add made-up points by hand, Propose from IEDs kept

Date: 2026-10-20. Owner request in [ADR-0091](../../decisions/ADR-0091-gateway-own-points.md).
Lane G (merged 9ff3397), log [lane-g-own-points](../../architecture/go-engine/streams/lane-g-own-points.md).

## Change

- Points tab: **Add point** beside Propose from IEDs (name, type in plain words, suggested free
  address, starting value, scale; for a command, which own status it changes). Own rows show
  "Own point" and a value control (OFF/ON, position list, number) usable while the gateway runs;
  filter chip "Own points". Propose keeps own points (moves one off a taken address, with a note).
- Go: own point validation, storage, serving (good quality, spontaneous on change), live value
  route, next-point route, commands on own command points (always accepted, logged, drive the
  target status), CSV columns only when needed.

## Evidence

- Go gateway/toolsapi/app tests pass incl. Python/Flask comparison tests for IED points;
  `TestOwnPointsOverIEC104` with the real `studio_gateway.exe` and lib60870 master: own points
  read, a value change arrives spontaneously, select-and-execute on an own command moves its
  own breaker.
- Frontend 235/235 in the lane (new `ownPoints.test.ts`), `npm run check` clean.
- Browser at 1366x768 on a scratch Go server: added status, position, command and measurement
  points; a 104 master read them with quality 0; ON in the page arrived spontaneously; Close
  command set "Closed (ON)" and was logged; values survived a restart.

## Limitations

- Own points are always good quality (no invalid/questionable yet); commands are always accepted
  (no select-before-operate model); one command drives at most one status.
- (Fixed after the merge, `gateway.css`: the point table fits 1366x768 without sideways scrolling;
  the protocol tag sits under the type, narrower number boxes, tighter cells; measured in the built
  app: table scroll width 832 = visible width 832.)
- Python engine: no own points (see ADR-0091).
