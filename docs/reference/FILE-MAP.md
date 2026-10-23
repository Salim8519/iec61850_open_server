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
| Simulation lifecycle | `simulation/coordinator.py`, `simulation/run_selection.py` (run/stop single IEDs, ADR-0067) |
| Studio equipment operations | `simulation/equipment_control.py` |
| Equipment test buttons (ADR-0067) | `domain/equipment_test.py`, `simulation/equipment_test.py`, `equipment/testButtons.ts`, `features/diagram/EquipmentTest.tsx`, `TestMenu.tsx` |
| Run one IED from Devices | `features/devices/RunThisIed.tsx`, `copy/running.ts` (`RUN_ONE_COPY`, `runStateWords`) |
| External-control feedback | `simulation/native_control_feedback.py` |
| Authoritative canvas state | `simulation/equipment_states.py`, `frontend/src/equipment/liveValues.ts` |
| Sources/scenarios/pulses | `simulation/automation.py`, `scenario_runner.py`, `pulses.py` |
| Native worker lifecycle/IPC | `runtime/supervisor.py`, `worker_process.py`, `worker_protocol.py` |
| MMS controls/selection | `native/worker/controls/controls.cpp` |
| Native server/tasks | `native/worker/server/server.cpp` |
| Native typed updates | `native/worker/updates/values.cpp` |
| Runtime evidence | `observability/log_writer.py`, `tools/query_runtime_logs.py`; Go: `engine/internal/runtime/runlog.go`, `runlog_summary.go` (generator `signal_summary`, ADR-0115) |
| Simulated IED Beh, batches, start times (ADR-0115) | Go: `engine/internal/runtime/behaviour.go` (Beh from Mod), `test_modes.go`, `sample_batches.go` (native batch limit), `start_times.go`, `signal/sample_rules.go` (Enum, VisString, quality checks); native: `worker/server/server.cpp` (start times, TimeQuality, idle RCB events), `worker/updates/values.cpp`, `control/canvas_command.cpp` (origin) |
| Frontend API/contracts | `frontend/src/api/client.ts`, `api/types.ts` |
| Shared tab/state flow | `frontend/src/state/useStudio.ts` |
| Equipment sheet | `frontend/src/features/diagram/EquipmentSheet.tsx`, `SheetSettings.tsx`, `equipmentFields.ts` |
| Live colouring / switching hints | `domain/energization.py`, `domain/interlock_hints.py`, `simulation/energization_states.py`, `equipment/energization.ts`, `features/diagram/EnergizationLegend.tsx`, `SwitchHints.tsx` |
| Switching symbols / distance readability | `equipment/symbols/` (`EquipmentSymbol.tsx`, `SwitchingGlyphs.tsx`), `equipment/readability.ts`, `features/diagram/useZoomTier.ts` |
| Bay templates | `equipment/bayTemplates.ts`, `features/diagram/AddMenu.tsx` |
| SLD tidy up (layout) | `equipment/tidy/` (graph, rows, position, wires, tidy), `features/diagram/TidyButton.tsx`, `tidyCopy.ts` |
| SLD point panel (ADR-0102) | `equipment/pointPanel.ts`, `pointDisplay.ts`, `pointFilter.ts`, `pointRequests.ts`, `PointPanelCard.tsx`, `features/diagram/points/`, `copy/pointPanel.ts`; Go: `engine/internal/project/contract/point_panel.go`, `project/validation_point_panel.go`, `acquisition/watched_point_panel.go`, `runtime/point_panel*.go` |
| SLD tap-changer transformer (ADR-0109) | `equipment/tapChanger.ts`, `tapCommands.ts` (the one command seam), `tapContext.ts`, `TapTransformerBody.tsx`, `tap-transformer.css`, `features/diagram/tap/`, `copy/tapChanger.ts`; Go: `project/contract/tap_changer.go`, `project/validation_tap_*.go`, `acquisition/watched_tap_changer.go`, `runtime/tapchanger/` (pure: ratio, regulator, routing), `runtime/tap_changer*.go`, `runtime/plant/energize_transformer.go`, `runtime/typed_command.go` (stub until ADR-0104) |
| Select before operate, typed commands (ADR-0108) | Go: `engine/internal/runtime/typed_command.go` (ADR-0104 adapters until merge), `typed_controls.go`, `typed_values.go`, `sbo_selections.go`, `equipment_select.go`, `command_options.go`, `point_panel_select.go`, `api/runtimeapi/select_routes.go`, `api/runtimeapi/typed_routes.go` (ADR-0104 adapter), `scl/controls_extended*.go` (ADR-0104 reader, unchanged copy), `project/contract/command_options.go`, `project/validation_command_options.go`; browser: `equipment/selectOperate.ts`, `commandOptions.ts`, `typedCommand.ts`, `features/diagram/sbo/` (`useSelectOperate`, `SboCountdown`, `StepButtons`), `SwitchOperate.tsx`, `CommandOptionsPanel.tsx`, `points/PointCommand.tsx`, `points/useTypedControls.ts`, `copy/selectOperate.ts` |
| Canvas selection/clipboard/shortcuts | `features/diagram/Canvas.tsx`, `DiagramPage.tsx`, `useCanvasShortcuts.ts`, `equipment/clipboard.ts` |
| App shell and places | `frontend/src/app/App.tsx`, `Sidebar.tsx`, `TopBar.tsx`, `places.ts`, `useExitConfirm.ts` |
| Tools place | `frontend/src/features/tools/ToolsPage.tsx`, `TrafficTool.tsx`, `copy/tools.ts` |
| IED Explorer / Find IEDs | `backend/studio_app/tools/` (explorer, explorer_auto, ied_scan, control_client, tool_memory), `domain/explorer_*.py`, `scan_range.py`, `native/live/explore.cpp`, `identify.cpp`, `native/control/`, `features/tools/explorer/`, `find/` |
| Gateway simulator | `backend/studio_app/gateway/`, `domain/gateway_*.py`, `api/gateway.py`, `native/gateway/`, `cmake/gateway_support.cmake`, `tools/fetch_gateway_deps.py`, `features/tools/gateway/`, `copy/gateway.ts` |
| Gateway Add points (ADR-0105, Go only) | `engine/internal/gateway/addpoints/` (`catalog.go` the one type list per protocol, candidates, plan, rows/table, own range, smart Propose, xlsx template/read), `gateway/addresses.go`, `api/toolsapi/add_points.go`, `smart_propose.go`, `frontend/src/api/gatewayAdd.ts`, `copy/gatewayAdd.ts`, `features/tools/gateway/add/` |
| SCADA master / converter | `native/master/`, `gateway/master_manager.py`, `modbus_probe.py`, `converter.py`, `domain/master_rules.py`, `master_points.py`, `passthrough.py`, `api/gateway_master.py`, `features/tools/gateway/` (`ScadaTool`, `RoleSwitch`, `master/`, `converter/`), `copy/master.ts`, `converter.ts` |
| SCADA master point list (ADR-0110, Go only) | `engine/internal/gateway/masterpoints/` (contract, list spaces, plan, range, rows, xlsx, csv, compare, received, command, ranges, copy, iedcheck), `addpoints/catalog_master.go`, `gateway/service/master_list.go`, `mapped.go`, `project/gateway/master_points.go`, `api/toolsapi/master_list.go`, `master_list_files.go`, `frontend/src/api/masterList.ts`, `copy/masterList.ts`, `features/tools/gateway/master/list/` |
| Value writes and generators | `features/test/signalActions.ts`, `ValueControl.tsx`, `SourceEditor.tsx` |
| Devices | `features/devices/` (list, `DeviceListRow` with a hints slot, detail, connection, data points, compare, files, `deviceFilters.ts`) |
| Fill-window layout and list filters (ADR-0064) | `styles/fill.css`, `ui/FilterBar.tsx`, `ui/filterState.ts`, `ui/HelpTip.tsx`, `ui/TabBar.tsx`, `state/useListFilter.ts`, `components/CompactNotice.tsx`, each list's `*Filters.ts`, `projects/list_filters.py` |
| Event explanations | `frontend/src/features/event-log/eventDetails.ts` |
| Engineering history | `tools/history_log.py`, `docs/history/ledger-*.jsonl` |
| Lab network workflow | `networking/service.py`, `address_plan.py`, `change_plan.py`, `journal.py`, `helper/` |
| Gateway firewall (ADR-0062) | `networking/gateway_firewall.py` (preview/apply), `networking/firewall_rules.py` (read-only checks), `features/tools/gateway/GatewayFirewall.tsx` |
| Adapter listing | `networking/ip_helper.py` (Windows IP Helper API), `networking/adapters.py` (normalise, PowerShell fallback) |
| Network/IED settings, format 11 | `domain/network_settings.py`, `projects/ied_settings.py` |
| SCL communication and GOOSE | `scl/communication.py`, `goose_controls.py`, `goose_links.py` |
| GOOSE publishing (native) | `native/worker/goose/publishing.cpp`, `native/hal/` (Ethernet dispatch: `udp:` → engine, numbers → Npcap), `native/common/npcap.cpp`, `native/cmake/goose_*.cmake` |
| GOOSE engine (ADR-0073, Go) | `goose_engine/` (`internal/frame`, `guard`, `bus`, `uplink`, `npcap`, `engine`, `ipc`, `cmd/studio_goose`), `tools/build_goose_engine.py`, `acquisition/goose_engine.py`, `goose_share.py`, `domain/goose_reach.py`, `projects/goose_reach.py`, `api/goose_share.py`, `features/network/GooseReach.tsx`, `GooseShareWarning.tsx`, `GooseConflicts.tsx`, `copy/gooseReach.ts`; tests `protocol/goose_engine_rig.py`, `engine_wire_rig.py`, `npcap_peer.py` |
| GOOSE observation and LGOS | `native/live/goose_traffic.cpp`, `goose_inbox.cpp`, `acquisition/goose_monitor.py`, `domain/goose_supervision.py` |
| GOOSE link map (ADR-0065) | `scl/goose_vendor.py`, `goose_inventory.py`, `goose_remote.py`, `goose_sources.py`, `domain/goose_link_map.py`, `goose_link_live.py`, `projects/goose_links.py` |
| GOOSE inputs (ADR-0066) | `domain/goose_inputs.py`, `acquisition/goose_inputs.py` |
| MMS client access (ADR-0066) | `native/worker/server/access_log.*`, `read_hook.*`, `domain/mms_access.py`, `simulation/mms_access.py` |
| SLD signal layers | `features/diagram/signals/`, `equipment/signalLayout.ts`, `signalPulses.ts`, `signalLayerStore.ts`, `copy/signalLayers.ts`, `api/signalLayers.ts`, `signals/OutsideSenders.tsx`, `features/devices/GooseHint.tsx`, `GoosePartners.tsx`, `ReadByClients.tsx`, `useGooseLinks.ts` |
| GOOSE IED components (ADR-0112) | browser: `equipment/IedCard.tsx`, `ied-card.css`, `iedContext.ts`, `iedStatus.ts` (card words, keyed), `gooseLeaves.ts`, `gooseTriggers.ts` (origins), `gooseLinkChoices.ts`, `signalPlan.ts`/`signalRoute.ts` (`routeVia`), `features/diagram/ied/` (sheet, GOOSE section, Add all, missing list), `features/diagram/signals/` (`FlyLines`, `IedFlash`, `MissingChip`, `useGooseMembers`, `useMeasuredBoxes`), `api/gooseMessages.ts`, `copy/iedComponent.ts`; Go: `engine/internal/goosemsg/` (FC-aware catalog, links, plans, refusals, proposal), `project/contract/goose_link.go`, `project/validation_ied_component.go`, `validation_goose_link.go`, `runtime/goose_send.go`, `api/runtimeapi/goose_routes.go`, `api/liveapi/goose_messages.go`, `api/projectapi/goose_ieds_route.go` |
| Real IED client | `native/live/`, `acquisition/real_session.py`, `live_values.py`, `domain/model_compare.py`, `access_policy.py` |
| Guided UI and wording | `frontend/src/ui/`, `frontend/src/copy/`, `features/network/`, `features/settings/` |
| Feature contracts | `docs/specs/`, `studio/tools/check_specs.py` |
| Repository verification | `studio/tools/verify.py` and `check_*.py` |
| Licensing (app) | `domain/license_policy.py`, `licensing/` (incl. `revoked_ids.py` cancelled IDs, `verify_answer.py` signed answers; Go twins in `engine/internal/licensing/`), `api/license.py`, `api/security.py` gate, `frontend/src/features/license/` |
| VM host key (ADR-0099, Go only) | `engine/internal/licensing/hostkey/` (key store with DPAPI, protocol, adapters, server, client, Run value, sharing), `licensing/hostkey_*.go` and `vm_*.go` (host responder, VM search, proof file, status), `cmd/scadabench_hostkey/` (helper exe), `api/hostkeyapi/`, `netcfg/hostkey_firewall.go`, `native/common/vm_proof.*`, `tools/build_hostkey_helper.py`, `frontend/src/features/settings/VmHost*.tsx`, `features/license/ActivationCode.tsx` |
| Licensing (native) | `native/common/license_guard.*`, `native/common/license_clock.*` (clock and cancelled IDs), `native/release/release_key.hpp`, `native/third_party/monocypher/` |
| License server | `license-server/` (Go): `internal/api`, `store`, `security`, `certs`, `paypal`, `mail`, `sales`, `deploy/` (owner commands: `deploy-server.cmd`, `set-payments.cmd`, `set-price.cmd`, `payments.cmd`) |
| Owner page (ADR-0072) | `license-server/internal/adminauth/` (password, codes, sessions, lockout), `internal/audit/`, `internal/store/subscribers.go`, `internal/api/owner*.go`, `pages/admin*.{html,css,js}`, `pages/unsubscribe.html`, `cmd/license-server/adminsetup.go`, `gateway/` (Cloudflare Worker), `deploy/admin.ps1`; owner commands `admin-setup.cmd`, `reset-admin.cmd` |
| Brand and website | `studio/branding/`, `tools/build_icons.py`, `tools/build_installer_art.py` (installer pictures), `website/` (Astro, Cloudflare Pages) |
| Update check (ADR-0071) | app: `updates/versions.py` (pure rules), `updates/service.py`, `api/updates.py`, `frontend/src/api/updates.ts`, `app/UpdateNote.tsx`, `features/settings/UpdateSection.tsx`; server: `license-server/internal/release/`, `internal/api/release.go`, `cmd/license-server/release.go`, `deploy/release-record.ps1`, `owner.ps1 -Action release` |
| Waiting indicators and pickers (change 0105) | `ui/Spinner.tsx`, `ui/useDelayed.ts`, `components/pickerMemory.ts`, `components/SignalSearchPicker.tsx`, `features/devices/DeleteIedsDialog.tsx`, `DeviceSelectBar.tsx` |
| Point kinds and smart picker (ADR-0095, ADR-0113) | Go: `engine/internal/acquisition/signalindex/` (`point_kind.go` the table, `kind_cache.go`, `logical_nodes.go`, `scan.go` one counting pass, `near.go` suggestions); browser: `points/pointKind.ts` (same table), `components/signalPicker/` (`preferredKind`, `pickerAnswer`, `useSignalSearch`, `KindChips`, `ObjectSelect`, `HelperToggle`, `SuggestionBlock`, `SignalPickRow`), `copy/signalPicker.ts`; shared vectors `studio/tests/fixtures/point_kinds/iec61850.json` |
| Run shapes (change 0106) | `copy/runShape.ts`, `features/devices/DeviceRunSummary.tsx`, `runGroups.ts`, `features/diagram/RunStrip.tsx`, `equipment/liveContext.ts` (`StoppedIeds`), `simulation/coordinator.py` (`start` adds the rest) |
| Guided tour | `frontend/src/features/tour/`, `copy/tour.ts`, `projects/preferences.py`, `api/preferences.py` |
| Desktop window (ADR-0048) | `studio/backend/studio_app/launcher/entrypoint.py`, `app_window.py`, `single_instance.py`, `window_focus.py`, `windowed.py` |
| Protected release | `tools/configure_release.py`, `compile_protected.py`, `pack_frontend.py`, `packaging/studio_protected.spec` (installed folder + one-file portable exe), `installer.iss` |

Tests mirror these boundaries under `studio/tests/unit`, `integration`, `protocol`,
`frontend`, `packaging` and `performance`. Keep each maintained file below 300
physical lines and split by responsibility before growth forces a catch-all module.
