# API routes 1/3: session, project, runtime, SCL/IEDs, signals, equipment, tests

All paths are under `/api/v1`. Handler files are in `studio/backend/studio_app/api/`.
Columns: **H** = handler file:line; **Svc** = service it delegates to; **Req** = request;
**Resp** = response keys (jsonify => keys sorted); **Err** = statuses and message sources;
**SE** = side effects (save = atomic `autosave.subsim` write; changed = `revision+1` and
`project_changed` event; emit = bus event; W = native worker IPC); **FE** = frontend caller
(`studio/frontend/src/...`), poll interval if any. "stopped-only" = `require_stopped()`
(api/context.py:14) -> 400 StudioError "Stop the simulation before editing its structure".
Generic errors (KeyError 400 `'key'`, 415, 500) follow http-conventions.md section 4.

## Session and project

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| POST /session | security.py:10 | - | JSON `{token}` (silent parse) | `{"ok":true}` | 401 "Invalid launch session. Reopen the app from its launcher." (non-str or wrong token) | sets cookie | api/client.ts:33 `connectSession` (token from `#session=`) |
| GET /project | projects.py:9 | ProjectService.snapshot (projects/service.py:117) | - | snapshot keys: `diagram, id, ieds[], name, network, revision, scenarios, session_id, sources[], test_panel` (+`migration_notice` when set) + `runtime` (state), `running_ieds` (sorted ids), `startup_recovery` (recovery result or null) | - | none | state/useStudio.ts (initial load and after `project_changed`/`runtime`/`resync`) |
| PUT /project/diagram | projects.py:16 | ProjectService.diagram -> validation.validate_diagram (pydantic Diagram) | `{diagram, revision:int, tab_id:str 1..64}` exactly these 3 keys | `{"ok":true,"revision":n}` | 400 "Diagram update requires its project revision and tab identity" / "Invalid project revision or tab identity"; 409 code `conflict` "Another tab changed this project. This tab must synchronize before saving."; pydantic 400 details; StudioErrors in projects/validation.py | stopped-only; save; changed(source=tab_id) | state/useStudio.ts |
| POST /project/save | projects.py:29 | ProjectService.save | none | `{"ok":true}` | - | save (no changed) | state/useStudio.ts |
| GET /project/download | projects.py:35 | ProjectService.export -> archive.encode | - | ZIP bytes, `substation.subsim` | - | none | app/TopBar.tsx (plain link `/api/v1/project/download`) |
| POST /project/open | projects.py:41 | ProjectService.restore | multipart `file` | snapshot (no runtime keys) | stopped-only; 400 StudioErrors from restore/migration/validation (projects/service.py:175-251, archive.decode); missing file -> 400 generic | save; changed; maybe `recovery/` migration backup | state/useStudio.ts |

Snapshot `ieds[]` = every IED field except `source`, `cfg`, `signals`, plus `signal_count`.
Runtime-only fields (`state`, `goose_state`) live on the IED dict and appear here.

## Runtime

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| POST /runtime/{start,stop,pause} | runtime.py:8 | Coordinator.start/stop/pause (simulation/coordinator.py:92-188) | none | `{"running":[sorted ids],"state":...}` | 404 `{"error":"Unknown runtime action"}` for other action; 400 ValueErrors: "Stop the current simulation first", "Import or create an IED before Run", "Set at least one IED to 'Be this IED' or 'Read real IED' before Run", endpoint conflict text (projects/ied_settings.py:25), "Start the simulation before pausing automation", live-client missing text; 503 native start errors / capability (runtime/worker_settings.py) | spawns studio_ied workers, studio_goose, studio_live; RunLog; emits `runtime`, `signal`... | state/useStudio.ts `/runtime/${action}`; features/devices/DeviceRunSummary.tsx |
| POST /runtime/ieds/{id}/{start,stop} | runtime.py:17 | Coordinator.start([id]) / stop([id]) -> run_selection.add_ieds/remove_ieds | none | same as above | 404 unknown action; 400 "Unknown IED", "<name> is left out. Choose 'Be this IED' or 'Read real IED' first.", "This IED is already running", "This IED is not running" | spawns/stops one worker/session; emits `ied_runtime`, `runtime` | features/devices/RunThisIed.tsx |

