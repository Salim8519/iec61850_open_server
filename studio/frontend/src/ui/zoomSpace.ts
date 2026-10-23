/** Fit to window (CSS zoom on <html>): screen pixels from getBoundingClientRect, clientX/Y and
 *  innerWidth/innerHeight are divided by an element's effective zoom to get the CSS pixels it is
 *  placed in. At zoom 1 (and in browsers without currentCSSZoom) every value is returned unchanged. */
export type ZoomBox = { left: number; top: number; right: number; bottom: number; width: number; height: number };

/** The effective CSS zoom of an element (its own and its parents'); 1 when unknown. */
export function zoomOf(element?: Element | null): number {
  return (element ?? document.documentElement).currentCSSZoom ?? 1;
}

/** A screen rectangle in the zoom space `zoom`. */
export function boxIn(rect: ZoomBox, zoom: number): ZoomBox {
  return { left: rect.left / zoom, top: rect.top / zoom, right: rect.right / zoom, bottom: rect.bottom / zoom,
    width: rect.width / zoom, height: rect.height / zoom };
}

/** A screen point (clientX/clientY) in the zoom space `zoom`. */
export const pointIn = (x: number, y: number, zoom: number) => ({ x: x / zoom, y: y / zoom });

/** The window's size in the zoom space `zoom` (CSS zoom does not change innerWidth/innerHeight). */
export const viewIn = (zoom: number) => ({ width: window.innerWidth / zoom, height: window.innerHeight / zoom });
