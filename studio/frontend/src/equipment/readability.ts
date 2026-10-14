/* Distance readability of the canvas (ADR-0045). Display only; never simulation input. */

export type ZoomTier = 'near' | 'mid' | 'far';

/** Zoom band used to thicken symbol strokes so state stays visible when zoomed out. */
export function zoomTier(zoom: number): ZoomTier {
  if (zoom >= 0.85) return 'near';
  return zoom >= 0.5 ? 'mid' : 'far';
}

const CONFIRMED = ['Open', 'Closed', 'Intermediate', 'Invalid'];

/** True when a confirmed switch position changed to another confirmed position.
    The first value after load, run start or a lost value is not a change. */
export function confirmedChange(previous: string, next: string): boolean {
  return previous !== next && CONFIRMED.includes(previous) && CONFIRMED.includes(next);
}
