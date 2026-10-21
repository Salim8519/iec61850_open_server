import assert from 'node:assert/strict';
import test from 'node:test';
import { coalesce } from '../../frontend/src/state/coalesce.ts';
import { sameRange, virtualRange } from '../../frontend/src/ui/virtualRange.ts';
import { stats, wantsPerf } from '../../frontend/src/app/perf.ts';
import { deliverLive, RELOADS } from '../../frontend/src/state/eventStream.ts';
import { createFrameBatch } from '../../frontend/src/state/frameBatch.ts';
import { createValueStore } from '../../frontend/src/state/valueStore.ts';
import { createEventLogStore } from '../../frontend/src/state/eventLogStore.ts';
import { fakeFrames, settle } from './fakeClock.ts';

test('refresh coalescing: seven asks in the first second send GET /project twice', async () => {
  const resolvers: (() => void)[] = [];
  let started = 0;
  const refresh = coalesce(() => { started += 1; return new Promise<void>(resolve => resolvers.push(resolve)); });
  const first = refresh();  // the stream opened
  const later = Array.from({ length: 6 }, () => refresh());  // replayed runtime, live_status, project_changed...
  assert.equal(started, 1, 'one in flight');
  resolvers[0](); await first; await settle();
  assert.equal(started, 2, 'one trailing run for all six');
  let done = 0;
  later.forEach(promise => promise.then(() => { done += 1; }));
  await settle();
  assert.equal(done, 0, 'callers wait for a run that started after their call');
  resolvers[1](); await Promise.all(later);
  assert.equal(done, 6);
  assert.equal(started, 2);
});

test('refresh coalescing: a failed run still lets the trailing one start', async () => {
  let calls = 0;
  const refresh = coalesce(async () => { calls += 1; if (calls === 1) throw new Error('offline'); });
  const first = refresh();
  const second = refresh();
  await assert.rejects(first, /offline/);
  await second;
  assert.equal(calls, 2);
});

test('virtual range: rows on screen plus overscan, spacers keep the full height', () => {
  const range = virtualRange({ count: 500, rowHeight: 30, scrollTop: 3000, viewport: 600, overscan: 5 });
  assert.equal(range.start, 95);
  assert.equal(range.end, 125, 'rows 100-119 are on screen, five more below');
  assert.equal(range.before, 95 * 30);
  assert.equal(range.before + (range.end - range.start) * 30 + range.after, 500 * 30);
  const top = virtualRange({ count: 3, rowHeight: 30, scrollTop: 0, viewport: 600 });
  assert.deepEqual(top, { start: 0, end: 3, before: 0, after: 0 });
  assert.deepEqual(virtualRange({ count: 0, rowHeight: 30, scrollTop: 50, viewport: 600 }), { start: 0, end: 0, before: 0, after: 0 });
  assert.ok(sameRange(range, { ...range }));
});

test('virtual range: measured heights of wrapped rows are used', () => {
  const heights = new Map([[0, 100], [1, 100]]);
  const range = virtualRange({ count: 100, rowHeight: 20, scrollTop: 210, viewport: 100, overscan: 0,
    heightOf: index => heights.get(index) });
  assert.equal(range.start, 2, 'rows 0 and 1 are 200 px tall together');
  assert.equal(range.before, 200);
  assert.equal(range.end, 8);
  assert.equal(range.after, (100 - 8) * 20);
  const end = virtualRange({ count: 10, rowHeight: 20, scrollTop: 10000, viewport: 100, overscan: 0 });
  assert.equal(end.end, 10, 'scrolled past the end still shows the last rows');
});

test('#perf: percentiles and the address switch', () => {
  assert.deepEqual(stats([]), { n: 0, p50: null, p95: null, max: null });
  const samples = Array.from({ length: 20 }, (_, index) => index + 1);
  assert.deepEqual(stats(samples), { n: 20, p50: 10, p95: 19, max: 20 });
  assert.ok(wantsPerf('#perf'));
  assert.ok(wantsPerf('#session=abc&perf'));
  assert.ok(!wantsPerf('#session=perfect'));
  assert.ok(!wantsPerf(''));
});

test('live events go to the stores, never to React state', () => {
  const frames = fakeFrames();
  const batch = createFrameBatch(frames.clock);
  const values = createValueStore(batch);
  const log = createEventLogStore(batch, { formatTime: () => 't' });
  const layers: string[] = [];
  const equipment: [string, unknown][] = [];
  const energized: string[] = [];
  const targets = { values, log, layers: { handle: (event: { kind: string }) => { layers.push(event.kind); } },
    equipment: { set: (key: string, value: unknown) => { equipment.push([key, value]); } },
    energize: (event: { kind: string }) => { energized.push(event.kind); } };
  deliverLive({ id: 1, kind: 'signal', timestamp: 1, ied_id: 'I', ref: 'LD/X.mag.f', value: 5 }, targets, 0);
  deliverLive({ id: 2, kind: 'mms_access', timestamp: 1, ied_id: 'I', clients: [] }, targets, 0);
  deliverLive({ id: 3, kind: 'equipment_state', timestamp: 1, node_id: 'QA1', value: 'closed' }, targets, 0);
  deliverLive({ id: 4, kind: 'energization', timestamp: 1, nodes: {}, edges: {} }, targets, 0);
  frames.frame();
  assert.equal(values.get('I:LD/X.mag.f'), 5);
  assert.equal(log.size(), 3, 'per-second MMS client summaries stay out of Activity');
  assert.deepEqual(layers, ['signal', 'mms_access', 'equipment_state', 'energization']);
  assert.deepEqual(equipment, [['QA1', 'closed']]);
  assert.deepEqual(energized, ['energization']);
  assert.ok(RELOADS.has('runtime') && RELOADS.has('live_status') && !RELOADS.has('signal'));
});
