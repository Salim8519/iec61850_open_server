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
| [0055](2026/0055-higher-contrast-design.md) | Higher-contrast surfaces, borders and single-line diagram | Browser check of four places; frontend tests and build; owner acceptance pending |
| [0056](2026/0056-fast-adapter-detection.md) | Fast, live adapter detection through the Windows IP Helper API | 12 ms vs 6 s; matches Get-NetAdapter on this PC; cable plug-in not yet observed |
| [0057](2026/0057-station-feature-specs.md) | Proposed specs for SLD import, signal flow, station health, SV, test mode | Documentation only; no code; specs not promoted |
| [0058](2026/0058-sld-import.md) | Build the single-line diagram from the SCD Substation section | 19 backend + frontend tests; browser on two SCDs; owner acceptance pending |
| [0059](2026/0059-signal-flow.md) | Signal flow: declared GOOSE and report links between IEDs | Unit/API/frontend tests; browser on model SCD found a real subscription error; owner acceptance pending |
| [0060](2026/0060-station-health.md) | Station health: live network and real IEDs compared with the SCD | Unit/protocol (incl. goID wire test on VMware adapter)/frontend; browser with stand-in; third-party relay pending |
| [0061](2026/0061-sampled-values.md) | Sampled Values listening: supervision, RMS/phasor, 50/60 Hz | 4 SV wire tests on VMware adapter, 15% of one core for 4 streams; real merging unit pending |
| [0062](2026/0062-test-mode.md) | Test mode: Mod/Beh/Sim badges, simulated IED modes, simulation notice; native integer write fix | Probe reads Beh 4/3 on a simulated IED; protocol 38 pass; real relay pending |
| [0063](2026/0063-sld-tidy-up.md) | Tidy up: one-press single-line diagram layout (busbars on top, straight bays) | 10 frontend tests, fuzz of 400 diagrams, isolated browser check on scrambled model SCD; owner acceptance pending |
| [0064](2026/0064-infeed-symbol.md) | Infeed symbol, isolated from SCL/SLD/runtime engines | 4 guard tests incl. byte-identical SCL export; owner acceptance pending |
| [0065](2026/0065-live-colouring.md) | Live colouring of wires from real switch positions | Unit, native protocol, frontend tests; isolated browser check; owner acceptance pending |
| [0066](2026/0066-interlock-hints.md) | Friendly interlock hints, never blocking; external switching noted | Unit, native protocol (probe), frontend tests; browser check; real SCADA pending |
| [0067](2026/0067-bay-templates.md) | Bay templates: line feeder, transformer bay, bus coupler | 4 frontend tests; browser check; owner acceptance pending |
| [0068](2026/0068-readable-symbols.md) | CB/DS/ES symbols readable from a distance: bold geometry, zoom bands, change pulse | 3 frontend tests; browser render at 3 zooms; live-run pulse and owner acceptance pending |
| [0069](2026/0069-licensing.md) | Licensing, Go license server (PayPal/Mailgun), licensed native workers, protected build pipeline | Go/Python/native/frontend tests, real Go e2e, enforced worker, browser flow; EXE, live PayPal pending |
| [0070](2026/0070-signal-picker-fits-panel.md) | Signal picker fits the properties panel; full point paths wrap, Choose always visible | 95 frontend tests; browser layout check at 330/380px; owner acceptance pending |
| [0070](2026/0070-scadabench-brand-website-tour.md) | ScadaBench rename, icon, live website, guided tour, tunnel/email/Mailgun, one-command server deploy | Tests pass; site live; server deploy waits for key authorisation |
| [0071](2026/0071-license-server-live-release.md) | License server live on own VPS via tunnel, one-command deploy, hidden-input payment secrets, first real release + download | Live health/403 admin, real self-test vs live server, download hash match, full tests; PayPal purchase pending |
| [0072](2026/0072-checkout-trust-sales-hardening.md) | PayPal-branded trustworthy checkout, live price (set-price.cmd), payment record, owner sale alerts, abuse/resource hardening | Go tests, local attack run, live limits/CSP/product/order checks; next real purchase to watch |
| [0073](2026/0073-desktop-app-window.md) | No console window; second launch focuses the running window; close asks then stops all IEDs; SLD tab after Devices | 311 backend + 101 frontend tests, verify PASS, real source double-launch (356 ms, exit 0), real Edge window restore and close-stops; built 0.1.1 EXE: windowed, 2nd launch 0.41 s restores window, close stops all; close prompt by owner |
| [0074](2026/0074-window-handoff-startup-crash.md) | Installed app no longer crashes (KeyboardInterrupt) when Edge relaunches/hands off its window at start; closing a handed-off window stops the app | 275 unit tests, verify PASS; no EXE rebuilt, real relaunch not reproduced |
| [0075](2026/0075-release-hardening.md) | Release hardening: installed app refuses dev mode, single version stamp (shown in Settings/health), stronger anti-debug, HWID v2 (SMBIOS UUID; re-activation), native workers verify the signed compiled module | 280 unit tests, verify PASS, native rebuilt: C++/Python HWID parity + enforced worker accepts v2 cert + module-signature round-trip (valid/tamper/wrong-key); Authenticode signing deferred; no EXE built |
| [0074](2026/0074-tutorial-videos.md) | Seven narrated teaching videos from the real app (2x capture, Gemini TTS via fal.ai, Remotion) on public OpenSCD demo data only | All rendered and frame-checked; no network change; tests pass; on hold before publishing |
| [0076](2026/0076-legacy-hwid-acceptance.md) | Licenses from older builds survive the HWID v2 update without re-activation or spending a move; weekly check no longer reports them moved | 29 Python + 17 native license tests (dev + enforced), real owner cert matches v1 ID; 0.1.2 installer built |

