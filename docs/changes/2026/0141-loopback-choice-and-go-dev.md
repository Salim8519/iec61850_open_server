# 0141: "This PC only (loopback)" in the network picker; development runs the Go engine

Date: 2026-10-20. Owner: "ok network adapters fine, but add options for loopback ips, means the
ieds talk to each other on this pc, add this option"; users hate differences between dev and
production, and 0.1.15 ships the Go engine ([ADR-0083](../../decisions/ADR-0083-go-release-packaging.md)).

## Change

- `features/network/NetworkSetup.tsx`: the network card picker's first choice is "This PC only
  (loopback): IEDs talk to each other on this PC". It shows hint `network.adapter.loopback` and a
  "Use this PC only" button that sends the existing `PUT /network {mode:'loopback'}` (same call as
  "Back to this PC only"); no preview, no Windows change, no approval text. `setupWords.ts`
  exports `LOOPBACK_CHOICE`. Works on both engines (existing endpoint).
- `start-studio.cmd` runs `run_dev_engine.py`, whose default engine is now Go (as released);
  `--engine python` or `SCADABENCH_ENGINE=python` still starts the frozen Python engine.
  `internal/launcher` `DefaultEngine` = Go. Frontend changes hot-reload; Go changes need a restart.

## Evidence

- `npm run check` clean; in the running app the picker lists the new first option, selecting it
  shows the hint and "Use this PC only" (screenshot in the session; nothing applied).
- `run_dev_engine.py --print-command` prints the Go build and server command;
  `go test ./internal/launcher` ok.

## Limitations

- The owner's running dev app keeps the engine it was started with until it is restarted.
