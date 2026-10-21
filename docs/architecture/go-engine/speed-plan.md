# Speed plan: blazing fast, ultra light, top network performance

Owner, 2026-10-20: "make the app ultra fast, ultra lightweight on CPU and RAM, top notch network
performance"; felt speed first; no user-visible differences; fix and check all bugs first.
Built from four read-only planning reviews (wiring, push/UI, native, startup/packaging) and the
first stress baseline ([0123](../../changes/2026/0123-stress-bench-scenarios.md)).

## Rules for every lane

- One lane = one agent = one worktree branch `speed-<lane>` = disjoint files (table below).
- Byte-identical for anything a user sees or saves; relaxed only for invisible details (header
  order, error text for requests the UI never sends). Python is frozen; push/host modes are Go-only
  behind capabilities, Python keeps today's behaviour.
- Never touch the owner's running app, `studio/native/build/Release`, or its data folder. Native
  builds go to `studio/native/build/speed-<lane>`. Loopback 127.67.x.y, ports >= 20000. No EXE.
- Laptop: `GOMAXPROCS=4`, `go test -p 2 -trimpath`, one heavy build at a time per lane, stop if
  free RAM < 2 GB. Each lane ends with its stream log under `streams/speed-<lane>.md` with
  measured before/after numbers; the integrator merges, records, runs `verify.py`.
- Every file < 300 lines, one responsibility; new backend code in Go.

## Targets (gates)

| Measure | Today (Python) | Target |
|---|---|---|
| CB click to symbol, p95 | ~1.2 s (0.8 s full save) | <= 50 ms |
| Value change to paint, p95 | up to 2-3 s (polling) | <= 20 ms |
| Project save on edit | 7.9 ms (+ full save per click) | <= 5 ms p95 (journal) |
| Backend idle CPU | 0.45-1 % | <= 0.5 % (no timers when idle) |
| Browser idle scripting | polls every 0.3-5 s | <= 0.5 %, 0 requests/min |
| 20 idle IEDs CPU | 31 % (old) / 1.9 % (ADR-0079) | <= 1.9 % without the global 1 ms timer |
| MMS read, 1 active + 4 idle clients | 42.8 ms (old) | <= 0.3 ms |
| GOOSE MinTime gaps, any load | 23.9 ms late frames (old) | max <= 6 ms, 0 past TAL |
| Run 20 IEDs | not measured (0.59 s at 5) | <= 0.5 s |
| Double-click to project shown | not measured | <= 1.5 s small, <= 3 s large |
| Gateway 5000 points idle CPU | 12.6-19 % | <= 0.1 % (Go measured 0.03 %) |

## Phase 0: bugs first (tonight)

- **Q - real-time UI test.** Separate launcher instance (temp data folder, demo station,
  127.67.x.y), built-in browser, every page and function (import, SLD, operate, Run/Stop, GOOSE,
  Health, signal flow, test panel, scenarios, tools, gateway/master, Explorer, Find IEDs,
  settings, license view) at 1366x768; click-to-paint timings; console errors. Output: bug list
  with repro steps and severity. Fixes: frontend bugs fixed directly; Python backend bugs fixed
  only when critical (owner approved "fix all bugs" 2026-10-20) and mirrored in Go parity tests.
- Already fixed: protected build lacked `studio_goose.exe` (0122).
- Known: benchmark S4 probe check always 0 (0123); `GET /project/test-panel` registered by both
  runtimeapi and toolsapi (composition would fail).

## Phase 1: one Go server (item 1, biggest felt gain)

Interfaces to close (from the wiring review): runtime Bus adapter; `runtime.Project` needs
`AssignIP`; `projectapi.Runtime` needs `CycleLock`, `Emit`, IED state overlay for `GET /project`;
gates (`StartupGate`, `LicenseGate`) adapters; acquisition needs runtime value `liveState`,
`Samples`, `PublishChanged`, `SetIEDState`; native `Request` argument order and timeouts;
`goose_sources`/`goose_links` declarations not ported; runtime.Live late binding; value hub
`Subscribe` missing; toolsapi `Source`, `Stopped`, `RunningSequences`; tools worker children not
in the Job Object; `autosave.Fold` never called; gateway autostart only after recovery.
Lock order: `lifeMu -> commandMu -> project.mutate -> table.mu -> emitMu/bus.mu`.

