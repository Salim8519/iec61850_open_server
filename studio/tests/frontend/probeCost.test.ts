import assert from 'node:assert/strict';
import test from 'node:test';
import { RANGES_COPY } from '../../frontend/src/copy/master.ts';
import { probeCost } from '../../frontend/src/features/tools/gateway/master/probeCost.ts';

// Change 0089: holding registers 0-5010 on a sparse gateway took more requests than the page's
// "up to about 5052" and minutes, not seconds. The estimate now mirrors the worker's split rule.
test('Find registers estimates the worst case with the worker split rule', () => {
  assert.deepEqual(probeCost('holding_registers', 0, 124), { fewest: 1, most: 140 });
  const sweep = probeCost('holding_registers', 0, 5010);
  assert.equal(sweep.fewest, 41);
  assert.ok(sweep.most > 5052 && sweep.most < 5700, String(sweep.most));
  assert.deepEqual(probeCost('coils', 0, 0), { fewest: 1, most: 1 });
});

test('the preview states minutes for long probes', () => {
  const sweep = probeCost('holding_registers', 0, 5010);
  const text = RANGES_COPY.preview(sweep.fewest, sweep.most);
  assert.match(text, /^At least 41 requests \(about 4 s\); up to \d+ \(about 9 min\)/);
});
