# 0224 Screen trials U1-U5: fewer requests and less work in the browser

Date: 2026-11-20. Owner: "do all of them, once i wake, i should see all ready and no coding left,
just testing". Plan: [speed and network trials plan](../../testing/speed-trials-plan-2026-11-20.md),
screen trials U1-U5. Results: [trial results](../../testing/speed-trials-results-2026-11-20.md).
Layers: frontend, Go engine (push catalog, project and live routes). No decision record: the plan
names none for these ideas, every change is additive and nothing the user sees changes.

## Change

- **U1, runtime-only actions (Consider, done).** Setting a value, Pulse, Resume generator, the SLD
  sheet's Test buttons, a GOOSE send and Run sequence go through `useStudio().command`: busy and
  the error line as before, but no `GET /project` afterwards (`state/runCommand.ts`, provided by
  App). They change only live values; `u1_revision_check.mjs` shows the project revision and body
  unchanged after each. Operate and tap commands stay a `perform`: the engine remembers the last
  position and command options in the project (`runtime/equipment_control.go`, `NodeData`).
- **U2, one diagram edit (Consider, done).** New Go-only push topics `project.history`,
  `project.autosave` and `goose.members` (`push/catalog_project.go`); the push catalog now also
  reads the project and platform routes (`app/compose.go`, `wire_push.go`). File › Save project and
  the auto save switch tell the stream through `projectapi.Deps.Saved` (change `project.saved`).
  In the browser, views keyed on the revision subscribe once and call `liveFeed.again(path)` when
  it changes (`api/useAskAgain.ts`): nothing for a pushed path (the engine re-reads every topic
  after `project_changed` and sends only a changed answer), one GET for a polled one as before.
  `useHistory` and `useAutoSaveSwitch` read through the live feed.
- **Found on the way (U2): Undo right after a drag.** Undo saves the drawing first; sending the
  same drawing again made the engine record an empty step, so the first Undo did nothing visible.
  `state/diagramSave.ts` now sends nothing when nothing changed and skips a queued save of a drawing
  the save ahead of it already sent.
- **U3, lazy pages (Drop).** Built and measured, then removed: no faster usable start, and loading
  page stylesheets later changed the look (see Evidence). The app loads as before.
- **U4, point tables (Consider, done).** A keyed topic's frame is handed to the live feed with its
  row texts (`pushStream` `parts`, `pushClient`); `api/rowParse.ts` parses only row texts it has
  not seen, so unchanged rows keep their objects. Gateway `PointRow` and the new `MasterPointRow`
  are memoised (`ui/sameProps.ts`); their handlers read the latest list (`usePointList.latest`,
  functional state updates) and the row index and own-point list are plain props.
- **U5, idle polling (Consider for two parts, done; Explorer dropped).** `network.lab` (re-read every
  3 s while shown, like the poll it replaces) and `goose.messages` (param `ied`, every value change
  and goose events, at most once a second) are push topics; `GET /goose/messages?ied=` answers like
  `/ieds/{id}/goose/messages` and is what the frontend now asks. The Explorer's 2 s read stays on its
  own timer: ADR-0135 fixes the 2 s read, the timer already skips hidden windows, and the shared
  poller would add an extra read whenever the window shows again, so no request can be saved.

## Evidence

Two engines from the same commit (today's frontend vs this change) on a 20-IED loopback project,
headless Chrome at 1366x768, rounds alternating; full set-up and per-round numbers in the
[trial results](../../testing/speed-trials-results-2026-11-20.md).

| Idea | Today | This change |
|---|---|---|
| U1 set a value: click to ready, requests | median 27-30 ms, 5 requests | 20-25.5 ms, 1 request |
| U1 SLD Test button | median 26.5-27 ms, 4 requests | 18.5-20 ms, 2 requests |
| U2 one diagram drag: requests | 8-9 | 2 |
| U3 lazy pages | ready median 319-357 ms | 318-494 ms, look changed: dropped |
| U4 5000-point gateway list: main thread | 19.2-20.5 ms/s | 10.3-11.8 ms/s |
| U5 Lab card page idle: requests a minute | 20 (`/network/lab`) | 0 |
| U5 SLD IED sheet idle: requests a minute | 30 (`/ieds/{id}/goose/messages`) | 0 |

- Go: `go vet` clean; `go test` of push, api/projectapi, api/liveapi, api/labapi, app all pass.
  New tests: `TestProjectTopics`, `TestSaveAndAutoSaveSwitchTellSaved`, `TestGooseMessagesByQuery`;
  `TestCatalogTicksOnlyWithCondition` and the live area coverage list updated.
- Frontend: `tsc --noEmit` clean; all 509 frontend tests pass under Node 24 (new
  `speedTrialsScreen.test.ts`, `speedTrialsLive.test.ts`, one more in `fixPlanA1A2.test.ts`; the
  Remove assertion in `saveAlways.test.ts` now expects the latest list). Production `npm run build`
  succeeds; `verify.py` passes (a first run under load failed two unrelated tests, licensing
  `cancelled_stays_cancelled` on a passing temp file and netcfg's live card listing while another
  test engine changed the Lab card; both pass alone and in the second full run).

## Limitations

- Measured on this PC only, headless Chrome, loopback IEDs; the packaged window (WebView2) and a
  real network were not used. The push topics need the Go engine; the frozen Python engine keeps
  polling exactly as before.
- Rows parse row by row only while the push stream feeds a table; a polled answer (push offline)
  is parsed whole, as before. A list with unsaved edits merges values into new row objects, so its
  rows render again each second, as before.
- U3 numbers were taken with the browser cache off (a first start); not repeated warm, as the
  look difference alone rules the idea out.
