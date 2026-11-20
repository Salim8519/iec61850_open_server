# Trial harness (speed and network trials, 2026-11-20)

Tools for `docs/testing/speed-trials-plan-2026-11-20.md`. They are **not** part of
`verify.py`: they start real engines and native workers and change addresses on the ScadaBench
Lab card. Results: `docs/testing/speed-trials-results-2026-11-20.md`. Raw output goes to
`.local/trials/results/` (not committed).

Every tool runs a **separate engine instance**: the Go server `studio/build/engine/studio_server.exe`
without `--dev` (packaged behaviour) and the development workers in `studio/native/build/Release`,
with its own data folder under `.local/trials/instances/` (deleted afterwards), its own
`LOCALAPPDATA`, an HTTP port >= 20000 and a Job object that kills every worker with it. The owner's
installed app and its data folder are never touched.

**One trial at a time.** There is one Lab card per PC, and every Stop (and Use) of an engine
built before change 0222 takes off the card every address that engine does not use itself
(ADR-0127 Release; since ADR-0141 an engine removes only the addresses its own journal added). Two
engines on the card at once (the owner's app, or two agents' trials) remove each other's
addresses: IEDs then fail with "MMS bind failed" or lose their address while running.
`Instance()` therefore refuses to start while another `studio_server.exe` runs or the Lab
card already holds addresses (`instance.busy_reasons()`). When other agents must run at the
same time, pass `--shared` (`Instance(shared=True)`): `step0_run_stop.py` then leaves out every
round in which Windows showed our addresses vanish (`disturbed_rounds`) and keeps refused Runs
with Windows' address changes (`failed_runs`); use medians.

A trial with its own address range, ports or workers sets `SCADABENCH_TRIAL_RANGE` (e.g. `198.18.26`),
`SCADABENCH_TRIAL_PORTS` (e.g. `26000-26999`) and `SCADABENCH_TRIAL_NATIVE` before starting a tool.

Rebuild the engine first when its source changed (from `studio/engine`):
`go build -o ..\build\engine\studio_server.exe ./cmd/studio_server`.

## Commands (from the repo root, any Python 3.12)

| Command | What it measures |
|---|---|
| `python studio/tools/trials/step0_run_stop.py --project small --rounds 10` | Every step of Run and Stop, median and worst (`--project large`, `--goose off`, `--hold`, `--gap`, `--warm`, `--shared`) |
| `python studio/tools/trials/card_watch.py --card Ethernet --seconds 300` | Owner check on a real card: every address change on the card and DNS Client CPU each second, read-only |
| `python studio/tools/trials/dnscache_cpu.py --repeats 3` | Windows DNS Client CPU for 30 s after one address added/removed and after a Run/Stop |
| `python studio/tools/trials/process_start.py --count 8 --repeats 10` | Bare `studio_ied.exe` start, N at once, until its first answer (`--exe` for another build) |
| `python studio/tools/trials/runstop_ab.py --project large --base <exe>=<workers> --trial <exe>=<workers>` | R1/R2/R3/R5: Run and Stop, today against a trial build, alternating sessions |
| `python studio/tools/trials/real_stop_ab.py --case r4\|r6 --base ... --trial ...` | R4/R6: Stop with a real-IED session (a separate `studio_ied`, or a listener that never answers) |
| `python studio/tools/trials/r7_master_connect.py --outstation gateway\|standin --base ... --trial ...` | R7: SCADA master Connect to an IEC 104 outstation with 5000 points (`r7_station.py`: no counter confirmation) |
| `python studio/tools/trials/r3_spare_cost.py --server <exe> --native <workers> --release <workers>` | R3: spares' memory and processor time, licence-check parts |
| `python studio/tools/trials/winaddr.py` | The Lab card's addresses as Windows lists them now |
| `python studio/tools/trials/winlisten.py` | Every TCP listener and its process |
| `python studio/tools/trials/projects.py` | IED and diagram counts of the two trial projects |
| `python studio/tools/trials/n3_n5_lock_races.py --idea n3 --mode add --server base` | N3/N5: a Release taking off an address the same engine is just putting on (gateway own address, gateway Start, SNMP Start) while IED and SNMP Stop loop 200 times (`--server trial`, `--mode start`, `--idea n5`) |
| `python studio/tools/trials/n4_launch_order.py --launches 30 --server base` | N4: launch clean-up against the gateway's and SNMP devices' automatic start (`--exit kill`) |
| `python studio/tools/trials/n9_gateway_address.py --cycles 10 --server base` | N9: is the gateway's own address on the card only while it runs; exit leaves nothing |
| `python studio/tools/trials/n10_session_addresses.py --rounds 10 --server base` | N10: Run/Stop steps and DNS Client CPU, then what stays after Stop, File › Close, exit and a kill + launch |
| `python studio/tools/trials/lab_cross_removal.py --rounds 10 --server base` | Finding: does one copy take another copy's addresses off at Stop, Use, exit, launch |

