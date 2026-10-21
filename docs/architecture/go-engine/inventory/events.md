# Event bus inventory (SSE `/api/v1/events` and run log)

Producer paths are relative to `studio/backend/studio_app/`. Frontend consumption details are
in frontend-events.md (FE = `studio/frontend/src`).

## Envelope and the two publish paths
- `EventBus.publish(kind, **payload)` (runtime/event_bus.py:13) builds
  `{"id": seq, "kind": kind, "timestamp": time.time(), **payload}`. A payload key named
  `id`/`kind`/`timestamp` would overwrite the envelope; native events are renamed first.
- `Coordinator.emit` (simulation/coordinator.py:61) = publish + `RunLog.write(event)` when a
  run log is open (from Run start until stop). So **run-log content = coordinator.emit events
  during a run**. Gateway/master/goose/live/simulation all use `coordinator.emit`.
- Direct `bus.publish` (never in the run log): `project_changed` (app_factory.py:48),
  `mms_access` (simulation/mms_access.py:28,49), `explorer_connected`/`explorer_command`
  (ExplorerService gets `bus.publish`, app_factory.py:61).
- SSE JSON uses insertion order (http-conventions.md section 3). Native payloads arrive with
  keys in **alphabetical** order (nlohmann::json = std::map), and Python keeps that order.
- IMPORTANT correction for the frontend-callers notes: HTTP `jsonify` responses are already
  key-sorted (Flask `sort_keys=True`), so FE-visible order of objects in REST responses is
  alphabetical today. Insertion order matters only for SSE events and files.

## Native worker events forwarded by `Coordinator.on_worker_event` (coordinator.py:67-90)
Every frame with an `event` key from a `studio_ied.exe` worker (and its stderr lines) becomes
`emit(<event>, ied_id=<id>, **payload)` where payload = frame minus `event`, with renames
`kind->native_kind`, `id->native_id`, `timestamp->source_timestamp`. Exceptions:
- `mms_access`: not emitted by coordinator; merged and re-published as
  `mms_access {ied_id, clients:[{first_seen,ip,last_read,own,peer,reads,rcbs,total_reads,writes,...}]}`
  (domain/mms_access.client_view), bus only.
- `control_applied`: adds `external` (bool, mms_access.control), cancels a pulse, updates the
  index value/timestamp, adds `updates:{timestamp_ref: ts}`, then enqueues native feedback.
- `worker_failed`: cancels pulses, sets IED `state="failed"`, emitted as `worker_failed {ied_id,error}`.

| Native kind (studio_ied) | Fields after rename (native file) |
| --- | --- |
| `control_requested` | ref, value, trigger "Oper", peer (worker/controls/controls.cpp:28) |
| `control_applied` | action, cdc, changed, ctl_value, peer, ref, source_timestamp, status_ref, trigger, value (+external, +updates) (controls.cpp:57) |
| `control_selection` | model, ref, sbo_timeout_ms, selected, state (controls.cpp:67) |
| `control_test` | ref (controls.cpp:83) |
| `connection` | connected, peer (worker/server/server.cpp:19) |
| `report_control` | error, native_kind (int), name, parameter, peer (server.cpp:27) |
| `goose_control` | enabled, gocb (worker/goose/publishing.cpp:13) |
| `goose_publishing` | state "starting"/"active" (publishing.cpp:46-48) |
| `native_diagnostic` | message (each stderr line, runtime/worker_process.py:124-126) |
| `worker_failed` | error (reader EOF/OSError/ValueError text or null) (worker_process.py:115) |

## Backend-originated kinds

