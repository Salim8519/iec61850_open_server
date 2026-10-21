# 0105: Easier IED delete, pickers that keep their place, visible waiting, GOOSE corner marks, update check

Date: 2026-10-19. Owner request (with screenshots): the IED delete button is almost hidden and
several IEDs cannot be deleted at once; in SLD properties a wrong point pick "resets everything";
network and other pages show no spinner, so users think the app is stuck; GOOSE stickers cover other
equipment ("space of SLD and clean look is important", option B chosen from drawn proposals); and
"add an endpoint that checks if a new version is available", secured against bots and floods,
without annoying users. Decision: [ADR-0071](../../decisions/ADR-0071-quiet-update-check.md).

## Findings

- Delete lived only in Devices › Files. The backend already had an atomic bulk removal
  (`POST /ieds/removal-preview`, `/ieds/remove`, fingerprint and one recovery backup) without UI.
- Picker reset: after a choice `SignalSearchPicker` unmounted; *Change* mounted a new one with
  "Choose an IED", empty search, all reports and page 1. Same pattern: relay *Add signals* after
  *Done* and Test › *Add test point* after *Close*. Also switching the connection type (one point ↔
  two contacts) cleared the saved point at once.
- 22 silent waits were found in the frontend (audit list in this change's session); the worst were
  the blank window at start, Run/Stop and file open with only greyed buttons, and lists that said
  "No data points match" or "No test points yet" before their first answer.
- The GOOSE tag sat beside the topmost bound equipment (`x + width + 10`), so in a tight bay it
  covered the next bay (owner screenshot: C03_Y1105 over QB2).

## Change

- Devices: *Select* beside Import gives tick boxes and a footer (count, All N, Cancel,
  *Delete N IEDs*, at most 20); a *Delete* button in the device header; one review dialog
  (`DeleteIedsDialog`) for one or many IEDs on the bulk endpoints, with plural-correct impact lines.
- Pickers: `pickerMemory.ts` remembers IED, search, report and page per picker for the browser tab;
  *Change* opens on the bound IED; after a choice the list stays open with the row marked
  *Chosen ✓* and the others *Use this*; *Undo* and *Done*. Switching connection type keeps the
  saved point until the new one is complete; only *Not connected* clears it (spec unchanged:
  choosing still saves immediately).
- Waiting: shared `ui/Spinner.tsx` (ring, words, calm "still working" after 8 s, reduced motion
  static) and `useDelayed`. `perform()` carries words ("Starting the IEDs…", "Opening the project…",
  "Reading the file…", "Importing the IEDs…"), shown in the top bar run state and on Run/Stop after
  400 ms. Spinners or busy words for: start window, network cards, Refresh, address check, Apply,
  signal picker, data points, live values and *Read these now*, Compare with SCD (no longer says
  "press Run" while running), Health check, Signal flow, Test points, License tab, Build from SCD.
- SLD signal layers (spec LAYER-10 changed first): a bound IED is an 18 px "G" mark inside the
  top-right corner of its topmost equipment (second IED steps left); ⚠ replaces G for a problem
  (dashed border), dotted when not running, "◉n" for readers, full name on hover. Unbound IEDs keep
  name tags left of the drawing. Marks scale at most 1.5×; links start at the mark centre.
- Update check (ADR-0071): license server `GET /api/v1/release` (precomputed signed body,
  `scadabench-release-v1\n` signature prefix, per-IP limit, ETag/304, 5-minute public cache, host
  allow-list), `license-server release` (stdin JSON), `deploy/release-record.ps1`,
  `owner.ps1 -Action release`, run at the end of `website/deploy.cmd`. App: `updates/` service
  (daily, 6 s timeout, nothing about the PC sent, last good answer kept), `/api/v1/app/update`,
  one dismissible line "Update 0.1.14" above Settings, Settings › About section with Check now and
  the once-a-day switch. `version.py` is `0.1.13+dev`; `protected_source.py` replaces any `+dev`.
- `studio/tests/conftest.py` keeps the background check off during tests.

## Evidence

| Check | Result |
| --- | --- |
| `python studio/tools/verify.py` | lengths, architecture, documentation, history, specifications PASS |
| Frontend tests (`node --test`) | 165 pass (new `pickerMemory.test.ts`; LAYER-10 test rewritten for corner marks) |
| Python tests (repo root) | 511 pass, 11 skipped (first run: 1 fail, a fake license service without `server_url`; fixed with a lazy fallback) |
| Go `go vet`, `go test ./...` | pass; new tests: signed with domain prefix and never valid as a plain payload, 304 on ETag, no CORS, third check in a minute 429, foreign host / http / bad version / bad hash / user-info / unknown field refused |
| Cross-language | record from `release-record.ps1` → `license-server release` → signed body → verified by the Python checker with a test key |
| Dev app in browser, 1366×768 (separate dev instance on 8775/5190 because the installed app held 8765) | Select mode ticks two IEDs, footer and review dialog with impact lines; Cancel deleted nothing. SLD: "G" inside QB1's corner, QB2 uncovered. QB1 *Change* opened on C03_Y1105 (was "Choose an IED"), *Chosen ✓* marked, *Use this* kept the list and search, *Undo* restored `Disc0XSWI1.Pos.stVal`, reopening kept "XSWI1.Pos". Network: "Looking for network cards…" and "Refreshing…" seen. About: live server answered "did not answer" (endpoint not deployed); with a stored sample 0.1.14 answer the sidebar showed "Update 0.1.14 ×", × hid it; real state restored |

## Limitations

- The live license server is not redeployed by this change: until `deploy-server.cmd` runs, the
  app shows "scadabench.com did not answer the version question" and `deploy.cmd` prints a warning.
- Cloudflare does not cache JSON by default; the 5-minute public cache helps only with a cache rule.
- Not covered by this change: gateway Connect/Start busy words, CSV playback, signal selects in
  Export SCL / sequences / sources, master and served point tables that show "0 points" before the
  first poll, GOOSE/SV traffic first load. Undo after a bulk delete is the recovery file, not a button.
- No EXE built (owner rule). Screenshots at 1920×1080 were not taken.
