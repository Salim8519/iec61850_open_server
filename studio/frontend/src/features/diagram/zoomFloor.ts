/** The SLD's smallest zoom (change 0236). A fixed floor of 0.2 stopped "Fit view" from zooming out on a
 *  large station: React Flow clamps the fitted zoom to minZoom and then only centres the drawing (the
 *  merged Al Kamil 400 + 132 kV SLD, 7024 x 5880, needs about 0.04 on a laptop screen). The floor is 0.2,
 *  or lower when the whole drawing needs it, so the fit button always shows every symbol. */
export const ZOOM_FLOOR = 0.2;

/** The lowest zoom this canvas may reach: ZOOM_FLOOR, or the zoom that fits the drawing when that is
 *  smaller (a little under it, so the padding still fits after rounding). */
export function zoomFloor(fitted: number): number {
  return Number.isFinite(fitted) && fitted > 0 && fitted < ZOOM_FLOOR ? fitted * 0.9 : ZOOM_FLOOR;
}
