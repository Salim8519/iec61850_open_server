# 0238 White sidebar with depth; What's new after an update; click-rule sweep

Date: 2026-11-25. Owner: "The side bar should be kept white to avoid huge changes in user ui, only this
change, just a bit depth too, everything else is amazing"; "at every update we show what's new in this
update and the old updates too, on small pop up at the start"; then "go ahead" on the plan (sidebar,
What's new, rule sweep, fit check). Decisions: [ADR-0147](../../decisions/ADR-0147-raised-panels.md)
(amended), [ADR-0148](../../decisions/ADR-0148-whats-new.md). Spec: [whats-new](../../specs/whats-new.md).
Layers: frontend, Go preferences, website changelog.

## What changed

- **Sidebar** (`styles/layout.css`, `styles/tokens.css`): white again, with a soft shadow on its right
  edge; the open place is framed, blue and raised. The dark rail tokens are gone.
- **What's new**: release notes moved from `website/src/pages/changelog.astro` into
  `frontend/src/copy/releaseNotes/` (three files by version series, plus `index.ts` with `versionCode`,
  `releaseOf`, `releasesUpTo`); a 0.1.37 entry lists this branch's changes. `features/whats-new/`:
  `whatsNewChoice` (show, mark or none), `useWhatsNew` (reads preferences and engine capabilities,
  stores `whats_new_seen` on close), `WhatsNewDialog` (this version, "Earlier updates" fold, link to
  scadabench.com, Got it). Help › What's new reopens it. The website changelog renders the same list up
  to `SITE.version`; `astro.config.mjs` lets its dev server read the studio folder.
- **Engine** (`internal/project/preferences.go`): Go-only `whats_new_seen`, 0 to 999999999.
- **R8**: `button.wire` (double border) on SLD Open/Close, raise/lower, point-panel commands, Explorer
  commands and Operate, SCADA master Send and real-device commands; disabled ones stay dashed.
- **R9**: table rows no longer light up on hover (no row is clickable as a whole).
- **R12**: pop-up dialogs carry `--shadow-pop`.

## Evidence

- Frontend tests 521/521 (new `whatsNew.test.ts`; sidebar-white check; interlock test accepts `wire`);
  `tsc --noEmit` clean; `go test ./internal/project/` passes (new `TestWhatsNewSeenPreference`;
  the default preferences file test unchanged). Specs, file lengths: PASS.
- Website `astro build`: the changelog has the same 31 headings and notes as the old hand-written page,
  and 0.1.37 is not shown while the site is 0.1.36.
- Chromium on the built UI with a mocked API at 1600x900 and 1366x768 (fit 0.95): white sidebar with
  edge shadow; What's new opened for `0.1.37+dev`, the fold listed 0.1.36 and older, Got it closed it;
  Help menu shows Take the tour, What's new, Video tutorials.

## Limitations

- Not yet seen in the real Windows app; the 80% fit size (about 1280x640) was not captured.
- R10 is not finished: 134 of 201 disabled buttons give no written reason (many only while busy). They
  need per-button wording, planned as its own change.
- Under the Python engine What's new opens only from Help.
- `studio_app/version.py` still says 0.1.35+dev while the site says 0.1.36 (left as found).
