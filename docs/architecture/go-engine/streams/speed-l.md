# Speed lane L: platform routes (license, update check, lab network, gateway firewall)

Branch `speed-l` (cut from `master` @ fb8f804). Owner of `studio/engine/internal/api/platformapi/**`
and this log ([speed plan](../speed-plan.md), step S8). `internal/licensing`, `internal/updates`,
`internal/netcfg` and `internal/project` were used read-only; nothing outside the package changed.

## Routes (17, all byte-identical to Flask in the replay)

| Python | Go handler | Service seam (Deps) |
|---|---|---|
| `GET /license/status` | `licenseStatus` | `License.Status()` + `app_version` |
| `POST /license/trial`, `/license/activate`, `/license/activate-offline`, `PUT /license/config` | `license*` | `License` actions, `_run` mapping via `licensing.ErrorResponse` |
| `GET /app/update`, `POST /app/update/check`, `PUT /app/update` | `update*` | `Updates` (`*updates.Service`) |
| `GET /network`, `PUT /network` | `networkStatus`, `networkMode` | `Projects` (`ProjectNetwork`), `Npcap` |
| `GET /network/adapters` | `networkAdapters` | `Network.Adapters(refresh == "1")` |
| `POST /network/plan/preview`, `/plan/apply`, `GET /network/cleanup/preview`, `POST /cleanup/apply` | `plan*`, `cleanup*` | `Network` (`*netcfg.NetworkService`) |
| `GET`/`POST /network/gateway-firewall` | `firewall*` | `Firewall` (`*netcfg.GatewayFirewall`) |

All paths carry the `/api/v1` prefix; none is `Polled`. `PUT /ieds/<key>/settings` (also in
`api/network.py`) belongs to lane P. Python's evaluation order is kept: body read, then
`require_stopped`, then `.get` (so a list body is 400 "Stop the simulation" while running and 500
while stopped); apply/clean-up/`PUT /network` run under `cycle_lock`.