Start semantics (must match): start with no ids while running adds every runnable IED not
yet active (change 0106); stopping the last selected IED stops the whole run; a failed
start rolls back and emits `runtime` `{state:"stopped", message}` then re-raises.

## SCL import, templates, IED files and removal

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| POST /scl/validate | scl.py:14 | scl/schema_validation.validate_schema (lxml XSD) | multipart `file` | `{"errors":[{"line","message"}<=100],"schema":ver|null,"status":"valid"|"invalid"|"unsupported"}` | 400 "SCL exceeds the 20 MB import limit", "Invalid or unsafe XML: <expat msg>", "Expected an SCL root element", "Duplicate IED name in SCL" | none | not called by the frontend (tests/tools only) |
| POST /scl/preview | scl.py:19 | scl/document.preview_ieds | multipart `file` | `{"available_slots":max(0,20-n),"ieds":[{description,logical_devices,manufacturer,name,type,unavailable_reason}]}`; existing names get "Already in this project" | stopped-only; "No IEDs found in SCL"; parse errors above | none | state/useStudio.ts |
| POST /scl/import | scl.py:31 | ProjectService.add_scl -> scl/importer.import_scl | multipart `file` + form `selected_names` (JSON array) | `{"ieds":[new ids]}` | stopped-only; "Preview the file and choose which IEDs to import first"; "Select between 1 and 20 IEDs"; json.loads errors -> 400 (JSONDecodeError is ValueError: "Expecting value: line 1 column 1 (char 0)"); importer StudioErrors; "An IED with this name already exists; rename before importing"; "This release supports up to 20 IEDs per project" | save; changed; loopback IPs assigned 127.50.0.x | state/useStudio.ts |
| POST /ieds/template | scl.py:44 | ProjectService.add_template -> scl/templates.make_template | JSON `{name}` (default "BayController"; non-silent get_json) | `{"ieds":[id]}` | stopped-only; 415 without JSON; identifier errors | save; changed | state/useStudio.ts |
| GET /ieds/{id}/source | scl.py:52 | find_ied | - | original XML bytes (UTF-8 re-encode of stored text), filename = original | 400 code `not_found` "Unknown IED" | none | features/devices/DeviceFiles.tsx |
| POST /ieds/{id}/export-scl | scl.py:59 | scl/exporter.export_ied (lxml) | `{overrides:{ref:value}}` exactly | edited XML `<name>.edited.scd` | "Provide an overrides object; engineering addresses remain unchanged"; exporter StudioErrors (exporter.py) | none (project unchanged) | features/devices/ExportScl.tsx |
| GET /ieds/{id}/removal-preview | ieds.py:9 | ProjectService.removal_preview | - | `{bindings,id,name,scenario_steps,scenarios_removed,sources,test_controls}` | stopped-only; not_found | none | (not used by FE; DeleteIedsDialog uses bulk) |
| DELETE /ieds/{id} | ieds.py:15 | ProjectService.remove_ied -> bulk_removal | JSON `{confirm:true}` | preview keys + `recovery_file` (absolute path) | 400 `{"error":"Confirm deletion after reviewing affected bindings and automation"}` (no code); stopped-only | backup `recovery/before-ied-delete-<hex>.subsim`; save; changed | not used by FE (launcher/self_test.py uses it) |
| POST /ieds/removal-preview | ieds.py:24 | bulk_removal.preview | `{ids:[1..20 unique str]}` | `{bindings,fingerprint(sha256),ieds:[{id,name}] sorted,scenario_steps,scenarios_removed,sources,test_controls}` | "Select 1 to 20 unique IEDs"; not_found | none | DeleteIedsDialog.tsx |
| POST /ieds/remove | ieds.py:31 | bulk_removal.remove | `{ids,fingerprint,confirm:true}` | preview keys + `recovery_file` | 400 "Confirm the deletion preview first"; "Project changed. Review a fresh deletion preview"; "Recovery directory must not be a symbolic link"; "Recovery backup verification failed; nothing deleted" | backup; save; changed; emit `ieds_removed {ied_ids,count}` | DeleteIedsDialog.tsx |

