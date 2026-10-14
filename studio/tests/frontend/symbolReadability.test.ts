import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { confirmedChange, zoomTier } from '../../frontend/src/equipment/readability.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('zoom bands: near at normal size, thicker strokes as the user zooms out', () => {
  assert.equal(zoomTier(2), 'near');
  assert.equal(zoomTier(0.85), 'near');
  assert.equal(zoomTier(0.84), 'mid');
  assert.equal(zoomTier(0.5), 'mid');
  assert.equal(zoomTier(0.49), 'far');
  assert.equal(zoomTier(0.2), 'far');
  const css = read('equipment/equipment-node.css');
  const stroke = (band: string) => parseFloat(css.match(new RegExp(`zoom-${band} \\.equipment-symbol \\{ --sym-stroke: ([\\d.]+)px`))![1]);
  assert.ok(stroke('far') > stroke('mid') && stroke('mid') > 3);
});

test('only a confirmed position change pulses, never the first value', () => {
  assert.ok(confirmedChange('Open', 'Closed'));
  assert.ok(confirmedChange('Closed', 'Intermediate'));
  assert.ok(confirmedChange('Closed', 'Invalid'));
  assert.ok(!confirmedChange('Closed', 'Closed'));
  assert.ok(!confirmedChange('Awaiting value', 'Closed'), 'run start or load');
  assert.ok(!confirmedChange('Mapping required', 'Open'));
  assert.ok(!confirmedChange('Closed', 'Awaiting value'), 'value lost');
});

test('switching geometry keeps a wide swing, a broken open conductor and reduced motion', () => {
  const glyphs = read('equipment/symbols/SwitchingGlyphs.tsx');
  const swing = glyphs.match(/Closed: 0, Intermediate: (\d+), Open: (\d+)/)!;
  assert.ok(Number(swing[2]) >= 40 && Number(swing[1]) < Number(swing[2]));
  assert.match(glyphs, /Closed: 1, Intermediate: [\d.]+, Open: 0 }/, 'open breaker bridge disappears');
  const css = read('equipment/equipment-node.css');
  assert.match(css, /prefers-reduced-motion: reduce\) \{\s*\.equipment-node\[data-pulse="a"\] \{ animation: change-ring-a/);
  assert.match(read('equipment/symbols/switch-motion.css'), /prefers-reduced-motion: reduce/);
});
