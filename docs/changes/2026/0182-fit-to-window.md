# 0182 Fit to window: small windows show the big-window layout evenly smaller

Date 2026-10-23. Frontend + one Go-only preference. Decision [ADR-0111](../../decisions/ADR-0111-fit-to-window.md),
spec [display-fit](../../specs/display-fit.md). No protocol code changed, so no MMS client check applies.

## Problem (owner request)

On a 14-inch laptop (1920×1080 at 150%, app window about 1280×640) Devices › Data points showed the
header in two rows (110 px), the filter bar in two rows (99 px), a cut subtitle (310/326 px) and a
149 px table (2 full rows). The owner wants the same view as on their laptop (maximized, ~1707×880)
on every screen, without a different compact design.

## Change

- `app/displayFit.ts` (pure): fit = min(w/1440, h/800, 1) in 0.05 steps, never below 0.8, 1 from
  0.975; `layoutLimits` gives `data-max` tokens for the fitted layout (none for an unmeasured window).
- `app/installFit.ts`: applied before the first render (`main.tsx`) and on each resize, directly
  (a minimized or hidden window gets no animation frames); loads `display_size`.
- `styles/tokens.css` `:root { zoom: var(--fit, 1) }`; `.app` and every vh/vw divided by `--fit`;
  `.react-flow` cancels the zoom, its panels and the stray note take it back (`canvas-interaction.css`).
- Every size `@media` → `:where(:root[data-max~="…"])` (layout, workspace, fill, explorer, gateway,
  license gate); `toolLayout.test.ts` LAYOUT-01 pins the new form.
- `ui/zoomSpace.ts` used by Menu, HelpTip, VirtualRows, TestMenu and the tour.
- Settings › Display (`features/settings/DisplaySection.tsx`, `copy/display.ts`): *Fit to window
  (recommended)* / *Always 100%* with "Now 80% · this window is 1280 × 640".
- Go: `display_size` added to `goOnlyPreferenceKeys`, range 0..1 (`internal/project/preferences.go`,
  `preferences_display_test.go`).
- Tools: `studio/tools/layout_snapshot.js` (element-by-element layout snapshot and compare);
  `clip_check.js` selectors fixed (`.segmented-control button`, `.tabbar button`, `.nav-item`,
  `.filter-select`, `.topbar-title p`, `.notice-text`; the old `.tab` and `.segmented button` matched nothing).

## Evidence

- Unchanged where it should be: master and this change on the same data copy, element by element
  (boxes + text) on Devices Overview, Data points, SLD, Test, Tools and Settings at 1920×1000,
  1707×880 (owner laptop), 1440×900 and 1600×800: all identical; Settings differs only by the new
  Display section tab.
- Same view when scaled: 1280×640 (fit 0.8) against 1600×800 × 0.8: SLD, Test, Tools, Devices keep
  every element on the same row (worst shift 1-6 px); Data points: one long reference wraps one row
  lower (the native scrollbar does not scale).
- Data points at 1280×640: header one row (51 px), filter bar 79 px, table 285 px, 7 full rows
  (was 2), subtitle not cut. 1280×587 (browser tab) no longer stacks the panes.
- clip_check: 0 clipped at 1280×640 (80%), 1366×690 (85%), 1280×587 and 1920×1000, all pages and
  Settings › Display.
- Interactions at 1280×640: File menu opens under its button (right edges equal), HelpTip bubble
  under its "?", Full view covers 1280×640, a 36 000-point list scrolled 2 000 px fills its box with
  rows (no gap), Activity drawer 448 px (560 × 0.8) full height, SLD symbol drag moves exactly with
  the cursor (100, −40 → 100, −40), node box = layout width, zoom buttons, canvas controls at 0.8.
- Settings: Always 100% applies at once, `preferences.json` gets `display_size: 1`, kept after a
  reload; back to Fit gives 80%.
- Frontend tests 460/460, tsc clean, Go `internal/project/...` tests, verify.py (see the ledger).

## Limitations

- The SLD drawing itself is not shrunk (React Flow); everything around it is.
- Wire drawing, the tour ring and the run-time test menu were not driven with a real mouse here
  (synthetic connect also fails at 100%); they use the same conversion as the checked popups.
- Native scrollbars keep their size, so scrolling panes are a few pixels narrower when fitted.
- Edge/Chrome older than 128 keep 100%. Not yet checked in the packaged window on a real 14-inch laptop.
