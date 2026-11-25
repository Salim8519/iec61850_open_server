# ADR-0148: What's new after an update, from one list of release notes

Accepted: 2026-11-25. Owner, 2026-11-25: "at every update we show what's new in this update and the old
updates too, on small pop up at the start". Change
[0238](../changes/2026/0238-white-sidebar-whats-new-rule-sweep.md). Spec [whats-new](../specs/whats-new.md).

## Problem

Release notes existed only as hand-written HTML on the website changelog page. The offline app could not
show them, and two copies would drift.

## Decision

1. **One list.** The notes move to `studio/frontend/src/copy/releaseNotes/` (TypeScript data, split by
   version series to keep files short). The app bundles them, so they work offline. The website changelog
   imports the same list and shows versions up to `SITE.version`, so the next version can be written
   before its release without appearing online.
2. **Preference.** `whats_new_seen` is a Go-only preference (like `display_size`): absent from the default
   document, a whole number `major*1000000 + minor*1000 + patch`, 0 to 999999999.
3. **Screen.** At start the app compares the engine's version (`/app/update` `current`) with
   `whats_new_seen`; a newer version with notes opens a pop-up; a first start records it quietly because
   the tour opens instead. Help › What's new reopens it at any time.
4. **Python engine.** It cannot store the preference and answers no capabilities, so the pop-up does not
   open by itself there (it would return at every start); Help still opens it. No browser storage: the
   packaged app's port can change (existing rule).

## Compatibility

The default preferences file stays byte-identical to the Python engine's. The website's changelog page
renders the same headings and notes as before (checked against the old page). The website dev server may
read the notes from the studio folder (`vite.server.fs.allow`).