| Kind | Payload fields (order) | Producer file:line | FE consumer |
| --- | --- | --- | --- |
| `project_changed` | revision, source (tab_id, "gateway" or null) | app_factory.py:48 via ProjectService.changed | useStudio.ts reload / revision echo |
| `runtime` | state, running (sorted ids) / state, message / state | coordinator.py:138,152,173,188,249; run_selection.py:70,84 | useStudio.ts (refresh; clear on stopped) |
| `ied_runtime` | ied_id, state "started"/"stopped", role | run_selection.py:69,83 | Activity |
| `signal` | ied_id, ref, value, source, quality, source_timestamp, updates | simulation/sample_writer.py:55 | useStudio values, SLD pulses |
| `signal` (initial) | ied_id, ref, value, quality, source_timestamp, source "initial_native_snapshot", updates | simulation/bindings.py:22 | same |
| `signal` (live) | ied_id, ref, value, source live_report/live_poll/live_goose, quality, source_timestamp, updates, live_state "fresh" | acquisition/live_values.py:43 | same |
| `live_read_error` | ied_id, ref, message | live_values.py:46; real_session.py:128 ("The device could not read this point (error N)") | Activity |
| `live_stale` | ied_id, message ("Connection to the device was lost") | live_values.py:67 | Activity |
| `live_status` | ied_id, state (starting/no_address/connecting/discovering/live/retrying/stopped), message | acquisition/real_session.py:35 | useStudio refresh |
| `source` | ied_id, ref, spec (summary without csv_text/points) | coordinator.py:225 | Activity |
| `source_resumed` | ied_id, ref | coordinator.py:213 | Activity |
| `source_failed` | ied_id, ref, message | simulation/automation.py:46 | Activity |
| `automation_late` | lateness_seconds, skipped_ticks | coordinator.py:242 (lateness >= 0.1 s) | Activity |
| `automation_failed` | message | coordinator.py:248 (then `runtime` paused "Automation failed; MMS remains active") | Activity |
| `pulse_started` | correlation_id, ied_id, ref, duration_ms | simulation/pulses.py:38 | Activity |
| `pulse_cancelled` | correlation_id, ied_id, ref, message ("Simulation stopped", "Replaced by a newer pulse", "Newer signal write", "New MMS control feedback", "IED worker failed", "IED stopped", "Automatic source resumed") | pulses.py:46 | Activity |
| `pulse_completed` / `pulse_failed` | correlation_id, ied_id, ref, message ("Previous value restored" / error) | pulses.py:62 | Activity |
| `scenario_started` | scenario_id, name | simulation/scenario_runner.py:29 | Activity |
| `scenario_cancelled` | scenario_id | scenario_runner.py:36 | Activity |
| `scenario_step_applied` | scenario_id, step_id, ied_id, ref, action, value, scheduled_seconds, observed_seconds | scenario_runner.py:57 | Activity |
| `scenario_completed` | scenario_id | scenario_runner.py:63 | Activity |
| `scenario_failed` | scenario_id, step_id, message ("Assertion failed: expected X, observed Y") | scenario_runner.py:66 | Activity |
| `equipment_state` | node_id (node or `node:status_id`), value | simulation/equipment_states.py:38 | useStudio equipment map |
| `energization` | nodes, edges, hints | simulation/energization_states.py:34 (only on change or forced) | useStudio energization |
| `interlock_hint_observed` | node_id, action, origin, code, message | energization_states.py:48 | Activity |
| `equipment_feedback_applied` | node_id, control_ref, target_count, changed_count, action, wire_value, origin "studio_api" | simulation/equipment_control.py:77 | Activity |
| `equipment_feedback_failed` | node_id, control_ref, message / ied_id, control_ref, message ("Native control event has no Boolean ctlVal") / node_id, control_ref, origin, message | equipment_control.py:73; native_control_feedback.py:37,94 | Activity |
| `equipment_feedback_observed` | node_id, control_ref, target_count, changed_count, action, wire_value, origin "native_control" | native_control_feedback.py:89 | Activity |
| `equipment_button_applied` | node_id, action "press", value | equipment_control.py:105,127 | Activity |
| `equipment_test` | node_id, action | simulation/equipment_test.py:40 | Activity |
| `ieds_removed` | ied_ids, count | api/ieds.py:39 | Activity |
| `goose_status` | goose_rx frame minus event/values: conf_rev, dataset, frames, go_id, gocb_ref, key, origin, problem, source_timestamp, sq_num, src_mac, st_num, state, tal_ms, test (timeout frames: frames, gocb_ref, key, problem, sq_num, st_num, state) | acquisition/goose_monitor.py:58 | SLD GOOSE layer |
| `goose_unknown` | gocb_ref, app_id | goose_monitor.py:72 | Activity |
| `goose_listener_failed` | message | goose_monitor.py:78; acquisition/manager.py:58; goose_engine.py:125 | Activity |
| `goose_supervision_failed` | ied_id, message | goose_monitor.py:90 | Activity |
| `goose_input` | ied_id, gocb_ref, st_num, applied[refs], problems[<=10] | acquisition/goose_inputs.py:71 | Activity |
| `goose_input_refused` | ied_id, gocb_ref, reason | goose_inputs.py:60 | Activity |
| `goose_conflict` | ied_id, gocb_ref, src_mac, sharing | acquisition/goose_engine.py:111 | GooseConflicts.tsx (count) |
| `goose_share` | state off/checking/sharing | goose_engine.py:115 | Activity |
| `goose_uplink` | state, message | goose_engine.py:119 | Activity |
| `goose_storm` | ied_id, limit_per_second | goose_engine.py:121 | Activity |
| `sv_listener_failed` | message | acquisition/manager.py:63 | Activity |
| `sv_status` | key, state, problems (only when verdict changed) | acquisition/sv_monitor.py:59 | Activity |
| `gateway_started` | protocol, port, points | gateway/manager.py:112 | Activity |
| `gateway_stopped` | (none) | gateway/manager.py:177 | Activity |
| `gateway_command` | point, result "relayed", ok, reason / point, result "refused", reason / point, ied, role, ok, result (summary) | gateway/commands.py:55,61,68 | Activity |
| `master_connected` | protocol, host, port | gateway/master_manager.py:85 | Activity |
| `master_disconnected` | (none) | master_manager.py:225 | Activity |
| `explorer_connected` | host, port | tools/explorer.py:96 (bus only) | Activity |
| `explorer_command` | host, ref, value, ok, result | tools/explorer.py:234 (bus only) | Activity |
| `resync` | (envelope only, not stored) | runtime/event_bus.py:26 | useStudio refresh |

