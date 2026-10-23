// Words of Settings › Display (fit to window).
export const DISPLAY_COPY = {
  section: 'Display',
  heading: 'Screen size',
  label: 'How big the app is drawn',
  fit: 'Fit to window (recommended)',
  fitTitle: 'On a small window everything is drawn evenly smaller, so the layout looks the same as on a big screen',
  full: 'Always 100%',
  fullTitle: 'Always draw at full size; on a small window some panels stack or need scrolling',
  hint: 'On small screens ScadaBench shrinks everything evenly, so you see the same layout as on a big screen. Text never goes below 80%.',
  oldBrowser: 'Fit to window needs a newer Edge or Chrome (version 128 or later). The app is shown at 100%.',
  notSaved: 'This choice could not be saved by the running backend, so it was put back. Restart ScadaBench and try again.',
};

/** The live line under the choice, e.g. "Now 80% · this window is 1280 × 640". */
export const displayNow = (fit: number, width: number, height: number) =>
  `Now ${Math.round(fit * 100)}% · this window is ${Math.round(width)} × ${Math.round(height)}`;
