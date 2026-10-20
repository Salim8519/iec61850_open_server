# Decisions

| ID | Decision |
| --- | --- |
| [0001](ADR-0001-flask-runtime.md) | Flask, native workers, portable browser UI |
| [0002](ADR-0002-color-refresh.md) | User-requested richer shared color palette |
| [0003](ADR-0003-scl-validation.md) | Offline XSD validation and schema provenance |
| [0004](ADR-0004-preserving-scl-export.md) | Verified scalar export preserves original document scope |
| [0005](ADR-0005-automation-cadence.md) | Bounded deadline recovery and explicit timing evidence |
| [0006](ADR-0006-native-array-parser.md) | Audited correction without modifying original vendor source |
| [0007](ADR-0007-startup-recovery.md) | Stopped engineering recovery and per-directory launcher lock |
| [0008](ADR-0008-compact-projects.md) | Deduplicated engineering archives and bounded legacy recovery |
| [0009](ADR-0009-report-mapping-and-signal-tests.md) | Report mapping, saved signal tests, guarded pulses and display motion |
| [0010](ADR-0010-development-reload.md) | Development reload mode and Vite hot reload |
| [0011](ADR-0011-typed-equipment-controls.md) | Typed DPC/SPC commands, constructed switch state and trip relay |
| [0012](ADR-0012-one-development-launcher.md) | One unambiguous source-development launcher and owned process tree |
| [0013](ADR-0013-type-driven-equipment-operation.md) | Type-driven equipment operation and selected feedback |
| [0014](ADR-0014-equipment-actions-and-source-editor.md) | Semantic actions, direct selection, push buttons and analog Source |
| [0015](ADR-0015-native-control-feedback.md) | Atomic native controls and shared mapped feedback |
| [0016](ADR-0016-format-6-trip-and-live-canvas.md) | Reset-high legacy Trip migration and isolated live canvas values |
| [0017](ADR-0017-trip-polarity-and-sbo-observability.md) | Independent Trip polarity, state hydration and enhanced-SBO lifecycle |
| [0018](ADR-0018-concen-sbow-compatibility.md) | Explicit origin-scoped CONCEN SBO adaptation (superseded by ADR-0069) |
| [0019](ADR-0019-single-session-project-revisions.md) | Shared tab identity and stale-write rejection |
| [0020](ADR-0020-append-only-evidence-ledgers.md) | Append-only engineering and runtime evidence |
| [0021](ADR-0021-sld-drawing-primitives.md) | SLD junctions, four-way terminals, loose wiring and drawn busbar length |
| [0022](ADR-0022-relay-status-sets.md) | Relay as a named status set, non-connectable, initiated through test paths |
| [0023](ADR-0023-push-button.md) | Terminal-free push button, latching default, pulse-backed self-reset |
| [0024](ADR-0024-relay-panel-usability.md) | Relay panel and signal-card usability refinement |
| [0025](ADR-0025-properties-cards-and-canvas-clarity.md) | Properties cards, setup checklist, canvas clarity, no-op save guard |
| [0026](ADR-0026-configurable-state-colours.md) | Configurable state colours for two-state equipment |
| [0027](ADR-0027-canvas-editing-ergonomics.md) | Canvas multi-select, clipboard shortcuts and terminal hit targets |
| [0028](ADR-0028-beginner-guidance-and-network-tabs.md) | Beginner guidance kit, Network/Real IEDs tabs, protected feature specs |
| [0029](ADR-0029-lab-network-and-format-11.md) | Guided lab network, elevated helper, append-only journal, format 11 |
| [0030](ADR-0030-goose-with-user-installed-npcap.md) | Layer-2 GOOSE with pinned SDK, delay-loaded user-installed Npcap |
| [0031](ADR-0031-read-only-live-acquisition.md) | Read-only live acquisition from real IEDs and SCD comparison |
| [0032](ADR-0032-clean-four-place-ui.md) | Clean four-place UI (Diagram, Devices, Test, Settings) and design system v2 |
| [0033](ADR-0033-feature-consolidation-format-12.md) | Feature consolidation, retired legacy kinds, project format 12 |
| [0034](ADR-0034-higher-contrast-surfaces.md) | Higher-contrast surfaces, borders and single-line diagram |
| [0035](ADR-0035-sld-import-from-scd.md) | Single-line diagram import from the SCD Substation section |
| [0036](ADR-0036-signal-flow-from-scl.md) | Station signal-flow graph from SCL |
| [0037](ADR-0037-station-health.md) | Station health check against the SCD (read-only, goID check) |
| [0038](ADR-0038-sampled-values-listening.md) | Listen to Sampled Values (no publishing), nominal frequency setting |
| [0039](ADR-0039-test-mode.md) | Test mode, behaviour and simulation; sized integer writes |
| [0040](ADR-0040-sld-tidy-up.md) | Tidy up the single-line diagram in the browser (layered SLD layout) |
| [0041](ADR-0041-infeed-symbol.md) | Infeed symbol, isolated from every engine (format stays 12) |
| [0042](ADR-0042-live-colouring.md) | Live colouring computed by the backend, shown by line style |
| [0043](ADR-0043-interlock-hints.md) | Interlocking is friendly hints only; never blocking |
| [0044](ADR-0044-bay-templates.md) | Bay templates in the Add menu |
| [0045](ADR-0045-readable-switching-symbols.md) | Switching symbols readable from a distance (geometry, zoom bands, change pulse) |
| [0046](ADR-0046-licensing-and-protected-release.md) | Machine licensing (trial, 1-PC lifetime), Go license server, protected release; development unchanged |
| [0047](ADR-0047-scadabench-brand-website-tour.md) | ScadaBench brand, icon pipeline, Astro website on Cloudflare Pages, guided tour, lazy server ops |
| [0048](ADR-0048-desktop-app-window.md) | Windowed EXE, one instance (second launch shows the window), confirm and stop on close, SLD tab after Devices |
| [0049](ADR-0049-tutorial-videos.md) | Narrated tutorial videos: real-app recordings (Playwright/Edge), Gemini TTS via fal.ai, Remotion edit, served from scadabench.com |
| [0050](ADR-0050-release-hardening.md) | Release hardening: installed app refuses dev mode, version stamp, stronger anti-debug, HWID v2 (SMBIOS UUID, re-activation); signing deferred |
| [0051](ADR-0051-legacy-hwid-acceptance.md) | Pre-v2 licenses stay valid on the same PC after update (v1 ID accepted for certificates issued before 2027-07-01); no move spent |

