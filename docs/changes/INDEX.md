# Change index

| ID | Summary | Status |
| --- | --- | --- |
| [0001](2026/0001-foundation.md) | Architecture and development foundation | Verified foundation |
| [0002](2026/0002-native-and-workspace.md) | Native MMS, CID audits, browser workspace | Implemented; partial acceptance |
| [0003](2026/0003-portable-preview.md) | Windows one-folder developer preview | Build verified |
| [0004](2026/0004-selective-import.md) | Explicit IED selection before CID import | Implemented; API tested |
| [0005](2026/0005-sequences-deletion-colors.md) | Timed scenarios, recoverable IED deletion, color refresh | Tested; visual acceptance pending |
| [0006](2026/0006-batched-runtime.md) | Batched automation and load benchmark | Short test passed; soak running |
| [0007](2026/0007-diagram-history.md) | Diagram history and native initial display | Tested; visual acceptance pending |
| [0008](2026/0008-csv-playback.md) | Portable CSV measurement playback | Native playback tested |
| [0009](2026/0009-scl-schema-validation.md) | Offline edition-aware SCL schema validation | Schemas/template tested |
| [0010](2026/0010-verified-scl-export.md) | Preserving edited SCL export and staged editor | API tested; visual acceptance pending |
| [0011](2026/0011-cadence-and-benchmark-identity.md) | Explicit timing recovery and benchmark provenance | Tested; full load acceptance pending |
| [0012](2026/0012-independent-mms-browsing.md) | MMS type/dataset discovery and indexed-member correction | Verified; top-level array regression open |
| [0013](2026/0013-top-level-array-candidate.md) | Top-level DO array correction and isolated candidate | Protocol tests pass; promotion pending |
| [0014](2026/0014-startup-recovery.md) | Stopped autosave recovery and launcher ownership lock | Source/API tested; packaged acceptance pending |
| [0015](2026/0015-portable-promotion-and-stale-backend.md) | Portable promotion and deletion 404 diagnosis | Rebuilt; old user backend awaits safe restart |
| [0016](2026/0016-compact-save-recovery.md) | Live restart, compact saves and oversized legacy recovery | Real project verified; rebuilt; 80 tests pass |
| [0017](2026/0017-report-mapping-and-signal-tests.md) | Report-first mapping, switch motion, saved signal tests and bulk deletion | Tested and live; visual acceptance pending |
| [0018](2026/0018-quick-start-script.md) | One-command start for the packaged app | Self-test verified |
| [0019](2026/0019-development-hot-reload.md) | Development hot reload (backend restart and Vite HMR) | Tested; interactive acceptance pending |
| [0020](2026/0020-typed-controls-and-trip-relay.md) | Typed DPC/SPC controls, constructed switch state and trip relay | Protocol tested; release acceptance pending |
| [0021](2026/0021-one-development-launcher.md) | One source-development launcher with deterministic cleanup | Source/proxy smoke verified; visual acceptance pending |
| [0022](2026/0022-type-driven-equipment-operation.md) | Type-driven operation updates the selected simulated feedback | 107 tests pass; visual acceptance pending |
| [0023](2026/0023-clean-mapping-push-buttons-and-sources.md) | Clean mappings, configurable buttons and analog Source | 111 tests pass; visual acceptance pending |
| [0024](2026/0024-external-controls-selection-and-launch-reuse.md) | External command feedback, stable selection and launcher reuse | Protocol tested; visual acceptance pending |
| [0025](2026/0025-format-6-trip-and-stable-canvas.md) | Reset-high legacy Trip migration and stable live canvas | Protocol and stress tested; visual acceptance pending |
| [0026](2026/0026-trip-state-and-sbo-lifecycle.md) | Independent Trip state and enhanced-SBO lifecycle | Protocol and regression tested; visual acceptance pending |
| [0027](2026/0027-concen-compatibility-and-tab-sync.md) | CONCEN SBO adapter and authoritative multi-tab revisions | Protocol and regression tested; visual acceptance pending |
| [0028](2026/0028-command-trace-5-diagnosis.md) | Trace 5 accepted Open was inverted back to Close | Packet/runtime evidence recorded; policy decision pending |
| [0029](2026/0029-append-only-llm-traceability.md) | Append-only logs and LLM repository map | Focused tests and repository verification recorded |
| [0030](2026/0030-stable-deferred-sbo-probe.md) | Bounded probe wait for deferred SBO adaptation | Independent protocol regression coverage |
| [0031](2026/0031-concentrator-standard-sequence-and-noop-feedback.md) | Concentrator standard sequence evidence and no-op feedback fix | Protocol and unit suites pass; two-direction capture pending |
| [0032](2026/0032-open-switch-start-positions.md) | Open switch start positions and connector-free push button | Tests and verify pass; visual acceptance pending |
| [0033](2026/0033-sld-drawing-primitives.md) | SLD junctions, four-way terminals and drawn busbar length | Tests and verify pass; visual acceptance pending |
| [0034](2026/0034-relay-status-sets.md) | Relay as a named, initiatable status panel instead of a wire-path object | Tests and verify pass; extended by 0035 |
| [0035](2026/0035-relay-single-ied-and-launcher.md) | Single-IED relay picker, legacy palette, no auto-open launcher | Tests and verify pass; visual acceptance pending |
| [0036](2026/0036-resizable-inspector.md) | Drag-resizable properties panel with wrapping rows | Tests and verify pass; visual acceptance pending |
| [0037](2026/0037-properties-polish.md) | Properties without path typing, polished buttons | Tests and verify pass; visual acceptance pending |
| [0038](2026/0038-section-tones.md) | Color-coded property sections per domain | Tests and verify pass; visual acceptance pending |
| [0039](2026/0039-grip-fix.md) | Resize grip pinned outside the scroll flow | Tests and verify pass; visual acceptance pending |
| [0040](2026/0040-push-button.md) | Terminal-free push button, latching default, self-reset | Protocol and unit tested; visual acceptance pending |
| [0041](2026/0041-dev-session-guidance.md) | Accurate missing-session guidance in development mode | Integration tests and verify pass |
| [0042](2026/0042-relay-panel-usability.md) | Relay panel and signal-card usability | Tests, build, verify and screenshots; owner acceptance pending |
| [0043](2026/0043-properties-and-canvas-clarity.md) | Properties cards, setup checklist, canvas clarity, no-op save guard | Tests, build, verify, screenshots; owner acceptance pending |
| [0044](2026/0044-configurable-state-colours.md) | Configurable state colours for two-state equipment | Tests, build, verify, running screenshots; owner acceptance pending |
| [0045](2026/0045-painted-state-parts.md) | Paint the state part of the symbol, not the whole card | Tests, build, verify, running screenshots; owner acceptance pending |
| [0046](2026/0046-canvas-editing-ergonomics.md) | Canvas multi-select, copy/paste/duplicate, easier terminals, one-commit drag | Tests, build, Playwright on isolated copy; owner acceptance pending |
| [0047](2026/0047-wires-vanish-and-junction-offset.md) | Wires no longer vanish until refresh; junction dot centred on terminals | Tests incl. library-level bounds test, build, Playwright on owner copy; race not reproducible on demand |
| [0048](2026/0048-junction-drag-grip.md) | Junction 40 px drag grip, stubs to wired terminals | Tests, build, Playwright grab-grid on owner copy; owner acceptance pending |
| [0049](2026/0049-lab-network-binding.md) | Guided lab network binding and project format 11 | Unit/integration tests, browser preview; real Apply on owner PC pending |
| [0050](2026/0050-goose-publish-and-listen.md) | GOOSE publishing, listening and supervision | Wire test on VMware adapter passes; physical/third-party evidence pending |
| [0051](2026/0051-real-ied-acquisition.md) | Read-only live acquisition and SCD comparison | Protocol tests against stand-in, browser check; real relay pending |
| [0052](2026/0052-beginner-guidance-and-specs.md) | Beginner guidance kit and protected feature specs | Frontend tests, build, spec check; owner visual acceptance pending |
| [0053](2026/0053-real-cid-import-fixes.md) | 134/134 real vendor CIDs import and run (PhyComAddr, unnamed reports) | Audit + native start of all 134; regression tests |
| [0054](2026/0054-clean-ui-restructure.md) | Clean four-place UI restructure with merged features and format 12 | Browser walkthrough with real CIDs, probe evidence; owner acceptance pending |