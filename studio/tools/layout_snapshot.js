/* Layout snapshot for the running app (change 0182, ADR-0111).
 *
 * Paste into the browser console (or run through a browser automation tool). It records the
 * box of every visible element inside .app, in document order, skipping display: contents
 * wrappers and everything inside the SLD drawing (.react-flow__viewport), which keeps its own
 * zoom. Use it to prove that a change leaves a page identical, or that a scaled page is the
 * reference page times the fit factor.
 *
 *   const a = layoutSnapshot();            // { page, viewport, fit, count, hash, boxes }
 *   layoutCompare(a, b)                     // same layout: { same, diffs: [...] }
 *   layoutCompare(reference, scaled, 0.8)   // scaled = reference x 0.8, within 1 px
 */
(() => {
  const round = value => Math.round(value * 10) / 10;
  function layoutSnapshot() {
    const app = document.querySelector('.app');
    const boxes = [];
    if (app) {
      for (const element of app.querySelectorAll('*')) {
        if (element.closest('.react-flow__viewport')) continue;
        const style = getComputedStyle(element);
        if (style.display === 'contents' || style.display === 'none' || style.visibility === 'hidden') continue;
        const box = element.getBoundingClientRect();
        if (!box.width || !box.height) continue;
        const name = typeof element.className === 'string' ? element.className.trim().split(/\s+/)[0] : '';
        boxes.push([element.tagName.toLowerCase() + (name ? '.' + name : ''),
          round(box.left), round(box.top), round(box.width), round(box.height)]);
      }
    }
    let hash = 0;
    for (const char of JSON.stringify(boxes)) hash = (hash * 31 + char.charCodeAt(0)) | 0;
    return { page: document.querySelector('.topbar-title h1')?.textContent ?? '', viewport: `${innerWidth}x${innerHeight}`,
      fit: getComputedStyle(document.documentElement).getPropertyValue('--fit').trim() || '1',
      count: boxes.length, hash, boxes };
  }
  function layoutCompare(reference, other, scale = 1, tolerance = scale === 1 ? 0.1 : 1) {
    const diffs = [];
    if (reference.boxes.length !== other.boxes.length) diffs.push({ count: [reference.boxes.length, other.boxes.length] });
    const length = Math.min(reference.boxes.length, other.boxes.length);
    for (let index = 0; index < length && diffs.length < 20; index += 1) {
      const a = reference.boxes[index], b = other.boxes[index];
      const off = a[0] !== b[0] || [1, 2, 3, 4].some(i => Math.abs(a[i] * scale - b[i]) > tolerance);
      if (off) diffs.push({ index, reference: a, other: b });
    }
    return { same: diffs.length === 0, diffs };
  }
  Object.assign(window, { layoutSnapshot, layoutCompare });
  return 'layoutSnapshot() and layoutCompare(a, b, scale) are ready';
})();
