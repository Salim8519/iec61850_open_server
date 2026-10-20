/** Opt-in speed marks for the owner and the speed lanes: open the app with `#perf` (speed lane F).
 *
 * Click-to-paint: an Open/Close press on the SLD (DiagramPage.operate) to the frame after the
 * symbol shows the new position (EquipmentNode, two animation frames). Long tasks and long
 * animation frames are collected while #perf is on. `window.__studioPerf.report()` gives p50/p95.
 * Without #perf every function here returns at once and nothing is observed. */
type Stats = { n: number; p50: number | null; p95: number | null; max: number | null };
type Report = { clickToPaint: Stats; longTasks: Stats & { totalMs: number }; longFrames: Stats & { totalMs: number };
  requests: Record<string, number> };

let enabled = false;
const pending = new Map<string, number>();
const clicks: number[] = [];
const longTasks: number[] = [];
const longFrames: number[] = [];

/** Nearest-rank percentiles of `values` (ms, rounded). */
export function stats(values: number[]): Stats {
  if (!values.length) return { n: 0, p50: null, p95: null, max: null };
  const sorted = [...values].sort((a, b) => a - b);
  const rank = (p: number) => sorted[Math.min(sorted.length - 1, Math.max(0, Math.ceil(p * sorted.length) - 1))];
  return { n: sorted.length, p50: Math.round(rank(0.5)), p95: Math.round(rank(0.95)), max: Math.round(sorted[sorted.length - 1]) };
}

/** True when the address asks for #perf (alone or next to other fragment parts). */
export function wantsPerf(hash: string) {
  return hash.replace(/^#/, '').split('&').some(part => part === 'perf' || part === 'perf=1');
}

function observe(type: string, into: number[]) {
  try {
    new PerformanceObserver(list => list.getEntries().forEach(entry => into.push(entry.duration)))
      .observe({ type, buffered: true });
  } catch { /* this browser does not report that entry type */ }
}

function requests(): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const entry of performance.getEntriesByType('resource')) {
    const path = new URL(entry.name).pathname;
    if (path.startsWith('/api/')) counts[path] = (counts[path] ?? 0) + 1;
  }
  return counts;
}

export function report(): Report {
  const total = (values: number[]) => Math.round(values.reduce((sum, value) => sum + value, 0));
  return { clickToPaint: stats(clicks), longTasks: { ...stats(longTasks), totalMs: total(longTasks) },
    longFrames: { ...stats(longFrames), totalMs: total(longFrames) }, requests: requests() };
}

/** Called once before the app renders; reads the address before the session step clears it. */
export function initPerf() {
  let on = wantsPerf(window.location.hash);
  try {
    if (on) sessionStorage.setItem('studio-perf', '1');
    else on = sessionStorage.getItem('studio-perf') === '1';
  } catch { /* storage blocked: #perf lasts for this page only */ }
  if (!on || enabled) return;
  enabled = true;
  performance.setResourceTimingBufferSize(20000);
  observe('longtask', longTasks);
  observe('long-animation-frame', longFrames);
  const reset = () => { clicks.length = 0; longTasks.length = 0; longFrames.length = 0; performance.clearResourceTimings(); };
  Object.assign(window, { __studioPerf: { report, reset, clicks, longTasks, longFrames } });
}

/** An operate command was pressed for `nodeId`. */
export function markClick(nodeId: string) {
  if (enabled) pending.set(nodeId, performance.now());
}

/** `nodeId` shows a new position; the sample ends when that frame is painted. */
export function markPaint(nodeId: string) {
  if (!enabled) return;
  const start = pending.get(nodeId);
  if (start === undefined) return;
  pending.delete(nodeId);
  requestAnimationFrame(() => requestAnimationFrame(() => clicks.push(performance.now() - start)));
}
