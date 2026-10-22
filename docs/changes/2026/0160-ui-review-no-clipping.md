# 0160: UI review of every page; nothing clipped at 1366×768, safer edits, plainer words

Date: 2026-10-22. A page-by-page review asked for by the owner after [0159](0159-snmp-cards-goose-skipped-words.md):
three reviewers clicked every tool, tab and button in private copies of the dev app (demo station only)
and measured clipped text with `scrollWidth`. Design: [layout](../../design/layout.md) (no clipped
text, 1366×768 first), [interactions](../../design/interactions.md). Frontend only; the Go engine and the
frozen Python backend are unchanged.

## What changed

- SNMP values: label and OID text wrap inside their fixed columns instead of running over the next
  column (`.gateway-points .editable-text` is one-line for the gateway; the SNMP tables reuse that class).
- SNMP values: the Changes column gets 14 px from the OID column so "Counts up time" shows whole; the
  search hints are shorter ("Search name, OID or value", "Search name, OID, value, meaning").
- SNMP "Import a device" panel: one full-width column; the file button wraps; the name field fits.
- Gateway command rows: the value cell no longer clips "Open…"; the two send buttons wrap inside it.
- Protocol converter: tighter tab padding so six tabs and Full view fit 1366 px; the row wraps on a
  narrower window instead of cutting "Full view".
- Icons in buttons never shrink (`button svg { flex-shrink: 0 }`).
- IED Explorer watch list: its "?" tip was restyled as a toolbar button and shrank to a dot.
- Filter and type dropdowns: an active dropdown no longer loses 2 px of text width (inset shadow);
  the filter dropdown may be 260 px; the Sequences point-type dropdown fills its cell.
- Gateway: "Save point list" is disabled while the list is locked (it always failed); a CSV with no
  usable rows leaves the list as it was and says so. Master: Watch all/none are off with no points.
- Words: SNMP "39 re-read" → "39 kept up to date"; Explorer chip "Bad quality" → "Needs attention"
  (it also lists breakers between positions); remove buttons are named "Remove <ref> from the watch
  list" and "Remove the point at address N" instead of "Stop watching" and an internal id.
- GOOSE traffic: a MAC address and "state N" never break in the middle.
- Activity: the event cell stays a table cell (rows line up) and times stay on one line.
- Collapsed sidebar (≤1100 px): only the engine dot shows; its words stay in the hover text.
- Integer test values use step 1 in the number field.
- SLD equipment sheet: Rotate, Duplicate, Show signal flow and Delete wrap inside the 380 px sheet
  (Delete was outside the window and the Advanced colour rows were cut).
- SLD equipment name: clearing the field no longer sends an empty name (the project refused it with an
  "Invalid request" banner and the typed text was appended to the old name); it says "Type a name: it
  cannot be empty." and leaving it empty brings the saved name back. Typing still shows on the diagram live.
- Build from SCD: a "Needs a link" line shows each reason once (display only; engine unchanged).
- Devices: "Copied" lasts 2 s and is cleared when another IED is selected; the Data points search hint
  is "Search name or path".

## Evidence

- Frontend tests 327/327 (new `uiNoClip.test.ts`); `tsc --noEmit` clean.
- Browser at 1366×768 (private dev copy, demo data): a clip scan of Gateway, SNMP Simulate/Read,
  Explorer, Find IEDs and GOOSE traffic finds nothing; SNMP name/OID cells fit; command cells hold both
  buttons with no sideways table scroll (816 = 816); converter tab row 47 px high (one line); import
  panel 193 px inside a 222 px column; Activity cells all 44 px; at 1000×640 the sidebar shows only the dot.
- SLD (private copy A): the sheet body is 364 = 364 px wide with Advanced open and Delete wraps
  inside; clearing the name shows the inline line and no banner, then typing gives "Feeder breaker"
  (renamed back afterwards, "All changes saved"); leaving it empty restores the saved name.
- Playwright with clipboard permission: "Copied" shows, is gone after 2.4 s, clipboard 127.50.0.2:102.
- verify.py: all PASS (one licensing scenario failed once while three test copies loaded the PC;
  it passed alone and in the full rerun).

## Not changed (reported, need a design decision or carry more risk)

- SNMP errors show in the shared top line, not beside the field; unsaved SNMP Settings are lost on
  tab switch; gateway Commands log shows "unknown type" after a protocol switch; Find registers has
  no Stop; menus and the Activity panel and the tour lack full keyboard focus handling; typed network
  addresses are not checked for duplicates (engine, Python and Go alike); Test point saves create
  no-op revisions; validation messages appear after Save instead of beside the field.
- Devices/SLD: the first diagram edit after opening the page cannot be undone (the push replay resets
  the undo history); Undo after deleting a wired item brings the item back but needs a second Undo for
  its wires; new items are placed at fixed spots (sometimes under the sheet); the GOOSE "?" bubble is
  cut by its scroll area; disabled controls while running lack a reason; Tidy up needs two presses;
  the SLD toolbar wraps with orphan dividers; raw validator text in File check; default tile names are
  raw ids; the Add menu scrolls at 1366×768; device list rows use role="listitem" on buttons.
- "address(es)" in Find IEDs comes from the engine and stays (Go must match Python byte for byte).
