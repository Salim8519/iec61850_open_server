// Change 0236: the SLD's Fit view button must show the whole drawing, also on a large station.
import assert from 'node:assert/strict';
import test from 'node:test';
// React Flow's own fit computation (the one fitView uses).
import { getViewportForBounds } from '../../frontend/node_modules/@xyflow/system/dist/esm/index.mjs';
import { ZOOM_FLOOR, zoomFloor } from '../../frontend/src/features/diagram/zoomFloor.ts';

// The merged Al Kamil 400 + 132 kV SLD (28 bay controllers) on a 1366 x 768 window's canvas.
const station = { x: -60, y: -1326, width: 7024, height: 5880 };
const canvas = { width: 1000, height: 560 };
const padding = 0.3; // Canvas.tsx fitOptions
const shown = (zoom: number) => station.width * zoom <= canvas.width && station.height * zoom <= canvas.height;

test('the old fixed floor only centres a large station', () => {
  const view = getViewportForBounds(station, canvas.width, canvas.height, ZOOM_FLOOR, 2, padding);
  assert.equal(view.zoom, ZOOM_FLOOR);
  assert.equal(shown(view.zoom), false, 'with minZoom 0.2 the drawing does not fit: the bug');
});

test('the floor follows the drawing, so Fit view shows every symbol', () => {
  const fitted = getViewportForBounds(station, canvas.width, canvas.height, 0, 1, padding).zoom;
  const floor = zoomFloor(fitted);
  const view = getViewportForBounds(station, canvas.width, canvas.height, floor, 2, padding);
  assert.ok(floor < ZOOM_FLOOR);
  assert.ok(shown(view.zoom), `zoom ${view.zoom} must fit ${station.width} x ${station.height}`);
});

test('a small drawing keeps the usual floor', () => {
  assert.equal(zoomFloor(0.8), ZOOM_FLOOR);
  assert.equal(zoomFloor(Number.NaN), ZOOM_FLOOR);
  assert.equal(zoomFloor(0), ZOOM_FLOOR);
});
