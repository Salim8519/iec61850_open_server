// Change 0187: Fit view keeps the drawing below the floating SLD toolbar.
import assert from 'node:assert/strict';
import test from 'node:test';
import { clearOfToolbar, TOOLBAR_GAP, toolbarSpace } from '../../frontend/src/features/diagram/fitPadding.ts';

const box = (top: number, bottom: number) => ({ getBoundingClientRect: () => ({ top, bottom }) });
const canvas = (bar: ReturnType<typeof box> | null) =>
  ({ ...box(64, 900), querySelector: () => bar } as unknown as Element);

test('the top padding is the toolbar plus a gap; the other sides keep the ratio', () => {
  assert.equal(toolbarSpace(canvas(box(80, 182))), 118);
  assert.deepEqual(clearOfToolbar(0.3, 118), { top: `${118 + TOOLBAR_GAP}px`, right: 0.3, bottom: 0.3, left: 0.3 });
});

test('without a toolbar the plain ratio is kept', () => {
  assert.equal(toolbarSpace(canvas(null)), 0);
  assert.equal(toolbarSpace(null), 0);
  assert.equal(clearOfToolbar(0.2, 0), 0.2);
});

test('a new card never lands in the toolbar strip at the top of the view', async () => {
  const { freeSpot, NEW_SIZE } = await import('../../frontend/src/equipment/placeNew.ts');
  const strip = { x: -400, y: -100, width: 1600, height: 180 };
  const spot = freeSpot([strip], { x: 200, y: 40 });
  assert.ok(spot.y >= strip.y + strip.height || spot.y + NEW_SIZE.height <= strip.y, `spot ${JSON.stringify(spot)}`);
});
