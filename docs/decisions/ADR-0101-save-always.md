# ADR-0101: Save always: tool settings and lists save by themselves

Accepted: 2026-10-22. Owner decision in chat ("save always", no delay). Change
[0161](../changes/2026/0161-save-always.md). Frontend only; engine routes and data are unchanged.

## Problem

The project saved by itself, but four places had their own Save button: SNMP device Settings (and
Traps), the gateway point list, Test sequences and the gateway Add point form. Edits were lost when the
user switched tabs, and the UI behaved two ways.

## Decision

1. A valid change is sent at once (no timer). One request is in flight per place; a newer value waits
   as the single trailing save, so the latest value wins and requests never run side by side
   (`state/autoSave.ts`, `state/useAutoSave.ts`). Leaving a tab never drops a valid edit.
2. A wrong value keeps its text, shows one red ⚠ line with words under the field (`role="alert"`) and is
   not sent. A quiet status beside the title reads "Saving…", "✓ Saved" or "⚠ Not saved yet".
3. The gateway list cannot change while the gateway runs: a refused edit is kept and reads "Will save
   when the gateway stops", then saves when it stops. Whole-list actions (Remove, Remove all, Propose,
   Import CSV) get a one-step Undo, replacing the old Discard.
4. Running SNMP devices restart on each save, so while they run a typed field saves when it is left
   (blur or Enter); stopped devices save on every change. The page says so.

## Compatibility

No data format change. Users no longer see Save buttons in these places; existing saved data loads as
before.