| Lane | Owns | Steps |
|---|---|---|
| R runtime | `internal/runtime`, `internal/native` | exports (S1), value hub `Subscribe` 0-alloc (S6) |
| P project | `internal/project`, `scl`, `store`, `api/projectapi` | `AssignIP`, NodeData benchmark p95 <= 5 ms (S2), declarations port, route gaps: IED settings, scenarios, SLD from SCL, signal flow (S9) |
| L platform | `internal/api/platformapi` (new) | license, app/update, network + firewall routes with Flask goldens (S8) |
| T tools | `internal/tools/worker`, `api/toolsapi` | spawns in the Job Object, single test-panel GET |
| I integrator | `internal/app`, `cmd/studio_server`, `run_dev_engine.py` | compose click/save path (S3), events parity (S4), live (S5), tools (S7), every route served (S10), parity build + `STUDIO_ENGINE=go` suite (S11) |

R, P, L, T run in parallel; I follows R+P. `studio_xsd.exe` (S12) needs the owner's OK to
download libxml2 2.11.9 + libiconv sources. Canvas command association stays closed (owner).

## Phase 2: push updates and repaint only changes (items 2 and 4)

Today 24 poll paths (only 2 shared), every SSE event re-renders the whole App, every `signal`
copies the full value map (up to 90k keys), polled tables rebuild 5000 rows per second.
- Frontend (both engines): values, event log, mode badges in stores outside React with
  per-key listeners and one flush per animation frame; memo fixes on the diagram; every poller on
  `liveFeed` (no overlap, unchanged answers ignored, paused when the tab is hidden); virtual rows
  for unpaged lists; `#perf` marks for click-to-paint.
- Go: `GET /api/v1/capabilities`; one stream per tab `GET /api/v1/push` carrying the unchanged bus
  events plus topic frames (`full`/delta with `rev`/`base`, resync, per-client topic subscriptions,
  100-250 ms coalescing, unchanged bytes not sent, no tickers when idle). Python keeps polling.
- Lanes: F1 stores + `useStudio`; F2 event log + diagram memo; F3 `liveFeed` + pollers; F4 push
  client; F5 virtual rows; G1 push core; G2 live topics; G3 tools topics. ADR for the push stream.

## Phase 3: native, network and timers (items 3, 5, 6 + network)

- **Rebuild** `studio_control` (control_mux_v1) and `studio_ied` (ADR-0079) into the app folder:
  only with the owner's app closed and the owner's OK.
- **N1 timers:** precise clock (`GetSystemTimePreciseAsFileTime` via `hal/time_studio.c`), loop
  wait on WSA events + `CREATE_WAITABLE_TIMER_HIGH_RESOLUTION`, power-throttling opt-out, remove
  global `timeBeginPeriod(1)` (fallback on old Windows). Keeps ADR-0070 TAL rules.
- **N2 GOOSE uplink:** `studio_goose` waits on `pcap_getevent` + events instead of 1 ms polling:
  0 idle wakes, outbound < 0.1 ms.
- **N3 waits:** `studio_control` CmdTerm condition variable (-10-15 ms per enhanced operate),
  `studio_live` SV reader event wait.
- **N4 IPC hot path:** cache ref -> attribute in `update_batch` (11-13 -> ~8 us/attr), `ReadFile`
  framing; Go start sends `update_batch` instead of one round trip per signal.
- **N5 IED host (later, measured):** many IEDs per process (shards of 8, loop groups, watchdog,
  version-2 `hello`, Go-only), est. 20 IEDs 190 -> 45-60 MB, Run <= 0.5 s. Crash isolation change
  is an owner decision.
- **Live network (real IEDs):** persistent MMS sessions, report-driven values, push to UI; measure
  with an independent client per AGENTS.md.

## Phase 4: startup and packaging (item 7)

- Now (both engines): static spinner in `index.html`; lazy pages and xyflow chunk (785 KB bundle
  -> <= 250 KB initial); one `/project` fetch (today twice); license and project in parallel;
  polls start after first paint; startup marks.
- Go launcher (P5 parity): lease -> listen -> window spawn -> compose -> serve, recovery in the
  background; exact Python single-instance event names, data folder rules, `_internal` layout.
- Go-only package after all gates (ADR amending ADR-0074/0063); no EXE without the owner's ask.

## Order tonight

1. Q (bug hunt) + R + P + L in parallel; N1 native build in its own folder when RAM allows.
2. Then T, F1/F3 (frontend hygiene helps Python today), I composition when R+P land.
3. Each lane merged only after its tests, parity and measurements; `verify.py` on master.