| [0052](ADR-0052-owner-license-helper.md) | Owner license helper: one portable, offline, password-locked EXE with its own owner signing key (server key never leaves the server) |
| [0053](ADR-0053-website-mobile-conversion.md) | Website phone-first: full navigation and phone menu, phone buy bar, hover prefetch, light videos, value/trust sections, visible conditional 7-day refund policy |
| [0054](ADR-0054-tools-place.md) | Fifth place Tools after Test: cards for delivered tools only; GOOSE/SV traffic moved there; Explorer, Find IEDs, inspector, 104/Modbus/DNP3 gateway proposed |
| [0056](ADR-0056-windows-tls-verification.md) | License client verifies HTTPS with the Windows certificate check (truststore); fixes "cannot reach the license server" after the 2026 Let's Encrypt chain change |
| [0057](ADR-0057-gateway-explorer-commands.md) | Gateway (lib60870, opendnp3, own Modbus), studio_control for confirmed commands to real IEDs from Tools, format 13 gateway section, autonomous tools with remembered settings |
| [0058](ADR-0058-window-first-startup.md) | Window first: the autosave restores on a background thread; engineering requests wait behind the security/license checks; a protective recovery stop refuses all work |
| [0059](ADR-0059-separate-run-running-dots.md) | Run and the gateway are separate; a running dot per place with words; gateway panel fills the window; page commands only after a warning, logged like SCADA commands |
| [0060](ADR-0060-portable-version.md) | Portable version = the same sealed build with a data folder beside the exe; zip plus an Inno self-extractor (fits the 25 MiB Pages limit), no registry or uninstaller |
| [0061](ADR-0061-scada-roles-master-converter.md) | SCADA roles in one tool (station gateway, SCADA master, protocol converter) with each protocol's own words; studio_master (104 controlling station, DNP3 master, Modbus client); discover all by default; format 14 |
| [0062](ADR-0062-gateway-firewall.md) | Gateway on all adapters opens Windows Firewall through the network helper: reviewed allow rule for the gateway worker, leftover Block rules switched off and re-enabled by clean-up |
| [0063](ADR-0063-official-installer-only.md) | One official installer (ScadaBench-Setup); no portable installer or zip; portable = one PyInstaller one-file exe from the same analysis that runs directly, data and workers beside it (supersedes the self-extractor/zip parts of ADR-0060) |
| [0064](ADR-0064-tool-pages-fill-window.md) | Open tools and Devices fill the window (side column, panes that scroll on their own, stacked below 1000×620); one FilterBar on every list, choices remembered by the backend; long hints behind "?" |
| [0065](ADR-0065-sld-signal-layers.md) | SLD signal layers: remembered switches GOOSE messages / Who is reading, one GOOSE link map by goCBRef (vendor-disabled, remote publishers, APPID/MAC hints), IED tags never saved, one quiet pulse per real stNum change, Devices partners |
| [0066](ADR-0066-goose-inputs-and-mms-access-events.md) | Simulated subscribers apply received GOOSE to their own ExtRef-bound inputs (refused on any problem); workers report MMS client address, reads per node and report blocks once per second without changing answers |
| [0067](ADR-0067-run-single-ied-and-test-buttons.md) | Run or connect one IED inside one run (`running_ieds`, `POST /runtime/ieds/{id}/start` or `/stop`); any running IED locks the drawing; equipment test buttons write simulated points via the sample writer, no command; outside senders leave the canvas for a toolbar list |
| [0068](ADR-0068-publish-wording-and-one-page-network.md) | Modes shown as Be this IED / Read real IED / Leave out (stored roles unchanged); role line and top-bar reachability; one-page network setup with one button and a suggested free address per problem row; no published IED names each mode |
| [0069](ADR-0069-standard-commands-and-listening-endpoint.md) | Standard IEC 61850 commands for every client (CONCEN profile removed, format 15 migrates it with a notice; worker advertises `standard_controls`); Operate and gateway commands go to the endpoint the worker confirmed, so lab-mode IEDs are reached |
| [0070](ADR-0070-goose-visible-between-machines.md) | GOOSE between machines visible and timed: outside-sender dot starts beside its receiver at screen size with a fading line and a "← sender · change" tag line; IED worker uses a 1 ms timer so MinTime retransmissions stay within TAL; one lab-reachability rule; live state for receiving outside senders; equipment names wrap |
| [0071](ADR-0071-quiet-update-check.md) | Quiet, signed daily update check: license server `GET /api/v1/release` (domain-separated Ed25519 signature, rate limit, ETag, cacheable), app checks at most daily without sending anything about the PC; one dismissible line above Settings, details in About; development reports `x.y.z+dev` |
| [0072](ADR-0072-owner-page.md) | Owner page at `scadabench.com/<owner path>` via a Cloudflare Worker that adds a gateway secret (otherwise 404); password (PBKDF2) plus authenticator code, recovery codes, no email reset; memory sessions, CSRF, lockouts, fresh code for risky actions, append-only audit; news only with consent and one-click unsubscribe; disk cleanup limited to old logs |
| [0073](ADR-0073-goose-engine-go.md) | GOOSE engine as a second process in Go (`studio_goose.exe`): workers publish to it over loopback UDP, `studio_live` decodes from its inbox; GOOSE inside the app by default in every mode without Npcap; one confirmed switch shares it on an auto-picked wired adapter after a 3 s check; smart guard (reflections, no wire-to-wire, conflict hold with Stop/Continue, rate cap); stale-engine refusal |
| [0074](ADR-0074-go-engine.md) | Go engine (`studio_server.exe`, stdlib only) replaces the Python backend with byte-identical behaviour; Python stays default until all parity/performance gates and owner sign-off, then remains a selectable fallback; phases ordered by speed gain (runtime first); persistent multiplexed MMS client, pinned libxml2 XSD helper, journal save, value hub; 20-IED limit unchanged |
| [0075](ADR-0075-python-compatible-json.md) | Go engine JSON is byte-identical to CPython: `pyjson` encoders for Flask `jsonify` (sorted, compact, ASCII, newline), `json.dumps` defaults and compact form; `repr(float)`, NaN/Infinity, big integers; ordered decoding with `json.loads` rules; parity proven by differential tests against CPython (20k in verify, 1M at the P0 gate) |
| [0076](ADR-0076-go-only-development-and-parallel-workstreams.md) | New backend code only in Go, Python frozen as the stable engine (critical fixes only); phases P0-P5 built in parallel by six workstreams with disjoint ownership and shared contracts (pyjson, ordered model trees, `api.Route`, `apperr`, native framing); only the integrator writes ledger/indexes/verify |
| [0079](ADR-0079-event-driven-ied-server-loop.md) | `studio_ied` runs the vendor single-threaded MMS server threadless from an event loop that sleeps until the next vendor control/report deadline, GOOSE retransmission or IPC wake; vendor socket HAL compiled unmodified under renamed symbols; private files pinned by SHA-256; threaded MMS rejected; no IPC/API/wire change |
| [0087](ADR-0087-push-stream-and-live-store.md) | Frontend live store (both engines: stores outside React, per-frame flush, liveFeed polling hygiene) and Go-only push stream (`/capabilities`, `/push` per tab with exact-GET topics and keyed text deltas, subscriptions per visible view, coalescing, unchanged bytes not sent); gateway sweep replaced by project_changed; Python keeps polling |
| [0088](ADR-0088-ied-timing-without-machine-timer.md) | studio_ied reads the precise system clock and sleeps on socket events + a high-resolution waitable timer at exact millisecond deadlines; power-throttling opt-out; `timeBeginPeriod(1)` only as fallback before Windows 10 1803; supersedes ADR-0070 decision 2; no wire change |
| [0089](ADR-0089-master-refused-connect-fails-at-once.md) | studio_master replaces lib60870 Windows `Socket_connect` (vendor file compiled unmodified under a renamed symbol, hash-pinned): wait on writable + except, first attempt without SYN repeats; refusals reported at once, other failures at the connect timeout; same bytes, text and events |
