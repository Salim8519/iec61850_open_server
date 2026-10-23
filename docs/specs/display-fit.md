# Spec: Fit to window

Decision: [ADR-0111](../decisions/ADR-0111-fit-to-window.md).
Change: [0182](../changes/2026/0182-fit-to-window.md).
Owner request (2026-10-23): keep the same best-fit app view on smaller screens such as 14-inch
laptops; normal screens must look exactly as before.

## Behaviour contract

- The app is designed for a page of at least 1440×800 CSS pixels. A smaller window shows that
  layout evenly smaller (CSS `zoom` on `<html>`): nothing moves, folds or is hidden.
- The fit is `min(width/1440, height/800, 1)` rounded to 0.05, never below 0.8 (15 px text becomes
  12 px, the size of Windows' own menu text). From 0.975 up it is 1: windows from about 1404×780 up
  are untouched. 1280×640 → 80%, 1366×690 → 85%, 1536×744 → 95%.
- Settings › Display: *Fit to window (recommended)* or *Always 100%*, with the zoom in use and the
  window size. The choice is the Go-only preference `display_size` (0 fit, 1 always 100%).
- Narrow-window rules (sidebar icons, stacked panes, gateway columns) follow the fitted layout size,
  through `data-max` tokens on `<html>` instead of `@media`; at 100% they are the old thresholds.
- The SLD drawing keeps its own zoom (React Flow cannot be measured under CSS zoom); everything
  around it is fitted. Popups placed by pixel positions convert screen pixels into their zoom.
- Browsers without `Element.currentCSSZoom` (Edge/Chrome older than 128) stay at 100%.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| FIT-01 | The fit is 1 from about 1404×780 up, 0.05 steps below, never under 0.8; Always 100% is 1. | `studio/tests/frontend/displayFit.test.ts::fitScale: 0.05 steps, never below 0.8, 1 from about 1404×780 up` |
| FIT-02 | Narrow-window rules use the fitted layout size with the old thresholds; an unmeasured window gets none. | `studio/tests/frontend/displayFit.test.ts::layoutLimits: the old @media thresholds on the fitted layout size`, `studio/tests/frontend/displayFit.test.ts::no size @media is left: narrow-window rules use the data-max tokens from installFit` |
| FIT-03 | Every vh/vw is divided by the fit, so a fitted page still fills the window. | `studio/tests/frontend/displayFit.test.ts::every vh/vw in the CSS is divided by the fit, so the fitted page still fills the window` |
| FIT-04 | The SLD canvas cancels the page zoom; its panels scale back with the page. | `studio/tests/frontend/displayFit.test.ts::the SLD canvas cancels the page zoom and its panels scale back` |
| FIT-05 | Fit is applied before the first render; popups placed by pixels convert into their zoom space. | `studio/tests/frontend/displayFit.test.ts::main.tsx installs fit before the first render`, `studio/tests/frontend/displayFit.test.ts::popups placed by pixels convert into their zoom space; no raw window size` |
| FIT-06 | `display_size` is Go-only, 0..1, stored only once set (the default file is unchanged). | `studio/engine/internal/project/preferences_display_test.go::TestDisplaySizePreference`, `studio/engine/internal/project/preferences_display_test.go::TestDisplaySizeKeepsDefaultFile` |

## Evidence still required

The packaged window on a real 14-inch laptop (1920×1080 at 150%); until then the evidence is the
dev browser at emulated window sizes (change 0182).
