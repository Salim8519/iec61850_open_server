# ADR-0111: Fit to window: small windows show the big-window layout evenly smaller

Accepted: 2026-10-23. Owner request: "keep same best fit designed app view even on smaller screens,
currently the app runs amazing on my laptop, i want the same for all". Change
[0182](../changes/2026/0182-fit-to-window.md), spec [display-fit](../specs/display-fit.md). Builds on
[ADR-0032](ADR-0032-clean-four-place-ui.md) (fixed design), [ADR-0064](ADR-0064-tool-pages-fill-window.md)
(fill-the-window pages and their fallback) and [ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)
(new backend code in Go only).

## Problem

A 14-inch laptop (1920×1080 at 150%) gives the app window about 1280×640 CSS pixels. Nothing broke,
but the polished layout was squeezed: on Devices › Data points the header wrapped to two rows, the
filter bar to two rows, the subtitle was cut and the table showed 2 full rows. The owner's laptop
(2560×1440 at 150%, the app maximized at about 1707×880) shows the intended view. A compact layout or
folding controls into menus was proposed and rejected: it changes the design.

## Decision

1. **Scale, never rearrange.** Below 1440×800 the whole app is laid out as on a bigger window and
   shown smaller with CSS `zoom` on `<html>` (`--fit`): `min(w/1440, h/800, 1)` rounded to 0.05,
   never below 0.8, and 1 from 0.975 up. Pure function `app/displayFit.ts`; `app/installFit.ts`
   applies it before the first render and on every resize (writes only on change).
2. **Corrections that are no-ops at 100%.** `.app` and every vh/vw are divided by `--fit`; the SLD
   canvas cancels the zoom (`.react-flow`) because React Flow measures nodes with
   getBoundingClientRect, and its panels take it back; Menu, HelpTip, VirtualRows, TestMenu and the
   tour convert screen pixels into their zoom with `ui/zoomSpace.ts` (`currentCSSZoom`).
3. **Breakpoints follow the fitted layout.** Every size `@media` becomes a `data-max` token
   (`w1599 w1440 w1100 w1000 w520 h620`) computed from window ÷ fit, written with `:where()` so
   specificity is unchanged. At fit 1 these are the old thresholds on the same numbers.
4. **A remembered choice.** Settings › Display: *Fit to window (recommended)* / *Always 100%*,
   saved as the Go-only preference `display_size` (0..1, stored only once set, like `sld_sbo_steps`).
5. **No launcher change.** Chromium already remembers a maximized app window, and fit adapts to any
   window, so the Go launcher keeps `--window-size=1440,900` (same as the frozen Python engine).

## Compatibility

- Windows from about 1404×780 up are unchanged: element-by-element layout snapshots of Devices
  (Overview, Data points), SLD, Test and Tools are identical to master at 1920×1000, 1707×880,
  1440×900 and 1600×800; Settings differs only by its new Display section.
- The frozen Python engine answers "Unknown preference" for `display_size`; the Settings control
  reverts and says so. The default preferences file stays byte-identical.
- Browsers without `currentCSSZoom` (Edge/Chrome before 128) keep 100%.
- Limits: the SLD drawing itself is not shrunk; native scrollbars do not scale, so a scrolling pane
  is a few pixels narrower in the fitted layout (a long name may wrap one row lower).
