# Spec: What's new

Decision: [ADR-0148](../decisions/ADR-0148-whats-new.md).
Change: [0238](../changes/2026/0238-white-sidebar-whats-new-rule-sweep.md).

## Purpose

After an update the user sees, once, what changed in the version they now run, with earlier versions
one click away, and can open the same list again from Help.

## Behaviour contract

- **One list.** The notes live in `studio/frontend/src/copy/releaseNotes/`, newest first, one entry per
  version (version, title, plain-text notes starting with New:, Fixed:, Changed:, Improved: or Faster:).
  The website changelog renders the same list up to its own version, so an unreleased entry stays hidden.
- **When it opens.** At start, for the version the engine reports (`0.1.37+dev` counts as 0.1.37), when it
  has notes and is newer than `whats_new_seen`. A first start shows the tour instead and records the
  version quietly. It never opens over the tour.
- **Remembered.** Closing stores the version code in `whats_new_seen` (Go engine only). The frozen Python
  engine cannot store it, so there it opens only from Help.
- **Layout.** A pop-up: this version's title and notes (kind as a flat label), an "Earlier updates" fold,
  a link to all updates on scadabench.com and one Got it button. Help › What's new reopens it.

| ID | Rule | Enforced by |
| --- | --- | --- |
| NEWS-01 | Notes are newest first, one per version, plain text with a title. | `studio/tests/frontend/whatsNew.test.ts::release notes are newest first, one entry per version, each with words` |
| NEWS-02 | A +dev build counts as its release; lookups stop at the given version. | `studio/tests/frontend/whatsNew.test.ts::version codes and lookups treat a +dev build as its release` |
| NEWS-03 | The website changelog renders the same notes up to the site version. | `studio/tests/frontend/whatsNew.test.ts::the website changelog is rendered from the same notes, up to the site version` |
| NEWS-04 | It opens once after an update and never on a first start. | `studio/tests/frontend/whatsNew.test.ts::What's new opens once after an update, never on a first start` |
| NEWS-05 | The engine stores whats_new_seen as a version code and refuses anything else. | `studio/engine/internal/project/preferences_whatsnew_test.go::TestWhatsNewSeenPreference` |
