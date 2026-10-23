/** Fit to window (pure, no DOM): how much the whole app is shrunk on a small window, and which
 *  narrow-window rules apply to the shrunk layout. Windows from about 1404×780 up are never touched. */

/** The saved Display setting: 0 = fit to window (default), 1 = always 100%. */
export type DisplaySetting = 0 | 1;

/** The layout the app is designed for; smaller windows show it evenly smaller. */
const DESIGN_WIDTH = 1440;
const DESIGN_HEIGHT = 800;
/** Text never goes below 80% (15 px × 0.8 = 12 px, Windows' own menu text size). */
export const MIN_FIT = 0.8;

/** The zoom for a window of `width` × `height` CSS pixels: 1 or a step of 0.05 from 0.8 to 0.95. */
export function fitScale(width: number, height: number, setting: DisplaySetting): number {
  if (setting === 1 || !(width > 0) || !(height > 0)) return 1;
  const ratio = Math.min(width / DESIGN_WIDTH, height / DESIGN_HEIGHT, 1);
  if (ratio >= 0.975) return 1;
  return Math.max(MIN_FIT, Math.round(ratio * 20) / 20);
}

/** Width limits in the order the CSS uses them (the old @media max-width rules). */
const WIDTH_LIMITS = [1599, 1440, 1100, 1000, 520];
const HEIGHT_LIMITS = [620];

/** data-max tokens for the layout size (window ÷ fit): 'w1100' means the layout is at most 1100 px wide.
 *  At fit 1 these are exactly today's @media thresholds on the same window size. */
export function layoutLimits(width: number, height: number, fit: number): string {
  const layoutWidth = width / fit;
  const layoutHeight = height / fit;
  return [...WIDTH_LIMITS.filter(limit => layoutWidth <= limit).map(limit => `w${limit}`),
    ...HEIGHT_LIMITS.filter(limit => layoutHeight <= limit).map(limit => `h${limit}`)].join(' ');
}
