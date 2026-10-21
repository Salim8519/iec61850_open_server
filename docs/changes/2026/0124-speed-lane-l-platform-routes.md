# 0124: Speed lane L merged: Go platform routes (license, update check, lab network, gateway firewall)

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 1, lane L. Branch
`speed-l` (ef5c6dc, 7a00f0e) merged into `master` (ed0f103). Stream log:
[speed-l](../../architecture/go-engine/streams/speed-l.md). Not wired into the Go server yet.

## Change

- `studio/engine/internal/api/platformapi` (`api.Area`): 17 routes. License: `/license/status`,
  `trial`, `activate`, `activate-offline`, `config`. Updates: `GET`/`PUT /app/update`,
  `POST /app/update/check`. Network: `GET`/`PUT /network`, `/network/adapters`,
  `/network/plan/{preview,apply}`, `/network/cleanup/{preview,apply}`,
  `GET`/`POST /network/gateway-firewall` (Python path; not `/tools/...`).
- `Deps` of small interfaces (License, Updates, Network, Firewall, Projects, Runtime, AppVersion,
  optional Npcap); `ProjectNetwork` adapter over the project service.

## Evidence

- Goldens recorded from the frozen Flask `create_app()` test client (`testdata/http_gen.py`):
  167 requests / 223 steps (network 81, update 32, license 54), deterministic across two runs, no
  temp paths or user data. `TestPlatformRoutesAgainstFlask`: 167/167 identical status, headers,
  body (a deliberately broken mapping fails 2 steps); `TestRoutesMatchFlask` 17/17.
- Handlers 9-127 us per request (Flask test client 557-843 us incl. hooks; not like for like).
- No test changes adapters, IPs or firewall rules, asks for elevation or contacts license/update
  servers (fakes, test seed, temp folders).
- Integrator: only owned paths; personal-path scan clean; `go vet` clean; package tests pass on
  `master`; `verify.py`: see ledger line.

## Limitations

- `licensing.UncaughtError` lacks a Python exception kind: `errors.go` bridges the two reachable
  cases (KeyError-shaped reason -> 400; non-UTF-8 `license.cert` -> CPython's exact text).
- `netcfg.Preview` takes `strategy string`; non-text JSON values are mapped to Python's answers.
- Known unreachable difference: after a failing `update_network`, Python kept unsaved `lab_ip` in
  memory; Go discards the whole change. Bodies over 100 MiB (413) not recorded here.
- Wiring left: one `*licensing.Service` for the 402 gate and these routes, updates `ServerURL` from
  it, background loops started by the composition.