Runtime handlers for delete/bulk take `runtime().cycle_lock` (serialised with Run/Stop).

## Signals, controls, equipment

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /ieds/{id}/signals | signals.py:10 | in-handler filter over `ied.signals` | query `q` (casefold substring of `ref+" "+description`), `fc`, `kind` (comma list), `writable=true` (writable and fc!=CF), `report` (members), `ref` (exact), `ld`, `bad_quality=true`, `in_dataset=true`, `offset` (>=0), `limit` (1..250, default 100) | `{"devices":[sorted LDs],"items":[signal + quality + timestamp],"total":n}` | not_found; bad int -> 400 int() text | none | many: DeviceSignals.tsx, LiveValueTable.tsx, SignalPicker.tsx, SignalSearchPicker.tsx, SourceEditor.tsx (`fc=MX&limit=200`), SequenceEditor.tsx, StateMappingEditor.tsx, RelayStatusEditor.tsx, ExportScl.tsx |
| POST /signals/update | signals.py:45 | Coordinator.update -> SampleWriter.apply (source "manual") | `{ied_id,ref,value,quality?}` | `{"value": actual}` | "IED is not running", "Run the IED before updating a live value", "This attribute is not runtime-editable", "Sample batch exceeds the native 256-attribute limit", domain/signal.validate_value texts; 503 native | W update_batch; emits `signal`; may emit `pulse_cancelled`, `equipment_state`, `energization` | features/test/signalActions.ts |
| POST /signals/source | signals.py:51 | Coordinator.configure_source | `{ied_id,ref,spec}` | `{"ok":true}` | "Automatic sources require numeric MX measurements", "Unsupported source mode", "Source parameters must be finite numbers", "Source period must be at least 0.1 seconds", csv_trace texts, "Unknown signal reference" | save; changed; emit `source {ied_id,ref,spec}` | signalActions.ts |
| POST /signals/pulse | test_panel.py:30 | Pulses.start (simulation/pulses.py:39) | `{ied_id,ref,active_value=true,duration_ms=1000}` | `{"id":hex,"value":bool}` | "Pulse requires a Boolean value and 100 to 60000 milliseconds", "Run the IED before triggering a pulse", "Pulses can only be sent to simulated IEDs", "Pulses require a Boolean signal" | W read_versioned + update_guarded; emits `pulse_started`, later `pulse_completed`/`pulse_failed` (20 ms timer) | signalActions.ts |
| POST /signals/release-override | test_panel.py:37 | Coordinator.release_override | `{ied_id,ref}` | `{"ok":true}` | "Run the IED before resuming a source", "This signal has no automatic source" | emit `source_resumed` | signalActions.ts |
| POST /controls/operate | signals.py:58 | runtime/mms_client.operate (subprocess studio_probe.exe) | `{ied_id,ref,value:bool}` | parsed stdout JSON of studio_probe | "Control value must be boolean", "Unknown control object", "IED must be running to operate equipment"; 503 "Bundled MMS client is unavailable", "MMS control failed or command termination was not received from <host>:<port>" | marks own command (20 s); external MMS control | not called by FE |
| GET /equipment/states | equipment.py:9 | EquipmentStates.snapshot + EnergizationStates.snapshot | - | `{"energization":{edges,hints,nodes},"states":[{node_id,value}]}` (empty list when stopped) | - | none | state/useStudio.ts (on run start / resync) |
| POST /equipment/{node}/operate | equipment.py:14 | EquipmentControl.operate (simulation/equipment_control.py:52) | `{action}` or `{value}` | `{action,command,feedback[],node_id,wire_value}` or push-button `{action:"press",feedback,node_id,wire_value:null}` | "Run the IEDs before operating equipment", domain/equipment_control texts (mixed action/value refused), "Equipment has no command mapping", command_refusal, "Equipment command mapping is incompatible", "Mapped equipment state point cannot be updated while running"; 503 "MMS command was applied, but simulated equipment feedback failed" | studio_probe subprocess; feedback writes; save (node initial_position, no changed); emits `equipment_feedback_applied`/`_failed`, `equipment_button_applied`, `equipment_state` | features/diagram/DiagramPage.tsx |
| POST /equipment/{node}/test | equipment.py:23 | simulation/equipment_test.apply_equipment_test | `{action, value?, row?}` | `{action,node_id,written:{ied_id:{ref:value}}}` | StudioError 400: "Unknown equipment", "Run the IED before using a test button", REAL_ONLY text, "<name> is not running. Run this IED first.", quality text; domain/equipment_test texts | sample writes (source equipment_test); emit `equipment_test` | DiagramPage.tsx (TestMenu) |

