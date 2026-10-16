import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import { TOUR_STEPS } from '../../frontend/src/copy/tour.ts';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap(name => {
    const path = join(dir, name);
    return statSync(path).isDirectory() ? sources(path) : path.endsWith('.tsx') ? [readFileSync(path, 'utf8')] : [];
  });
}
const all = sources(SRC).join('\n');

test('every tour step points at an element that exists in the app', () => {
  for (const step of TOUR_STEPS) {
    if (!step.target) continue;
    const literal = all.includes(`data-tour="${step.target}"`);
    const place = step.target.startsWith('place-') && all.includes('data-tour={`place-${id}`}');
    const menu = all.includes(`triggerTour="${step.target}"`);
    assert.ok(literal || place || menu, `missing data-tour target ${step.target}`);
  }
});

test('tour wording is plain and complete', () => {
  assert.ok(TOUR_STEPS.length >= 5 && TOUR_STEPS.length <= 8);
  for (const step of TOUR_STEPS) {
    assert.ok(step.title.length > 3 && step.title.length < 50, step.id);
    assert.ok(step.text.length > 40 && step.text.length < 220, step.id);
  }
  assert.equal(TOUR_STEPS[0].target, undefined);
});

test('the tour opens from preferences and can always be reopened from Help', () => {
  const app = readFileSync(join(SRC, 'app/App.tsx'), 'utf8');
  const sidebar = readFileSync(join(SRC, 'app/Sidebar.tsx'), 'utf8');
  assert.match(app, /prefs\.tour_seen_version < prefs\.tour_version/);
  assert.match(app, /markTourSeen/);
  assert.match(sidebar, /onSelect: openTour/);
});
