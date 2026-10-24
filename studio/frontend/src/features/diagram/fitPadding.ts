/** Fit view keeps the drawing clear of the floating SLD toolbar (change 0187). With one padding ratio on
 *  every side, the top row of the drawing (such as the GOOSE IED components that Add all GOOSE IEDs puts
 *  above it, and their GOOSE lines) was fitted under the toolbar and could not be seen. */
export type FitPadding = number | { top: `${number}px`; right: number; bottom: number; left: number };

/** Space between the toolbar's lower edge and the first symbol: room for the GOOSE lane drawn above the
 *  top row (LANE_MARGIN, 36 flow units, at zoom up to 1) and its chips. */
export const TOOLBAR_GAP = 64;

/** Pixels from the canvas top down to the toolbar's lower edge; 0 when there is no toolbar. */
export function toolbarSpace(canvas: Element | null | undefined): number {
  const bar = canvas?.querySelector('.canvas-toolbar');
  if (!canvas || !bar) return 0;
  return Math.max(0, Math.round(bar.getBoundingClientRect().bottom - canvas.getBoundingClientRect().top));
}

/** The ratio on every side, but at least the toolbar plus a gap at the top. */
export function clearOfToolbar(ratio: number, space: number): FitPadding {
  return space > 0 ? { top: `${space + TOOLBAR_GAP}px` as const, right: ratio, bottom: ratio, left: ratio } : ratio;
}

/** clearOfToolbar for the SLD canvas on screen, measured now (for fitView calls after an action). */
export function fitPadding(ratio: number): FitPadding {
  return clearOfToolbar(ratio, toolbarSpace(globalThis.document?.querySelector('.canvas')));
}