## Scenarios, test panel, traces, test mode

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| PUT /scenarios | scenarios.py:8 | ProjectService.scenarios -> domain/scenario.validate_scenarios | JSON array | `{"ok":true}` | stopped-only; "Expected at most 100 scenarios", name 1..100, steps 1..1000, step time 0..86400 ... | save, changed, save again | features/test/SequenceEditor.tsx |
| GET /scenarios/runs | scenarios.py:15 | ScenarioRunner.snapshot | - | top-level array `[{completed_steps,id,state}]` | - | none | SequenceEditor.tsx (polled 1000 ms) |
| POST /scenarios/{id}/start | scenarios.py:20 | Coordinator.start_scenario | - | `{"ok":true}` | "Run the IEDs before starting a scenario", "Unknown scenario", "This scenario is already running" | emit `scenario_started` | SequenceEditor.tsx |
| GET /project/test-panel | test_panel.py:16 | Coordinator.signal + sources | - | `{"items":[entry + automatic + overridden + signal]}` | entry signal lookup errors (400) | none | features/test/TestPoints.tsx (1000 ms while running), features/devices/DeviceSignals.tsx |
| PUT /project/test-panel | test_panel.py:8 | projects/test_panel.validate_panel | JSON array (<=500) | `{"ok":true}` | stopped-only; texts in projects/test_panel.py:16-40 | save; changed (rolled back on save failure) | TestPoints.tsx |
| POST /traces/preview | traces.py:8 | domain/csv_trace.read_rows | multipart `file` (reads 5 MiB+1) | `{"columns":[...],"row_count":n,"rows":[first 5]}` | "CSV trace exceeds 5 MB"; UnicodeDecodeError (ValueError) text; "CSV needs 2 to 256 named columns", "CSV column names must be unique", "CSV row N has the wrong number of columns", "CSV exceeds 100,000 data rows" | none | features/test/CsvPlayback.tsx |
| PUT /ieds/{id}/test-mode | test_mode.py:9 | simulation/test_mode_apply.set_test_mode | JSON (silent) | `{"test_mode":{...},"written":n}` | StudioError "Mode and simulation can only be set on IEDs that ScadaBench simulates. ScadaBench never changes a real device."; "<name>: <domain/test_mode error>" | save; changed; worker writes when running (source test_mode) | features/devices/TestModeCard.tsx |

## Parity risks
- `/project` snapshot is the biggest payload; nested dicts come from the importer/model and
  pydantic `model_dump` (field order irrelevant due to sorting, but values/float forms matter).
- Pydantic lax coercion and its error `details` (type/loc/msg/url) on `PUT /project/diagram`.
- `/scl/validate` and import depend on lxml (libxml2) XSD validation texts and line numbers.
- `/signals` search uses Python `casefold`; paging and `devices` sorting are byte-visible.
- DELETE `/ieds/{id}` returns 400 without `code`, while StudioErrors include `code`/`details`.