Not forwarded to the bus (consumed internally): studio_live `live_report`, `live_connection`,
`goose_rx` (becomes goose_status), `goose_stats`, `goose_unknown_cap`, `sv_rx`, `sv_unknown*`;
studio_gateway `gw_traffic`, `gw_client`, `gw_command`; studio_master `master_values`,
`master_link`, `master_probe`, `master_interrogation`, `gw_traffic`; studio_goose events are
re-emitted with selected keys only; stderr `native_diagnostic` of non-IED processes is dropped.

## Ordering and timing semantics to keep
- Emission is synchronous on the producing thread; the bus sequence is the global order.
- `signal` events come one per written signal (with companions inside `updates`), after the
  native batch acknowledges; `equipment_state` follows (only for changed values unless forced),
  then at most one `energization` per publish cycle.
- Run start order: `runtime{state:running,running}` -> initial `signal`s -> `equipment_state`
  (all, forced) -> `energization` (forced) -> initial-state/test-mode writes -> live/GOOSE start.
- Stop (coordinator.py:158-176): `pulse_cancelled`* -> live stop (`live_status stopped` per
  real IED) -> `scenario_cancelled`* -> workers closed -> `runtime{state:stopped}`; the run log
  is closed after that final `runtime` event. State is `stopping` in between.
- `mms_access` arrives about once per second per simulated IED while clients exist.
- Pulses checked every 20 ms; automation cadence 0.1 s with backlog 5.

## Parity risks
- Key renames and passthrough of arbitrary native keys (future native fields appear automatically).
- Which events reach the run log (only coordinator.emit while a run is open).
- Alphabetical native key order vs insertion order of backend payloads in the SSE JSON.
- Event text (`message`) is shown verbatim in Activity; FE picks `message||ref||status_ref||state`.