The last five (group C, change 0222) use `labgroup.py`: ports 24000-24999, addresses
198.18.24.0/24, `--server base` = `.local/trials/c/base/studio_server.exe` (build today's master
there first), `--server trial` = `studio/build/engine/studio_server.exe`. They run beside other
engines (`shared`) and tell by their own lab journal whether *this* engine removed an address.

## Projects

`projects.build("small")` is the 8-IED panels project from `test_cids/substation`;
`projects.build("large")` is the 14-IED 132 kV project from `test_cids/Al Kamil` plus six
IEDs of the 400 kV project (20 IEDs). Local timing only: never put this data in public material.
Every IED gets MMS port 20102 and, through `POST /network/lab/use`, a Lab-card address
`198.18.20.<10+n>`. The Lab card refuses loopback (127.x) addresses, so trials use the
198.18.0.0/15 test range; this harness owns 198.18.20.0/24 (other trials use other /24s).

## Building blocks for a new trial

| Module | Use |
|---|---|
| `instance.py` | `with Instance() as engine:` then `engine.json(method, path, body)`, `engine.upload(...)`, `engine.data` (data folder), `engine.job` |
| `projects.py` | `open_in(engine, build("small"))`, `use_lab_card(engine)` -> `{ied_id: ip}` |
| `sse.py` | `EventRecorder(engine).start()`; `.since(moment, kind=None)` -> engine events with their `timestamp` |
| `watcher.py` | `Watcher(engine.job).start()`; `.mark()` before an action; `.first("address"\|"listen"\|"process", "added"\|"removed")`; `.kernel_times(since=t)`; `.windows_created()` |
| `winaddr.py` | `addresses(prefix, interface)` -> `{ip: Row(state, created, ...)}` from GetUnicastIpAddressTable; `lab_card_index()` |
| `winlisten.py` | `listeners(prefix)` -> `{(ip, port): pid}` from GetExtendedTcpTable |
| `winproc.py` | `Tracker` (kernel creation/exit times), `cpu_seconds(pids)`, `system_busy()`, `service_pid(name)` |
| `winclock.py` | `now()`: the one clock (precise Windows wall time, Unix seconds) every record uses |
| `timeline.py` | `run_steps(...)`, `stop_steps(...)`, `journal(data_dir, "applied"\|"removed", since)` |
| `summary.py` | `describe(rounds, names)` (median, worst, share), `markdown(rows)` |

Windows is the witness: address checks read Windows' own address list (with Windows' creation
time of each address), never the project or the lab journal; the journal only says when the
engine got the service's answer.

## How the Run and Stop steps are cut

Each step ends at an observed moment, so the steps add up to the click-to-answer total.

| Step | Ends when |
|---|---|
| addresses | the lab journal's last `applied` record (the service answered; Windows' creation time is a few ms earlier) |
| listen check + GOOSE bus | the first `studio_ied.exe` is created (kernel time). Run `--goose off` to see the listen check alone |
| process start | the last `studio_ied.exe` is created |
| model load | the last IED listens on its MMS port (Windows' listener table); includes each process's own start-up |
| saved values | the engine's `runtime` `running` event (every IED got its saved positions) |
| first values read | the first `initial_state` signal (the mapped points were read back from the workers) |
| initial states | the first test-mode/behaviour signal, or the last `initial_state` signal |
| test modes | the last `test_mode`/`behaviour` signal (0 when the project has none to write) |
| rest (listeners, reply) | the HTTP answer to `POST /runtime/start` |
| workers stop | the last worker process exits (kernel time) |
| address release | the last address leaves Windows' list (or the journal's last `removed`) |
| stopped shown | the engine's `runtime` `stopped` event |
| reply | the HTTP answer to `POST /runtime/stop` |
