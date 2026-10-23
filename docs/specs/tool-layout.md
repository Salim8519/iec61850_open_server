# Spec: Tool layout and list filters

Decision: [ADR-0064](../decisions/ADR-0064-tool-pages-fill-window.md).
Change: [0093](../changes/2026/0093-tool-layout-and-filters.md).
Owner request (2026-10-18): lists must be visible without page scrolling; filters on every list;
remember every choice; beginner hints everywhere, but not as blocks that push lists down.

## Shell

- **Fill page.** Open tools (SCADA gateway & master, Find IEDs, IED Explorer) and Devices use
  `content fill-page`: the page is exactly the window height and never scrolls.
- **Panes** (`styles/fill.css`):
  - `fill-split`: a side column (`--side-width`, 256 px; SCADA 240 px, Devices 280 px, Explorer
    300 px) and the main area. Its single row is `minmax(0, 1fr)`.
  - `side-pane`: the settings or list column, with its own scroll. Its `side-foot` (status,
    inline errors, the primary button) stays in view at the bottom.
  - `fill-panel` + `panel-body`: a card whose last part (table, log, tree) takes the rest of the
    height. `.table-scroll` inside it has no fixed `max-height`.
  - `fill-column` / `fill-grow`: panes stacked in one column (Explorer: node over watch list).
- **Breakpoints.** Targets 1366×768 and 1920×1080. Sizes are the fitted layout (ADR-0111, `data-max`
  tokens on `<html>`). Below 1000 px wide or 620 px high the panes
  stack, the page scrolls and tables are capped at 70 vh.
- **Hints.** A paragraph longer than one line moves behind a `HelpTip` "?" next to the thing it
  explains. Empty states keep their explanation as text.

## Filter bar

- One component, `ui/FilterBar.tsx`; the model is `ui/filterState.ts` (pure functions).
- Parts, left to right: search box, one-of chip groups (optional counts, e.g. "Real 2"), toggle
  chips, dropdowns (shown only when there is a real choice), the count ("12 of 340" while
  filtered, otherwise the total) and Clear (only while filtered). Lists may add buttons (Pause).
- An active chip shows a tick and a solid accent border; an inactive toggle has a dashed border.
- Search matches every typed word in any shown field, ignoring case.
- Lists and their filters:

| List | Filters | Where it filters |
| --- | --- | --- |
| SCADA points (gateway) | search, IED, point type (ADR-0095), Bad quality, Not served, Own points | page |
| SCADA points (master, converter) | search, point group, point type (ADR-0095), Has a command, Bad quality, Watched only | page |
| Traffic | search, Both/Sent/Received, Errors only, Hide keep-alives (+ Pause, not remembered) | page |
| Commands | search, All/Done/Refused, Anyone/From SCADA/From me | page |
| Find IEDs results | search, Answered, IEC 61850 only, Not in project, vendor | page |
| Explorer tree | search, All/Controls/Measured/Settings (by logical-node class) | loaded nodes |
| Explorer watch list | search, Changed in last minute, Bad quality | page |
| Devices list | search, All/Real/Sim/Off with counts, Has problems | page |
| Data points (simulated IED) | search, logical device, report, point type with counts (ADR-0095), Bad quality, In a dataset, Hide quality / time | backend, all points |
| Data points (real IED) | search, point type | backend, first 50 shown |
| Compare with SCD | search, Differences only (default)/Missing/Extra/Type mismatch/Show all, point type | page |

## Remembered filters

- Each list has its own name (`useListFilter('<name>', spec)`). Chips and dropdowns are saved in
  `preferences.json` (`list_filters`) through `PUT /preferences`, never in browser storage.
- The typed search and Traffic's Pause are not remembered. A remembered dropdown value that is not
  in today's list counts as "all"; an unknown or damaged stored value falls back to the default.

| ID | Rule | Enforced by |
| --- | --- | --- |
| LAYOUT-01 | Open tools and Devices fill the window; only panes scroll; narrow windows fall back to page scroll. | `studio/tests/frontend/toolLayout.test.ts::open tools and Devices use the fill-the-window page, only lists scroll` |
| LAYOUT-02 | SCADA views keep settings in a scrolling column with status, inline errors and Start/Connect at its bottom; the converter has two steps. | `studio/tests/frontend/toolLayout.test.ts::the SCADA tool keeps settings in a column and status with Start at its bottom` |
| LAYOUT-03 | Every list uses the one FilterBar with its own remembered list name. | `studio/tests/frontend/toolLayout.test.ts::every list uses the one FilterBar with its own remembered list name` |
| LAYOUT-04 | Filter choices are remembered by the backend, never by browser storage; search and Pause are not remembered. | `studio/tests/frontend/toolLayout.test.ts::filters are remembered by the backend, not by the browser`, `studio/tests/frontend/filters.test.ts::remembered filters keep chips and dropdowns, never the search text or Pause`, `studio/tests/integration/test_preferences_api.py::test_list_filters_are_remembered_per_list_and_survive_a_restart`, `studio/tests/integration/test_preferences_api.py::test_bad_list_filters_are_refused_and_damaged_ones_ignored` |
| LAYOUT-05 | Active chips show a tick and a solid border, not colour alone; Clear appears only while filtered. | `studio/tests/frontend/toolLayout.test.ts::active chips are shown with a tick and a solid border, not colour alone`, `studio/tests/frontend/filters.test.ts::the filter bar starts empty and offers Clear only after a change` |
| LAYOUT-06 | Each list's filter rule matches the table above. | `studio/tests/frontend/filters.test.ts::SCADA points filter by point type, IED, bad quality and not served`, `studio/tests/frontend/filters.test.ts::master points filter by group, point type, quality, command and watched`, `studio/tests/frontend/filters.test.ts::traffic filters direction, errors and keep-alives`, `studio/tests/frontend/filters.test.ts::commands filter by result and by who sent them`, `studio/tests/frontend/filters.test.ts::Find IEDs results filter by answer, IEC 61850, project and vendor`, `studio/tests/frontend/filters.test.ts::Explorer tree groups logical nodes by class; the watch list tracks changes`, `studio/tests/frontend/filters.test.ts::Devices list, data points and comparison filters` |
| LAYOUT-07 | The Data points filters are read-only backend filters over all points. | `studio/tests/integration/test_mapping_api.py::test_data_point_display_filters_only_narrow_the_list` |
| LAYOUT-08 | The Devices list row is its own component with a hints slot; detail header and stats are one line. | `studio/tests/frontend/toolLayout.test.ts::the Devices list row is its own component with a slot for extra hints` |
| LAYOUT-09 | The migration and recovery notices are one dismissible line with details one click away. | `studio/tests/frontend/toolLayout.test.ts::the migration notice is one dismissible line with its details one click away` |
