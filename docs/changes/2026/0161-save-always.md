# 0161: Save always: SNMP settings, gateway point list, sequences and Add point save by themselves

Date: 2026-10-22. Decision [ADR-0101](../../decisions/ADR-0101-save-always.md); asked by the owner after
the UI review [0160](0160-ui-review-no-clipping.md) found SNMP Settings edits lost on a tab switch.
Frontend only; engine routes and data unchanged.

## What changed

- New `state/autoSave.ts` (pure) and `state/useAutoSave.ts`: a valid value is sent at once; one request
  in flight, a newer value waits as the single trailing save; `ui/SaveStatus.tsx` shows "Saving…",
  "✓ Saved" or "⚠ Not saved yet" beside the title.
- SNMP device Settings and Traps share one draft that survives tab switches; engine rules are mirrored in
  `deviceRules.ts` so a wrong address, port, community or v3 user shows a red line under its field and
  is not sent. While devices run (each save restarts them) a field saves when it is left.
- Gateway point list: edits, Propose, Import CSV and Add point save themselves; Remove, Remove all,
  Propose and Import CSV get a one-step Undo; an edit refused because the gateway runs reads "Will save
  when the gateway stops" and saves after Stop; an invalid list keeps its problems listed with "Discard
  unsaved changes". Send buttons say "Wait until the point list is saved" while a save is pending.
- Test sequences save when their first step is added; an empty name or removing the last step shows a
  red line and is not saved.

## Evidence

- Frontend tests 372/372 on the merged branch (new `autoSave.test.ts`, `saveAlways.test.ts`); tsc clean.
- Worker browser run (Playwright, 1366×768, sample IED and demo switch): rename saved at once and
  survived tab switch and reload; 18 keystrokes gave 18 PUTs with never more than one in flight; bad
  address, port 70000 and short v3 passwords kept their text and were not saved; gateway Remove all then
  Undo restored all rows; locked case (refusal forced at HTTP level, then a real Start) kept the edit
  across a tab switch and saved after Stop; sequences saved on step add.
- Merged-branch check (demo station): SNMP "✓ Saved" with no Save button, rename survives tab switch and
  reload, "999.1.1" shows the red line; gateway Propose shows "✓ Saved" with no Save button.

## Limitations

- An invalid gateway draft is not kept when you leave the tool; a draft held because the gateway runs is
  kept for the page session only (lost on a browser reload).
- While SNMP devices run, a typed value that has not left its field yet is lost on an immediate reload.
