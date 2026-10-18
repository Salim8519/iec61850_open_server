# 0093: Tool pages and Devices fill the window; one remembered filter bar on every list

Date: 2026-10-18. Decision: [ADR-0064](../../decisions/ADR-0064-tool-pages-fill-window.md).
Spec: [tool layout](../../specs/tool-layout.md); updated [tools page](../../specs/tools-page.md),
[UI structure](../../specs/ui-structure.md), [gateway](../../specs/gateway.md),
[SCADA master](../../specs/scada-master.md), [Find IEDs](../../specs/find-ieds.md),
[IED Explorer](../../specs/ied-explorer.md), [layout](../../design/layout.md),
[interactions](../../design/interactions.md).

Owner request: the SCADA tool, Find IEDs, IED Explorer and Devices need a lot of page scrolling
before a list is visible. Chosen mockups: settings column + list filling the rest (B), results
directly under a one-row form (C), and filters on every list, remembered like other settings.

## What changed

- **Shell.** `styles/fill.css` adds the fill-the-window page and its panes (side column, panel
  with a filling table, stacked panes). `.table-scroll` keeps its old height on other pages but
  fills its pane here. Narrow windows (below 1000×620) stack the panes and scroll the page.
- **SCADA gateway & master.** Act as moved into the header row with a one-line caption; the
  settings column holds protocol, address/port/ASDU/minutes, who may connect, real commands,
  Advanced, the compact "IEDs are not running · Run the IEDs" line, the firewall check, and at its
  bottom status, inline errors and Start/Connect. The converter shows two numbered steps. Long
  paragraphs moved behind "?". `GatewayStatusBar` and `MasterStatusBar` are removed.
- **Find IEDs.** One toolbar row with the "N addresses" preview, a thin progress line, results
  filling the window, and an empty state. **IED Explorer.** Toolbar row; tree column; node with the
  watch list pinned below (the watch interval is now a compact "Every 1 s" dropdown and Clear is a
  labelled trash button).
- **Devices.** Searchable, filterable list in its own column (`DeviceListRow.tsx` with a `hints`
  slot for the parallel GOOSE/MMS work), one-line detail header, stats line, full-height Data points.
- **Filters.** `ui/FilterBar.tsx` + `ui/filterState.ts` on 12 lists (see the spec table), each
  list's rules in a pure `*Filters.ts`. Remembered per list in `preferences.json` through
  `/preferences` (`projects/list_filters.py`), restored on the next start.
- **Data points filters in the backend.** `GET /ieds/{id}/signals` gained the read-only `ld`,
  `bad_quality` and `in_dataset` parameters and returns `devices`, so these filters cover all
  points of a large IED, not only the loaded page of 100.
- **Notices.** Migration and recovery notices are one dismissible line; their details (and the
  recovery file path) open over the page.

## Evidence

- Source app in development mode, own backend (port 8781) and data folder, public demo station
  `marketing/videos/demo/scadabench_demo.scd` (FEED1, FEED2, BUS1, BUS2, TR1, TR2) only.
  Measured in the in-app browser with `.page` scrollHeight − clientHeight:

| Page | 1366×768 | 1920×1080 |
| --- | --- | --- |
| Devices (Overview, Data points) | page scroll 0; Data points table 394 px with 100 rows inside | page scroll 0; list and detail 986 px |
| SCADA gateway, running, 374 points | page scroll 0; header 43 px and filter bar 58 px (one row each), table 389 px (≈6 rows), Stop visible | page scroll 0; table 701 px (≈13 rows) |
| SCADA master, not connected | page scroll 0; empty state "Points appear here by themselves after Connect." | — |
| Master connect to a closed port | inline in the column: "No answer from 127.0.0.1:2404 within 10 s. …" | — |
| Converter | page scroll 0; "1 · Read the device" and "2 · Serve to SCADA" with their own buttons | — |
| Find IEDs, This PC, 6 found | page scroll 0; toolbar 93 px, results table 409 px | page scroll 0; table 721 px |
| IED Explorer, 127.50.0.2, 40 watched | page scroll 0; tree, node and watch list each scroll | page scroll 0; tree 630 px, watch table 285 px |
| 900×700 fallback (gateway) | panes stacked, page scrolls, table capped at 70 vh | — |

- Filters on the live list: Commands → "14 of 374", Commands + Bad quality → "0 of 374", Clear →
  "374". Measured → saved as `{"kind": "measured", …}` in `list_filters`; after a page reload the
  gateway opened with Measured active, "232 of 374".
- Found and fixed while measuring: screen-reader labels inside long tables stretched the page
  (2,220 px of page scroll in Explorer) because they were positioned against `.page`; scroll
  panes now contain them. The split row is pinned to the window height.
- Tests: frontend `node --experimental-strip-types --test studio/tests/frontend/*.test.ts` 140 pass
  (new `filters.test.ts` 11, `toolLayout.test.ts` 7; `testMode.test.ts` follows the moved row);
  Python unit + integration 378 pass, 15 skipped (new
  `test_list_filters_are_remembered_per_list_and_survive_a_restart`,
  `test_bad_list_filters_are_refused_and_damaged_ones_ignored`,
  `test_data_point_display_filters_only_narrow_the_list`); `tsc --noEmit` clean;
  `python studio/tools/verify.py` all PASS. No protocol tests were needed: no wire behaviour changed.

## Limitations

- **Filtered on the page, not in the backend:** SCADA points, master points, traffic (the last
  1,000 lines), commands, Find IEDs results, the Explorer tree (only nodes already loaded, by
  logical-node class letter), the watch list and Compare. "Changed in last minute" counts changes
  seen while the page was open. Real-IED Data points still list the first 50 matches.
- **Compare** shows at most 400 rows as before; its "Differences only" count includes dataset and
  report differences that are listed under the table, not as rows.
- At 1366×768 the Traffic filter bar and the converter's header caption wrap to a second row.
- Removed: the old Data points "Descriptions (DC)" kind (search still finds those points).
- **Pre-existing, not changed here:** after a failed Explorer connect, a worker that is still
  associated answers "Already connected" until it restarts; in development React runs the "open
  from Find IEDs" effect twice, which can trigger it. Recorded in known issues.
- Screenshots were checked visually in the in-app browser, not saved into the repository (they
  would show local network adapters). The owner's visual acceptance is pending.
