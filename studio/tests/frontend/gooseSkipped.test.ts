import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GooseWatch } from '../../frontend/src/api/network.ts';
import type { Ied } from '../../frontend/src/api/types.ts';
import { noListener, publisherRole, skippedGoose } from '../../frontend/src/features/network/gooseSkipped.ts';

const control = (gocb_ref: string) => ({ name: 'GoCB01', gocb_ref, go_id: '', dataset_ref: '', app_id: 0, mac: '',
  vlan_id: 0, vlan_priority: 4, conf_rev: 1 });
const ied = (name: string, role: Ied['role'], publishes: string[], subscribes: string[] = []) => ({ name, role,
  goose: { publishes: publishes.map(control), subscribes: subscribes.map(ref => ({ ...control(ref), publisher: '', inputs: [] })),
    diagnostics: [] } }) as unknown as Ied;
const watch = (gocb_ref: string, publisher: GooseWatch['publisher']) => ({ key: gocb_ref, gocb_ref, app_id: 0, mac: '',
  go_id: '', dataset_ref: '', conf_rev: 1, state: 'receiving', problem: '', frames: 1, publisher, subscribers: [] }) as GooseWatch;

const C11 = 'C11Master/LLN0$GO$GoCB01';
const C19 = 'C19Master/LLN0$GO$GoCB01';
const ieds = [ied('C11', 'simulated', [C11]), ied('C19', 'off', [C19], [C11]), ied('C30', 'simulated', [], [C19])];

test('a skipped IED is named as skipped, never as outside the project', () => {
  const skipped = skippedGoose(ieds);
  assert.equal(publisherRole(watch(C19, { ied_id: null, name: 'C19', role: 'outside_project' }), skipped), 'skipped in this project');
  assert.equal(publisherRole(watch(C19, { ied_id: null, name: 'C06', role: 'outside_project' }), skipped), 'not in project');
  assert.equal(publisherRole(watch(C11, { ied_id: 'a', name: 'C11', role: 'simulated' }), skipped), 'simulated');
  assert.equal(publisherRole(watch(C11, { ied_id: 'a', name: 'C11', role: 'real' }), skipped), 'real device');
});

test('a block only skipped IEDs listen to names them instead of "nobody"', () => {
  const skipped = skippedGoose(ieds);
  assert.equal(noListener(watch(C11, undefined), skipped), 'C19 (skipped, so not listening)');
  assert.equal(noListener(watch('Other/LLN0$GO$GoCB01', undefined), skipped), 'Nobody in project');
  assert.equal(skipped.publishers.get(C19), 'C19');
});

test('the traffic list uses these words and splits GOOSE from skipped IEDs off the unknown list', () => {
  const source = readFileSync(new URL('../../frontend/src/features/network/GooseTraffic.tsx', import.meta.url), 'utf8');
  assert.match(source, /publisherRole\(watch, skipped\)/);
  assert.match(source, /noListener\(watch, skipped\)/);
  assert.match(source, /<Hint id="goose\.fromSkipped" \/>/);
  assert.doesNotMatch(source, /'not in project'/);
  const tool = readFileSync(new URL('../../frontend/src/features/tools/TrafficTool.tsx', import.meta.url), 'utf8');
  assert.match(tool, /ieds=\{project\.ieds\}/);
});
