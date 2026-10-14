# File ownership map

Use this map to find the one layer that owns a change. Detailed boundaries are in
[architecture/boundaries.md](../architecture/boundaries.md).

| Concern | Owning files |
| --- | --- |
| Flask composition/security | `studio/backend/studio_app/app_factory.py`, `api/security.py` |
| HTTP request shapes | `studio/backend/studio_app/api/` |
| Pure equipment semantics | `domain/equipment_control.py`, `domain/equipment_state.py` |
| Project format/migration | `projects/document.py`, `archive.py`, `mapping_migration.py` |
| Autosave/recovery/revisions | `projects/service.py`, `startup_recovery.py`, `migration_backup.py` |
| SCL preservation/model | `scl/document.py`, `importer.py`, `model_builder.py`, `type_resolver.py` |
| SCL validation/export | `scl/schema_validation.py`, `exporter.py` |
| Simulation lifecycle | `simulation/coordinator.py` |
| Studio equipment operations | `simulation/equipment_control.py` |
| External-control feedback | `simulation/native_control_feedback.py` |
| Authoritative canvas state | `simulation/equipment_states.py`, `frontend/src/equipment/liveValues.ts` |
| Sources/scenarios/pulses | `simulation/automation.py`, `scenario_runner.py`, `pulses.py` |
| Native worker lifecycle/IPC | `runtime/supervisor.py`, `worker_process.py`, `worker_protocol.py` |
| MMS controls/selection | `native/worker/controls/controls.cpp` |
| Native server/tasks | `native/worker/server/server.cpp` |
| Native typed updates | `native/worker/updates/values.cpp` |
| Runtime evidence | `observability/log_writer.py`, `tools/query_runtime_logs.py` |
| Frontend API/contracts | `frontend/src/api/client.ts`, `api/types.ts` |
| Shared tab/state flow | `frontend/src/state/useStudio.ts` |
| Equipment sheet | `frontend/src/features/diagram/EquipmentSheet.tsx`, `SheetSettings.tsx`, `equipmentFields.ts` |
| Canvas selection/clipboard/shortcuts | `features/diagram/Canvas.tsx`, `DiagramPage.tsx`, `useCanvasShortcuts.ts`, `equipment/clipboard.ts` |
| App shell and places | `frontend/src/app/App.tsx`, `Sidebar.tsx`, `TopBar.tsx`, `places.ts` |
| Value writes and generators | `features/test/signalActions.ts`, `ValueControl.tsx`, `SourceEditor.tsx` |
| Devices | `features/devices/` (list, detail, connection, data points, compare, files) |
| Event explanations | `frontend/src/features/event-log/eventDetails.ts` |
| Engineering history | `tools/history_log.py`, `docs/history/ledger-*.jsonl` |
| Lab network workflow | `networking/service.py`, `address_plan.py`, `change_plan.py`, `journal.py`, `helper/` |
| Adapter listing | `networking/ip_helper.py` (Windows IP Helper API), `networking/adapters.py` (normalise, PowerShell fallback) |
| Network/IED settings, format 11 | `domain/network_settings.py`, `projects/ied_settings.py` |
| SCL communication and GOOSE | `scl/communication.py`, `goose_controls.py`, `goose_links.py` |
| GOOSE publishing (native) | `native/worker/goose/publishing.cpp`, `native/common/npcap.cpp`, `native/cmake/goose_*.cmake` |
| GOOSE observation and LGOS | `native/live/goose_traffic.cpp`, `acquisition/goose_monitor.py`, `domain/goose_supervision.py` |
| Real IED client | `native/live/`, `acquisition/real_session.py`, `live_values.py`, `domain/model_compare.py`, `access_policy.py` |
| Guided UI and wording | `frontend/src/ui/`, `frontend/src/copy/`, `features/network/`, `features/settings/` |
| Feature contracts | `docs/specs/`, `studio/tools/check_specs.py` |
| Repository verification | `studio/tools/verify.py` and `check_*.py` |

Tests mirror these boundaries under `studio/tests/unit`, `integration`, `protocol`,
`frontend`, `packaging` and `performance`. Keep each maintained file below 300
physical lines and split by responsibility before growth forces a catch-all module.
