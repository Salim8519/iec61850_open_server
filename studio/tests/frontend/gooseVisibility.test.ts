// Change 0102: GOOSE between two machines is visible on the SLD and described truthfully.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GooseLink } from '../../frontend/src/api/signalLayers.ts';
import { linkStatus } from '../../frontend/src/copy/signalLayers.ts';
import { edgePoint, ENTRY_DISTANCE, tagScale } from '../../frontend/src/equipment/signalLayout.ts';
import { receivedLabel } from '../../frontend/src/equipment/signalPulses.ts';
import { endpointOf, onLabNetwork } from '../../frontend/src/features/network/endpoints.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const lab = { mode: 'lab' as const, adapter_guid: 'G', adapter_label: 'VMnet8', nominal_hz: 50 };

test('an outside sender starts beside its receiver at every zoom, never off the drawing edge', () => {
  const tag = { x: 5000, y: 3000 };
  for (const zoom of [0.2, 0.5, 1, 2]) {
    const scale = tagScale(zoom);
    const from = edgePoint(tag, scale);
    // On screen the dot travels 130-440 px whatever the zoom (scale grows as zoom shrinks).
    const screen = (tag.x - from.x) * zoom;
    assert.ok(screen >= 130 && screen <= 2 * ENTRY_DISTANCE, `zoom ${zoom}: ${screen} px`);
  }
});

test('dot, line and words keep their screen size; the receiver tag names the sender for ~2 s', () => {
  const overlay = read('features/diagram/signals/SignalOverlay.tsx');
  assert.match(overlay, /radius=\{5 \* scale\}/);
  assert.match(overlay, /--signal-scale/);
  assert.match(overlay, /received=\{received &&/);
  assert.match(read('features/diagram/signals/signal-layers.css'), /font-size: calc\(11px \* var\(--signal-scale, 1\)\)/);
  assert.match(read('features/diagram/signals/IedTag.tsx'), /ied-tag-received/);
  assert.equal(receivedLabel('FEED1', 'stNum 8', 'observed'), '← FEED1 · stNum 8');
  assert.equal(receivedLabel('FEED1', '', 'expected'), '← FEED1 · new state (expected)');
});

test('a lab-mode IED without a lab address is not called reachable from the adapter', () => {
  const added = { ip: '127.50.0.3', port: 102, lab_ip: '' };
  assert.equal(onLabNetwork(added, lab), false);
  assert.equal(endpointOf(added, lab).host, '127.50.0.3');
  assert.equal(onLabNetwork({ lab_ip: '192.168.98.28' }, lab), true);
  assert.match(read('app/TopBar.tsx'), /onLabNetwork/);
  assert.match(read('features/devices/DeviceOverview.tsx'), /no address on the lab network yet/);
});

test('a receiving sender outside the project shows its live state, a silent one says Outside project', () => {
  const link: GooseLink = { id: 'L', gocb_ref: 'FEED1LD1/LLN0$GO$Feed1SwiPosCtl', publisher: 'FEED1', subscriber: 'BUS1',
    block: 'Feed1SwiPosCtl', status: 'declared', remote: false, match: 'named', inputs: [],
    live: { state: 'receiving', problem: '', st_num: 41, sq_num: 3, frames: 56 } } as unknown as GooseLink;
  assert.equal(linkStatus(link, true, true, true).label, 'Receiving');
  assert.match(linkStatus(link, true, true, true).hint, /FEED1 is not in this project/);
  const waiting = { ...link, live: { ...link.live!, state: 'waiting' } } as GooseLink;
  assert.equal(linkStatus(waiting, true, true, true).label, 'Outside project');
  assert.equal(linkStatus(link, false, true, true).label, 'Outside project', 'stopped: nothing live to show');
});

test('equipment names and states wrap instead of being cut', () => {
  const css = read('equipment/equipment-node.css');
  const rule = (name: string) => css.slice(css.indexOf(`.${name} {`), css.indexOf('}', css.indexOf(`.${name} {`)));
  for (const name of ['equipment-label', 'equipment-state']) {
    assert.doesNotMatch(rule(name), /ellipsis|nowrap/);
    assert.match(rule(name), /overflow-wrap: anywhere/);
  }
});
