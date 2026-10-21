// Change 0106: stopped, partly running and fully running never look alike.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { Ied } from '../../frontend/src/api/types.ts';
import { nameList, RUN_MARK, RUN_SHAPE_COPY, runSummary } from '../../frontend/src/copy/runShape.ts';
import { runGroups } from '../../frontend/src/features/devices/runGroups.ts';
import { placeActivity } from '../../frontend/src/app/running.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const ied = (id: string, role = 'simulated') => ({ id, name: `C${id}`, role } as unknown as Ied);
const ieds = [ied('1'), ied('2'), ied('3'), ied('4', 'off')];
const project = (runtime: string, running?: string[]) => ({ ieds, runtime, running_ieds: running });

test('the run has three shapes and left-out IEDs never count', () => {
  assert.equal(runSummary(project('stopped')).shape, 'stopped');
  assert.equal(runSummary(project('running', ['1'])).shape, 'partial');
  assert.equal(runSummary(project('running', ['1', '2', '3'])).shape, 'all');
  assert.equal(runSummary(project('paused', ['1'])).total, 3);
  assert.equal(runSummary(null).shape, 'stopped');
  assert.deepEqual(Object.values(RUN_MARK), ['○', '◐', '●'], 'a shape for each state, not colour alone');
});

test('names are full and lists say how many more', () => {
  assert.equal(nameList([ied('1')]), 'C1');
  assert.equal(nameList(ieds, 2), 'C1, C2 and 2 more');
  const words = RUN_SHAPE_COPY.devices(runSummary(project('running', ['1'])));
  assert.equal(words, '1 of 3 running: C1. The other 2 are stopped.');
  assert.match(RUN_SHAPE_COPY.devices(runSummary(project('running', ['1', '2']))), /The other one is stopped\./);
});

test('Devices groups running IEDs first only during a partial run', () => {
  const partial = runGroups(ieds, runSummary(project('running', ['3'])));
  assert.deepEqual(partial.map(group => [group.label, group.ieds.map(item => item.id)]),
    [['Running · 1', ['3']], ['Not running · 3', ['1', '2', '4']]]);
  assert.deepEqual(runGroups(ieds, runSummary(project('running', ['1', '2', '3']))), [{ label: null, ieds }]);
  assert.deepEqual(runGroups(ieds, runSummary(project('stopped'))), [{ label: null, ieds }]);
});

test('sidebar dots are half filled during a partial run', () => {
  const partial = placeActivity(project('running', ['1']) as never, null);
  assert.equal(partial.diagram?.partial, true);
  assert.equal(partial.diagram?.labels[0], 'Live diagram: 1 of 3 IEDs running');
  assert.equal(placeActivity(project('running', ['1', '2', '3']) as never, null).diagram?.partial, undefined);
  assert.match(read('styles/layout.css'), /\.live-dot\.partial \{/);
});

test('top bar is green only when all run; SLD greys equipment of stopped IEDs in words', () => {
  const bar = read('app/TopBar.tsx');
  assert.match(bar, /shape === 'all' \? 'running' : 'partial'/);
  assert.match(read('features/diagram/RunStrip.tsx'), /Run all/);
  assert.match(read('features/devices/DeviceRunSummary.tsx'), /runRest/);
  assert.match(read('styles/layout.css'), /\.run-state\.shape-partial \{[^}]*dashed/);
  const node = read('equipment/EquipmentNode.tsx');
  assert.match(node, /RUN_SHAPE_COPY\.equipmentStopped/);
  assert.match(read('equipment/equipment-node.css'), /\.equipment-node\.ied-stopped \{[^}]*dashed/);
  assert.match(read('features/diagram/Canvas.tsx'), /lockNote \?\? 'Running · drawing is locked'/);
});