Files: `area.go` (Deps, interfaces, route table), `body.go` (`get_json(silent=True)`, `or {}`,
`.get`, Python truthiness), `errors.go` (`_run`, `configure`'s ValueError, uncaught-error bridge),
`license_routes.go`, `update_routes.go`, `network_routes.go`, `projects.go` (`ProjectNetwork`).

## Golden replay

- Generator `testdata/http_gen.py` (+ `gen_fakes.py`, `gen_license.py`, `gen_network.py`) runs the
  frozen `create_app()` with its test client. Only the outside world is faked: license server
  (canned answers), update server (opener), adapter list, rule lists, ARP probe, firewall state and
  the UAC helper (writes the `.result.json` the mode asks for; never starts a process). Certificates
  and release answers are signed with the test seed of `tests/unit/license_signer.py`; clocks are set
  by `now`/`tick` steps; IEDs are public dummies (`192.168.50.x`, `127.50.0.x` strings, no binding).
- `studio/.venv/Scripts/python.exe -I -B studio/engine/internal/api/platformapi/testdata/http_gen.py`
  writes `testdata/generated/http_golden.json`: **167 requests, 223 steps** (network 81, update 32,
  license 54 requests). Two runs are byte-identical; no temp path or user data in the file.
- `TestPlatformRoutesAgainstFlask` wires the real Go services (licensing, updates, netcfg network
  service and gateway firewall, project service through `ProjectNetwork`) to the same fakes and
  replays every step in order: **167/167** identical status, headers (Content-Type; no extras) and
  body bytes. A deliberate break of the UTF-8 bridge makes 2 steps fail (the check is not vacuous).
- `TestRoutesMatchFlask`: the 17 (method, path) pairs equal the Flask url_map entries.

Covered: dev and enforced licensing, trial/activate/transfer/offline/config with every server code
class (409/503/400, `code: null`), bad bodies (none, list, string, invalid JSON, wrong types),
60 s status cache, trial countdown, clock rollback, revoked state, non-UTF-8 `license.cert`, list
`client_config.json`; update check ok/304/404/500/offline/unreadable answer/important release,
manual 60 s gap, every `configure` refusal, hand-edited `update_check.json` (KeyError 400,
TypeError 500); network mode changes and refusals, endpoint conflicts with details, running
refusals, unknown/unhashable strategies, the whole apply flow (stale fingerprint 409, confirm
rules, UAC cancelled 400, helper timeout 503, refused, no result, partial failure, success, lab
mode in `GET /network`), clean-up flow back to loopback, gateway firewall add/unblock/blocked,
`RuntimeError`/`ValueError`/`KeyError` from Windows reads (503/400/400).

## Wiring notes for the integrator

```go
projectNetwork := platformapi.ProjectNetwork{Service: projects}
network := netcfg.NewNetworkService(projectNetwork, dataDir, nativeIEDExe, appExe)
platformapi.New(platformapi.Deps{
    License: licenseService,                 // the same *licensing.Service as the 402 gate
    Updates: updates.New(updates.Options{Directory: dataDir, ServerURL: licenseService.ServerURL, Current: version}),
    Network: network,
    Firewall: netcfg.NewGatewayFirewall(network, gatewayExe, netcfg.SystemExecutor{}),
    Projects: projectNetwork, Runtime: coordinatorAdapter, // State(), CycleLock()
    AppVersion: version,                     // Npcap nil = this PC
})
```

- `ProjectNetwork` also implements `netcfg.Projects` (simulated IEDs, enter/leave lab mode), so the
  network service and `PUT /network` share the project actor (`service.Update`). Enter-lab-mode is
  one committed change (lab_ip of every row + `update_network`); Python mutated the IEDs in place
  before `update_network`, so a failing `update_network` left unsaved `lab_ip`s in memory there.
  Not reachable after a successful preview (the plan already refuses clashing addresses).
- Python's update service read `license_service.server_url` at check time; pass the method value.
- Start `updates.StartBackground` and the license background verify in the composition, not here.
- Security order (403/401/402, startup wait with `/api/v1/license` exempt) is httpapi's; handlers
  assume they run after it.

## Needs from other packages (not changed here)

1. **licensing / updates: typed uncaught errors.** `licensing.UncaughtError` only carries a reason,
   but inside `create_app` a Python `KeyError` or `ValueError` is answered 400 by the app's handlers,
   not 500 (the package generators used a bare Flask app). Two cases are reachable through these
   routes and are bridged in `errors.go`: a reason shaped `'name'` (KeyError: `'version'` from a
   hand-edited `update_check.json`, `'certificate'` from a server answer) answers 400
   `{"error": "'version'"}`; `license.cert is not UTF-8` re-reads the file and answers CPython's
   `UnicodeDecodeError` text via `pycompat.DecodeUTF8` (400). Proposal: give `UncaughtError` a
   `Kind` (KeyError/ValueError/TypeError/AttributeError/OverflowError) and the exact Python message,
   then delete the bridge.
2. **netcfg `Preview(guid any, strategy string, …)`**: Python accepted any JSON value. Non-text
   strategies are mapped here (a sentinel gives the same ValueError; lists/objects become the
   TypeError 500 after the adapter and IED checks, as in Python). A `strategy any` parameter would
   remove the mapping.

## Safety

No test or tool changes adapters, addresses or firewall rules, asks for elevation, or reaches the
license/update servers: `Elevator`, `AdapterSource`, `Prober`, rule lists and firewall state are
fakes, and the update opener and license client are canned. Nothing touches the owner's app or
data folder (temp folders only).

## Evidence (2026-10-20, Go 1.25.6, CPython 3.12 venv, GOMAXPROCS=4)

| Check | Result |
|---|---|
| `go vet ./...` (studio/engine) | clean; `gofmt -l` on the package: clean |
| `go test -p 2 -trimpath -count=1 ./...` | all packages ok except one run of `internal/store/xsd` `TestGoldenSeedIsServedWithoutHelper` (Windows TempDir cleanup "directory is not empty", not this lane's code); that package passes when run again |
| `TestPlatformRoutesAgainstFlask` | 167/167 Flask answers reproduced (status, headers, body bytes) |
| `TestRoutesMatchFlask` | 17/17 routes, none polled |
| Golden regeneration | two runs byte-identical (101 KB, 223 steps) |

Read routes a settings page calls, per request (dev license, the fakes above). Before: Flask test
client in-process, which includes Flask's security/startup hooks and session cookie. After: the Go
handler only, through `http.ServeMux` (`BenchmarkPlatformReads`, 2000 iterations). The two
numbers are not the same measure; the end-to-end gate is the integrator's.

| Route | Python (Flask) | Go handler | Go allocs |
|---|---|---|---|
| `GET /network` | 707 µs | 95 µs (mostly the Npcap file check) | 46 |
| `GET /network/adapters` (cached) | 557 µs | 49 µs | 95 |
| `GET /app/update` | 788 µs | 72 µs (reads `update_check.json`) | 44 |
| `GET /license/status` (dev) | 597 µs | 9 µs | 42 |
| `GET /network/cleanup/preview` | 843 µs | 127 µs (reads the journal) | 47 |

## Open items

- Integrator: compose the area (see wiring), delete the bridge in `errors.go` once licensing has
  typed uncaught errors (need 1), and decide about the netcfg strategy parameter (need 2).
- The 413 path (bodies over 100 MiB) is httpapi's `ReadJSON` and was not recorded here.
- `GET /network` reads the Npcap DLL on every call, as Python did; a cached value would be a
  behaviour change (Npcap installed while the app runs) and needs the owner's decision.
