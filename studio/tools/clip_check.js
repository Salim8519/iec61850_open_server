/* Clipped-text check for the running app (change 0097).
 *
 * Paste into the browser console (or run through a browser automation tool) on any page,
 * with the window at 1366x768. It returns { ok, checked, clipped: [...] } and fails (ok=false)
 * when a pill, chip, tag or label is cut: its content is wider than its box (scrollWidth >
 * clientWidth). Record the result in the change record.
 *
 * Run it on Devices, on the SLD with both signal-layer switches on, and on the tool pages.
 */
(() => {
  const SELECTORS = [
    '.status-pill', '.pill', '.chip', '.partner-chip', '.ied-tag', '.ied-tag-name', '.list-pills > *',
    '.layer-switch', '.run-state', 'label', '.tab', '.segmented button', '.outside-senders button',
  ];
  const clipped = [];
  let checked = 0;
  for (const element of document.querySelectorAll(SELECTORS.join(','))) {
    const box = element.getBoundingClientRect();
    if (!box.width || !box.height || getComputedStyle(element).visibility === 'hidden') continue;
    checked += 1;
    if (element.scrollWidth > element.clientWidth + 1) {
      clipped.push({
        text: (element.textContent || '').trim().slice(0, 60),
        className: String(element.className).slice(0, 60),
        scrollWidth: element.scrollWidth,
        clientWidth: element.clientWidth,
      });
    }
  }
  return { ok: clipped.length === 0, page: location.hash || location.pathname,
    viewport: `${innerWidth}x${innerHeight}`, checked, clipped };
})();