| [0077](2026/0077-owner-license-helper.md) | Owner license helper EXE: license this PC, license file for another PC, remove license; offline, Argon2id+AES-GCM password vault, no network code | 10 Go tests incl. real sealed-exe end-to-end, Studio Python + native accept/refuse, no-network dependency check, 16 Python licensing tests; owner setup + key add pending; no ScadaBench EXE built |
| [0078](2026/0078-website-mobile-conversion.md) | Website: full navigation + phone menu, phone buy bar, readable phone hero, tap-to-zoom screenshots, demo video, bench comparison, built-by, trust row, refund policy (EULA §6), grouped FAQ, 720p tutorial videos (98 to 16 MB) | Build OK, verify PASS, Lighthouse mobile perf 95-100 / a11y 100 on 6 pages, Chrome at 360/390/768/desktop; not deployed yet |
| [0079](2026/0079-tools-place.md) | Tools place (5th sidebar place): tool cards with hints, GOOSE & SV traffic moved from Settings, tour step; proposed specs for IED Explorer, Find IEDs, inspector, gateway simulator | tsc clean, 105 frontend tests pass, verify PASS; owner visual acceptance pending; no backend/native change |
| [0080](2026/0080-windows-tls-verification.md) | "Cannot reach the license server" was a TLS chain issue (ISRG Root X2 missing, expired cross-sign); client now verifies like Windows/browsers | Live health via app client OK, badssl expired/wrong-host/self-signed still refused, 290 unit tests; installed 0.1.x needs a new build |
| [0081](2026/0081-ied-explorer-find-ieds.md) | IED Explorer (warm client, lazy tree, autonomous key-point watch, confirmed commands via new studio_control) and Find IEDs (previewed range, socket probe, MMS identify); every tool setting remembered | 6 explorer-wire + 5 control protocol tests, API test with confirmed operate; dev app on owner project: connect 0.9 s, 40 points watched; 358 Python + 108 frontend pass |
| [0082](2026/0082-gateway-simulator.md) | Gateway simulator: IEC 104 (lib60870), Modbus TCP (own), DNP3 (opendnp3); smart point list, autonomous start (points + Run), auto-resume, real-IED command switch; project format 13 | Independent lib60870/opendnp3 masters + Modbus socket client; 104 master operates simulated breaker end to end; owner project 3,240 points, GI 2,652 objects good; no third-party SCADA yet |
| [0084](2026/0084-fast-project-restore.md) | Faster launch: restoring the autosave imports each shared SCL source once instead of once per IED (owner project: backend ready 29 s, restore step 28.7 to 6.2 s) | Identical models on the real autosave, 48 restore tests + 290 unit pass, verify PASS; ships with next installer |
| [0085](2026/0085-window-first-startup.md) | Window opens in ~1 s (was 29 s on owner project); project restores in background with "Opening your last project…"; one XML parse per import | 341 Python + 107 frontend tests, tsc clean, verify PASS; source-run timing 0.9 s window / 6.7 s ready; installed build to re-measure |
| [0086](2026/0086-separate-run-running-dots.md) | Run and the gateway are separate (Start never starts IEDs); running dots per place and tool-card badges; gateway points fill the window; Close/Open and On/Off commands from the points table after a warning | 360 Python + 50 protocol + 111 frontend pass; source app at 1440×900: Run stays stopped after Start, dots on Devices/SLD/Tools, page command operated, no page scroll |
| [0087](2026/0087-release-0.1.6-portable.md) | Release 0.1.6 (separate Run, running dots, page commands) with installer and new portable version (self-extractor on the website, zip locally); website published | Built-app and unpacked-portable checks: self-test, data beside exe, no uninstall key, 104 e2e; live downloads match SHA-256 |
